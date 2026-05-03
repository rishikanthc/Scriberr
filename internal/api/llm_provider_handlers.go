package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"scriberr/internal/llmprovider"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
)

const llmProviderTimeout = 10 * time.Second

var llmProviderHTTPClient = &http.Client{Timeout: llmProviderTimeout}

type llmProviderTestResult struct {
	Provider string
	BaseURL  string
	Models   []string
}

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

type LLMProviderConnectionTester struct{}

func (LLMProviderConnectionTester) TestLLMProviderConnection(ctx context.Context, rawBaseURL, apiKey string) (llmprovider.TestResult, error) {
	result, err := testLLMProviderConnection(ctx, rawBaseURL, apiKey)
	if err != nil {
		return llmprovider.TestResult{}, err
	}
	return llmprovider.TestResult{Provider: result.Provider, BaseURL: result.BaseURL, Models: result.Models}, nil
}

func (h *Handler) getLLMProvider(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}

	result, err := h.llmProvider.Get(c.Request.Context(), user.ID)
	if errors.Is(err, llmprovider.ErrNotConfigured) {
		c.JSON(http.StatusOK, gin.H{
			"configured":  false,
			"provider":    "openai_compatible",
			"base_url":    "",
			"has_api_key": false,
			"key_preview": nil,
			"model_count": 0,
			"models":      []string{},
			"large_model": nil,
			"small_model": nil,
		})
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load LLM provider", nil)
		return
	}
	if result.ConnectionError != nil {
		response := llmProviderResponse(result.Config, nil)
		response["connection_error"] = result.ConnectionError.Error()
		c.JSON(http.StatusOK, response)
		return
	}
	c.JSON(http.StatusOK, llmProviderResponse(result.Config, result.Models))
}

func (h *Handler) updateLLMProvider(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}

	var req updateLLMProviderRequest
	if !bindJSON(c, &req) {
		return
	}

	if strings.TrimSpace(req.BaseURL) == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "base_url is required", stringPtr("base_url"))
		return
	}
	result, err := h.llmProvider.Save(c.Request.Context(), user.ID, llmprovider.SaveRequest{
		BaseURL:    req.BaseURL,
		APIKey:     req.APIKey,
		LargeModel: req.LargeModel,
		SmallModel: req.SmallModel,
	})
	if errors.Is(err, llmprovider.ErrLargeModelUnavailable) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "large_model is not available from this provider", stringPtr("large_model"))
		return
	}
	if errors.Is(err, llmprovider.ErrSmallModelUnavailable) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "small_model is not available from this provider", stringPtr("small_model"))
		return
	}
	if err != nil {
		writeError(c, http.StatusUnprocessableEntity, "PROVIDER_CONNECTION_FAILED", err.Error(), stringPtr("base_url"))
		return
	}

	response := llmProviderResponse(result.Config, result.Models)
	h.publishEvent("settings.updated", gin.H{"llm_provider_configured": true})
	c.JSON(http.StatusOK, response)
}

func testLLMProviderConnection(ctx context.Context, rawBaseURL, apiKey string) (llmProviderTestResult, error) {
	candidates, err := llmProviderCandidates(rawBaseURL)
	if err != nil {
		return llmProviderTestResult{}, err
	}

	var lastErr error
	for _, candidate := range candidates {
		models, err := fetchOpenAICompatibleModels(ctx, candidate, apiKey)
		if err == nil {
			return llmProviderTestResult{Provider: "openai_compatible", BaseURL: candidate, Models: models}, nil
		}
		lastErr = err
	}

	if ollamaBaseURL, ok := ollamaNativeCandidate(rawBaseURL); ok {
		models, err := fetchOllamaNativeModels(ctx, ollamaBaseURL)
		if err == nil {
			return llmProviderTestResult{Provider: "ollama", BaseURL: ollamaBaseURL, Models: models}, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return llmProviderTestResult{}, fmt.Errorf("could not list models from provider: %w", lastErr)
	}
	return llmProviderTestResult{}, fmt.Errorf("could not list models from provider")
}

func llmProviderCandidates(rawBaseURL string) ([]string, error) {
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

	resp, err := llmProviderHTTPClient.Do(req)
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

	resp, err := llmProviderHTTPClient.Do(req)
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

func llmProviderResponse(config *models.LLMConfig, models []string) gin.H {
	baseURL := llmprovider.BaseURL(config)
	hasKey := config.APIKey != nil && strings.TrimSpace(*config.APIKey) != ""
	var keyPreviewValue any
	if hasKey {
		keyPreviewValue = keyPreview(*config.APIKey)
	}
	if models == nil {
		models = []string{}
	}
	return gin.H{
		"configured":  true,
		"provider":    config.Provider,
		"base_url":    baseURL,
		"has_api_key": hasKey,
		"key_preview": keyPreviewValue,
		"model_count": len(models),
		"models":      models,
		"large_model": config.LargeModel,
		"small_model": config.SmallModel,
		"updated_at":  config.UpdatedAt,
	}
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

func llmProviderBaseURL(config *models.LLMConfig) string {
	return llmprovider.BaseURL(config)
}

func stringInSlice(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
