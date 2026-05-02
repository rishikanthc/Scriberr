package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scriberr/internal/auth"
	"scriberr/internal/config"

	"github.com/stretchr/testify/require"
)

func TestCanonicalRouteRegistration(t *testing.T) {
	engine := SetupRoutes(NewHandler(&config.Config{Environment: "test"}, auth.NewAuthService("test-secret")), nil)

	registered := map[string]bool{}
	for _, route := range engine.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	expected := []string{
		"GET /health",
		"GET /api/v1/health",
		"GET /api/v1/ready",
		"GET /api/v1/auth/registration-status",
		"POST /api/v1/auth/register",
		"POST /api/v1/auth/login",
		"POST /api/v1/auth/refresh",
		"POST /api/v1/auth/logout",
		"GET /api/v1/auth/me",
		"POST /api/v1/auth/change-password",
		"POST /api/v1/auth/change-username",
		"GET /api/v1/api-keys",
		"POST /api/v1/api-keys",
		"DELETE /api/v1/api-keys/:id",
		"GET /api/v1/tags",
		"POST /api/v1/tags",
		"GET /api/v1/tags/:tag_id",
		"PATCH /api/v1/tags/:tag_id",
		"DELETE /api/v1/tags/:tag_id",
		"POST /api/v1/files",
		"GET /api/v1/files",
		"GET /api/v1/files/:id",
		"PATCH /api/v1/files/:id",
		"DELETE /api/v1/files/:id",
		"GET /api/v1/files/:id/audio",
		"POST /api/v1/recordings",
		"GET /api/v1/recordings",
		"GET /api/v1/recordings/:id",
		"PUT /api/v1/recordings/:id/chunks/:chunk_index",
		"POST /api/v1/transcriptions",
		"GET /api/v1/transcriptions",
		"GET /api/v1/transcriptions/:id",
		"PATCH /api/v1/transcriptions/:id",
		"DELETE /api/v1/transcriptions/:id",
		"GET /api/v1/transcriptions/:id/transcript",
		"GET /api/v1/transcriptions/:id/tags",
		"PUT /api/v1/transcriptions/:id/tags",
		"POST /api/v1/transcriptions/:id/tags/:tag_id",
		"DELETE /api/v1/transcriptions/:id/tags/:tag_id",
		"GET /api/v1/transcriptions/:id/annotations",
		"POST /api/v1/transcriptions/:id/annotations",
		"GET /api/v1/transcriptions/:id/annotations/:annotation_id",
		"PATCH /api/v1/transcriptions/:id/annotations/:annotation_id",
		"DELETE /api/v1/transcriptions/:id/annotations/:annotation_id",
		"POST /api/v1/transcriptions/:id/annotations/:annotation_id/entries",
		"PATCH /api/v1/transcriptions/:id/annotations/:annotation_id/entries/:entry_id",
		"DELETE /api/v1/transcriptions/:id/annotations/:annotation_id/entries/:entry_id",
		"GET /api/v1/transcriptions/:id/summary",
		"GET /api/v1/transcriptions/:id/audio",
		"GET /api/v1/transcriptions/:id/events",
		"GET /api/v1/transcriptions/:id/logs",
		"GET /api/v1/transcriptions/:id/executions",
		"GET /api/v1/profiles",
		"POST /api/v1/profiles",
		"GET /api/v1/profiles/:id",
		"PATCH /api/v1/profiles/:id",
		"DELETE /api/v1/profiles/:id",
		"POST /api/v1/profiles/:idAction",
		"GET /api/v1/settings",
		"PATCH /api/v1/settings",
		"GET /api/v1/settings/llm-provider",
		"PUT /api/v1/settings/llm-provider",
		"GET /api/v1/chat/models",
		"GET /api/v1/chat/sessions",
		"POST /api/v1/chat/sessions",
		"GET /api/v1/chat/sessions/:session_id",
		"PATCH /api/v1/chat/sessions/:session_id",
		"DELETE /api/v1/chat/sessions/:session_id",
		"GET /api/v1/chat/sessions/:session_id/context",
		"POST /api/v1/chat/sessions/:session_id/context/transcripts",
		"PATCH /api/v1/chat/sessions/:session_id/context/transcripts/:context_source_id",
		"DELETE /api/v1/chat/sessions/:session_id/context/transcripts/:context_source_id",
		"POST /api/v1/chat/sessions/:session_id/title:generate",
		"GET /api/v1/events",
		"GET /api/v1/models/transcription",
		"GET /api/v1/admin/queue",
	}
	for _, route := range expected {
		require.True(t, registered[route], "missing route %s", route)
	}

	for route := range registered {
		require.NotContains(t, route, "/api/v1/transcription/", "legacy singular transcription route must not be registered")
		require.NotContains(t, route, "/api/v1/transcription ", "legacy singular transcription route must not be registered")
	}
}

