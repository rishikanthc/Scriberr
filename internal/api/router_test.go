package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"scriberr/internal/auth"
	"scriberr/internal/config"
	"scriberr/internal/models"

	"github.com/stretchr/testify/require"
)

func newTestRouter(t *testing.T, ready func() error) http.Handler {
	t.Helper()

	authService := auth.NewAuthService("test-secret")
	handler := NewHandler(&config.Config{
		Environment:    "test",
		AllowedOrigins: []string{"http://localhost:5173"},
	}, authService)
	handler.readinessCheck = ready

	return SetupRoutes(handler, authService)
}

func testToken(t *testing.T) string {
	t.Helper()

	token, err := auth.NewAuthService("test-secret").GenerateToken(&models.User{
		ID:       1,
		Username: "admin",
	})
	require.NoError(t, err)
	return token
}

func serveTestRequest(t *testing.T, router http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func decodeBody(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&body))
	return body
}

func TestHealthAndReadiness(t *testing.T) {
	router := newTestRouter(t, func() error { return nil })

	req, err := http.NewRequest(http.MethodGet, "/health", nil)
	require.NoError(t, err)
	resp := serveTestRequest(t, router, req)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, map[string]any{"status": "ok"}, decodeBody(t, resp))

	req, err = http.NewRequest(http.MethodGet, "/api/v1/health", nil)
	require.NoError(t, err)
	resp = serveTestRequest(t, router, req)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, map[string]any{"status": "ok"}, decodeBody(t, resp))

	req, err = http.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	require.NoError(t, err)
	resp = serveTestRequest(t, router, req)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, map[string]any{"database": "ok", "status": "ready"}, decodeBody(t, resp))
}

func TestRequestIDIsEchoedAndIncludedInErrors(t *testing.T) {
	router := newTestRouter(t, func() error { return nil })

	req, err := http.NewRequest(http.MethodGet, "/api/v1/files", nil)
	require.NoError(t, err)
	req.Header.Set("X-Request-ID", "req_test")

	resp := serveTestRequest(t, router, req)
	require.Equal(t, "req_test", resp.Header().Get("X-Request-ID"))
	require.Equal(t, http.StatusUnauthorized, resp.Code)

	body := decodeBody(t, resp)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "UNAUTHORIZED", errBody["code"])
	require.Equal(t, "req_test", errBody["request_id"])
}

func TestProtectedPlaceholderRouteUsesCanonicalErrorShape(t *testing.T) {
	router := newTestRouter(t, func() error { return nil })

	req, err := http.NewRequest(http.MethodGet, "/api/v1/files", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testToken(t))

	resp := serveTestRequest(t, router, req)
	require.Equal(t, http.StatusNotImplemented, resp.Code)

	body := decodeBody(t, resp)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "NOT_IMPLEMENTED", errBody["code"])
	require.NotEmpty(t, errBody["request_id"])
}

func TestAPINotFoundUsesCanonicalErrorShape(t *testing.T) {
	router := newTestRouter(t, func() error { return nil })

	req, err := http.NewRequest(http.MethodGet, "/api/v1/does-not-exist", nil)
	require.NoError(t, err)
	resp := serveTestRequest(t, router, req)
	require.Equal(t, http.StatusNotFound, resp.Code)

	body := decodeBody(t, resp)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "NOT_FOUND", errBody["code"])
	require.NotEmpty(t, errBody["request_id"])
}

func TestMalformedJSONUsesCanonicalErrorShape(t *testing.T) {
	router := newTestRouter(t, func() error { return nil })

	req, err := http.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("{"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp := serveTestRequest(t, router, req)
	require.Equal(t, http.StatusBadRequest, resp.Code)

	body := decodeBody(t, resp)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "INVALID_REQUEST", errBody["code"])
	require.NotEmpty(t, errBody["request_id"])
}
