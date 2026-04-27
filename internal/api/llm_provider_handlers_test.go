package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/stretchr/testify/require"
)

func TestLLMProviderSettingsEmptyAndAuth(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, _ := s.request(t, http.MethodGet, "/api/v1/settings/llm-provider", nil, "", "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)

	resp, body := s.request(t, http.MethodGet, "/api/v1/settings/llm-provider", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, false, body["configured"])
	require.Equal(t, "openai_compatible", body["provider"])
	require.Equal(t, false, body["has_api_key"])
}

func TestLLMProviderSettingsSaveTestsConnectionAndMasksKey(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	var authHeader string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/models", r.URL.Path)
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1-mini"},{"id":"qwen2.5"}]}`))
	}))
	defer provider.Close()

	resp, body := s.request(t, http.MethodPut, "/api/v1/settings/llm-provider", map[string]any{
		"base_url": provider.URL,
		"api_key":  "sk-test-secret",
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, true, body["configured"])
	require.Equal(t, "openai_compatible", body["provider"])
	require.Equal(t, provider.URL, body["base_url"])
	require.Equal(t, true, body["has_api_key"])
	require.Equal(t, "sk-t...cret", body["key_preview"])
	require.Equal(t, float64(2), body["model_count"])
	require.NotContains(t, resp.Body.String(), "sk-test-secret")
	require.Equal(t, "Bearer sk-test-secret", authHeader)

	var stored models.LLMConfig
	require.NoError(t, database.DB.First(&stored).Error)
	require.Equal(t, "openai_compatible", stored.Provider)
	require.NotNil(t, stored.APIKey)
	require.Equal(t, "sk-test-secret", *stored.APIKey)

	resp, body = s.request(t, http.MethodGet, "/api/v1/settings/llm-provider", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, true, body["configured"])
	require.Equal(t, true, body["has_api_key"])
	require.NotContains(t, resp.Body.String(), "sk-test-secret")
}

func TestLLMProviderSettingsFallsBackToV1Models(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"llama"}]}`))
	}))
	defer provider.Close()

	resp, body := s.request(t, http.MethodPut, "/api/v1/settings/llm-provider", map[string]any{
		"base_url": provider.URL,
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, provider.URL+"/v1", body["base_url"])
	require.Equal(t, float64(1), body["model_count"])
}

func TestLLMProviderSettingsSupportsOllamaNativeTags(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2:latest"}]}`))
	}))
	defer provider.Close()

	resp, body := s.request(t, http.MethodPut, "/api/v1/settings/llm-provider", map[string]any{
		"base_url": provider.URL,
	}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "ollama", body["provider"])
	require.Equal(t, provider.URL, body["base_url"])
	require.Equal(t, float64(1), body["model_count"])
}

func TestLLMProviderSettingsValidationAndConnectionFailure(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodPut, "/api/v1/settings/llm-provider", map[string]any{
		"base_url": "file:///tmp/models",
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, "base_url", errBody["field"])

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no models", http.StatusBadGateway)
	}))
	defer provider.Close()

	resp, body = s.request(t, http.MethodPut, "/api/v1/settings/llm-provider", map[string]any{
		"base_url": provider.URL,
	}, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	errBody = body["error"].(map[string]any)
	require.Equal(t, "PROVIDER_CONNECTION_FAILED", errBody["code"])
	require.Equal(t, "base_url", errBody["field"])

	var count int64
	require.NoError(t, database.DB.Model(&models.LLMConfig{}).Count(&count).Error)
	require.Equal(t, int64(0), count)
	require.False(t, strings.Contains(resp.Body.String(), provider.URL+"/models"))
}