func TestEndpointContractSmoke(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	cases := []struct {
		name   string
		method string
		path   string
		body   any
		token  string
		want   int
	}{
		{name: "health", method: http.MethodGet, path: "/api/v1/health", want: http.StatusOK},
		{name: "me requires auth", method: http.MethodGet, path: "/api/v1/auth/me", want: http.StatusUnauthorized},
		{name: "me authenticated", method: http.MethodGet, path: "/api/v1/auth/me", token: token, want: http.StatusOK},
		{name: "api keys list", method: http.MethodGet, path: "/api/v1/api-keys", token: token, want: http.StatusOK},
		{name: "api keys create", method: http.MethodPost, path: "/api/v1/api-keys", body: map[string]any{"name": "contract"}, token: token, want: http.StatusCreated},
		{name: "files list", method: http.MethodGet, path: "/api/v1/files", token: token, want: http.StatusOK},
		{name: "files invalid cursor", method: http.MethodGet, path: "/api/v1/files?cursor=bad", token: token, want: http.StatusUnprocessableEntity},
		{name: "recordings list", method: http.MethodGet, path: "/api/v1/recordings", token: token, want: http.StatusOK},
		{name: "recordings invalid create", method: http.MethodPost, path: "/api/v1/recordings", body: map[string]any{"mime_type": "video/webm"}, token: token, want: http.StatusUnprocessableEntity},
		{name: "tags list", method: http.MethodGet, path: "/api/v1/tags", token: token, want: http.StatusOK},
		{name: "tags invalid create", method: http.MethodPost, path: "/api/v1/tags", body: map[string]any{"name": ""}, token: token, want: http.StatusUnprocessableEntity},
		{name: "transcriptions list", method: http.MethodGet, path: "/api/v1/transcriptions", token: token, want: http.StatusOK},
		{name: "transcriptions invalid sort", method: http.MethodGet, path: "/api/v1/transcriptions?sort=size", token: token, want: http.StatusUnprocessableEntity},
		{name: "transcriptions invalid tag match", method: http.MethodGet, path: "/api/v1/transcriptions?tag_match=both&tag=missing", token: token, want: http.StatusUnprocessableEntity},
		{name: "annotations invalid transcription", method: http.MethodGet, path: "/api/v1/transcriptions/tr_missing/annotations", token: token, want: http.StatusNotFound},
		{name: "annotations invalid kind", method: http.MethodGet, path: "/api/v1/transcriptions/tr_missing/annotations?kind=bookmark", token: token, want: http.StatusUnprocessableEntity},
		{name: "profiles list", method: http.MethodGet, path: "/api/v1/profiles", token: token, want: http.StatusOK},
		{name: "settings get", method: http.MethodGet, path: "/api/v1/settings", token: token, want: http.StatusOK},
		{name: "llm provider get", method: http.MethodGet, path: "/api/v1/settings/llm-provider", token: token, want: http.StatusOK},
		{name: "events stream requires auth", method: http.MethodGet, path: "/api/v1/events", want: http.StatusUnauthorized},
		{name: "models list", method: http.MethodGet, path: "/api/v1/models/transcription", token: token, want: http.StatusOK},
		{name: "queue stats", method: http.MethodGet, path: "/api/v1/admin/queue", token: token, want: http.StatusOK},
		{name: "youtube import", method: http.MethodPost, path: "/api/v1/files:import-youtube", body: map[string]any{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}, token: token, want: http.StatusAccepted},
		{name: "transcription submit malformed upload", method: http.MethodPost, path: "/api/v1/transcriptions:submit", token: token, want: http.StatusBadRequest},
		{name: "legacy list absent", method: http.MethodGet, path: "/api/v1/transcription/list", token: token, want: http.StatusNotFound},
		{name: "legacy upload absent", method: http.MethodPost, path: "/api/v1/transcription/upload", token: token, want: http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name+" "+tc.method+" "+tc.path, func(t *testing.T) {
			resp, body := s.request(t, tc.method, tc.path, tc.body, tc.token, "")
			require.Equal(t, tc.want, resp.Code)
			require.NotEmpty(t, resp.Header().Get("X-Request-ID"))
			if tc.want >= 400 {
				errBody := body["error"].(map[string]any)
				require.NotEmpty(t, errBody["code"])
				require.NotEmpty(t, errBody["message"])
				require.NotContains(t, errBody["message"], s.uploadDir)
			}
		})
	}
}

