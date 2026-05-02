package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"scriberr/internal/auth"
	"scriberr/internal/database"
	"scriberr/internal/models"
)

func rawRecordingChunkRequest(t *testing.T, s *authTestServer, token, path, contentType string, body []byte, checksum string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	if checksum != "" {
		req.Header.Set("X-Chunk-SHA256", checksum)
	}
	req.Header.Set("X-Chunk-Duration-Ms", "3000")
	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)
	var response map[string]any
	if recorder.Body.Len() > 0 {
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("Decode response returned error: %v body=%s", err, recorder.Body.String())
		}
	}
	return recorder, response
}

func TestRecordingCreateUploadListGetStopCancel(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodPost, "/api/v1/recordings", map[string]any{
		"title":             " Team sync ",
		"source_kind":       "microphone",
		"mime_type":         "audio/webm;codecs=opus",
		"chunk_duration_ms": 3000,
		"options": map[string]any{
			"language":    "en",
			"diarization": true,
		},
	}, token, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%v", resp.Code, body)
	}
	recordingID := body["id"].(string)
	if !strings.HasPrefix(recordingID, "rec_") {
		t.Fatalf("recording id = %q", recordingID)
	}
	if body["title"] != "Team sync" {
		t.Fatalf("title = %v", body["title"])
	}
	if body["status"] != "recording" {
		t.Fatalf("status = %v", body["status"])
	}
	if body["file_id"] != nil || body["transcription_id"] != nil {
		t.Fatalf("file/transcription ids should be null: %v", body)
	}

	checksum := sha256Hex("chunk")
	resp, body = rawRecordingChunkRequest(t, s, token, "/api/v1/recordings/"+recordingID+"/chunks/0", "audio/webm;codecs=opus", []byte("chunk"), checksum)
	if resp.Code != http.StatusCreated {
		t.Fatalf("chunk status = %d body=%v", resp.Code, body)
	}
	if body["status"] != "stored" {
		t.Fatalf("chunk status = %v", body["status"])
	}
	if body["received_chunks"] != float64(1) {
		t.Fatalf("received_chunks = %v", body["received_chunks"])
	}

	resp, body = rawRecordingChunkRequest(t, s, token, "/api/v1/recordings/"+recordingID+"/chunks/0", "audio/webm;codecs=opus", []byte("chunk"), checksum)
	if resp.Code != http.StatusCreated {
		t.Fatalf("retry chunk status = %d body=%v", resp.Code, body)
	}
	if body["status"] != "already_stored" {
		t.Fatalf("retry status = %v", body["status"])
	}

	resp, body = s.request(t, http.MethodGet, "/api/v1/recordings/"+recordingID, nil, token, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("get status = %d body=%v", resp.Code, body)
	}
	if body["received_chunks"] != float64(1) {
		t.Fatalf("get received_chunks = %v", body["received_chunks"])
	}
	if _, ok := body["source_file_path"]; ok {
		t.Fatal("recording response leaked source_file_path")
	}

	resp, body = s.request(t, http.MethodGet, "/api/v1/recordings", nil, token, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%v", resp.Code, body)
	}
	items := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items len = %d", len(items))
	}

	resp, body = s.request(t, http.MethodPost, "/api/v1/recordings/"+recordingID+":stop", map[string]any{
		"final_chunk_index": 0,
		"duration_ms":       3000,
	}, token, "")
	if resp.Code != http.StatusAccepted {
		t.Fatalf("stop status = %d body=%v", resp.Code, body)
	}
	if body["status"] != "stopping" {
		t.Fatalf("stop response status = %v", body["status"])
	}

	resp, body = rawRecordingChunkRequest(t, s, token, "/api/v1/recordings/"+recordingID+"/chunks/1", "audio/webm;codecs=opus", []byte("chunk"), checksum)
	if resp.Code != http.StatusConflict {
		t.Fatalf("late chunk status = %d body=%v", resp.Code, body)
	}

	resp, body = s.request(t, http.MethodPost, "/api/v1/recordings/"+recordingID+":cancel", nil, token, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("cancel status = %d body=%v", resp.Code, body)
	}
	if body["status"] != "canceled" {
		t.Fatalf("cancel status = %v", body["status"])
	}

	var stored models.RecordingSession
	if err := database.DB.First(&stored, "id = ?", strings.TrimPrefix(recordingID, "rec_")).Error; err != nil {
		t.Fatalf("recording was not persisted: %v", err)
	}
	if stored.TranscriptionOptionsJSON == "" || !strings.Contains(stored.TranscriptionOptionsJSON, `"language":"en"`) {
		t.Fatalf("options json = %q", stored.TranscriptionOptionsJSON)
	}
}

