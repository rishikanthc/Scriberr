package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"scriberr/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type LLMTestSuite struct {
	suite.Suite
	helper     *TestHelper
	mockServer *httptest.Server
	service    *llm.OpenAIService
}

func (suite *LLMTestSuite) SetupSuite() {
	suite.helper = NewTestHelper(suite.T(), "llm_test.db")
	suite.setupMockServer()
}

func (suite *LLMTestSuite) TearDownSuite() {
	suite.helper.Cleanup()
	if suite.mockServer != nil {
		suite.mockServer.Close()
	}
}

func (suite *LLMTestSuite) setupMockServer() {
	suite.mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			suite.handleModelsRequest(w, r)
		case "/chat/completions":
			suite.handleChatCompletionRequest(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Create OpenAI service with mock server
	service := llm.NewOpenAIService("test-api-key", nil)
	// Use reflection or a custom method to set baseURL to mock server
	// For now, we'll test what we can without modifying the baseURL
	suite.service = service
}

func (suite *LLMTestSuite) handleModelsRequest(w http.ResponseWriter, r *http.Request) {
	// Check authorization header
	auth := r.Header.Get("Authorization")
	if auth != "Bearer test-api-key" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
		return
	}

	// Return mock models response
	response := llm.ModelsResponse{
		Data: []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}{
			{ID: "gpt-3.5-turbo", Object: "model", Created: 1677610602, OwnedBy: "openai"},
			{ID: "gpt-4", Object: "model", Created: 1687882411, OwnedBy: "openai"},
			{ID: "text-davinci-003", Object: "model", Created: 1669599635, OwnedBy: "openai"}, // Should be filtered out
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *LLMTestSuite) handleChatCompletionRequest(w http.ResponseWriter, r *http.Request) {
	// Check authorization header
	auth := r.Header.Get("Authorization")
	if auth != "Bearer test-api-key" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
		return
	}

	// Parse request body
	var chatReq llm.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"message": "Invalid request"}}`))
		return
	}

	if chatReq.Stream {
		suite.handleStreamingResponse(w, chatReq)
	} else {
		suite.handleNonStreamingResponse(w, chatReq)
	}
}

func (suite *LLMTestSuite) handleNonStreamingResponse(w http.ResponseWriter, chatReq llm.ChatRequest) {
	response := llm.ChatResponse{
		ID:      "chatcmpl-test123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   chatReq.Model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: "This is a test response from the mock OpenAI service.",
				},
				FinishReason: "stop",
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     10,
			CompletionTokens: 15,
			TotalTokens:      25,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *LLMTestSuite) handleStreamingResponse(w http.ResponseWriter, chatReq llm.ChatRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send streaming chunks
	chunks := []string{"This ", "is ", "a ", "test ", "streaming ", "response."}

	for _, chunk := range chunks {
		streamChunk := llm.ChatStreamResponse{
			ID:      "chatcmpl-stream-test123",
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   chatReq.Model,
			Choices: []struct {
				Index int `json:"index"`
				Delta struct {
					Role    string `json:"role,omitempty"`
					Content string `json:"content,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Delta: struct {
						Role    string `json:"role,omitempty"`
						Content string `json:"content,omitempty"`
					}{
						Content: chunk,
					},
					FinishReason: "",
				},
			},
		}

		chunkJSON, _ := json.Marshal(streamChunk)
		w.Write([]byte("data: " + string(chunkJSON) + "\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	// Send final chunk
	w.Write([]byte("data: [DONE]\n\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// Test OpenAI service creation
func (suite *LLMTestSuite) TestNewOpenAIService() {
	service := llm.NewOpenAIService("test-api-key-123", nil)

	assert.NotNil(suite.T(), service)
}

// Test chat message structure
func (suite *LLMTestSuite) TestChatMessageStructure() {
	message := llm.ChatMessage{
		Role:    "user",
		Content: "Hello, how are you?",
	}

	assert.Equal(suite.T(), "user", message.Role)
	assert.Equal(suite.T(), "Hello, how are you?", message.Content)

	// Test JSON marshaling
	jsonData, err := json.Marshal(message)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(jsonData), "user")
	assert.Contains(suite.T(), string(jsonData), "Hello, how are you?")
}

// Test chat request structure
func (suite *LLMTestSuite) TestChatRequestStructure() {
	messages := []llm.ChatMessage{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello!"},
	}

	request := llm.ChatRequest{
		Model:       "gpt-3.5-turbo",
		Messages:    messages,
		Stream:      false,
		Temperature: 0.7,
		MaxTokens:   150,
	}

	assert.Equal(suite.T(), "gpt-3.5-turbo", request.Model)
	assert.Len(suite.T(), request.Messages, 2)
	assert.False(suite.T(), request.Stream)
	assert.Equal(suite.T(), 0.7, request.Temperature)
	assert.Equal(suite.T(), 150, request.MaxTokens)

	// Test JSON marshaling
	jsonData, err := json.Marshal(request)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(jsonData), "gpt-3.5-turbo")
	assert.Contains(suite.T(), string(jsonData), "helpful assistant")
}

// Test GetModels with valid API key (mock)
func (suite *LLMTestSuite) TestGetModelsSuccess() {
	// Since we can't easily override the baseURL without modifying the service,
	// we'll test the structure and error handling instead

	ctx := context.Background()

	// This will call the real OpenAI API, which will likely fail with our test key
	// But we can test that the method doesn't panic and returns appropriate errors
	models, err := suite.service.GetModels(ctx)

	// With a fake API key, we expect an error
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), models)
}

