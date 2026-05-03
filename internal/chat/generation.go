package chat

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"scriberr/internal/llm"
	"scriberr/internal/models"
)

const providerTimeout = 10 * time.Second

var (
	ErrEmptyMessage     = errors.New("chat message content is required")
	ErrModelUnavailable = errors.New("chat model is not available")
	publicPathPattern   = regexp.MustCompile(`(?:[A-Za-z]:\\|/)[^\s:;,'")]+`)
	publicAPIKeyPattern = regexp.MustCompile(`(?i)\b([A-Za-z0-9_]*(?:token|api_key|apikey)[A-Za-z0-9_]*)=[^\s]+`)
)

type ProviderModel struct {
	ID                  string
	DisplayName         string
	ContextWindow       int
	ContextWindowSource string
	SupportsStreaming   bool
	SupportsReasoning   bool
}

type ProviderModels struct {
	Provider   string
	Configured bool
	Models     []ProviderModel
}

type StreamMessageCommand struct {
	UserID      uint
	SessionID   string
	Content     string
	Model       string
	Temperature float64
}

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	ReasoningTokens  int
	TotalTokens      int
}

type StreamEvent struct {
	Name             string
	SessionID        string
	RunID            string
	MessageID        string
	UserMessage      *models.ChatMessage
	AssistantMessage *models.ChatMessage
	Delta            string
	Error            string
	Status           models.ChatGenerationRunStatus
	Usage            *TokenUsage
}

func (s *Service) SetLLMClientFactory(factory LLMClientFactory) {
	if factory == nil {
		factory = ClientForConfig
	}
	s.llmFactory = factory
}

func (s *Service) ListProviderModels(ctx context.Context, userID uint) (*ProviderModels, error) {
	config, err := s.ActiveLLMConfig(ctx, userID)
	if err != nil {
		return nil, err
	}
	client, err := s.llmFactory(config)
	if err != nil {
		return nil, err
	}
	providerCtx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	availableModels, err := client.GetModels(providerCtx)
	if err != nil {
		return nil, err
	}
	items := make([]ProviderModel, 0, len(availableModels))
	for _, model := range availableModels {
		window, _ := client.GetContextWindow(providerCtx, model)
		items = append(items, ProviderModel{
			ID:                  model,
			DisplayName:         model,
			ContextWindow:       window,
			ContextWindowSource: "provider",
			SupportsStreaming:   true,
			SupportsReasoning:   true,
		})
	}
	return &ProviderModels{Provider: config.Provider, Configured: true, Models: items}, nil
}

func (s *Service) EnsureModelAvailable(ctx context.Context, config *models.LLMConfig, model string) error {
	client, err := s.llmFactory(config)
	if err != nil {
		return err
	}
	providerCtx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	availableModels, err := client.GetModels(providerCtx)
	if err != nil {
		return err
	}
	if !stringInSlice(availableModels, model) {
		return ErrModelUnavailable
	}
	return nil
}

