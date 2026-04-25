package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecurityRegressionAuthRequiredRoutes(t *testing.T) {
	s := newAuthTestServer(t)

	cases := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/files"},
		{method: http.MethodGet, path: "/api/v1/transcriptions"},
		{method: http.MethodGet, path: "/api/v1/profiles"},
		{method: http.MethodGet, path: "/api/v1/settings"},
		{method: http.MethodGet, path: "/api/v1/events"},
		{method: http.MethodGet, path: "/api/v1/models/transcription"},
		{method: http.MethodGet, path: "/api/v1/admin/queue"},
		{method: http.MethodPost, path: "/api/v1/files:import-youtube"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			resp, body := s.request(t, tc.method, tc.path, nil, "", "")
			require.Equal(t, http.StatusUnauthorized, resp.Code)
			errBody := body["error"].(map[string]any)
			require.Equal(t, "UNAUTHORIZED", errBody["code"])
			require.NotEmpty(t, errBody["request_id"])
		})
	}
}

func TestSecurityRegressionMalformedInputsUseStableErrors(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		contentTyp string
		headerKey  string
		headerVal  string
		want       int
		field      string
		code       string
	}{
		{name: "malformed cursor", method: http.MethodGet, path: "/api/v1/files?cursor=not-a-cursor", want: http.StatusUnprocessableEntity, field: "cursor", code: "VALIDATION_ERROR"},
		{name: "bad idempotency key", method: http.MethodPost, path: "/api/v1/api-keys", body: `{"name":"cli"}`, contentTyp: "application/json", headerKey: "Idempotency-Key", headerVal: "bad key", want: http.StatusUnprocessableEntity, field: "Idempotency-Key", code: "VALIDATION_ERROR"},
		{name: "malformed json", method: http.MethodPatch, path: "/api/v1/settings", body: `{`, contentTyp: "application/json", want: http.StatusBadRequest, code: "INVALID_REQUEST"},
		{name: "malformed multipart", method: http.MethodPost, path: "/api/v1/files", body: "not multipart", contentTyp: "multipart/form-data; boundary=missing", want: http.StatusBadRequest, field: "file", code: "INVALID_REQUEST"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)
			if tc.contentTyp != "" {
				req.Header.Set("Content-Type", tc.contentTyp)
			}
			if tc.headerKey != "" {
				req.Header.Set(tc.headerKey, tc.headerVal)
			}

			recorder := httptest.NewRecorder()
			s.router.ServeHTTP(recorder, req)
			require.Equal(t, tc.want, recorder.Code)

			var body map[string]any
			require.NoError(t, json.NewDecoder(recorder.Body).Decode(&body))
			errBody := body["error"].(map[string]any)
			require.Equal(t, tc.code, errBody["code"])
			require.NotEmpty(t, errBody["request_id"])
			if tc.field != "" {
				require.Equal(t, tc.field, errBody["field"])
			}
			require.NotContains(t, errBody["message"], s.uploadDir)
		})
	}
}

func TestSecurityRegressionPathLeakageInRepresentativeErrors(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := uploadMultipart(t, s, token, "file", "../secret.wav", "audio/wav", []byte("RIFF----WAVEfmt data"), "")
	require.Equal(t, http.StatusCreated, resp.Code)
	require.NotContains(t, body["title"], "..")

	req, err := http.NewRequest(http.MethodGet, "/api/v1/files/"+body["id"].(string)+"/audio", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Range", "bytes=999-1000")
	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusRequestedRangeNotSatisfiable, recorder.Code)
	require.NotContains(t, recorder.Body.String(), s.uploadDir)

	resp, body = s.request(t, http.MethodGet, "/api/v1/files/file_missing/audio", nil, token, "")
	require.Equal(t, http.StatusNotFound, resp.Code)
	errBody := body["error"].(map[string]any)
	require.NotContains(t, errBody["message"], s.uploadDir)
}

func TestSecurityRegressionSSEDisconnectCleanup(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	_, cancel, done := startEventStream(t, s, token, "/api/v1/events")
	stopEventStream(t, cancel, done)
	require.Equal(t, 0, s.handler.events.subscriberCount())
}

func TestRepresentativeResponseShapes(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodGet, "/api/v1/files", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Contains(t, body, "items")
	require.Contains(t, body, "next_cursor")

	var upload bytes.Buffer
	writer := multipart.NewWriter(&upload)
	require.NoError(t, writer.Close())
	req, err := http.NewRequest(http.MethodPost, "/api/v1/files", &upload)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, decodeBody(t, recorder), "error")
}