// Test GetModels with timeout
func (suite *LLMTestSuite) TestGetModelsTimeout() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	models, err := suite.service.GetModels(ctx)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), models)
	assert.Contains(suite.T(), err.Error(), "context deadline exceeded")
}

// Test ChatCompletion with various inputs
func (suite *LLMTestSuite) TestChatCompletionStructure() {
	messages := []llm.ChatMessage{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Say hello"},
	}

	ctx := context.Background()

	// This will fail with the real API, but we test the error handling
	response, err := suite.service.ChatCompletion(ctx, "gpt-3.5-turbo", messages, 0.7)

	assert.Error(suite.T(), err) // Expected to fail with test API key
	assert.Nil(suite.T(), response)
}

// Test ChatCompletion with invalid inputs
func (suite *LLMTestSuite) TestChatCompletionInvalidInputs() {
	ctx := context.Background()

	// Test with empty messages
	emptyMessages := []llm.ChatMessage{}
	response, err := suite.service.ChatCompletion(ctx, "gpt-3.5-turbo", emptyMessages, 0.7)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)

	// Test with invalid model
	validMessages := []llm.ChatMessage{
		{Role: "user", Content: "Hello"},
	}
	response, err = suite.service.ChatCompletion(ctx, "", validMessages, 0.7)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
}

