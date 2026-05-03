package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/transcription/engineprovider"
	"scriberr/internal/transcription/worker"

	"github.com/stretchr/testify/require"
)

type fakeQueueService struct {
	mu        sync.Mutex
	enqueued  []string
	canceled  []string
	stats     worker.QueueStats
	err       error
	cancelErr error
}

func (q *fakeQueueService) Enqueue(ctx context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enqueued = append(q.enqueued, jobID)
	return q.err
}
func (q *fakeQueueService) Cancel(ctx context.Context, userID uint, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.canceled = append(q.canceled, jobID)
	return q.cancelErr
}
func (q *fakeQueueService) Start(context.Context) error { return nil }
func (q *fakeQueueService) Stop(context.Context) error  { return nil }
func (q *fakeQueueService) Stats(context.Context, uint) (worker.QueueStats, error) {
	return q.stats, q.err
}

func setTestQueueService(s *authTestServer, queue *fakeQueueService) {
	s.handler.queueService = queue
	if s.handler.transcriptions != nil {
		s.handler.transcriptions.SetQueue(queue)
	}
}

type fakeCapabilityProvider struct {
	caps []engineprovider.ModelCapability
}

func (p fakeCapabilityProvider) ID() string { return "local" }
func (p fakeCapabilityProvider) Capabilities(context.Context) ([]engineprovider.ModelCapability, error) {
	return p.caps, nil
}
func (p fakeCapabilityProvider) Prepare(context.Context) error { return nil }
func (p fakeCapabilityProvider) Transcribe(context.Context, engineprovider.TranscriptionRequest) (*engineprovider.TranscriptionResult, error) {
	return nil, nil
}
func (p fakeCapabilityProvider) Diarize(context.Context, engineprovider.DiarizationRequest) (*engineprovider.DiarizationResult, error) {
	return nil, nil
}
func (p fakeCapabilityProvider) Close() error { return nil }

func TestCreateSubmitRetryUseQueueService(t *testing.T) {
	s := newAuthTestServer(t)
	queue := &fakeQueueService{}
	setTestQueueService(s, queue)
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": fileID,
		"title":   "Queued by service",
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	firstID := strings.TrimPrefix(body["id"].(string), "tr_")

	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+body["id"].(string)+":retry", nil, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	retryID := strings.TrimPrefix(body["id"].(string), "tr_")

	require.Len(t, queue.enqueued, 2)
	require.Equal(t, firstID, queue.enqueued[0])
	require.Equal(t, retryID, queue.enqueued[1])
}

func TestCreateReturnsServiceUnavailableWhenQueueStopped(t *testing.T) {
	s := newAuthTestServer(t)
	setTestQueueService(s, &fakeQueueService{err: worker.ErrQueueStopped})
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{"file_id": fileID}, token, "")

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	require.Equal(t, "SERVICE_UNAVAILABLE", body["error"].(map[string]any)["code"])

	var count int64
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).
		Where("source_file_hash IS NOT NULL").
		Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestRetryPreservesNewJobWhenQueueStopped(t *testing.T) {
	s := newAuthTestServer(t)
	queue := &fakeQueueService{}
	setTestQueueService(s, queue)
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{"file_id": fileID}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	transcriptionID := body["id"].(string)

	queue.err = worker.ErrQueueStopped
	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+":retry", nil, token, "")
	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	require.Equal(t, "SERVICE_UNAVAILABLE", body["error"].(map[string]any)["code"])

	var count int64
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).
		Where("source_file_hash IS NOT NULL").
		Count(&count).Error)
	require.Equal(t, int64(2), count)
}

func TestCancelUsesQueueServiceAndMapsConflict(t *testing.T) {
	s := newAuthTestServer(t)
	queue := &fakeQueueService{cancelErr: worker.ErrStateConflict}
	setTestQueueService(s, queue)
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)
	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{"file_id": fileID}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	transcriptionID := body["id"].(string)

	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+":cancel", nil, token, "")

	require.Equal(t, http.StatusConflict, resp.Code)
	require.Equal(t, "CONFLICT", body["error"].(map[string]any)["code"])
	require.Equal(t, strings.TrimPrefix(transcriptionID, "tr_"), queue.canceled[0])
}

