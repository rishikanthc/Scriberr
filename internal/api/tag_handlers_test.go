package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func createTranscriptionForTagTest(t *testing.T, s *authTestServer, token string, title string) string {
	t.Helper()
	fileID, _ := createUploadedFileForTranscription(t, s, token)
	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": fileID,
		"title":   title,
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	return body["id"].(string)
}

func TestTagCreateListGetUpdateDelete(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodPost, "/api/v1/tags", map[string]any{
		"name":        " Client Call ",
		"color":       "#E87539",
		"description": " customer calls ",
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	tagID := body["id"].(string)
	require.True(t, strings.HasPrefix(tagID, "tag_"))
	require.Equal(t, "Client Call", body["name"])
	require.Equal(t, "#E87539", body["color"])
	require.Equal(t, "customer calls", body["description"])
	require.NotContains(t, body, "normalized_name")
	require.NotContains(t, body, "user_id")
	require.NotContains(t, body, "deleted_at")
	require.NotContains(t, body, "metadata_json")

	resp, body = s.request(t, http.MethodPost, "/api/v1/tags", map[string]any{"name": "client   call"}, token, "")
	require.Equal(t, http.StatusConflict, resp.Code)
	require.Equal(t, "CONFLICT", body["error"].(map[string]any)["code"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/tags?q=client&limit=1", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, tagID, items[0].(map[string]any)["id"])
	require.Nil(t, body["next_cursor"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/tags/"+tagID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, tagID, body["id"])

	resp, body = s.request(t, http.MethodPatch, "/api/v1/tags/"+tagID, map[string]any{
		"name":  "Customer Call",
		"color": "",
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "Customer Call", body["name"])
	require.Nil(t, body["color"])

	resp, _ = s.request(t, http.MethodDelete, "/api/v1/tags/"+tagID, nil, token, "")
	require.Equal(t, http.StatusNoContent, resp.Code)

	resp, _ = s.request(t, http.MethodGet, "/api/v1/tags/"+tagID, nil, token, "")
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestTranscriptionTagsAndTagFilteredList(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	firstTranscriptionID := createTranscriptionForTagTest(t, s, token, "Research meeting")
	secondTranscriptionID := createTranscriptionForTagTest(t, s, token, "Sales call")

	resp, body := s.request(t, http.MethodPost, "/api/v1/tags", map[string]any{"name": "Research"}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	researchTagID := body["id"].(string)
	resp, body = s.request(t, http.MethodPost, "/api/v1/tags", map[string]any{"name": "Meeting"}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	meetingTagID := body["id"].(string)

	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+firstTranscriptionID+"/tags/"+researchTagID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, body["items"].([]any), 1)

	resp, body = s.request(t, http.MethodPut, "/api/v1/transcriptions/"+firstTranscriptionID+"/tags", map[string]any{
		"tag_ids": []string{researchTagID, meetingTagID},
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, body["items"].([]any), 2)

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+firstTranscriptionID+"/tags", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, body["items"].([]any), 2)

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions?tags=Research,"+meetingTagID+"&tag_match=all", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, firstTranscriptionID, items[0].(map[string]any)["id"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions?tag="+researchTagID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items = body["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, firstTranscriptionID, items[0].(map[string]any)["id"])
	require.NotEqual(t, secondTranscriptionID, items[0].(map[string]any)["id"])

	resp, _ = s.request(t, http.MethodDelete, "/api/v1/transcriptions/"+firstTranscriptionID+"/tags/"+researchTagID, nil, token, "")
	require.Equal(t, http.StatusNoContent, resp.Code)

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions?tags=Research,"+meetingTagID+"&tag_match=all", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Empty(t, body["items"].([]any))
}

func TestTagValidationAndErrorEnvelope(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	transcriptionID := createTranscriptionForTagTest(t, s, token, "Private transcript")

	resp, body := s.request(t, http.MethodPost, "/api/v1/tags", map[string]any{"name": "", "color": "orange"}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	require.Equal(t, "VALIDATION_ERROR", body["error"].(map[string]any)["code"])
	require.NotEmpty(t, body["error"].(map[string]any)["request_id"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/tags", map[string]any{"name": "Private"}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	tagID := body["id"].(string)

	resp, _ = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+"/tags/tag_missing", nil, token, "")
	require.Equal(t, http.StatusNotFound, resp.Code)

	resp, _ = s.request(t, http.MethodGet, "/api/v1/transcriptions?tag_match=both&tag="+tagID, nil, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}
