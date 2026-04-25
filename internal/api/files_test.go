package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/stretchr/testify/require"
)

func registerForFileTests(t *testing.T, s *authTestServer) string {
	t.Helper()

	resp, body := s.request(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":         "admin",
		"password":         "password123",
		"confirm_password": "password123",
	}, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	return body["access_token"].(string)
}

func uploadMultipart(t *testing.T, s *authTestServer, token, fieldName, filename, contentType string, content []byte, title string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="`+fieldName+`"; filename="`+filename+`"`)
	partHeader.Set("Content-Type", contentType)
	part, err := writer.CreatePart(partHeader)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	if title != "" {
		require.NoError(t, writer.WriteField("title", title))
	}
	require.NoError(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, "/api/v1/files", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)

	var response map[string]any
	if recorder.Body.Len() > 0 {
		require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	}
	return recorder, response
}

func TestFileUploadListGetPatchDelete(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := uploadMultipart(t, s, token, "file", "meeting.wav", "audio/wav", []byte("RIFF----WAVEfmt data"), "Team sync")
	require.Equal(t, http.StatusCreated, resp.Code)
	fileID := body["id"].(string)
	require.True(t, strings.HasPrefix(fileID, "file_"))
	require.Equal(t, "Team sync", body["title"])
	require.Equal(t, "audio", body["kind"])
	require.Equal(t, "ready", body["status"])
	require.Equal(t, "audio/wav", body["mime_type"])
	require.NotContains(t, body, "audio_path")
	require.NotContains(t, body, "source_file_path")

	var stored models.TranscriptionJob
	require.NoError(t, database.DB.First(&stored, "id = ?", strings.TrimPrefix(fileID, "file_")).Error)
	require.NotEmpty(t, stored.AudioPath)
	require.NotContains(t, fileID, filepath.Base(stored.AudioPath))

	resp, body = s.request(t, http.MethodGet, "/api/v1/files", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.Len(t, items, 1)

	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": fileID,
		"title":   "Team sync transcript",
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	transcriptionID := body["id"].(string)

	resp, body = s.request(t, http.MethodGet, "/api/v1/files", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items = body["items"].([]any)
	require.Len(t, items, 1)

	resp, _ = s.request(t, http.MethodGet, "/api/v1/files/"+strings.Replace(transcriptionID, "tr_", "file_", 1), nil, token, "")
	require.Equal(t, http.StatusNotFound, resp.Code)

	resp, body = s.request(t, http.MethodGet, "/api/v1/files/"+fileID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, fileID, body["id"])
	require.NotContains(t, body, "source_file_path")

	resp, body = s.request(t, http.MethodPatch, "/api/v1/files/"+fileID, map[string]any{"title": "Renamed"}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "Renamed", body["title"])

	resp, _ = s.request(t, http.MethodDelete, "/api/v1/files/"+fileID, nil, token, "")
	require.Equal(t, http.StatusNoContent, resp.Code)

	resp, _ = s.request(t, http.MethodGet, "/api/v1/files/"+fileID, nil, token, "")
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestFileUploadValidationAndSecurity(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, _ := uploadMultipart(t, s, token, "wrong", "meeting.wav", "audio/wav", []byte("RIFF----WAVEfmt data"), "")
	require.Equal(t, http.StatusBadRequest, resp.Code)

	resp, body := uploadMultipart(t, s, token, "file", "../secret.wav", "audio/wav", []byte("RIFF----WAVEfmt data"), "")
	require.Equal(t, http.StatusCreated, resp.Code)
	require.NotContains(t, body["title"], "..")

	resp, body = uploadMultipart(t, s, token, "file", "notes.txt", "text/plain", []byte("plain text"), "")
	require.Equal(t, http.StatusUnsupportedMediaType, resp.Code)
	errBody := body["error"].(map[string]any)
	require.NotContains(t, errBody["message"], "/")
}

func TestYouTubeImportReturnsProcessingPlaceholder(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodPost, "/api/v1/files:import-youtube", map[string]any{
		"url":   "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"title": "Talk",
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	require.True(t, strings.HasPrefix(body["id"].(string), "file_"))
	require.Equal(t, "Talk", body["title"])
	require.Equal(t, "youtube", body["kind"])
	require.Equal(t, "processing", body["status"])
	require.NotContains(t, body, "source_file_path")

	resp, body = s.request(t, http.MethodGet, "/api/v1/files", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, "youtube", items[0].(map[string]any)["kind"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/files:import-youtube", map[string]any{
		"url": "file:///etc/passwd",
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "url", errBody["field"])
}

func TestFileListFiltersSortingPaginationAndValidation(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	uploads := []struct {
		filename string
		title    string
	}{
		{filename: "alpha.wav", title: "Alpha meeting"},
		{filename: "bravo.mp3", title: "Bravo notes"},
		{filename: "charlie.wav", title: "Charlie sync"},
	}
	for _, upload := range uploads {
		resp, _ := uploadMultipart(t, s, token, "file", upload.filename, "audio/wav", []byte("RIFF----WAVEfmt data"), upload.title)
		require.Equal(t, http.StatusCreated, resp.Code)
	}
	resp, _ := s.request(t, http.MethodPost, "/api/v1/files:import-youtube", map[string]any{
		"url":   "https://www.youtube.com/watch?v=abc123",
		"title": "YouTube talk",
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)

	resp, body := s.request(t, http.MethodGet, "/api/v1/files?kind=audio&q=bravo&sort=title", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, "Bravo notes", items[0].(map[string]any)["title"])
	require.Equal(t, "audio", items[0].(map[string]any)["kind"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/files?kind=youtube", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items = body["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, "youtube", items[0].(map[string]any)["kind"])

	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	resp, body = s.request(t, http.MethodGet, "/api/v1/files?updated_after="+future, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Empty(t, body["items"].([]any))

	resp, body = s.request(t, http.MethodGet, "/api/v1/files?limit=2&sort=title", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	firstPage := body["items"].([]any)
	require.Len(t, firstPage, 2)
	require.Equal(t, "Alpha meeting", firstPage[0].(map[string]any)["title"])
	require.Equal(t, "Bravo notes", firstPage[1].(map[string]any)["title"])
	nextCursor, ok := body["next_cursor"].(string)
	require.True(t, ok)
	require.NotEmpty(t, nextCursor)

	resp, body = s.request(t, http.MethodGet, "/api/v1/files?limit=2&sort=title&cursor="+nextCursor, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	secondPage := body["items"].([]any)
	require.Len(t, secondPage, 2)
	require.Equal(t, "Charlie sync", secondPage[0].(map[string]any)["title"])
	require.Equal(t, "YouTube talk", secondPage[1].(map[string]any)["title"])
	require.Nil(t, body["next_cursor"])

	validationCases := []string{
		"/api/v1/files?limit=0",
		"/api/v1/files?kind=document",
		"/api/v1/files?sort=size",
		"/api/v1/files?updated_after=not-a-time",
		"/api/v1/files?cursor=not-a-cursor",
	}
	for _, path := range validationCases {
		resp, body := s.request(t, http.MethodGet, path, nil, token, "")
		require.Equal(t, http.StatusUnprocessableEntity, resp.Code, path)
		errBody := body["error"].(map[string]any)
		require.NotEmpty(t, errBody["field"])
	}
}

func TestFileAudioRangeStreaming(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	content := []byte("RIFF----WAVEfmt 0123456789abcdef")

	resp, body := uploadMultipart(t, s, token, "file", "meeting.wav", "audio/wav", content, "Team sync")
	require.Equal(t, http.StatusCreated, resp.Code)
	fileID := body["id"].(string)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/files/"+fileID+"/audio", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	full := httptest.NewRecorder()
	s.router.ServeHTTP(full, req)
	require.Equal(t, http.StatusOK, full.Code)
	require.Equal(t, "bytes", full.Header().Get("Accept-Ranges"))
	require.Equal(t, content, full.Body.Bytes())

	req, err = http.NewRequest(http.MethodGet, "/api/v1/files/"+fileID+"/audio", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Range", "bytes=5-9")
	partial := httptest.NewRecorder()
	s.router.ServeHTTP(partial, req)
	require.Equal(t, http.StatusPartialContent, partial.Code)
	require.Equal(t, "bytes 5-9/32", partial.Header().Get("Content-Range"))
	require.Equal(t, content[5:10], partial.Body.Bytes())

	req, err = http.NewRequest(http.MethodGet, "/api/v1/files/"+fileID+"/audio", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Range", "bytes=99-100")
	invalid := httptest.NewRecorder()
	s.router.ServeHTTP(invalid, req)
	require.Equal(t, http.StatusRequestedRangeNotSatisfiable, invalid.Code)
	require.Equal(t, "bytes */32", invalid.Header().Get("Content-Range"))
}
