package transcription

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWebhookIntegration_Failure(t *testing.T) {
	// Setup mock repository
	mockRepo := new(MockJobRepository)

	// Setup webhook server
	webhookCalled := make(chan bool, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload webhook.WebhookPayload
		json.NewDecoder(r.Body).Decode(&payload)

		// Verify payload
		if payload.Status == models.StatusFailed {
			webhookCalled <- true
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Setup service
	service := NewUnifiedTranscriptionService(mockRepo, "data/temp", "data/transcripts")

	// Setup test job
	callbackURL := server.URL
	jobID := "test-job-id"
	job := &models.TranscriptionJob{
		ID:        jobID,
		AudioPath: "/non/existent/file.wav", // This will cause processing to fail
		Status:    models.StatusPending,
		Parameters: models.WhisperXParams{
			CallbackURL: &callbackURL,
			ModelFamily: "whisper",
		},
	}

	// Mock expectations
	mockRepo.On("FindWithAssociations", mock.Anything, jobID).Return(job, nil)
	mockRepo.On("CreateExecution", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("UpdateExecution", mock.Anything, mock.Anything).Return(nil)

	// Execute
	// We expect an error because the file doesn't exist
	err := service.ProcessJob(context.Background(), jobID)
	assert.Error(t, err)

	// Verify webhook was called
	select {
	case <-webhookCalled:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Webhook was not called within timeout")
	}

	mockRepo.AssertExpectations(t)
}
