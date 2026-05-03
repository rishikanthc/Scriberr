package chat

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/llm"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeStreamingLLM struct {
	models []string
	events []llm.StreamEvent
}

func (f *fakeStreamingLLM) GetModels(context.Context) ([]string, error) {
	return f.models, nil
}

func (f *fakeStreamingLLM) ChatCompletion(context.Context, string, []llm.ChatMessage, float64) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{}, nil
}

func (f *fakeStreamingLLM) ChatCompletionStream(context.Context, string, []llm.ChatMessage, float64) (<-chan string, <-chan error) {
	content := make(chan string)
	errs := make(chan error)
	close(content)
	close(errs)
	return content, errs
}

func (f *fakeStreamingLLM) ChatCompletionStreamEvents(context.Context, string, []llm.ChatMessage, float64) (<-chan llm.StreamEvent, <-chan error) {
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

func (f *fakeStreamingLLM) GetContextWindow(context.Context, string) (int, error) {
	return 4096, nil
}

func TestServiceStreamMessagePersistsRunAndEmitsEvents(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "scriberr.db"))
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	repo := repository.NewChatRepository(db)
	llmConfigs := repository.NewLLMConfigRepository(db)
	service := NewService(repo, llmConfigs)
	service.SetLLMClientFactory(func(*models.LLMConfig) (llm.Service, error) {
		return &fakeStreamingLLM{
			models: []string{"qwen"},
			events: []llm.StreamEvent{
				{Type: llm.StreamEventReasoningDelta, ReasoningDelta: "thinking"},
				{Type: llm.StreamEventContentDelta, ContentDelta: "answer"},
				{Type: llm.StreamEventUsage, Usage: &llm.TokenUsage{PromptTokens: 10, CompletionTokens: 2, ReasoningTokens: 1, TotalTokens: 13}},
				{Type: llm.StreamEventDone},
			},
		}, nil
	})

	user := models.User{Username: "chat-stream-user", Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	parent := completedTranscriptForGenerationTest(t, db, user.ID, "parent transcript")
	model := "qwen"
	baseURL := "https://example.test/v1"
	apiKey := "sk-test"
	require.NoError(t, db.Create(&models.LLMConfig{UserID: user.ID, Name: "Default", Provider: "openai_compatible", BaseURL: &baseURL, OpenAIBaseURL: &baseURL, APIKey: &apiKey, LargeModel: &model, IsDefault: true}).Error)
	session := models.ChatSession{UserID: user.ID, ParentTranscriptionID: parent.ID, Title: "Chat", Provider: "openai_compatible", Model: model}
	require.NoError(t, service.CreateSession(context.Background(), &session))

	events, err := service.StreamMessage(context.Background(), StreamMessageCommand{
		UserID:      user.ID,
		SessionID:   session.ID,
		Content:     "What happened?",
		Model:       model,
		Temperature: 0.2,
	})
	require.NoError(t, err)

	var names []string
	for event := range events {
		names = append(names, event.Name)
	}
	require.Equal(t, []string{
		"chat.run.started",
		"chat.message.created",
		"chat.delta.reasoning",
		"chat.delta.content",
		"chat.run.completed",
	}, names)

	messages, err := service.ListMessages(context.Background(), user.ID, session.ID, 10)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, models.ChatMessageRoleUser, messages[0].Role)
	require.Equal(t, models.ChatMessageRoleAssistant, messages[1].Role)
	require.Equal(t, "answer", messages[1].Content)
	require.Equal(t, "thinking", messages[1].ReasoningContent)
	require.NotNil(t, messages[1].TotalTokens)
	require.Equal(t, 13, *messages[1].TotalTokens)
}

func completedTranscriptForGenerationTest(t *testing.T, db *gorm.DB, userID uint, text string) models.TranscriptionJob {
	t.Helper()
	title := text
	transcript := `{"text":"` + text + `","segments":[{"id":"s1","speaker":"SPEAKER_00","text":"` + text + `"}]}`
	job := models.TranscriptionJob{UserID: userID, Title: &title, Status: models.StatusCompleted, AudioPath: filepath.Join(t.TempDir(), "chat.wav"), Transcript: &transcript, CreatedAt: time.Now()}
	require.NoError(t, db.Create(&job).Error)
	return job
}