func TestTranscriptExecutionsLogsModelsAndStatsUseEngineServices(t *testing.T) {
	s := newAuthTestServer(t)
	queue := &fakeQueueService{stats: worker.QueueStats{Queued: 2, Processing: 1, Completed: 3, Failed: 4, Canceled: 5, Running: 1}}
	setTestQueueService(s, queue)
	registry, err := engineprovider.NewRegistry("local", fakeCapabilityProvider{caps: []engineprovider.ModelCapability{
		{ID: "whisper-base", Name: "Whisper Base", Provider: "local", Installed: true, Default: true, Capabilities: []string{"transcription", "word_timestamps"}},
	}})
	require.NoError(t, err)
	s.handler.modelRegistry = registry

	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)
	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{"file_id": fileID}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	transcriptionID := body["id"].(string)
	jobID := strings.TrimPrefix(transcriptionID, "tr_")

	now := time.Now().UTC().Truncate(time.Millisecond)
	transcript := `{"text":"hello","segments":[{"id":"seg_000001","start":0,"end":1,"speaker":"SPEAKER_00","text":"hello"}]}`
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":          models.StatusCompleted,
		"transcript_text": transcript,
		"progress":        1.0,
		"progress_stage":  "completed",
		"started_at":      now.Add(-time.Minute),
		"completed_at":    now,
	}).Error)
	errorMessage := "failed at /tmp/private/model.bin api_key=secret-value"
	require.NoError(t, database.DB.Create(&models.TranscriptionJobExecution{
		TranscriptionJobID: jobID,
		Status:             models.StatusFailed,
		Provider:           "local",
		ModelName:          "whisper-base",
		StartedAt:          now.Add(-time.Minute),
		FailedAt:           &now,
		ErrorMessage:       &errorMessage,
	}).Error)

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, float64(1), body["progress"])
	require.Equal(t, "completed", body["progress_stage"])
	require.NotNil(t, body["started_at"])
	require.NotNil(t, body["completed_at"])

	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":     models.StatusFailed,
		"last_error": errorMessage,
	}).Error)
	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.NotContains(t, body["error"], "/tmp/private")
	require.NotContains(t, body["error"], "secret-value")
	require.Contains(t, body["error"], "[redacted-path]")

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/transcript", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "hello", body["text"])
	require.Empty(t, body["words"])
	require.Len(t, body["segments"].([]any), 1)

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/executions", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	execution := body["items"].([]any)[0].(map[string]any)
	require.Equal(t, "local", execution["provider"])
	require.Equal(t, "whisper-base", execution["model"])
	require.NotContains(t, execution["error"], "/tmp/private")
	require.NotContains(t, execution["error"], "secret-value")

	resp, rawLogs := s.rawRequest(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/logs", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.NotContains(t, rawLogs, "/tmp/private")
	require.NotContains(t, rawLogs, "secret-value")
	require.Contains(t, rawLogs, "[redacted-path]")
	require.Contains(t, rawLogs, "\nfailed_at=")

	resp, body = s.request(t, http.MethodGet, "/api/v1/models/transcription", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	model := body["items"].([]any)[0].(map[string]any)
	require.Equal(t, "whisper-base", model["id"])
	require.Equal(t, true, model["installed"])
	require.Equal(t, true, model["default"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/admin/queue", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, float64(2), body["queued"])
	require.Equal(t, float64(1), body["running"])
}

func TestQueueServiceErrorDoesNotLeakInternals(t *testing.T) {
	s := newAuthTestServer(t)
	setTestQueueService(s, &fakeQueueService{err: errors.New("open /tmp/private/socket token=secret failed")})
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{"file_id": fileID}, token, "")

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	message := body["error"].(map[string]any)["message"].(string)
	require.NotContains(t, message, "/tmp/private")
	require.NotContains(t, message, "secret")
}