func TestRecordingValidationAndAuth(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, _ := s.request(t, http.MethodPost, "/api/v1/recordings", map[string]any{
		"mime_type": "audio/webm;codecs=opus",
	}, "", "")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("unauth create status = %d", resp.Code)
	}

	resp, _ = s.request(t, http.MethodPost, "/api/v1/recordings", map[string]any{
		"mime_type": "video/webm",
	}, token, "")
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid mime status = %d", resp.Code)
	}

	resp, body := s.request(t, http.MethodPost, "/api/v1/recordings", map[string]any{
		"mime_type": "audio/webm;codecs=opus",
	}, token, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%v", resp.Code, body)
	}
	recordingID := body["id"].(string)

	resp, _ = rawRecordingChunkRequest(t, s, token, "/api/v1/recordings/"+recordingID+"/chunks/bad", "audio/webm;codecs=opus", []byte("chunk"), "")
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("bad index status = %d", resp.Code)
	}

	resp, _ = rawRecordingChunkRequest(t, s, token, "/api/v1/recordings/"+recordingID+"/chunks/0", "audio/webm;codecs=opus", []byte("too-large"), "")
	if resp.Code != http.StatusRequestEntityTooLarge && resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("oversized status = %d", resp.Code)
	}

	resp, _ = rawRecordingChunkRequest(t, s, token, "/api/v1/recordings/"+recordingID+"/chunks/1", "audio/webm;codecs=opus", []byte("chunk"), strings.Repeat("0", 64))
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("checksum status = %d", resp.Code)
	}

	resp, _ = s.request(t, http.MethodPost, "/api/v1/recordings/"+recordingID+":stop", map[string]any{
		"final_chunk_index": 0,
		"duration_ms":       int64((2 * time.Hour) / time.Millisecond),
	}, token, "")
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("too long stop status = %d", resp.Code)
	}
}

func TestRecordingCommandRoutesAndRetryContract(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodPost, "/api/v1/recordings", map[string]any{
		"mime_type": "audio/webm;codecs=opus",
	}, token, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%v", resp.Code, body)
	}
	recordingID := body["id"].(string)
	checksum := sha256Hex("chunk")
	resp, body = rawRecordingChunkRequest(t, s, token, "/api/v1/recordings/"+recordingID+"/chunks/0", "audio/webm;codecs=opus", []byte("chunk"), checksum)
	if resp.Code != http.StatusCreated {
		t.Fatalf("chunk status = %d body=%v", resp.Code, body)
	}
	resp, body = s.request(t, http.MethodPost, "/api/v1/recordings/"+recordingID+":stop", map[string]any{
		"final_chunk_index": 0,
	}, token, "")
	if resp.Code != http.StatusAccepted {
		t.Fatalf("stop status = %d body=%v", resp.Code, body)
	}

	internalID := strings.TrimPrefix(recordingID, "rec_")
	if err := database.DB.Model(&models.RecordingSession{}).Where("id = ?", internalID).UpdateColumns(map[string]any{
		"status":         models.RecordingStatusFailed,
		"failed_at":      time.Now(),
		"progress_stage": "failed",
	}).Error; err != nil {
		t.Fatalf("mark failed returned error: %v", err)
	}
	resp, body = s.request(t, http.MethodPost, "/api/v1/recordings/"+recordingID+":retry-finalize", nil, token, "")
	if resp.Code != http.StatusAccepted {
		t.Fatalf("retry-finalize status = %d body=%v", resp.Code, body)
	}
	if body["status"] != "stopping" {
		t.Fatalf("retry-finalize status body = %v", body["status"])
	}

	resp, body = s.request(t, http.MethodPost, "/api/v1/recordings", map[string]any{
		"mime_type": "audio/webm;codecs=opus",
	}, token, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("second create status = %d body=%v", resp.Code, body)
	}
	secondID := body["id"].(string)
	resp, body = s.request(t, http.MethodPost, "/api/v1/recordings/"+secondID+":cancel", nil, token, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("cancel command status = %d body=%v", resp.Code, body)
	}
	if body["status"] != "canceled" {
		t.Fatalf("cancel command status body = %v", body["status"])
	}
}

