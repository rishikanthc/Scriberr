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
		client:  &http.Client{Timeout: 60 * time.Minute},
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
		Role      string `json:"role"`
		Content   string `json:"content"`
		Thinking  string `json:"thinking,omitempty"`
		Reasoning string `json:"reasoning,omitempty"`
	} `json:"message"`
	Done             bool   `json:"done"`
	DoneReason       string `json:"done_reason,omitempty"`
	PromptEvalCount  int    `json:"prompt_eval_count,omitempty"`
	EvalCount        int    `json:"eval_count,omitempty"`
	CompletionTokens int    `json:"completion_tokens,omitempty"`
	PromptTokens     int    `json:"prompt_tokens,omitempty"`
	TotalTokens      int    `json:"total_tokens,omitempty"`
	ReasoningTokens  int    `json:"reasoning_tokens,omitempty"`
}

// ChatCompletion performs a non-streaming chat completion against Ollama
func (s *OllamaService) ChatCompletion(ctx context.Context, model string, messages []ChatMessage, temperature float64) (*ChatResponse, error) {
	// Map to Ollama messages
	msgs := make([]ollamaChatMessage, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, ollamaChatMessage(m))
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
	events, errors := s.ChatCompletionStreamEvents(ctx, model, messages, temperature)

	go func() {
		defer close(contentChan)
		defer close(errorChan)
		for event := range events {
			if event.Type == StreamEventContentDelta && event.ContentDelta != "" {
				select {
				case contentChan <- event.ContentDelta:
				case <-ctx.Done():
					return
				}
			}
		}
		for err := range errors {
			if err != nil {
				errorChan <- err
				return
			}
		}
	}()

	return contentChan, errorChan
}

func (s *OllamaService) ChatCompletionStreamEvents(ctx context.Context, model string, messages []ChatMessage, temperature float64) (<-chan StreamEvent, <-chan error) {
	eventChan := make(chan StreamEvent, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(eventChan)
		defer close(errorChan)

		msgs := make([]ollamaChatMessage, 0, len(messages))
		for _, m := range messages {
			msgs = append(msgs, ollamaChatMessage(m))
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
			errorChan <- normalizeOllamaError(resp.StatusCode, body)
			return
		}

		if err := parseOllamaStream(ctx, resp.Body, eventChan); err != nil {
			errorChan <- err
		}
	}()

	return eventChan, errorChan
}

func parseOllamaStream(ctx context.Context, reader io.Reader, eventChan chan<- StreamEvent) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var chunk ollamaChatResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		for _, event := range ollamaChunkEvents(chunk) {
			if err := sendStreamEvent(ctx, eventChan, event); err != nil {
				return err
			}
		}
		if chunk.Done {
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}
	return nil
}

func ollamaChunkEvents(chunk ollamaChatResponse) []StreamEvent {
	events := make([]StreamEvent, 0, 4)
	if chunk.Message.Thinking != "" {
		events = append(events, StreamEvent{Type: StreamEventReasoningDelta, ReasoningDelta: chunk.Message.Thinking, Model: chunk.Model})
	}
	if chunk.Message.Reasoning != "" {
		events = append(events, StreamEvent{Type: StreamEventReasoningDelta, ReasoningDelta: chunk.Message.Reasoning, Model: chunk.Model})
	}
	if chunk.Message.Content != "" {
		events = append(events, StreamEvent{Type: StreamEventContentDelta, ContentDelta: chunk.Message.Content, Model: chunk.Model})
	}
	if hasOllamaUsage(chunk) {
		promptTokens := chunk.PromptTokens
		if promptTokens == 0 {
			promptTokens = chunk.PromptEvalCount
		}
		completionTokens := chunk.CompletionTokens
		if completionTokens == 0 {
			completionTokens = chunk.EvalCount
		}
		totalTokens := chunk.TotalTokens
		if totalTokens == 0 {
			totalTokens = promptTokens + completionTokens
		}
		events = append(events, StreamEvent{
			Type:  StreamEventUsage,
			Model: chunk.Model,
			Usage: &TokenUsage{
				PromptTokens:     promptTokens,
				CompletionTokens: completionTokens,
				ReasoningTokens:  chunk.ReasoningTokens,
				TotalTokens:      totalTokens,
			},
		})
	}
	if chunk.Done {
		events = append(events, StreamEvent{Type: StreamEventDone, FinishReason: chunk.DoneReason, Model: chunk.Model})
	}
	return events
}

func hasOllamaUsage(chunk ollamaChatResponse) bool {
	return chunk.PromptEvalCount != 0 || chunk.EvalCount != 0 || chunk.PromptTokens != 0 || chunk.CompletionTokens != 0 || chunk.TotalTokens != 0 || chunk.ReasoningTokens != 0
}

type ollamaErrorResponse struct {
	Error string `json:"error"`
}

func normalizeOllamaError(statusCode int, body []byte) error {
	var parsed ollamaErrorResponse
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error != "" {
		return &ProviderError{StatusCode: statusCode, Message: parsed.Error}
	}
	return &ProviderError{StatusCode: statusCode, Message: fmt.Sprintf("provider returned status %d", statusCode)}
}

// ollamaShowRequest represents the request to show model info
type ollamaShowRequest struct {
	Name string `json:"name"`
}

// ollamaShowResponse represents the response from show model info
type ollamaShowResponse struct {
	ModelInfo map[string]interface{} `json:"model_info"`
	Details   struct {
		ContextLength int `json:"context_length"` // Some versions return this
	} `json:"details"`
	Parameters string `json:"parameters"`
}

// GetContextWindow returns the context window size for a given Ollama model
func (s *OllamaService) GetContextWindow(ctx context.Context, model string) (int, error) {
	// Default to 4096 if we can't determine
	defaultContext := 4096

	reqBody := ollamaShowRequest{
		Name: model,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return defaultContext, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/show", bytes.NewBuffer(data))
	if err != nil {
		return defaultContext, nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return defaultContext, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return defaultContext, nil
	}

	var showResp ollamaShowResponse
	if err := json.NewDecoder(resp.Body).Decode(&showResp); err != nil {
		return defaultContext, nil
	}

	// Try to find context length in details
	// Note: Ollama API response format varies.
	// Sometimes it's in model_info -> llama.context_length
	// Sometimes it's in parameters string "num_ctx 8192"

	// Check model_info
	if showResp.ModelInfo != nil {
		for k, v := range showResp.ModelInfo {
			if strings.Contains(k, "context_length") {
				if f, ok := v.(float64); ok {
					fmt.Printf("Debug: Found context length in model_info: %f\n", f)
					return int(f), nil
				}
			}
		}
	}

	// Parse parameters string
	if showResp.Parameters != "" {
		lines := strings.Split(showResp.Parameters, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "num_ctx") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					var ctxLen int
					if _, err := fmt.Sscanf(parts[1], "%d", &ctxLen); err == nil {
						fmt.Printf("Debug: Found context length in parameters: %d\n", ctxLen)
						return ctxLen, nil
					}
				}
			}
		}
	}

	fmt.Printf("Debug: Ollama context window for model %s: %d (default: %d)\n", model, defaultContext, 4096)
	return defaultContext, nil
}
