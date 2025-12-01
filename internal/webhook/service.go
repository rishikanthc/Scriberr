package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"scriberr/internal/models"
	"scriberr/pkg/logger"
)

// WebhookPayload represents the data sent to the callback URL
type WebhookPayload struct {
	JobID        string                 `json:"job_id"`
	Status       models.JobStatus       `json:"status"`
	AudioPath    string                 `json:"audio_path"`
	Transcript   *string                `json:"transcript,omitempty"`
	Summary      *string                `json:"summary,omitempty"`
	ErrorMessage *string                `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CompletedAt  time.Time              `json:"completed_at"`
}

// Service handles webhook operations
type Service struct {
	client *http.Client
}

// NewService creates a new webhook service
func NewService() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendWebhook sends a webhook notification to the specified URL
func (s *Service) SendWebhook(ctx context.Context, url string, payload WebhookPayload) error {
	if url == "" {
		return nil
	}

	logger.Info("Sending webhook", "job_id", payload.JobID, "url", url, "status", payload.Status)

	// Marshal payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Scriberr-Webhook/1.0")

	// Send request with retry logic
	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i) * time.Second) // Simple backoff
			logger.Info("Retrying webhook", "job_id", payload.JobID, "attempt", i+1)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			logger.Warn("Webhook request failed", "error", err, "attempt", i+1)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logger.Info("Webhook sent successfully", "job_id", payload.JobID, "status_code", resp.StatusCode)
			return nil
		}

		lastErr = fmt.Errorf("webhook returned non-success status: %d", resp.StatusCode)
		logger.Warn("Webhook returned error status", "status_code", resp.StatusCode, "attempt", i+1)
	}

	return fmt.Errorf("failed to send webhook after %d attempts: %w", maxRetries, lastErr)
}
