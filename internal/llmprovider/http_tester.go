package llmprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const providerProbeTimeout = 10 * time.Second

var providerProbeHTTPClient = &http.Client{Timeout: providerProbeTimeout}

type HTTPConnectionTester struct{}

type llmModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func (HTTPConnectionTester) TestLLMProviderConnection(ctx context.Context, rawBaseURL, apiKey string) (TestResult, error) {
	candidates, err := providerCandidates(rawBaseURL)
	if err != nil {
		return TestResult{}, err
	}

	var lastErr error
	for _, candidate := range candidates {
		models, err := fetchOpenAICompatibleModels(ctx, candidate, apiKey)
		if err == nil {
			return TestResult{Provider: "openai_compatible", BaseURL: candidate, Models: models}, nil
		}
		lastErr = err
	}

	if ollamaBaseURL, ok := ollamaNativeCandidate(rawBaseURL); ok {
		models, err := fetchOllamaNativeModels(ctx, ollamaBaseURL)
		if err == nil {
			return TestResult{Provider: "ollama", BaseURL: ollamaBaseURL, Models: models}, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return TestResult{}, fmt.Errorf("could not list models from provider: %w", lastErr)
	}
	return TestResult{}, fmt.Errorf("could not list models from provider")
}

func providerCandidates(rawBaseURL string) ([]string, error) {
	normalized, parsed, err := normalizeProviderURL(rawBaseURL)
	if err != nil {
		return nil, err
	}
	candidates := []string{normalized}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	if path == "" {
		withV1 := *parsed
		withV1.Path = "/v1"
		candidates = append(candidates, strings.TrimRight(withV1.String(), "/"))
	}
	return uniqueStrings(candidates), nil
}

func ollamaNativeCandidate(rawBaseURL string) (string, bool) {
	normalized, parsed, err := normalizeProviderURL(rawBaseURL)
	if err != nil {
		return "", false
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	if path == "/v1" {
		withoutV1 := *parsed
		withoutV1.Path = ""
		normalized = strings.TrimRight(withoutV1.String(), "/")
	}
	if strings.Contains(strings.ToLower(parsed.Host), "11434") || path == "" || path == "/v1" {
		return normalized, true
	}
	return "", false
}

func normalizeProviderURL(rawBaseURL string) (string, *url.URL, error) {
	value := strings.TrimSpace(rawBaseURL)
	if value == "" {
		return "", nil, fmt.Errorf("base_url is required")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", nil, fmt.Errorf("base_url must be an absolute http or https URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", nil, fmt.Errorf("base_url must use http or https")
	}
	if parsed.User != nil {
		return "", nil, fmt.Errorf("base_url must not include credentials")
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return strings.TrimRight(parsed.String(), "/"), parsed, nil
}

func fetchOpenAICompatibleModels(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := providerProbeHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("models endpoint returned HTTP %d", resp.StatusCode)
	}

	var data llmModelsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return nil, fmt.Errorf("models endpoint returned invalid JSON")
	}
	models := make([]string, 0, len(data.Data))
	for _, model := range data.Data {
		if strings.TrimSpace(model.ID) != "" {
			models = append(models, model.ID)
		}
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("models endpoint returned no models")
	}
	return models, nil
}

func fetchOllamaNativeModels(ctx context.Context, baseURL string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := providerProbeHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("ollama tags endpoint returned HTTP %d", resp.StatusCode)
	}

	var data ollamaTagsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return nil, fmt.Errorf("ollama tags endpoint returned invalid JSON")
	}
	models := make([]string, 0, len(data.Models))
	for _, model := range data.Models {
		if strings.TrimSpace(model.Name) != "" {
			models = append(models, model.Name)
		}
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("ollama tags endpoint returned no models")
	}
	return models, nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
