package api

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/stretchr/testify/require"
)

func createUploadedFileForTranscription(t *testing.T, s *authTestServer, token string) (string, []byte) {
	t.Helper()

	content := []byte("RIFF----WAVEfmt transcription-source")
	resp, body := uploadMultipart(t, s, token, "file", "source.wav", "audio/wav", content, "Source audio")
	require.Equal(t, http.StatusCreated, resp.Code)
	return body["id"].(string), content
}

func TestTranscriptionSubmitUploadsFileAndQueuesTranscription(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "submit.wav")
	require.NoError(t, err)
	_, err = part.Write([]byte("RIFF----WAVEfmt submit-source"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("title", "Submitted transcript"))
	require.NoError(t, writer.WriteField("options", `{"language":"en","diarization":true}`))
	require.NoError(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, "/api/v1/transcriptions:submit", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusAccepted, recorder.Code)

	response := decodeBody(t, recorder)
	transcriptionID := response["id"].(string)
	fileID := response["file_id"].(string)
	require.True(t, strings.HasPrefix(transcriptionID, "tr_"))
	require.True(t, strings.HasPrefix(fileID, "file_"))
	require.Equal(t, "queued", response["status"])

	resp, list := s.request(t, http.MethodGet, "/api/v1/files", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, list["items"].([]any), 1)

	resp, transcription := s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, fileID, transcription["file_id"])
	require.Equal(t, "en", transcription["language"])
	require.Equal(t, true, transcription["diarization"])
}

