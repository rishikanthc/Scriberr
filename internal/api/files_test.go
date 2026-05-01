package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/mediaimport"
	"scriberr/internal/models"

	"github.com/stretchr/testify/require"
)

type fakeYouTubeImporter struct {
	mu        sync.Mutex
	once      sync.Once
	doneOnce  sync.Once
	calls     []mediaimport.YouTubeImportJob
	content   []byte
	filename  string
	mimeType  string
	err       error
	block     chan struct{}
	completed chan struct{}
}

type fakeMediaExtractor struct {
	content []byte
	err     error
	done    chan struct{}
}

func (f *fakeMediaExtractor) ExtractAudio(ctx context.Context, inputPath, outputPath string) error {
	defer func() {
		if f.done != nil {
			close(f.done)
		}
	}()
	if f.err != nil {
		return f.err
	}
	content := f.content
	if content == nil {
		content = []byte("extracted audio")
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outputPath, content, 0600)
}

func (f *fakeYouTubeImporter) Import(ctx context.Context, job mediaimport.YouTubeImportJob, onProgress mediaimport.ProgressFunc) (mediaimport.YouTubeImportResult, error) {
	f.mu.Lock()
	f.calls = append(f.calls, job)
	f.mu.Unlock()
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return mediaimport.YouTubeImportResult{}, ctx.Err()
		}
	}
	defer func() {
		if f.completed != nil {
			f.doneOnce.Do(func() { close(f.completed) })
		}
	}()
	if f.err != nil {
		return mediaimport.YouTubeImportResult{}, f.err
	}
	if onProgress != nil {
		onProgress(25)
	}
	content := f.content
	if content == nil {
		content = []byte("youtube audio")
	}
	if err := os.MkdirAll(filepath.Dir(job.OutputPath), 0755); err != nil {
		return mediaimport.YouTubeImportResult{}, err
	}
	if err := os.WriteFile(job.OutputPath, content, 0600); err != nil {
		return mediaimport.YouTubeImportResult{}, err
	}
	filename := f.filename
	if filename == "" {
		filename = "download.mp3"
	}
	mimeType := f.mimeType
	if mimeType == "" {
		mimeType = "audio/mpeg"
	}
	return mediaimport.YouTubeImportResult{Filename: filename, MimeType: mimeType}, nil
}

func (f *fakeYouTubeImporter) unblock() {
	if f.block != nil {
		f.once.Do(func() { close(f.block) })
	}
}

func (f *fakeYouTubeImporter) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

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

