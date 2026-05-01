package llm

import "context"

// Service is a provider-agnostic LLM interface
type Service interface {
	GetModels(ctx context.Context) ([]string, error)
	ChatCompletion(ctx context.Context, model string, messages []ChatMessage, temperature float64) (*ChatResponse, error)
	ChatCompletionStream(ctx context.Context, model string, messages []ChatMessage, temperature float64) (<-chan string, <-chan error)
	ChatCompletionStreamEvents(ctx context.Context, model string, messages []ChatMessage, temperature float64) (<-chan StreamEvent, <-chan error)
	GetContextWindow(ctx context.Context, model string) (int, error)
}

type StreamEventType string

const (
	StreamEventContentDelta   StreamEventType = "content_delta"
	StreamEventReasoningDelta StreamEventType = "reasoning_delta"
	StreamEventUsage          StreamEventType = "usage"
	StreamEventDone           StreamEventType = "done"
)

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	ReasoningTokens  int
	TotalTokens      int
}

type StreamEvent struct {
	Type              StreamEventType
	ContentDelta      string
	ReasoningDelta    string
	Usage             *TokenUsage
	FinishReason      string
	ProviderRequestID string
	Model             string
}

type ProviderError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}
