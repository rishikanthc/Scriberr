package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaService handles Ollama API interactions
type OllamaService struct {
	baseURL string
	client  *http.Client
}

// NewOllamaService creates a new Ollama service
func NewOllamaService(baseURL string) *OllamaService {
	// Normalize base URL: remove trailing slash
	b := strings.TrimRight(baseURL, "/")
	return &OllamaService{
		baseURL: b,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Ollama tags response
type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// GetModels retrieves available chat models from Ollama
func (s *OllamaService) GetModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var tags ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	out := make([]string, 0, len(tags.Models))
	for _, m := range tags.Models {
		if m.Name != "" {
			out = append(out, m.Name)
		}
	}
	return out, nil
}

// Ollama chat API payloads
type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	Options  map[string]any      `json:"options,omitempty"`
}

type ollamaChatResponse struct {
	Model   string `json:"model"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

// ChatCompletion performs a non-streaming chat completion against Ollama
func (s *OllamaService) ChatCompletion(ctx context.Context, model string, messages []ChatMessage, temperature float64) (*ChatResponse, error) {
	// Map to Ollama messages
	msgs := make([]ollamaChatMessage, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, ollamaChatMessage{Role: m.Role, Content: m.Content})
	}
	reqBody := ollamaChatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   false,
	}
	if temperature > 0 {
		reqBody.Options = map[string]any{"temperature": temperature}
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/chat", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}
	var oResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&oResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	// Map to generic ChatResponse
	cr := &ChatResponse{Model: oResp.Model}
	cr.Choices = []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	}{{
		Index: 0,
	}}
	cr.Choices[0].Message.Role = oResp.Message.Role
	cr.Choices[0].Message.Content = oResp.Message.Content
	return cr, nil
}

// ChatCompletionStream performs a streaming chat completion against Ollama
func (s *OllamaService) ChatCompletionStream(ctx context.Context, model string, messages []ChatMessage, temperature float64) (<-chan string, <-chan error) {
	contentChan := make(chan string, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errorChan)

		msgs := make([]ollamaChatMessage, 0, len(messages))
		for _, m := range messages {
			msgs = append(msgs, ollamaChatMessage{Role: m.Role, Content: m.Content})
		}
		reqBody := ollamaChatRequest{Model: model, Messages: msgs, Stream: true}
		if temperature > 0 {
			reqBody.Options = map[string]any{"temperature": temperature}
		}

		data, err := json.Marshal(reqBody)
		if err != nil {
			errorChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}
		req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/chat", bytes.NewBuffer(data))
		if err != nil {
			errorChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			errorChan <- fmt.Errorf("failed to make request: %w", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errorChan <- fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if ctx.Err() != nil {
				return
			}
			line := scanner.Text()
			// Ollama streams JSON objects per line
			if strings.TrimSpace(line) == "" {
				continue
			}
			var chunk ollamaChatResponse
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				continue
			}
			if chunk.Message.Content != "" {
				select {
				case contentChan <- chunk.Message.Content:
				case <-ctx.Done():
					return
				}
			}
			if chunk.Done {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			errorChan <- fmt.Errorf("error reading stream: %w", err)
		}
	}()

	return contentChan, errorChan
}
