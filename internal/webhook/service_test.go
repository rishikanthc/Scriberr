package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"scriberr/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSubscribesTo(t *testing.T) {
	tests := []struct {
		name       string
		eventsJSON string
		event      Event
		want       bool
	}{
		{name: "subscribed", eventsJSON: `["recording.uploaded","transcription.completed"]`, event: EventTranscriptionSuccess, want: true},
		{name: "not subscribed", eventsJSON: `["recording.uploaded"]`, event: EventTranscriptionSuccess, want: false},
		{name: "invalid configuration", eventsJSON: `not-json`, event: EventTranscriptionSuccess, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, subscribesTo(tt.eventsJSON, tt.event))
		})
	}
}

func newConfiguredWebhookTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:webhook-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Webhook{}, &models.WebhookDelivery{}))
	service := NewService()
	service.db = db
	service.retryDelays = []time.Duration{0, 10 * time.Millisecond, 10 * time.Millisecond}
	return service, db
}

func waitForDelivery(t *testing.T, db *gorm.DB, status string) models.WebhookDelivery {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var delivery models.WebhookDelivery
		if err := db.Order("created_at DESC").First(&delivery).Error; err == nil && delivery.Status == status {
			return delivery
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("webhook delivery did not reach %q", status)
	return models.WebhookDelivery{}
}

func TestConfiguredWebhookRetriesAndRecordsHistory(t *testing.T) {
	const secret = "hmactest"
	var attempts atomic.Int32
	requests := make(chan struct {
		body       []byte
		signature  string
		deliveryID string
	}, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests <- struct {
			body       []byte
			signature  string
			deliveryID string
		}{body: body, signature: r.Header.Get("X-Scriberr-Signature"), deliveryID: r.Header.Get("X-Scriberr-Delivery")}
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	service, db := newConfiguredWebhookTestService(t)
	hook := models.Webhook{Name: "Sidecarr", URL: server.URL, Events: `["transcription.completed"]`, Enabled: true, Secret: ptr(secret)}
	require.NoError(t, db.Create(&hook).Error)
	service.Dispatch(context.Background(), EventTranscriptionSuccess, &models.TranscriptionJob{
		ID: "job-123", Status: models.StatusCompleted, AudioPath: "/path/to/audio.wav",
	}, nil, "")

	delivery := waitForDelivery(t, db, DeliveryStatusSucceeded)
	assert.Equal(t, 3, delivery.AttemptCount)
	assert.Equal(t, http.StatusAccepted, *delivery.ResponseStatus)
	assert.NotNil(t, delivery.DeliveredAt)
	assert.Equal(t, int32(3), attempts.Load())

	for range 3 {
		request := <-requests
		assert.Equal(t, delivery.ID, request.deliveryID)
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(request.body)
		assert.Equal(t, "sha256="+hex.EncodeToString(mac.Sum(nil)), request.signature)
		var payload EventPayload
		require.NoError(t, json.Unmarshal(request.body, &payload))
		assert.Equal(t, SchemaVersion, payload.SchemaVersion)
		assert.Equal(t, EventTranscriptionSuccess, payload.Event)
	}
}

func TestConfiguredWebhookDoesNotRetryClientError(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	service, db := newConfiguredWebhookTestService(t)
	hook := models.Webhook{Name: "Invalid receiver", URL: server.URL, Events: `["recording.uploaded"]`, Enabled: true}
	require.NoError(t, db.Create(&hook).Error)
	service.Dispatch(context.Background(), EventRecordingUploaded, &models.TranscriptionJob{ID: "job-400"}, nil, "")

	delivery := waitForDelivery(t, db, DeliveryStatusFailed)
	assert.Equal(t, 1, delivery.AttemptCount)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestResumePendingDelivery(t *testing.T) {
	called := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called <- struct{}{}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	service, db := newConfiguredWebhookTestService(t)
	now := time.Now().UTC()
	delivery := models.WebhookDelivery{
		WebhookID: "hook-1", WebhookName: "Recovered", Event: string(EventRecordingUploaded),
		JobID: "job-recovered", DestinationURL: server.URL, Payload: `{}`,
		Status: DeliveryStatusPending, NextAttemptAt: &now,
	}
	require.NoError(t, db.Create(&delivery).Error)
	service.resumePendingDeliveries()

	waitForDelivery(t, db, DeliveryStatusSucceeded)
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("recovered webhook was not sent")
	}
}

func ptr[T any](value T) *T {
	return &value
}

func TestSendWebhook(t *testing.T) {
	// Setup
	service := NewService()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		// Mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "Scriberr-Webhook/1.0", r.Header.Get("User-Agent"))

			var payload WebhookPayload
			err := json.NewDecoder(r.Body).Decode(&payload)
			assert.NoError(t, err)
			assert.Equal(t, "job-123", payload.JobID)
			assert.Equal(t, models.StatusCompleted, payload.Status)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Test payload
		payload := WebhookPayload{
			JobID:       "job-123",
			Status:      models.StatusCompleted,
			AudioPath:   "/path/to/audio.wav",
			CompletedAt: time.Now(),
		}

		// Execute
		err := service.SendWebhook(ctx, server.URL, payload)

		// Verify
		assert.NoError(t, err)
	})

	t.Run("RetryLogic", func(t *testing.T) {
		attempts := 0
		// Mock server that fails twice then succeeds
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		payload := WebhookPayload{
			JobID:  "job-retry",
			Status: models.StatusFailed,
		}

		// Execute
		err := service.SendWebhook(ctx, server.URL, payload)

		// Verify
		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("FailureAfterRetries", func(t *testing.T) {
		// Mock server that always fails
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		payload := WebhookPayload{
			JobID: "job-fail",
		}

		// Execute
		err := service.SendWebhook(ctx, server.URL, payload)

		// Verify
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send webhook after 3 attempts")
	})

	t.Run("EmptyURL", func(t *testing.T) {
		err := service.SendWebhook(ctx, "", WebhookPayload{})
		assert.NoError(t, err)
	})
}
