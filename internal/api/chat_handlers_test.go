package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/llm"
	"scriberr/internal/models"

	"github.com/stretchr/testify/require"
)

type fakeChatLLM struct {
	models []string
	events []llm.StreamEvent
}

func (f *fakeChatLLM) GetModels(ctx context.Context) ([]string, error) {
	return f.models, nil
}

func (f *fakeChatLLM) ChatCompletion(ctx context.Context, model string, messages []llm.ChatMessage, temperature float64) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Model: model}, nil
}

func (f *fakeChatLLM) ChatCompletionStream(ctx context.Context, model string, messages []llm.ChatMessage, temperature float64) (<-chan string, <-chan error) {
	content := make(chan string, 1)
	errs := make(chan error, 1)
	go func() {
		defer close(content)
		defer close(errs)
		for _, event := range f.events {
			if event.Type == llm.StreamEventContentDelta {
				content <- event.ContentDelta
			}
		}
	}()
	return content, errs
}

func (f *fakeChatLLM) ChatCompletionStreamEvents(ctx context.Context, model string, messages []llm.ChatMessage, temperature float64) (<-chan llm.StreamEvent, <-chan error) {
	events := make(chan llm.StreamEvent, len(f.events))
	errs := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(errs)
		for _, event := range f.events {
			events <- event
		}
	}()
	return events, errs
}

func (f *fakeChatLLM) GetContextWindow(ctx context.Context, model string) (int, error) {
	return 4096, nil
}

func TestChatModelsRequireProviderAndReturnCapabilities(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodGet, "/api/v1/chat/models", nil, token, "")
	require.Equal(t, http.StatusConflict, resp.Code)
	require.Equal(t, "LLM_PROVIDER_NOT_CONFIGURED", body["error"].(map[string]any)["code"])

	userID := firstUserID(t)
	saveChatLLMConfig(t, userID, "qwen")
	s.handler.chatLLMFactory = func(config *models.LLMConfig) (llm.Service, error) {
		return &fakeChatLLM{models: []string{"qwen"}}, nil
	}

	resp, body = s.request(t, http.MethodGet, "/api/v1/chat/models", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, true, body["configured"])
	models := body["models"].([]any)
	require.Len(t, models, 1)
	require.Equal(t, "qwen", models[0].(map[string]any)["id"])
}