func TestAPIDocsContainOnlyCanonicalRoutes(t *testing.T) {
	docsPath := filepath.Join("..", "..", "docs", "api", "openapi.json")
	data, err := os.ReadFile(docsPath)
	require.NoError(t, err)

	var doc struct {
		OpenAPI string                    `json:"openapi"`
		Paths   map[string]map[string]any `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(data, &doc))
	require.NotEmpty(t, doc.OpenAPI)
	require.NotEmpty(t, doc.Paths)

	for path := range doc.Paths {
		require.True(t, strings.HasPrefix(path, "/api/v1/") || path == "/health", "unexpected path in API docs: %s", path)
		require.NotContains(t, path, "/api/v1/transcription/", "legacy singular transcription path must not be documented")
	}
	require.Contains(t, doc.Paths, "/api/v1/files")
	require.Contains(t, doc.Paths, "/api/v1/recordings")
	require.Contains(t, doc.Paths, "/api/v1/recordings/{id}/chunks/{chunk_index}")
	require.Contains(t, doc.Paths, "/api/v1/recordings/{id}:stop")
	require.Contains(t, doc.Paths, "/api/v1/recordings/{id}:cancel")
	require.Contains(t, doc.Paths, "/api/v1/recordings/{id}:retry-finalize")
	require.Contains(t, doc.Paths, "/api/v1/transcriptions")
	require.Contains(t, doc.Paths, "/api/v1/transcriptions/{id}/annotations")
	require.Contains(t, doc.Paths, "/api/v1/transcriptions/{id}/annotations/{annotation_id}")
	require.Contains(t, doc.Paths, "/api/v1/transcriptions/{id}/annotations/{annotation_id}/entries")
	require.Contains(t, doc.Paths, "/api/v1/transcriptions/{id}/annotations/{annotation_id}/entries/{entry_id}")
	require.Contains(t, doc.Paths, "/api/v1/profiles")
	require.Contains(t, doc.Paths, "/api/v1/settings")
	require.Contains(t, doc.Paths, "/api/v1/chat/models")
	require.Contains(t, doc.Paths, "/api/v1/chat/sessions")
	require.Contains(t, doc.Paths, "/api/v1/chat/sessions/{session_id}/context")
	require.Contains(t, doc.Paths, "/api/v1/chat/sessions/{session_id}/messages:stream")
	require.Contains(t, doc.Paths, "/api/v1/chat/runs/{run_id}:cancel")

	chatStream := doc.Paths["/api/v1/chat/sessions/{session_id}/messages:stream"]["post"].(map[string]any)
	require.Contains(t, chatStream["description"], "chat.delta.reasoning")
	require.Contains(t, chatStream["description"], "assistant_message.content")

	globalEvents := doc.Paths["/api/v1/events"]["get"].(map[string]any)
	globalEventResponses := globalEvents["responses"].(map[string]any)
	require.Contains(t, globalEventResponses, "200")
	require.NotContains(t, globalEventResponses, "501")

	transcriptionEvents := doc.Paths["/api/v1/transcriptions/{id}/events"]["get"].(map[string]any)
	transcriptionEventResponses := transcriptionEvents["responses"].(map[string]any)
	require.Contains(t, transcriptionEventResponses, "200")
	require.NotContains(t, transcriptionEventResponses, "501")

	filesList := doc.Paths["/api/v1/files"]["get"].(map[string]any)
	require.NotEmpty(t, filesList["parameters"], "files list docs must describe pagination/filter/sort query params")

	importYouTube := doc.Paths["/api/v1/files:import-youtube"]["post"].(map[string]any)
	require.NotEmpty(t, importYouTube["description"], "youtube import docs must describe async lifecycle")
}
