package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"scriberr/internal/models"
	"scriberr/pkg/logger"

	"gorm.io/gorm"
)

type Event string

const (
	SchemaVersion                   = "1"
	EventRecordingUploaded    Event = "recording.uploaded"
	EventTranscriptionSuccess Event = "transcription.completed"
	EventTranscriptionFailed  Event = "transcription.failed"
	EventSummarySuccess       Event = "summary.completed"
	EventSummaryFailed        Event = "summary.failed"

	DeliveryStatusPending    = "pending"
	DeliveryStatusProcessing = "processing"
	DeliveryStatusSucceeded  = "succeeded"
	DeliveryStatusFailed     = "failed"
)

var AllEvents = []Event{EventRecordingUploaded, EventTranscriptionSuccess, EventTranscriptionFailed, EventSummarySuccess, EventSummaryFailed}

type EventPayload struct {
	SchemaVersion string                 `json:"schema_version"`
	Event         Event                  `json:"event"`
	JobID         string                 `json:"job_id"`
	Title         *string                `json:"title,omitempty"`
	Status        models.JobStatus       `json:"status"`
	AudioPath     string                 `json:"audio_path"`
	Transcript    *string                `json:"transcript,omitempty"`
	Summary       *string                `json:"summary,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	OccurredAt    time.Time              `json:"occurred_at"`
}

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
	client      *http.Client
	db          *gorm.DB
	retryDelays []time.Duration
}

func (s *Service) SetDatabase(db *gorm.DB) {
	s.db = db
	go s.resumePendingDeliveries()
}

// Dispatch sends an event to every enabled webhook subscribed to it.
func (s *Service) Dispatch(ctx context.Context, event Event, job *models.TranscriptionJob, metadata map[string]interface{}, errorMessage string) {
	if s.db == nil || job == nil {
		return
	}
	var hooks []models.Webhook
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Find(&hooks).Error; err != nil {
		logger.Error("Failed to load webhooks", "error", err)
		return
	}
	for _, hook := range hooks {
		if !subscribesTo(hook.Events, event) {
			continue
		}
		payload := EventPayload{
			SchemaVersion: SchemaVersion,
			Event:         event, JobID: job.ID, Title: job.Title, Status: job.Status,
			AudioPath: job.AudioPath, Transcript: job.Transcript, Summary: job.Summary,
			Error: errorMessage, Metadata: metadata, OccurredAt: time.Now().UTC(),
		}
		if err := s.queueDelivery(ctx, hook, payload); err != nil {
			logger.Error("Failed to queue configured webhook", "webhook_id", hook.ID, "event", event, "error", err)
		}
	}
}

func subscribesTo(eventsJSON string, event Event) bool {
	var events []string
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		return false
	}
	for _, configured := range events {
		if configured == string(event) {
			return true
		}
	}
	return false
}

func (s *Service) queueDelivery(ctx context.Context, hook models.Webhook, payload EventPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	secret := ""
	if hook.Secret != nil {
		secret = *hook.Secret
	}
	now := time.Now().UTC()
	delivery := &models.WebhookDelivery{
		WebhookID: hook.ID, WebhookName: hook.Name, Event: string(payload.Event), JobID: payload.JobID,
		DestinationURL: hook.URL, Payload: string(data), Signature: signPayload(data, secret),
		Status: DeliveryStatusPending, NextAttemptAt: &now,
	}
	if err := s.db.WithContext(ctx).Create(delivery).Error; err != nil {
		return err
	}
	go s.processDelivery(delivery.ID)
	return nil
}

func signPayload(data []byte, secret string) string {
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(data)
	return "sha256=" + fmt.Sprintf("%x", mac.Sum(nil))
}

func (s *Service) sendDelivery(ctx context.Context, delivery models.WebhookDelivery) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, delivery.DestinationURL, bytes.NewReader([]byte(delivery.Payload)))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Scriberr-Webhook/1.0")
	req.Header.Set("X-Scriberr-Delivery", delivery.ID)
	if delivery.Signature != "" {
		req.Header.Set("X-Scriberr-Signature", delivery.Signature)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func (s *Service) processDelivery(id string) {
	for {
		var delivery models.WebhookDelivery
		if err := s.db.First(&delivery, "id = ?", id).Error; err != nil {
			logger.Error("Failed to load queued webhook", "delivery_id", id, "error", err)
			return
		}
		if delivery.Status == DeliveryStatusSucceeded || delivery.Status == DeliveryStatusFailed {
			return
		}
		if delivery.NextAttemptAt != nil && time.Now().Before(*delivery.NextAttemptAt) {
			timer := time.NewTimer(time.Until(*delivery.NextAttemptAt))
			<-timer.C
		}

		claimedAt := time.Now().UTC()
		claim := s.db.Model(&models.WebhookDelivery{}).
			Where("id = ? AND status = ?", id, DeliveryStatusPending).
			Updates(map[string]interface{}{"status": DeliveryStatusProcessing, "updated_at": claimedAt})
		if claim.Error != nil {
			logger.Error("Failed to claim queued webhook", "delivery_id", id, "error", claim.Error)
			return
		}
		if claim.RowsAffected == 0 {
			return
		}

		attempt := delivery.AttemptCount + 1
		requestCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		startedAt := time.Now()
		statusCode, sendErr := s.sendDelivery(requestCtx, delivery)
		cancel()
		durationMs := time.Since(startedAt).Milliseconds()

		if sendErr == nil {
			deliveredAt := time.Now().UTC()
			updates := map[string]interface{}{
				"status": DeliveryStatusSucceeded, "attempt_count": attempt,
				"response_status": statusCode, "last_error": nil,
				"next_attempt_at": nil, "delivered_at": deliveredAt,
			}
			if err := s.db.Model(&models.WebhookDelivery{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				logger.Error("Failed to record webhook success", "delivery_id", id, "error", err)
				return
			}
			logger.Info("Configured webhook sent successfully", "delivery_id", id, "webhook_id", delivery.WebhookID, "event", delivery.Event, "attempt", attempt, "status_code", statusCode, "duration_ms", durationMs)
			return
		}

		errorMessage := sendErr.Error()
		if isRetryable(statusCode, sendErr) && attempt < len(s.retryDelays) {
			nextAttemptAt := time.Now().UTC().Add(s.retryDelays[attempt])
			updates := map[string]interface{}{
				"status": DeliveryStatusPending, "attempt_count": attempt,
				"response_status": nullableStatus(statusCode), "last_error": errorMessage,
				"next_attempt_at": nextAttemptAt,
			}
			if err := s.db.Model(&models.WebhookDelivery{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				logger.Error("Failed to schedule webhook retry", "delivery_id", id, "error", err)
				return
			}
			logger.Warn("Configured webhook delivery will retry", "delivery_id", id, "webhook_id", delivery.WebhookID, "event", delivery.Event, "attempt", attempt, "status_code", statusCode, "duration_ms", durationMs, "next_attempt_at", nextAttemptAt, "error", sendErr)
			continue
		}

		updates := map[string]interface{}{
			"status": DeliveryStatusFailed, "attempt_count": attempt,
			"response_status": nullableStatus(statusCode), "last_error": errorMessage,
			"next_attempt_at": nil,
		}
		if err := s.db.Model(&models.WebhookDelivery{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			logger.Error("Failed to record webhook failure", "delivery_id", id, "error", err)
			return
		}
		logger.Error("Configured webhook delivery failed", "delivery_id", id, "webhook_id", delivery.WebhookID, "event", delivery.Event, "attempt", attempt, "status_code", statusCode, "duration_ms", durationMs, "error", sendErr)
		return
	}
}

func nullableStatus(statusCode int) interface{} {
	if statusCode == 0 {
		return nil
	}
	return statusCode
}

func isRetryable(statusCode int, err error) bool {
	return err != nil && (statusCode == 0 || statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooManyRequests || statusCode >= 500)
}

func (s *Service) resumePendingDeliveries() {
	if s.db == nil {
		return
	}
	var deliveries []models.WebhookDelivery
	if err := s.db.Where("status IN ?", []string{DeliveryStatusPending, DeliveryStatusProcessing}).Find(&deliveries).Error; err != nil {
		logger.Error("Failed to load pending webhooks", "error", err)
		return
	}
	for _, delivery := range deliveries {
		if delivery.Status == DeliveryStatusPending {
			go s.processDelivery(delivery.ID)
			continue
		}
		go s.recoverProcessingDelivery(delivery)
	}
}

func (s *Service) recoverProcessingDelivery(delivery models.WebhookDelivery) {
	leaseExpiresAt := delivery.UpdatedAt.Add(time.Minute)
	if delay := time.Until(leaseExpiresAt); delay > 0 {
		timer := time.NewTimer(delay)
		<-timer.C
	}
	recovery := s.db.Model(&models.WebhookDelivery{}).
		Where("id = ? AND status = ? AND updated_at <= ?", delivery.ID, DeliveryStatusProcessing, delivery.UpdatedAt).
		Update("status", DeliveryStatusPending)
	if recovery.Error != nil {
		logger.Error("Failed to recover interrupted webhook", "delivery_id", delivery.ID, "error", recovery.Error)
		return
	}
	if recovery.RowsAffected == 1 {
		s.processDelivery(delivery.ID)
	}
}

func (s *Service) List(ctx context.Context) ([]models.Webhook, error) {
	var hooks []models.Webhook
	if s.db == nil {
		return hooks, fmt.Errorf("webhook database is not configured")
	}
	err := s.db.WithContext(ctx).Order("created_at DESC").Find(&hooks).Error
	return hooks, err
}

func (s *Service) Create(ctx context.Context, hook *models.Webhook) error {
	if s.db == nil {
		return fmt.Errorf("webhook database is not configured")
	}
	return s.db.WithContext(ctx).Create(hook).Error
}

func (s *Service) Update(ctx context.Context, hook *models.Webhook) error {
	if s.db == nil {
		return fmt.Errorf("webhook database is not configured")
	}
	return s.db.WithContext(ctx).Save(hook).Error
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if s.db == nil {
		return fmt.Errorf("webhook database is not configured")
	}
	return s.db.WithContext(ctx).Delete(&models.Webhook{}, "id = ?", id).Error
}

func (s *Service) Get(ctx context.Context, id string) (*models.Webhook, error) {
	var hook models.Webhook
	if s.db == nil {
		return nil, fmt.Errorf("webhook database is not configured")
	}
	if err := s.db.WithContext(ctx).First(&hook, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &hook, nil
}

func (s *Service) ListDeliveries(ctx context.Context, limit int) ([]models.WebhookDelivery, error) {
	var deliveries []models.WebhookDelivery
	if s.db == nil {
		return deliveries, fmt.Errorf("webhook database is not configured")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	err := s.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&deliveries).Error
	return deliveries, err
}

// NewService creates a new webhook service
func NewService() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		retryDelays: []time.Duration{0, time.Second, 5 * time.Second},
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