func TestRecordingSecurityRegressions(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodPost, "/api/v1/recordings", map[string]any{
		"mime_type": "audio/webm;codecs=opus",
	}, token, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%v", resp.Code, body)
	}
	recordingID := body["id"].(string)

	otherUser := models.User{Username: "recording-other-user", Password: "pw"}
	if err := database.DB.Create(&otherUser).Error; err != nil {
		t.Fatalf("create other user returned error: %v", err)
	}
	otherToken, err := auth.NewAuthService("test-secret").GenerateToken(&otherUser)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	resp, _ = s.request(t, http.MethodGet, "/api/v1/recordings/"+recordingID, nil, otherToken, "")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("cross-user get status = %d", resp.Code)
	}
	resp, _ = rawRecordingChunkRequest(t, s, otherToken, "/api/v1/recordings/"+recordingID+"/chunks/0", "audio/webm;codecs=opus", []byte("chunk"), "")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("cross-user upload status = %d", resp.Code)
	}
	resp, _ = s.request(t, http.MethodPost, "/api/v1/recordings/"+recordingID+":stop", map[string]any{"final_chunk_index": 0}, otherToken, "")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("cross-user stop status = %d", resp.Code)
	}

	resp, body = rawRecordingChunkRequest(t, s, token, "/api/v1/recordings/"+recordingID+"/chunks/0", "audio/ogg", []byte("chunk"), "")
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("mime spoof status = %d body=%v", resp.Code, body)
	}
	assertRecordingErrorSafe(t, body, s.handler.config.Recordings.Dir)

	resp, body = s.request(t, http.MethodGet, "/api/v1/recordings/rec_..", nil, token, "")
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("traversal id status = %d body=%v", resp.Code, body)
	}
	assertRecordingErrorSafe(t, body, s.handler.config.Recordings.Dir)

	resp, body = rawRecordingChunkRequest(t, s, token, "/api/v1/recordings/"+recordingID+"/chunks/1", "audio/webm;codecs=opus", []byte("chunk"), strings.Repeat("0", 64))
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("checksum status = %d body=%v", resp.Code, body)
	}
	assertRecordingErrorSafe(t, body, s.handler.config.Recordings.Dir)
}

func TestRecordingChunkRequestCancellationDoesNotPersist(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodPost, "/api/v1/recordings", map[string]any{
		"mime_type": "audio/webm;codecs=opus",
	}, token, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%v", resp.Code, body)
	}
	recordingID := body["id"].(string)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "/api/v1/recordings/"+recordingID+"/chunks/0", strings.NewReader("chunk"))
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "audio/webm;codecs=opus")
	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)
	if recorder.Code != statusClientClosedRequest {
		t.Fatalf("canceled upload status = %d body=%s", recorder.Code, recorder.Body.String())
	}

	var count int64
	if err := database.DB.Model(&models.RecordingChunk{}).Where("session_id = ?", strings.TrimPrefix(recordingID, "rec_")).Count(&count).Error; err != nil {
		t.Fatalf("Count chunks returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("canceled request persisted %d chunks", count)
	}
}

func TestRecordingEventsDoNotLeakPaths(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	recorder, cancel, done := startEventStream(t, s, token, "/api/v1/events")
	resp, body := s.request(t, http.MethodPost, "/api/v1/recordings", map[string]any{
		"mime_type": "audio/webm;codecs=opus",
	}, token, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%v", resp.Code, body)
	}
	recordingID := body["id"].(string)
	checksum := sha256Hex("chunk")
	resp, _ = rawRecordingChunkRequest(t, s, token, "/api/v1/recordings/"+recordingID+"/chunks/0", "audio/webm;codecs=opus", []byte("chunk"), checksum)
	if resp.Code != http.StatusCreated {
		t.Fatalf("chunk status = %d", resp.Code)
	}
	stopEventStream(t, cancel, done)

	stream := recorder.Body.String()
	if !strings.Contains(stream, "event: recording.created") || !strings.Contains(stream, "event: recording.chunk.stored") {
		t.Fatalf("recording events missing: %s", stream)
	}
	if strings.Contains(stream, s.handler.config.Recordings.Dir) || strings.Contains(stream, "source_file_path") {
		t.Fatalf("event stream leaked storage details: %s", stream)
	}
}

func assertRecordingErrorSafe(t *testing.T, body map[string]any, forbidden string) {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Marshal body returned error: %v", err)
	}
	text := string(encoded)
	if strings.Contains(text, forbidden) || strings.Contains(text, "recordings/") || strings.Contains(text, "source_file_path") {
		t.Fatalf("recording error leaked storage details: %s", text)
	}
}