func TestChatSessionContextAndStreamingLifecycle(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	userID := firstUserID(t)
	parent := createCompletedChatTranscript(t, userID, "parent transcript")
	extra := createCompletedChatTranscript(t, userID, "extra transcript")
	saveChatLLMConfig(t, userID, "qwen")
	s.handler.chatLLMFactory = func(config *models.LLMConfig) (llm.Service, error) {
		return &fakeChatLLM{
			models: []string{"qwen"},
			events: []llm.StreamEvent{
				{Type: llm.StreamEventReasoningDelta, ReasoningDelta: "thinking"},
				{Type: llm.StreamEventContentDelta, ContentDelta: "final answer"},
				{Type: llm.StreamEventUsage, Usage: &llm.TokenUsage{PromptTokens: 10, CompletionTokens: 2, ReasoningTokens: 1, TotalTokens: 13}},
				{Type: llm.StreamEventDone, FinishReason: "stop"},
			},
		}, nil
	}

	resp, body := s.request(t, http.MethodPost, "/api/v1/chat/sessions", map[string]any{
		"parent_transcription_id": "tr_" + parent.ID,
		"title":                   "Chat",
		"model":                   "qwen",
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	sessionID := body["id"].(string)
	require.True(t, strings.HasPrefix(sessionID, "chat_"))

	resp, body = s.request(t, http.MethodGet, "/api/v1/chat/sessions?parent_transcription_id=tr_"+parent.ID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Len(t, body["items"].([]any), 1)

	resp, body = s.request(t, http.MethodGet, "/api/v1/chat/sessions/"+sessionID+"/context", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	contextItems := body["items"].([]any)
	require.Len(t, contextItems, 1)
	parentContext := contextItems[0].(map[string]any)
	require.Equal(t, "active", parentContext["status"])
	require.Equal(t, true, parentContext["has_plain_text_snapshot"])
	require.NotContains(t, parentContext, "plain_text_snapshot")

	resp, body = s.request(t, http.MethodPost, "/api/v1/chat/sessions/"+sessionID+"/context/transcripts", map[string]any{
		"transcription_id": "tr_" + extra.ID,
	}, token, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	contextID := body["id"].(string)

	resp, body = s.request(t, http.MethodPatch, "/api/v1/chat/sessions/"+sessionID+"/context/transcripts/"+contextID, map[string]any{"enabled": false}, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, false, body["enabled"])

	recorder, raw := s.rawRequest(t, http.MethodPost, "/api/v1/chat/sessions/"+sessionID+"/messages:stream", map[string]any{"content": "What happened?", "model": "qwen"}, token, "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/event-stream")
	require.Contains(t, raw, "event: chat.run.started")
	require.Contains(t, raw, "event: chat.delta.reasoning")
	require.Contains(t, raw, "event: chat.delta.content")
	require.Contains(t, raw, "event: chat.run.completed")
	require.Contains(t, raw, "final answer")
	require.Contains(t, raw, "reasoning_content")
	require.Contains(t, raw, "total_tokens")

	var assistant models.ChatMessage
	require.NoError(t, database.DB.Where("role = ? AND chat_session_id = ?", models.ChatMessageRoleAssistant, strings.TrimPrefix(sessionID, "chat_")).First(&assistant).Error)
	require.Equal(t, "final answer", assistant.Content)
	require.Equal(t, "thinking", assistant.ReasoningContent)
	require.NotNil(t, assistant.TotalTokens)
	require.Equal(t, 13, *assistant.TotalTokens)

	resp, body = s.request(t, http.MethodGet, "/api/v1/chat/sessions/"+sessionID+"/messages", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	messageItems := body["items"].([]any)
	require.Len(t, messageItems, 2)
	require.Equal(t, "user", messageItems[0].(map[string]any)["role"])
	require.Equal(t, "assistant", messageItems[1].(map[string]any)["role"])
}

func TestChatRunCancelPersistsTerminalState(t *testing.T) {
	s := newAuthTestServer(t)
	token := registerForFileTests(t, s)
	userID := firstUserID(t)
	parent := createCompletedChatTranscript(t, userID, "parent")
	session := models.ChatSession{UserID: userID, ParentTranscriptionID: parent.ID, Title: "Chat", Provider: "openai_compatible", Model: "qwen"}
	require.NoError(t, database.DB.Create(&session).Error)
	run := models.ChatGenerationRun{UserID: userID, ChatSessionID: session.ID, Status: models.ChatGenerationRunStatusPending, Provider: "openai_compatible", Model: "qwen", ContextWindow: 4096}
	require.NoError(t, database.DB.Create(&run).Error)

	resp, body := s.request(t, http.MethodPost, "/api/v1/chat/runs/chatrun_"+run.ID+":cancel", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "canceled", body["status"])
}

func firstUserID(t *testing.T) uint {
	t.Helper()
	var user models.User
	require.NoError(t, database.DB.Order("id ASC").First(&user).Error)
	return user.ID
}

func saveChatLLMConfig(t *testing.T, userID uint, model string) {
	t.Helper()
	baseURL := "https://example.test/v1"
	apiKey := "sk-test"
	config := models.LLMConfig{UserID: userID, Name: "Default", Provider: "openai_compatible", BaseURL: &baseURL, OpenAIBaseURL: &baseURL, APIKey: &apiKey, LargeModel: &model, SmallModel: &model, IsDefault: true}
	require.NoError(t, database.DB.Create(&config).Error)
}

func createCompletedChatTranscript(t *testing.T, userID uint, text string) models.TranscriptionJob {
	t.Helper()
	title := text
	sourceHash := "hash-" + strings.ReplaceAll(text, " ", "-") + time.Now().Format("150405.000000000")
	transcript := `{"text":"` + text + `","segments":[{"id":"s1","speaker":"SPEAKER_00","text":"` + text + `"}]}`
	job := models.TranscriptionJob{UserID: userID, Title: &title, Status: models.StatusCompleted, AudioPath: "/tmp/chat.wav", SourceFileHash: &sourceHash, Transcript: &transcript}
	require.NoError(t, database.DB.Create(&job).Error)
	return job
}
