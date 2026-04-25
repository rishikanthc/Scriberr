package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/stretchr/testify/require"
)

func idempotentJSONRequest(t *testing.T, s *authTestServer, method, path string, body any, token, key string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	var payload bytes.Buffer
	require.NoError(t, json.NewEncoder(&payload).Encode(body))
	req, err := http.NewRequest(method, path, &payload)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", key)

	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)

	var response map[string]any
	if recorder.Code != http.StatusNoContent {
		require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	}
	return recorder, response
}

func idempotentUploadRequest(t *testing.T, s *authTestServer, token, key string, body []byte, contentType string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, "/api/v1/files", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", key)

	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)

	var response map[string]any
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	return recorder, response
}

func fixedMultipartUpload(t *testing.T, filename string, content []byte, title string) ([]byte, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.SetBoundary("scriberr-test-boundary"))
	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("title", title))
	require.NoError(t, writer.Close())
	return body.Bytes(), writer.FormDataContentType()
}

func TestIdempotencyCachesJSONCreateAndRejectsBodyMismatch(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := idempotentJSONRequest(t, s, http.MethodPost, "/api/v1/api-keys", map[string]any{
		"name":        "CLI",
		"description": "first",
	}, token, "idem-api-key")
	require.Equal(t, http.StatusCreated, resp.Code)
	firstRawKey := body["key"].(string)
	firstID := body["id"].(string)

	resp, body = idempotentJSONRequest(t, s, http.MethodPost, "/api/v1/api-keys", map[string]any{
		"name":        "CLI",
		"description": "first",
	}, token, "idem-api-key")
	require.Equal(t, http.StatusCreated, resp.Code)
	require.Equal(t, firstRawKey, body["key"])
	require.Equal(t, firstID, body["id"])

	var count int64
	require.NoError(t, database.DB.Model(&models.APIKey{}).Count(&count).Error)
	require.Equal(t, int64(1), count)

	resp, body = idempotentJSONRequest(t, s, http.MethodPost, "/api/v1/api-keys", map[string]any{
		"name":        "CLI",
		"description": "changed",
	}, token, "idem-api-key")
	require.Equal(t, http.StatusConflict, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "IDEMPOTENCY_CONFLICT", errBody["code"])
}

func TestIdempotencyCoalescesConcurrentCreates(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	const workers = 8
	start := make(chan struct{})
	type result struct {
		status int
		body   map[string]any
		err    error
	}
	responses := make(chan result, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			payload, err := json.Marshal(map[string]any{
				"name":        "Concurrent CLI",
				"description": "same payload",
			})
			if err != nil {
				responses <- result{err: err}
				return
			}
			req, err := http.NewRequest(http.MethodPost, "/api/v1/api-keys", bytes.NewReader(payload))
			if err != nil {
				responses <- result{err: err}
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Idempotency-Key", "idem-concurrent-api-key")

			recorder := httptest.NewRecorder()
			s.router.ServeHTTP(recorder, req)

			var body map[string]any
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				responses <- result{status: recorder.Code, err: err}
				return
			}
			responses <- result{status: recorder.Code, body: body}
		}()
	}
	close(start)
	wg.Wait()
	close(responses)

	var firstID string
	for response := range responses {
		require.NoError(t, response.err)
		require.Equal(t, http.StatusCreated, response.status)
		id := response.body["id"].(string)
		if firstID == "" {
			firstID = id
		}
		require.Equal(t, firstID, id)
		require.NotEmpty(t, response.body["key"])
	}

	var count int64
	require.NoError(t, database.DB.Model(&models.APIKey{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestIdempotencyCachesMultipartUpload(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	body, contentType := fixedMultipartUpload(t, "meeting.wav", []byte("RIFF----WAVEfmt data"), "Meeting")

	resp, first := idempotentUploadRequest(t, s, token, "idem-upload", body, contentType)
	require.Equal(t, http.StatusCreated, resp.Code)

	resp, second := idempotentUploadRequest(t, s, token, "idem-upload", body, contentType)
	require.Equal(t, http.StatusCreated, resp.Code)
	require.Equal(t, first["id"], second["id"])

	var count int64
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).Where("source_file_hash IS NULL").Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestIdempotencyCachesTranscriptionCreate(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	payload := map[string]any{
		"file_id": fileID,
		"title":   "Queued once",
	}
	resp, first := idempotentJSONRequest(t, s, http.MethodPost, "/api/v1/transcriptions", payload, token, "idem-transcription")
	require.Equal(t, http.StatusAccepted, resp.Code)

	resp, second := idempotentJSONRequest(t, s, http.MethodPost, "/api/v1/transcriptions", payload, token, "idem-transcription")
	require.Equal(t, http.StatusAccepted, resp.Code)
	require.Equal(t, first["id"], second["id"])

	var count int64
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).Where("source_file_hash IS NOT NULL").Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestIdempotencyValidation(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := idempotentJSONRequest(t, s, http.MethodPost, "/api/v1/api-keys", map[string]any{"name": "CLI"}, token, "bad key with spaces")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "Idempotency-Key", errBody["field"])
}
