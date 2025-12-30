package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"scriberr/internal/models"

	"github.com/stretchr/testify/assert"
)

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
