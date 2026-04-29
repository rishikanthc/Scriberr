package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"scriberr/internal/auth"
	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/stretchr/testify/require"
)

func createTranscriptionForAnnotationTest(t *testing.T, s *authTestServer, token string) string {
	t.Helper()
	fileID, _ := createUploadedFileForTranscription(t, s, token)
	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": fileID,
		"title":   "Annotated transcript",
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	return body["id"].(string)
}

func TestAnnotationCreateListGetUpdateDelete(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	transcriptionID := createTranscriptionForAnnotationTest(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+"/annotations", map[string]any{
		"kind":    "note",
		"content": " Follow up ",
		"color":   " yellow ",
		"quote":   "quoted transcript text",
		"anchor": map[string]any{
			"start_ms":   1000,
			"end_ms":     2200,
			"start_word": 3,
			"end_word":   8,
			"text_hash":  "sha256:test",
		},
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	annotationID := body["id"].(string)
	require.True(t, strings.HasPrefix(annotationID, "ann_"))
	require.Equal(t, transcriptionID, body["transcription_id"])
	require.Equal(t, "note", body["kind"])
	require.Nil(t, body["content"])
	require.Equal(t, "yellow", body["color"])
	require.Equal(t, "stale", body["status"])
	entries := body["entries"].([]any)
	require.Len(t, entries, 1)
	firstEntry := entries[0].(map[string]any)
	require.True(t, strings.HasPrefix(firstEntry["id"].(string), "annent_"))
	require.Equal(t, annotationID, firstEntry["annotation_id"])
	require.Equal(t, "Follow up", firstEntry["content"])
	require.NotContains(t, firstEntry, "user_id")
	require.NotContains(t, body, "user_id")
	require.NotContains(t, body, "deleted_at")
	require.NotContains(t, body, "metadata_json")

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/annotations?kind=note&limit=1", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, annotationID, items[0].(map[string]any)["id"])
	require.Len(t, items[0].(map[string]any)["entries"].([]any), 1)
	require.Nil(t, body["next_cursor"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/annotations/"+annotationID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, annotationID, body["id"])

	resp, body = s.request(t, http.MethodPatch, "/api/v1/transcriptions/"+transcriptionID+"/annotations/"+annotationID, map[string]any{
		"quote": "updated quote",
		"anchor": map[string]any{
			"start_ms": 1100,
			"end_ms":   2300,
		},
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Nil(t, body["content"])
	require.Equal(t, "updated quote", body["quote"])
	require.Equal(t, float64(1100), body["anchor"].(map[string]any)["start_ms"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+"/annotations/"+annotationID+"/entries", map[string]any{
		"content": " second note ",
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	secondEntryID := body["id"].(string)
	require.True(t, strings.HasPrefix(secondEntryID, "annent_"))
	require.Equal(t, annotationID, body["annotation_id"])
	require.Equal(t, "second note", body["content"])

	resp, body = s.request(t, http.MethodPatch, "/api/v1/transcriptions/"+transcriptionID+"/annotations/"+annotationID+"/entries/"+secondEntryID, map[string]any{
		"content": " updated second note ",
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "updated second note", body["content"])

	resp, _ = s.request(t, http.MethodDelete, "/api/v1/transcriptions/"+transcriptionID+"/annotations/"+annotationID+"/entries/"+secondEntryID, nil, token, "")
	require.Equal(t, http.StatusNoContent, resp.Code)

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/annotations/"+annotationID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, body["entries"].([]any), 1)

	resp, _ = s.request(t, http.MethodDelete, "/api/v1/transcriptions/"+transcriptionID+"/annotations/"+annotationID, nil, token, "")
	require.Equal(t, http.StatusNoContent, resp.Code)

	resp, _ = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/annotations/"+annotationID, nil, token, "")
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestAnnotationValidationPaginationAndErrorEnvelope(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	transcriptionID := createTranscriptionForAnnotationTest(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+"/annotations", map[string]any{
		"kind":  "bookmark",
		"quote": "quoted text",
		"anchor": map[string]any{
			"start_ms": 10,
			"end_ms":   20,
		},
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errBody["code"])
	require.NotEmpty(t, errBody["request_id"])

	for i := 0; i < 2; i++ {
		resp, _ = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+"/annotations", map[string]any{
			"kind":  "highlight",
			"quote": "quoted text",
			"anchor": map[string]any{
				"start_ms": i * 100,
				"end_ms":   i*100 + 50,
			},
		}, token, "")
		require.Equal(t, http.StatusCreated, resp.Code)
	}

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/annotations?limit=1", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, body["items"].([]any), 1)
	nextCursor := body["next_cursor"].(string)
	require.NotEmpty(t, nextCursor)

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/annotations?limit=1&cursor="+nextCursor, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, body["items"].([]any), 1)

	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/annotations?updated_after="+future, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Empty(t, body["items"].([]any))

	resp, _ = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/annotations?cursor=bad", nil, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}

func TestAnnotationsAreScopedToAuthenticatedUser(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	transcriptionID := createTranscriptionForAnnotationTest(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+"/annotations", map[string]any{
		"kind":    "note",
		"content": "private note",
		"quote":   "quoted text",
		"anchor": map[string]any{
			"start_ms": 10,
			"end_ms":   20,
		},
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	annotationID := body["id"].(string)

	otherUser := models.User{Username: "annotation-other-user", Password: "pw"}
	require.NoError(t, database.DB.Create(&otherUser).Error)
	otherToken, err := auth.NewAuthService("test-secret").GenerateToken(&otherUser)
	require.NoError(t, err)

	resp, _ = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/annotations/"+annotationID, nil, otherToken, "")
	require.Equal(t, http.StatusNotFound, resp.Code)

	resp, _ = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/annotations", nil, otherToken, "")
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestAnnotationsStayScopedAcrossUsersWithDifferentTranscriptions(t *testing.T) {
	s := newAuthTestServer(t)
	firstToken := registerForFileTests(t, s)
	firstTranscriptionID := createTranscriptionForAnnotationTest(t, s, firstToken)

	secondUser := models.User{Username: "annotation-second-user", Password: "pw"}
	require.NoError(t, database.DB.Create(&secondUser).Error)
	secondToken, err := auth.NewAuthService("test-secret").GenerateToken(&secondUser)
	require.NoError(t, err)
	secondTranscriptionID := createTranscriptionForAnnotationTest(t, s, secondToken)

	resp, firstBody := s.request(t, http.MethodPost, "/api/v1/transcriptions/"+firstTranscriptionID+"/annotations", map[string]any{
		"kind":    "note",
		"content": "first user note",
		"quote":   "first quote",
		"anchor": map[string]any{
			"start_ms": 10,
			"end_ms":   20,
		},
	}, firstToken, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	firstAnnotationID := firstBody["id"].(string)

	resp, secondBody := s.request(t, http.MethodPost, "/api/v1/transcriptions/"+secondTranscriptionID+"/annotations", map[string]any{
		"kind":    "note",
		"content": "second user note",
		"quote":   "second quote",
		"anchor": map[string]any{
			"start_ms": 30,
			"end_ms":   40,
		},
	}, secondToken, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	secondAnnotationID := secondBody["id"].(string)

	resp, body := s.request(t, http.MethodGet, "/api/v1/transcriptions/"+firstTranscriptionID+"/annotations", nil, firstToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, firstAnnotationID, items[0].(map[string]any)["id"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+secondTranscriptionID+"/annotations", nil, secondToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items = body["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, secondAnnotationID, items[0].(map[string]any)["id"])

	resp, _ = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+secondTranscriptionID+"/annotations/"+firstAnnotationID, nil, secondToken, "")
	require.Equal(t, http.StatusNotFound, resp.Code)
	resp, _ = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+firstTranscriptionID+"/annotations/"+secondAnnotationID, nil, firstToken, "")
	require.Equal(t, http.StatusNotFound, resp.Code)
}