func TestTranscriptionCreateListGetPatchCancelDelete(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": fileID,
		"title":   "Team sync transcript",
		"options": map[string]any{
			"language":    "en",
			"diarization": true,
		},
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	transcriptionID := body["id"].(string)
	require.True(t, strings.HasPrefix(transcriptionID, "tr_"))
	require.Equal(t, fileID, body["file_id"])
	require.Equal(t, "queued", body["status"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	require.Equal(t, transcriptionID, item["id"])
	require.Equal(t, fileID, item["file_id"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "Team sync transcript", body["title"])
	require.Equal(t, "en", body["language"])
	require.Equal(t, true, body["diarization"])

	resp, body = s.request(t, http.MethodPatch, "/api/v1/transcriptions/"+transcriptionID, map[string]any{"title": "Renamed transcript"}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "Renamed transcript", body["title"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+":cancel", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "canceled", body["status"])

	resp, _ = s.request(t, http.MethodDelete, "/api/v1/transcriptions/"+transcriptionID, nil, token, "")
	require.Equal(t, http.StatusNoContent, resp.Code)

	resp, _ = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID, nil, token, "")
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestTranscriptionCreateAppliesDefaultAndSelectedProfiles(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/profiles", map[string]any{
		"name":       "Default profile",
		"is_default": true,
		"options": map[string]any{
			"model":       "whisper-small",
			"language":    "fr",
			"diarization": true,
			"threads":     2,
		},
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	defaultProfileID := body["id"].(string)

	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": fileID,
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	defaultJobID := strings.TrimPrefix(body["id"].(string), "tr_")

	var defaultJob models.TranscriptionJob
	require.NoError(t, database.DB.First(&defaultJob, "id = ?", defaultJobID).Error)
	require.Equal(t, "whisper-small", defaultJob.Parameters.Model)
	require.Equal(t, 2, defaultJob.Parameters.Threads)
	require.NotNil(t, defaultJob.Parameters.Language)
	require.Equal(t, "fr", *defaultJob.Parameters.Language)
	require.True(t, defaultJob.Diarization)

	resp, body = s.request(t, http.MethodPost, "/api/v1/profiles", map[string]any{
		"name": "Selected profile",
		"options": map[string]any{
			"model":       "parakeet-v2",
			"language":    "es",
			"diarization": true,
			"threads":     4,
		},
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	selectedProfileID := body["id"].(string)
	require.NotEqual(t, defaultProfileID, selectedProfileID)

	disableDiarization := false
	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id":    fileID,
		"profile_id": selectedProfileID,
		"options": map[string]any{
			"language":    "en",
			"diarization": disableDiarization,
		},
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	selectedJobID := strings.TrimPrefix(body["id"].(string), "tr_")

	var selectedJob models.TranscriptionJob
	require.NoError(t, database.DB.First(&selectedJob, "id = ?", selectedJobID).Error)
	require.Equal(t, "parakeet-v2", selectedJob.Parameters.Model)
	require.Equal(t, 4, selectedJob.Parameters.Threads)
	require.NotNil(t, selectedJob.Parameters.Language)
	require.Equal(t, "en", *selectedJob.Parameters.Language)
	require.False(t, selectedJob.Diarization)
}

func TestTranscriptionValidationTranscriptRetryAndAudioAlias(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	fileID, content := createUploadedFileForTranscription(t, s, token)

	resp, _ := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": "file_missing",
	}, token, "")
	require.Equal(t, http.StatusNotFound, resp.Code)

	resp, _ = s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": fileID,
		"options": map[string]any{"language": "english"},
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id":    fileID,
		"profile_id": "profile_missing",
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "profile_id", errBody["field"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": fileID,
		"title":   "Transcript",
		"options": map[string]any{"language": "en"},
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	transcriptionID := body["id"].(string)

	var job models.TranscriptionJob
	require.NoError(t, database.DB.First(&job, "id = ?", strings.TrimPrefix(transcriptionID, "tr_")).Error)
	transcript := "hello world"
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", job.ID).Updates(map[string]any{
		"status":          models.StatusCompleted,
		"transcript_text": transcript,
	}).Error)

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/transcript", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, transcriptionID, body["transcription_id"])
	require.Equal(t, transcript, body["text"])
	require.Empty(t, body["segments"])
	require.Empty(t, body["words"])

	req, err := http.NewRequest(http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/audio", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Range", "bytes=0-3")
	audio := httptest.NewRecorder()
	s.router.ServeHTTP(audio, req)
	require.Equal(t, http.StatusPartialContent, audio.Code)
	require.Equal(t, content[:4], audio.Body.Bytes())

	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+":retry", nil, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	require.Equal(t, transcriptionID, body["source_transcription_id"])
	require.Equal(t, "queued", body["status"])
	require.True(t, strings.HasPrefix(body["id"].(string), "tr_"))
}

func TestTranscriptionListFiltersSortingPaginationAndValidation(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	fileID, _ := createUploadedFileForTranscription(t, s, token)
	transcriptions := []struct {
		title  string
		status models.JobStatus
	}{
		{title: "Alpha transcript", status: models.StatusCompleted},
		{title: "Bravo transcript", status: models.StatusPending},
		{title: "Charlie transcript", status: models.StatusFailed},
	}
	for _, transcription := range transcriptions {
		resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
			"file_id": fileID,
			"title":   transcription.title,
		}, token, "")
		require.Equal(t, http.StatusAccepted, resp.Code)
		id := strings.TrimPrefix(body["id"].(string), "tr_")
		require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", id).Update("status", transcription.status).Error)
	}

	resp, body := s.request(t, http.MethodGet, "/api/v1/transcriptions?status=completed&q=alpha", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, "Alpha transcript", items[0].(map[string]any)["title"])
	require.Equal(t, "completed", items[0].(map[string]any)["status"])

	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions?updated_after="+future, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Empty(t, body["items"].([]any))

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions?limit=2&sort=-title", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	firstPage := body["items"].([]any)
	require.Len(t, firstPage, 2)
	require.Equal(t, "Charlie transcript", firstPage[0].(map[string]any)["title"])
	require.Equal(t, "Bravo transcript", firstPage[1].(map[string]any)["title"])
	nextCursor, ok := body["next_cursor"].(string)
	require.True(t, ok)
	require.NotEmpty(t, nextCursor)

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions?limit=2&sort=-title&cursor="+nextCursor, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	secondPage := body["items"].([]any)
	require.Len(t, secondPage, 1)
	require.Equal(t, "Alpha transcript", secondPage[0].(map[string]any)["title"])
	require.Nil(t, body["next_cursor"])

	validationCases := []string{
		"/api/v1/transcriptions?limit=200",
		"/api/v1/transcriptions?status=uploaded",
		"/api/v1/transcriptions?sort=size",
		"/api/v1/transcriptions?updated_after=not-a-time",
		"/api/v1/transcriptions?cursor=not-a-cursor",
	}
	for _, path := range validationCases {
		resp, body := s.request(t, http.MethodGet, path, nil, token, "")
		require.Equal(t, http.StatusUnprocessableEntity, resp.Code, path)
		errBody := body["error"].(map[string]any)
		require.NotEmpty(t, errBody["field"])
	}
}
