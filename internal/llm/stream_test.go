package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOpenAIStreamEmitsReasoningContentUsageAndDone(t *testing.T) {
	body := strings.Join([]string{
		`data: {"id":"chatcmpl-1","model":"qwen","choices":[{"index":0,"delta":{"reasoning_content":"thinking"},"finish_reason":""}]}`,
		`data: {"id":"chatcmpl-1","model":"qwen","choices":[{"index":0,"delta":{"content":"answer"},"finish_reason":""}]}`,
		`data: {"id":"chatcmpl-1","model":"qwen","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":5,"completion_tokens_details":{"reasoning_tokens":2},"total_tokens":8}}`,
		`data: [DONE]`,
		``,
	}, "\n")
	events := make(chan StreamEvent, 10)
	require.NoError(t, parseOpenAIStream(context.Background(), strings.NewReader(body), "req_123", events))
	close(events)

	collected := drainEvents(events)
	require.Len(t, collected, 4)
	assert.Equal(t, StreamEventReasoningDelta, collected[0].Type)
	assert.Equal(t, "thinking", collected[0].ReasoningDelta)
	assert.Equal(t, "req_123", collected[0].ProviderRequestID)
	assert.Equal(t, StreamEventContentDelta, collected[1].Type)
	assert.Equal(t, "answer", collected[1].ContentDelta)
	assert.Equal(t, StreamEventUsage, collected[2].Type)
	require.NotNil(t, collected[2].Usage)
	assert.Equal(t, 2, collected[2].Usage.ReasoningTokens)
	assert.Equal(t, StreamEventDone, collected[3].Type)
	assert.Equal(t, "stop", collected[3].FinishReason)
}

func TestOpenAIStreamEventsNormalizesProviderError(t *testing.T) {
	service := NewOpenAIService("secret-key", ptrString("https://example.test/v1"))
	service.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "Bearer secret-key", req.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"model unavailable","code":"model_not_found"}}`)),
		}, nil
	})}

	events, errors := service.ChatCompletionStreamEvents(context.Background(), "missing", []ChatMessage{{Role: "user", Content: "hi"}}, 0)
	assert.Empty(t, drainEvents(events))
	errs := drainErrors(errors)
	require.Len(t, errs, 1)
	var providerErr *ProviderError
	require.True(t, errorsAs(errs[0], &providerErr))
	assert.Equal(t, http.StatusUnprocessableEntity, providerErr.StatusCode)
	assert.Equal(t, "model_not_found", providerErr.Code)
	assert.Equal(t, "model unavailable", providerErr.Message)
	assert.NotContains(t, providerErr.Error(), "secret-key")
}

func TestParseOllamaStreamEmitsThinkingContentUsageAndDone(t *testing.T) {
	body := strings.Join([]string{
		`{"model":"qwen","message":{"role":"assistant","thinking":"thinking"},"done":false}`,
		`{"model":"qwen","message":{"role":"assistant","content":"answer"},"done":false}`,
		`{"model":"qwen","message":{"role":"assistant"},"done":true,"done_reason":"stop","prompt_eval_count":7,"eval_count":11}`,
		``,
	}, "\n")
	events := make(chan StreamEvent, 10)
	require.NoError(t, parseOllamaStream(context.Background(), strings.NewReader(body), events))
	close(events)

	collected := drainEvents(events)
	require.Len(t, collected, 4)
	assert.Equal(t, StreamEventReasoningDelta, collected[0].Type)
	assert.Equal(t, "thinking", collected[0].ReasoningDelta)
	assert.Equal(t, StreamEventContentDelta, collected[1].Type)
	assert.Equal(t, "answer", collected[1].ContentDelta)
	assert.Equal(t, StreamEventUsage, collected[2].Type)
	require.NotNil(t, collected[2].Usage)
	assert.Equal(t, 18, collected[2].Usage.TotalTokens)
	assert.Equal(t, StreamEventDone, collected[3].Type)
	assert.Equal(t, "stop", collected[3].FinishReason)
}

func TestLegacyStringStreamOnlyReturnsContentDeltas(t *testing.T) {
	service := NewOllamaService("https://example.test")
	service.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(strings.Join([]string{
				`{"model":"qwen","message":{"thinking":"hidden"},"done":false}`,
				`{"model":"qwen","message":{"content":"visible"},"done":false}`,
				`{"model":"qwen","done":true}`,
			}, "\n"))),
		}, nil
	})}

	content, errors := service.ChatCompletionStream(context.Background(), "qwen", []ChatMessage{{Role: "user", Content: "hi"}}, 0)
	assert.Equal(t, []string{"visible"}, drainStrings(content))
	assert.Empty(t, drainErrors(errors))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func ptrString(value string) *string {
	return &value
}

func drainEvents(events <-chan StreamEvent) []StreamEvent {
	var collected []StreamEvent
	for event := range events {
		collected = append(collected, event)
	}
	return collected
}

func drainStrings(values <-chan string) []string {
	var collected []string
	for value := range values {
		collected = append(collected, value)
	}
	return collected
}

func drainErrors(errors <-chan error) []error {
	var collected []error
	for err := range errors {
		if err != nil {
			collected = append(collected, err)
		}
	}
	return collected
}

func errorsAs(err error, target any) bool {
	return errors.As(err, target)
}