func (s *Service) StreamMessage(ctx context.Context, cmd StreamMessageCommand) (<-chan StreamEvent, error) {
	session, err := s.GetSession(ctx, cmd.UserID, cmd.SessionID)
	if err != nil {
		return nil, err
	}
	content := strings.TrimSpace(cmd.Content)
	if content == "" {
		return nil, ErrEmptyMessage
	}
	config, err := s.ActiveLLMConfig(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	model := strings.TrimSpace(cmd.Model)
	if model == "" {
		model = session.Model
	}
	client, err := s.llmFactory(config)
	if err != nil {
		return nil, err
	}
	modelCtx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	availableModels, err := client.GetModels(modelCtx)
	if err != nil {
		return nil, err
	}
	if !stringInSlice(availableModels, model) {
		return nil, ErrModelUnavailable
	}

	userMessage := &models.ChatMessage{UserID: session.UserID, ChatSessionID: session.ID, Role: models.ChatMessageRoleUser, Content: content}
	if err := s.CreateMessage(ctx, userMessage); err != nil {
		return nil, err
	}
	provider := config.Provider
	assistant := &models.ChatMessage{UserID: session.UserID, ChatSessionID: session.ID, Role: models.ChatMessageRoleAssistant, Status: models.ChatMessageStatusStreaming, Provider: &provider, Model: &model}
	if err := s.CreateMessage(ctx, assistant); err != nil {
		return nil, err
	}
	window, _ := client.GetContextWindow(ctx, model)
	if window <= 0 {
		window = 4096
	}
	run := &models.ChatGenerationRun{UserID: session.UserID, ChatSessionID: session.ID, AssistantMessageID: &assistant.ID, Status: models.ChatGenerationRunStatusPending, Provider: provider, Model: model, ContextWindow: window, ContextWindowSource: "provider"}
	if err := s.CreateGenerationRun(ctx, run); err != nil {
		return nil, err
	}
	assistant.RunID = &run.ID
	_ = s.UpdateMessage(ctx, assistant)

	out := make(chan StreamEvent, 16)
	go s.runMessageStream(ctx, out, session, userMessage, assistant, run, client, model, content, window, cmd.Temperature)
	return out, nil
}

func (s *Service) runMessageStream(ctx context.Context, out chan<- StreamEvent, session *models.ChatSession, userMessage, assistant *models.ChatMessage, run *models.ChatGenerationRun, client llm.Service, model, content string, window int, temperature float64) {
	defer close(out)
	sendChatEvent(ctx, out, StreamEvent{Name: "chat.run.started", SessionID: session.ID, RunID: run.ID, MessageID: assistant.ID, Status: models.ChatGenerationRunStatusStreaming})
	sendChatEvent(ctx, out, StreamEvent{Name: "chat.message.created", SessionID: session.ID, RunID: run.ID, MessageID: userMessage.ID, UserMessage: userMessage, AssistantMessage: assistant})

	now := time.Now()
	_ = s.UpdateGenerationRunStatus(context.Background(), session.UserID, run.ID, models.ChatGenerationRunStatusStreaming, now, nil)
	messages, _ := s.ListMessages(ctx, session.UserID, session.ID, 100)
	llmMessages := s.buildLLMMessages(ctx, session, messages, content, window)
	events, errorsChan := client.ChatCompletionStreamEvents(ctx, model, llmMessages, temperature)
	var responseContent, reasoningContent strings.Builder
	var usage *llm.TokenUsage
	for event := range events {
		switch event.Type {
		case llm.StreamEventReasoningDelta:
			reasoningContent.WriteString(event.ReasoningDelta)
			sendChatEvent(ctx, out, StreamEvent{Name: "chat.delta.reasoning", SessionID: session.ID, RunID: run.ID, MessageID: assistant.ID, Delta: event.ReasoningDelta})
		case llm.StreamEventContentDelta:
			responseContent.WriteString(event.ContentDelta)
			sendChatEvent(ctx, out, StreamEvent{Name: "chat.delta.content", SessionID: session.ID, RunID: run.ID, MessageID: assistant.ID, Delta: event.ContentDelta})
		case llm.StreamEventUsage:
			usage = event.Usage
		}
	}
	if err := firstStreamError(errorsChan); err != nil {
		message := sanitizePublicText(err.Error())
		assistant.Status = models.ChatMessageStatusFailed
		assistant.Content = responseContent.String()
		assistant.ReasoningContent = reasoningContent.String()
		_ = s.UpdateMessage(context.Background(), assistant)
		_ = s.UpdateGenerationRunStatus(context.Background(), session.UserID, run.ID, models.ChatGenerationRunStatusFailed, time.Now(), &message)
		sendChatEvent(ctx, out, StreamEvent{Name: "chat.run.failed", SessionID: session.ID, RunID: run.ID, MessageID: assistant.ID, Error: message, Status: models.ChatGenerationRunStatusFailed})
		return
	}
	assistant.Status = models.ChatMessageStatusCompleted
	assistant.Content = responseContent.String()
	assistant.ReasoningContent = reasoningContent.String()
	if usage != nil {
		assistant.PromptTokens = intPtr(usage.PromptTokens)
		assistant.CompletionTokens = intPtr(usage.CompletionTokens)
		assistant.ReasoningTokens = intPtr(usage.ReasoningTokens)
		assistant.TotalTokens = intPtr(usage.TotalTokens)
		run.ContextTokensEstimated = usage.PromptTokens
	}
	_ = s.UpdateMessage(context.Background(), assistant)
	_ = s.UpdateGenerationRunStatus(context.Background(), session.UserID, run.ID, models.ChatGenerationRunStatusCompleted, time.Now(), nil)
	sendChatEvent(ctx, out, StreamEvent{Name: "chat.run.completed", SessionID: session.ID, RunID: run.ID, MessageID: assistant.ID, AssistantMessage: assistant, Status: models.ChatGenerationRunStatusCompleted, Usage: tokenUsage(usage)})
}

func (s *Service) buildLLMMessages(ctx context.Context, session *models.ChatSession, messages []models.ChatMessage, current string, window int) []llm.ChatMessage {
	var out []llm.ChatMessage
	if session.SystemPrompt != nil && strings.TrimSpace(*session.SystemPrompt) != "" {
		out = append(out, llm.ChatMessage{Role: "system", Content: strings.TrimSpace(*session.SystemPrompt)})
	}
	built, err := s.BuildContext(ctx, session.UserID, session.ID, window)
	if err == nil && strings.TrimSpace(built) != "" {
		out = append(out, llm.ChatMessage{Role: "system", Content: "Active transcript contexts:\n" + built})
	}
	start := 0
	if len(messages) > 12 {
		start = len(messages) - 12
	}
	for _, message := range messages[start:] {
		if message.ID != "" && strings.TrimSpace(message.Content) != "" {
			out = append(out, llm.ChatMessage{Role: string(message.Role), Content: message.Content})
		}
	}
	if len(out) == 0 || out[len(out)-1].Content != current {
		out = append(out, llm.ChatMessage{Role: "user", Content: current})
	}
	return out
}

func sendChatEvent(ctx context.Context, out chan<- StreamEvent, event StreamEvent) {
	select {
	case out <- event:
	case <-ctx.Done():
	}
}

func firstStreamError(errors <-chan error) error {
	for err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

func intPtr(value int) *int { return &value }

func tokenUsage(usage *llm.TokenUsage) *TokenUsage {
	if usage == nil {
		return nil
	}
	return &TokenUsage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		ReasoningTokens:  usage.ReasoningTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

func stringInSlice(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func sanitizePublicText(value string) string {
	value = publicPathPattern.ReplaceAllString(value, "[redacted-path]")
	return publicAPIKeyPattern.ReplaceAllString(value, "$1=[redacted]")
}