func TestVideoUploadExtractsAudioInBackground(t *testing.T) {
	s := newAuthTestServer(t)
	extractor := &fakeMediaExtractor{content: []byte("mp3 audio"), done: make(chan struct{})}
	s.handler.mediaExtractor = extractor
	token := registerForFileTests(t, s)

	resp, body := uploadMultipart(t, s, token, "file", "lecture.mp4", "video/mp4", []byte("video bytes"), "Lecture")
	require.Equal(t, http.StatusAccepted, resp.Code)
	fileID := body["id"].(string)
	require.Equal(t, "Lecture", body["title"])
	require.Equal(t, "video", body["kind"])
	require.Equal(t, "processing", body["status"])

	select {
	case <-extractor.done:
	case <-time.After(time.Second):
		t.Fatal("video extraction did not complete")
	}

	resp, body = s.request(t, http.MethodGet, "/api/v1/files/"+fileID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "ready", body["status"])
	require.Equal(t, "audio", body["kind"])
	require.Equal(t, "audio/mpeg", body["mime_type"])

	req, err := http.NewRequest(http.MethodGet, "/api/v1/files/"+fileID+"/audio", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	stream := httptest.NewRecorder()
	s.router.ServeHTTP(stream, req)
	require.Equal(t, http.StatusOK, stream.Code)
	require.Equal(t, []byte("mp3 audio"), stream.Body.Bytes())
}

func TestFileUploadSizeLimit(t *testing.T) {
	s := newAuthTestServer(t)
	s.handler.maxUploadBytes = 128
	token := registerForFileTests(t, s)

	resp, body := uploadMultipart(t, s, token, "file", "large.wav", "audio/wav", bytes.Repeat([]byte("x"), 1024), "")
	require.Equal(t, http.StatusRequestEntityTooLarge, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "PAYLOAD_TOO_LARGE", errBody["code"])
	require.Equal(t, "file", errBody["field"])
	require.NotContains(t, errBody["message"], s.uploadDir)
}

func TestYouTubeImportDownloadsWithFakeImporterAndStreamsResult(t *testing.T) {
	s := newAuthTestServer(t)
	importer := &fakeYouTubeImporter{content: []byte("ID3 youtube audio"), completed: make(chan struct{})}
	s.handler.youtubeImporter = importer
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
	fileID := body["id"].(string)

	select {
	case <-importer.completed:
	case <-time.After(time.Second):
		t.Fatal("youtube import did not complete")
	}

	resp, body = s.request(t, http.MethodGet, "/api/v1/files/"+fileID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "ready", body["status"])
	require.Equal(t, "youtube", body["kind"])
	require.Equal(t, "audio/mpeg", body["mime_type"])
	require.Equal(t, float64(len("ID3 youtube audio")), body["size_bytes"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/files", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, "youtube", items[0].(map[string]any)["kind"])

	req, err := http.NewRequest(http.MethodGet, "/api/v1/files/"+fileID+"/audio", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	stream := httptest.NewRecorder()
	s.router.ServeHTTP(stream, req)
	require.Equal(t, http.StatusOK, stream.Code)
	require.Equal(t, []byte("ID3 youtube audio"), stream.Body.Bytes())
	require.Equal(t, 1, importer.callCount())
}

func TestYouTubeImportFailureIsSanitizedAndPublishesFailedEvent(t *testing.T) {
	s := newAuthTestServer(t)
	importer := &fakeYouTubeImporter{err: errors.New("yt-dlp failed /tmp/private/raw-url"), completed: make(chan struct{})}
	s.handler.youtubeImporter = importer
	token := registerForFileTests(t, s)

	recorder, cancel, done := startEventStream(t, s, token, "/api/v1/events")
	resp, body := s.request(t, http.MethodPost, "/api/v1/files:import-youtube", map[string]any{
		"url":   "https://youtu.be/dQw4w9WgXcQ",
		"title": "Broken import",
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	fileID := body["id"].(string)

	select {
	case <-importer.completed:
	case <-time.After(time.Second):
		t.Fatal("youtube import did not fail")
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		resp, body = s.request(t, http.MethodGet, "/api/v1/files/"+fileID, nil, token, "")
		require.Equal(t, http.StatusOK, resp.Code)
		if body["status"] == "failed" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	require.Equal(t, "failed", body["status"])
	require.NotContains(t, body, "/tmp/private")
	stopEventStream(t, cancel, done)

	stream := recorder.Body.String()
	require.Contains(t, stream, "event: file.failed")
	require.Contains(t, stream, `"id":"`+fileID+`"`)
	require.NotContains(t, stream, "/tmp/private")
	require.NotContains(t, stream, "raw-url")
}

func TestYouTubeImportURLValidation(t *testing.T) {
	s := newAuthTestServer(t)
	s.handler.youtubeImporter = &fakeYouTubeImporter{completed: make(chan struct{})}
	token := registerForFileTests(t, s)

	for _, rawURL := range []string{
		"file:///etc/passwd",
		"https://example.com/video",
		"https://youtube.evil.test/watch?v=dQw4w9WgXcQ",
	} {
		resp, body := s.request(t, http.MethodPost, "/api/v1/files:import-youtube", map[string]any{
			"url": rawURL,
		}, token, "")
		require.Equal(t, http.StatusUnprocessableEntity, resp.Code, rawURL)
		errBody := body["error"].(map[string]any)
		require.Equal(t, "url", errBody["field"])
	}
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

	resp, body = s.request(t, http.MethodGet, "/api/v1/files?status=processing", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items = body["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, "processing", items[0].(map[string]any)["status"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/files?status=ready", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items = body["items"].([]any)
	require.Len(t, items, 3)
	for _, raw := range items {
		require.Equal(t, "ready", raw.(map[string]any)["status"])
	}

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
		"/api/v1/files?status=completed",
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
