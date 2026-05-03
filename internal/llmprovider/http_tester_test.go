package llmprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPConnectionTesterUsesOpenAICompatibleModelsAndAuthorization(t *testing.T) {
	var authHeader string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/models", r.URL.Path)
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1-mini"},{"id":"qwen2.5"}]}`))
	}))
	defer provider.Close()

	result, err := (HTTPConnectionTester{}).TestLLMProviderConnection(context.Background(), provider.URL, "sk-test-secret")

	require.NoError(t, err)
	require.Equal(t, "openai_compatible", result.Provider)
	require.Equal(t, provider.URL, result.BaseURL)
	require.Equal(t, []string{"gpt-4.1-mini", "qwen2.5"}, result.Models)
	require.Equal(t, "Bearer sk-test-secret", authHeader)
}

func TestHTTPConnectionTesterFallsBackToV1Models(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"llama"}]}`))
	}))
	defer provider.Close()

	result, err := (HTTPConnectionTester{}).TestLLMProviderConnection(context.Background(), provider.URL, "")

	require.NoError(t, err)
	require.Equal(t, "openai_compatible", result.Provider)
	require.Equal(t, provider.URL+"/v1", result.BaseURL)
	require.Equal(t, []string{"llama"}, result.Models)
}

func TestHTTPConnectionTesterSupportsOllamaNativeTags(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2:latest"}]}`))
	}))
	defer provider.Close()

	result, err := (HTTPConnectionTester{}).TestLLMProviderConnection(context.Background(), provider.URL, "")

	require.NoError(t, err)
	require.Equal(t, "ollama", result.Provider)
	require.Equal(t, provider.URL, result.BaseURL)
	require.Equal(t, []string{"llama3.2:latest"}, result.Models)
}

func TestHTTPConnectionTesterRejectsInvalidURL(t *testing.T) {
	_, err := (HTTPConnectionTester{}).TestLLMProviderConnection(context.Background(), "file:///tmp/models", "")
	require.ErrorContains(t, err, "base_url must")
}
