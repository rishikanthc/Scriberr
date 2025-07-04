package summary_tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OllamaClient handles communication with Ollama API
type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
}

// OllamaModel represents a model available in Ollama
type OllamaModel struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    struct {
		Format            string `json:"format"`
		Family            string `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string `json:"parameter_size"`
		QuantizationLevel string `json:"quantization_level"`
	} `json:"details"`
}

// OllamaModelsResponse represents the response from Ollama models endpoint
type OllamaModelsResponse struct {
	Models []OllamaModel `json:"models"`
}

// OllamaGenerateRequest represents a request to generate text
type OllamaGenerateRequest struct {
	Model    string   `json:"model"`
	Prompt   string   `json:"prompt"`
	Stream   bool     `json:"stream"`
	Options  *OllamaOptions `json:"options,omitempty"`
}

// OllamaOptions represents generation options for Ollama
type OllamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// OllamaGenerateResponse represents the response from Ollama generate endpoint
type OllamaGenerateResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	Context            []int     `json:"context,omitempty"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient() *OllamaClient {
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434" // Default Ollama URL
	}

	return &OllamaClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 300 * time.Second, // 5 minutes timeout for generation
		},
	}
}

// GetModels retrieves available models from Ollama
func (c *OllamaClient) GetModels(ctx context.Context) ([]OllamaModel, error) {
	url := fmt.Sprintf("%s/api/tags", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get models from Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp OllamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	return modelsResp.Models, nil
}

// GenerateText generates text using the specified Ollama model
func (c *OllamaClient) GenerateText(ctx context.Context, model, prompt string, options *OllamaOptions) (string, error) {
	url := fmt.Sprintf("%s/api/generate", c.baseURL)
	
	request := OllamaGenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
		Options: options,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to generate text from Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var generateResp OllamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&generateResp); err != nil {
		return "", fmt.Errorf("failed to decode generate response: %w", err)
	}

	return generateResp.Response, nil
}

// IsAvailable checks if Ollama is available and responding
func (c *OllamaClient) IsAvailable(ctx context.Context) bool {
	url := fmt.Sprintf("%s/api/tags", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
} 