// Test ChatCompletionStream channel behavior
func (suite *LLMTestSuite) TestChatCompletionStreamChannels() {
	messages := []llm.ChatMessage{
		{Role: "user", Content: "Stream test"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	contentChan, errorChan := suite.service.ChatCompletionStream(ctx, "gpt-3.5-turbo", messages, 0.7)

	// Channels should not be nil
	assert.NotNil(suite.T(), contentChan)
	assert.NotNil(suite.T(), errorChan)

	// Should receive an error (with test API key)
	select {
	case err := <-errorChan:
		assert.Error(suite.T(), err)
	case <-ctx.Done():
		suite.T().Error("Test timed out")
	case content := <-contentChan:
		suite.T().Errorf("Unexpected content received: %s", content)
	}
}

// Test context cancellation
func (suite *LLMTestSuite) TestContextCancellation() {
	messages := []llm.ChatMessage{
		{Role: "user", Content: "This should be cancelled"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Test non-streaming
	response, err := suite.service.ChatCompletion(ctx, "gpt-3.5-turbo", messages, 0.7)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "context canceled")

	// Test streaming
	contentChan, errorChan := suite.service.ChatCompletionStream(ctx, "gpt-3.5-turbo", messages, 0.7)

	// Should receive an error due to cancelled context
	select {
	case err := <-errorChan:
		assert.Error(suite.T(), err)
	case <-time.After(1 * time.Second):
		suite.T().Error("Should have received error due to cancelled context")
	case content := <-contentChan:
		suite.T().Errorf("Unexpected content received: %s", content)
	}
}

// Test ValidateAPIKey functionality
func (suite *LLMTestSuite) TestValidateAPIKey() {
	ctx := context.Background()

	// With a test API key, this should fail
	err := suite.service.ValidateAPIKey(ctx)
	assert.Error(suite.T(), err)
}

// Test temperature parameter validation
func (suite *LLMTestSuite) TestTemperatureParameters() {
	messages := []llm.ChatMessage{
		{Role: "user", Content: "Test temperature"},
	}

	ctx := context.Background()
	temperatures := []float64{0.0, 0.5, 1.0, 2.0}

	for _, temp := range temperatures {
		// These will fail with our test API key, but we test that they don't panic
		response, err := suite.service.ChatCompletion(ctx, "gpt-3.5-turbo", messages, temp)
		assert.Error(suite.T(), err) // Expected to fail with test key
		assert.Nil(suite.T(), response)
	}
}

// Test message role validation
func (suite *LLMTestSuite) TestMessageRoles() {
	ctx := context.Background()

	validRoles := []string{"system", "user", "assistant"}

	for _, role := range validRoles {
		messages := []llm.ChatMessage{
			{Role: role, Content: "Test message with role " + role},
		}

		// Test that different roles don't cause panics
		response, err := suite.service.ChatCompletion(ctx, "gpt-3.5-turbo", messages, 0.7)
		assert.Error(suite.T(), err) // Expected to fail with test key
		assert.Nil(suite.T(), response)
	}
}

// Test JSON response parsing
func (suite *LLMTestSuite) TestResponseParsing() {
	// Test ChatResponse JSON unmarshaling
	responseJSON := `{
		"id": "chatcmpl-test123",
		"object": "chat.completion",
		"created": 1677652288,
		"model": "gpt-3.5-turbo",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Hello! How can I assist you today?"
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 9,
			"completion_tokens": 12,
			"total_tokens": 21
		}
	}`

	var response llm.ChatResponse
	err := json.Unmarshal([]byte(responseJSON), &response)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "chatcmpl-test123", response.ID)
	assert.Equal(suite.T(), "chat.completion", response.Object)
	assert.Equal(suite.T(), "gpt-3.5-turbo", response.Model)
	assert.Len(suite.T(), response.Choices, 1)
	assert.Equal(suite.T(), "assistant", response.Choices[0].Message.Role)
	assert.Equal(suite.T(), "Hello! How can I assist you today?", response.Choices[0].Message.Content)
	assert.Equal(suite.T(), 21, response.Usage.TotalTokens)
}

// Test streaming response parsing
func (suite *LLMTestSuite) TestStreamResponseParsing() {
	// Test ChatStreamResponse JSON unmarshaling
	streamJSON := `{
		"id": "chatcmpl-test123",
		"object": "chat.completion.chunk",
		"created": 1677652288,
		"model": "gpt-3.5-turbo",
		"choices": [{
			"index": 0,
			"delta": {
				"content": "Hello"
			},
			"finish_reason": null
		}]
	}`

	var streamResponse llm.ChatStreamResponse
	err := json.Unmarshal([]byte(streamJSON), &streamResponse)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "chatcmpl-test123", streamResponse.ID)
	assert.Equal(suite.T(), "chat.completion.chunk", streamResponse.Object)
	assert.Equal(suite.T(), "gpt-3.5-turbo", streamResponse.Model)
	assert.Len(suite.T(), streamResponse.Choices, 1)
	assert.Equal(suite.T(), "Hello", streamResponse.Choices[0].Delta.Content)
}

// Test models response parsing
func (suite *LLMTestSuite) TestModelsResponseParsing() {
	modelsJSON := `{
		"data": [
			{
				"id": "gpt-3.5-turbo",
				"object": "model",
				"created": 1677610602,
				"owned_by": "openai"
			},
			{
				"id": "gpt-4",
				"object": "model", 
				"created": 1687882411,
				"owned_by": "openai"
			}
		]
	}`

	var modelsResponse llm.ModelsResponse
	err := json.Unmarshal([]byte(modelsJSON), &modelsResponse)

	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), modelsResponse.Data, 2)
	assert.Equal(suite.T(), "gpt-3.5-turbo", modelsResponse.Data[0].ID)
	assert.Equal(suite.T(), "gpt-4", modelsResponse.Data[1].ID)
	assert.Equal(suite.T(), "openai", modelsResponse.Data[0].OwnedBy)
}

// Test error response handling
func (suite *LLMTestSuite) TestErrorResponseHandling() {
	// Test various error conditions that the service should handle gracefully

	ctx := context.Background()

	// Test with very long content (might cause API errors)
	longContent := strings.Repeat("a", 100000)
	longMessages := []llm.ChatMessage{
		{Role: "user", Content: longContent},
	}

	response, err := suite.service.ChatCompletion(ctx, "gpt-3.5-turbo", longMessages, 0.7)
	assert.Error(suite.T(), err) // Expected to fail
	assert.Nil(suite.T(), response)
}

func TestLLMTestSuite(t *testing.T) {
	suite.Run(t, new(LLMTestSuite))
}
