package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	chatdomain "scriberr/internal/chat"
	"scriberr/internal/database"
	"scriberr/internal/llm"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const chatProviderTimeout = 10 * time.Second

type createChatSessionRequest struct {
	ParentTranscriptionID string  `json:"parent_transcription_id"`
	Title                 string  `json:"title"`
	Model                 string  `json:"model"`
	SystemPrompt          *string `json:"system_prompt"`
}

type updateChatSessionRequest struct {
	Title        *string `json:"title"`
	Status       *string `json:"status"`
	SystemPrompt *string `json:"system_prompt"`
}

type addChatContextTranscriptRequest struct {
	TranscriptionID string `json:"transcription_id"`
}

type updateChatContextTranscriptRequest struct {
	Enabled *bool `json:"enabled"`
}

type streamChatMessageRequest struct {
	Content     string  `json:"content"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
}

func (h *Handler) listChatModels(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	config, ok := h.activeLLMConfig(c, userID, true)
	if !ok {
		return
	}
	client, err := h.chatLLMFactory(config)
	if err != nil {
		writeError(c, http.StatusUnprocessableEntity, "LLM_PROVIDER_UNAVAILABLE", "LLM provider is not available.", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), chatProviderTimeout)
	defer cancel()
	models, err := client.GetModels(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "LLM_PROVIDER_UNAVAILABLE", "LLM provider is not available.", nil)
		return
	}
	items := make([]gin.H, 0, len(models))
	for _, model := range models {
		window, _ := client.GetContextWindow(ctx, model)
		items = append(items, gin.H{
			"id":                    model,
			"display_name":          model,
			"context_window":        window,
			"context_window_source": "provider",
			"supports_streaming":    true,
			"supports_reasoning":    true,
		})
	}
	c.JSON(http.StatusOK, gin.H{"provider": config.Provider, "configured": true, "models": items})
}

func (h *Handler) createChatSession(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req createChatSessionRequest
	if !bindJSON(c, &req) {
		return
	}
	parentID, ok := parsePublicID(req.ParentTranscriptionID, "tr_")
	if !ok {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "parent_transcription_id is invalid", stringPtr("parent_transcription_id"))
		return
	}
	config, ok := h.activeLLMConfig(c, userID, true)
	if !ok {
		return
	}
	model := strings.TrimSpace(req.Model)
	if model == "" && config.LargeModel != nil {
		model = strings.TrimSpace(*config.LargeModel)
	}
	if model == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "model is required", stringPtr("model"))
		return
	}
	if !h.chatModelAvailable(c, config, model) {
		return
	}
	repo := repository.NewChatRepository(database.DB)
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Transcript chat"
	}
	session := &models.ChatSession{
		UserID:                userID,
		ParentTranscriptionID: parentID,
		Title:                 title,
		Provider:              config.Provider,
		Model:                 model,
		SystemPrompt:          req.SystemPrompt,
	}
	if err := repo.CreateSession(c.Request.Context(), session); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "parent transcription not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create chat session", nil)
		return
	}
	builder := chatdomain.NewContextBuilder(repo, chatdomain.ApproxTokenEstimator{})
	if _, err := builder.AddParentSource(c.Request.Context(), userID, session.ID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not add parent transcript context", nil)
		return
	}
	c.JSON(http.StatusCreated, chatSessionResponse(session))
}

func (h *Handler) listChatSessions(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	parentID, ok := parsePublicID(c.Query("parent_transcription_id"), "tr_")
	if !ok {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "parent_transcription_id is required", stringPtr("parent_transcription_id"))
		return
	}
	sessions, _, err := repository.NewChatRepository(database.DB).ListSessionsForTranscription(c.Request.Context(), userID, parentID, 0, 100)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list chat sessions", nil)
		return
	}
	items := make([]gin.H, 0, len(sessions))
	for i := range sessions {
		items = append(items, chatSessionResponse(&sessions[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}

func (h *Handler) getChatSession(c *gin.Context) {
	session, ok := h.chatSessionByPublicID(c, c.Param("session_id"))
	if !ok {
		return
	}
	c.JSON(http.StatusOK, chatSessionResponse(session))
}

func (h *Handler) updateChatSession(c *gin.Context) {
	session, ok := h.chatSessionByPublicID(c, c.Param("session_id"))
	if !ok {
		return
	}
	var req updateChatSessionRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.Title != nil {
		session.Title = strings.TrimSpace(*req.Title)
	}
	if req.SystemPrompt != nil {
		session.SystemPrompt = req.SystemPrompt
	}
	if req.Status != nil {
		switch models.ChatSessionStatus(*req.Status) {
		case models.ChatSessionStatusActive, models.ChatSessionStatusArchived:
			session.Status = models.ChatSessionStatus(*req.Status)
		default:
			writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "status is invalid", stringPtr("status"))
			return
		}
	}
	if err := repository.NewChatRepository(database.DB).UpdateSession(c.Request.Context(), session); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update chat session", nil)
		return
	}
	c.JSON(http.StatusOK, chatSessionResponse(session))
}

func (h *Handler) deleteChatSession(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	sessionID, ok := parsePublicID(c.Param("session_id"), "chat_")
	if !ok {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "chat session not found", nil)
		return
	}
	err := repository.NewChatRepository(database.DB).DeleteSession(c.Request.Context(), userID, sessionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "chat session not found", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete chat session", nil)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listChatMessages(c *gin.Context) {
	session, ok := h.chatSessionByPublicID(c, c.Param("session_id"))
	if !ok {
		return
	}
	messages, _, err := repository.NewChatRepository(database.DB).ListMessages(c.Request.Context(), session.UserID, session.ID, 0, 200)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list chat messages", nil)
		return
	}
	items := make([]gin.H, 0, len(messages))
	for i := range messages {
		items = append(items, chatMessageResponse(&messages[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}

func (h *Handler) getChatContext(c *gin.Context) {
	session, ok := h.chatSessionByPublicID(c, c.Param("session_id"))
	if !ok {
		return
	}
	sources, err := repository.NewChatRepository(database.DB).ListContextSources(c.Request.Context(), session.UserID, session.ID, false)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list chat context", nil)
		return
	}
	items := make([]gin.H, 0, len(sources))
	for i := range sources {
		items = append(items, chatContextSourceResponse(&sources[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}

func (h *Handler) addChatContextTranscript(c *gin.Context) {
	session, ok := h.chatSessionByPublicID(c, c.Param("session_id"))
	if !ok {
		return
	}
	var req addChatContextTranscriptRequest
	if !bindJSON(c, &req) {
		return
	}
	transcriptionID, ok := parsePublicID(req.TranscriptionID, "tr_")
	if !ok {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "transcription_id is invalid", stringPtr("transcription_id"))
		return
	}
	builder := chatdomain.NewContextBuilder(repository.NewChatRepository(database.DB), chatdomain.ApproxTokenEstimator{})
	mutation, err := builder.AddTranscriptSource(c.Request.Context(), session.UserID, session.ID, transcriptionID, models.ChatContextSourceKindTranscript)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "transcription not found", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not add context transcript", nil)
		return
	}
	c.JSON(http.StatusCreated, chatContextSourceResponse(mutation.Source))
}

func (h *Handler) updateChatContextTranscript(c *gin.Context) {
	session, ok := h.chatSessionByPublicID(c, c.Param("session_id"))
	if !ok {
		return
	}
	sourceID, ok := parsePublicID(c.Param("context_source_id"), "chatctx_")
	if !ok {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "context source not found", nil)
		return
	}
	var req updateChatContextTranscriptRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.Enabled != nil {
		err := repository.NewChatRepository(database.DB).SetContextSourceEnabled(c.Request.Context(), session.UserID, session.ID, sourceID, *req.Enabled)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "context source not found", nil)
			return
		}
		if err != nil {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update context source", nil)
			return
		}
	}
	source, err := repository.NewChatRepository(database.DB).FindContextSourceForUser(c.Request.Context(), session.UserID, session.ID, sourceID)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "context source not found", nil)
		return
	}
	c.JSON(http.StatusOK, chatContextSourceResponse(source))
}

func (h *Handler) deleteChatContextTranscript(c *gin.Context) {
	session, ok := h.chatSessionByPublicID(c, c.Param("session_id"))
	if !ok {
		return
	}
	sourceID, ok := parsePublicID(c.Param("context_source_id"), "chatctx_")
	if !ok {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "context source not found", nil)
		return
	}
	err := repository.NewChatRepository(database.DB).DeleteContextSource(c.Request.Context(), session.UserID, session.ID, sourceID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "context source not found", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete context source", nil)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) streamChatMessage(c *gin.Context, publicSessionID string) {
	session, ok := h.chatSessionByPublicID(c, publicSessionID)
	if !ok {
		return
	}
	var req streamChatMessageRequest
	if !bindJSON(c, &req) {
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "content is required", stringPtr("content"))
		return
	}
	config, ok := h.activeLLMConfig(c, session.UserID, true)
	if !ok {
		return
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = session.Model
	}
	if !h.chatModelAvailable(c, config, model) {
		return
	}
	client, err := h.chatLLMFactory(config)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "LLM_PROVIDER_UNAVAILABLE", "LLM provider is not available.", nil)
		return
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "streaming is not supported", nil)
		return
	}
	repo := repository.NewChatRepository(database.DB)
	ctx := c.Request.Context()
	userMessage := &models.ChatMessage{UserID: session.UserID, ChatSessionID: session.ID, Role: models.ChatMessageRoleUser, Content: content}
	if err := repo.CreateMessage(ctx, userMessage); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not persist user message", nil)
		return
	}
	provider := config.Provider
	assistant := &models.ChatMessage{UserID: session.UserID, ChatSessionID: session.ID, Role: models.ChatMessageRoleAssistant, Status: models.ChatMessageStatusStreaming, Provider: &provider, Model: &model}
	if err := repo.CreateMessage(ctx, assistant); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not persist assistant message", nil)
		return
	}
	window, _ := client.GetContextWindow(ctx, model)
	if window <= 0 {
		window = 4096
	}
	run := &models.ChatGenerationRun{UserID: session.UserID, ChatSessionID: session.ID, AssistantMessageID: &assistant.ID, Status: models.ChatGenerationRunStatusPending, Provider: provider, Model: model, ContextWindow: window, ContextWindowSource: "provider"}
	if err := repo.CreateGenerationRun(ctx, run); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create chat run", nil)
		return
	}
	assistant.RunID = &run.ID
	_ = repo.UpdateMessage(ctx, assistant)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	chatWriteSSE(c, flusher, "chat.run.started", chatRunPayload(session.ID, run.ID, assistant.ID, gin.H{"status": string(models.ChatGenerationRunStatusStreaming)}))
	chatWriteSSE(c, flusher, "chat.message.created", gin.H{
		"session_id":           publicChatSessionID(session.ID),
		"run_id":               publicChatRunID(run.ID),
		"message_id":           publicChatMessageID(userMessage.ID),
		"assistant_message_id": publicChatMessageID(assistant.ID),
		"user_message":         chatMessageResponse(userMessage),
		"assistant_message":    chatMessageResponse(assistant),
	})

	now := time.Now()
	_ = repo.UpdateGenerationRunStatus(context.Background(), session.UserID, run.ID, models.ChatGenerationRunStatusStreaming, now, nil)
	messages, _, _ := repo.ListMessages(ctx, session.UserID, session.ID, 0, 100)
	llmMessages := h.buildLLMMessages(ctx, repo, session, messages, content, window)
	events, errorsChan := client.ChatCompletionStreamEvents(ctx, model, llmMessages, req.Temperature)
	var responseContent, reasoningContent strings.Builder
	var usage *llm.TokenUsage
	for event := range events {
		switch event.Type {
		case llm.StreamEventReasoningDelta:
			reasoningContent.WriteString(event.ReasoningDelta)
			chatWriteSSE(c, flusher, "chat.delta.reasoning", chatRunPayload(session.ID, run.ID, assistant.ID, gin.H{"delta": event.ReasoningDelta}))
		case llm.StreamEventContentDelta:
			responseContent.WriteString(event.ContentDelta)
			chatWriteSSE(c, flusher, "chat.delta.content", chatRunPayload(session.ID, run.ID, assistant.ID, gin.H{"delta": event.ContentDelta}))
		case llm.StreamEventUsage:
			usage = event.Usage
		}
	}
	if err := firstStreamError(errorsChan); err != nil {
		message := sanitizePublicText(err.Error())
		assistant.Status = models.ChatMessageStatusFailed
		assistant.Content = responseContent.String()
		assistant.ReasoningContent = reasoningContent.String()
		_ = repo.UpdateMessage(context.Background(), assistant)
		_ = repo.UpdateGenerationRunStatus(context.Background(), session.UserID, run.ID, models.ChatGenerationRunStatusFailed, time.Now(), &message)
		chatWriteSSE(c, flusher, "chat.run.failed", chatRunPayload(session.ID, run.ID, assistant.ID, gin.H{"error": message}))
		return
	}
	assistant.Status = models.ChatMessageStatusCompleted
	assistant.Content = responseContent.String()
	assistant.ReasoningContent = reasoningContent.String()
	completionPayload := gin.H{"status": string(models.ChatGenerationRunStatusCompleted)}
	if usage != nil {
		assistant.PromptTokens = intPtr(usage.PromptTokens)
		assistant.CompletionTokens = intPtr(usage.CompletionTokens)
		assistant.ReasoningTokens = intPtr(usage.ReasoningTokens)
		assistant.TotalTokens = intPtr(usage.TotalTokens)
		run.ContextTokensEstimated = usage.PromptTokens
		completionPayload["usage"] = chatUsageResponse(usage)
	}
	_ = repo.UpdateMessage(context.Background(), assistant)
	_ = repo.UpdateGenerationRunStatus(context.Background(), session.UserID, run.ID, models.ChatGenerationRunStatusCompleted, time.Now(), nil)
	completionPayload["assistant_message"] = chatMessageResponse(assistant)
	chatWriteSSE(c, flusher, "chat.run.completed", chatRunPayload(session.ID, run.ID, assistant.ID, completionPayload))
}

func (h *Handler) cancelChatRun(c *gin.Context, publicRunID string) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	runID, ok := parsePublicID(publicRunID, "chatrun_")
	if !ok {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "chat run not found", nil)
		return
	}
	repo := repository.NewChatRepository(database.DB)
	run, err := repo.FindGenerationRunForUser(c.Request.Context(), userID, runID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "chat run not found", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load chat run", nil)
		return
	}
	if run.Status == models.ChatGenerationRunStatusCompleted || run.Status == models.ChatGenerationRunStatusFailed || run.Status == models.ChatGenerationRunStatusCanceled {
		c.JSON(http.StatusOK, chatRunResponse(run))
		return
	}
	message := "canceled"
	if err := repo.UpdateGenerationRunStatus(c.Request.Context(), userID, run.ID, models.ChatGenerationRunStatusCanceled, time.Now(), &message); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not cancel chat run", nil)
		return
	}
	run.Status = models.ChatGenerationRunStatusCanceled
	c.JSON(http.StatusOK, chatRunResponse(run))
}

func (h *Handler) generateChatTitle(c *gin.Context) {
	session, ok := h.chatSessionByPublicID(c, c.Param("session_id"))
	if !ok {
		return
	}
	messages, _, _ := repository.NewChatRepository(database.DB).ListMessages(c.Request.Context(), session.UserID, session.ID, 0, 1)
	title := session.Title
	if len(messages) > 0 && strings.TrimSpace(messages[0].Content) != "" {
		title = summarizeTitle(messages[0].Content)
		session.Title = title
		_ = repository.NewChatRepository(database.DB).UpdateSession(c.Request.Context(), session)
	}
	c.JSON(http.StatusOK, gin.H{"id": publicChatSessionID(session.ID), "title": title})
}

func (h *Handler) buildLLMMessages(ctx context.Context, repo repository.ChatRepository, session *models.ChatSession, messages []models.ChatMessage, current string, window int) []llm.ChatMessage {
	var out []llm.ChatMessage
	if session.SystemPrompt != nil && strings.TrimSpace(*session.SystemPrompt) != "" {
		out = append(out, llm.ChatMessage{Role: "system", Content: strings.TrimSpace(*session.SystemPrompt)})
	}
	built, err := chatdomain.NewContextBuilder(repo, chatdomain.ApproxTokenEstimator{}).Build(ctx, session.UserID, session.ID, chatdomain.BuildOptions{Budget: chatdomain.ContextBudget{ContextWindow: window, ReservedResponse: 512, ReservedChat: 1024, SafetyMarginTokens: 128}})
	if err == nil && strings.TrimSpace(built.Content) != "" {
		out = append(out, llm.ChatMessage{Role: "system", Content: "Active transcript contexts:\n" + built.Content})
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

func (h *Handler) activeLLMConfig(c *gin.Context, userID uint, write bool) (*models.LLMConfig, bool) {
	var config models.LLMConfig
	err := database.DB.WithContext(c.Request.Context()).Where("user_id = ? AND is_default = ?", userID, true).First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if write {
			writeError(c, http.StatusConflict, "LLM_PROVIDER_NOT_CONFIGURED", "Configure an LLM provider before starting chat.", nil)
		}
		return nil, false
	}
	if err != nil {
		if write {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load LLM provider", nil)
		}
		return nil, false
	}
	return &config, true
}

func (h *Handler) chatModelAvailable(c *gin.Context, config *models.LLMConfig, model string) bool {
	client, err := h.chatLLMFactory(config)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "LLM_PROVIDER_UNAVAILABLE", "LLM provider is not available.", nil)
		return false
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), chatProviderTimeout)
	defer cancel()
	models, err := client.GetModels(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "LLM_PROVIDER_UNAVAILABLE", "LLM provider is not available.", nil)
		return false
	}
	if !stringInSlice(models, model) {
		writeError(c, http.StatusUnprocessableEntity, "MODEL_NOT_AVAILABLE", "model is not available from the configured provider", stringPtr("model"))
		return false
	}
	return true
}

func (h *Handler) chatSessionByPublicID(c *gin.Context, publicID string) (*models.ChatSession, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return nil, false
	}
	sessionID, ok := parsePublicID(publicID, "chat_")
	if !ok {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "chat session not found", nil)
		return nil, false
	}
	session, err := repository.NewChatRepository(database.DB).FindSessionForUser(c.Request.Context(), userID, sessionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "chat session not found", nil)
		return nil, false
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load chat session", nil)
		return nil, false
	}
	return session, true
}

func chatClientForConfig(config *models.LLMConfig) (llm.Service, error) {
	baseURL := llmProviderBaseURL(config)
	switch config.Provider {
	case "ollama":
		return llm.NewOllamaService(baseURL), nil
	case "openai", "openai_compatible":
		apiKey := ""
		if config.APIKey != nil {
			apiKey = strings.TrimSpace(*config.APIKey)
		}
		return llm.NewOpenAIService(apiKey, &baseURL), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider")
	}
}

func chatWriteSSE(c *gin.Context, flusher http.Flusher, name string, payload gin.H) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(c.Writer, "event: %s\n", name)
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	flusher.Flush()
}

func chatRunPayload(sessionID, runID, messageID string, payload gin.H) gin.H {
	if payload == nil {
		payload = gin.H{}
	}
	payload["session_id"] = publicChatSessionID(sessionID)
	payload["run_id"] = publicChatRunID(runID)
	payload["message_id"] = publicChatMessageID(messageID)
	return payload
}

func firstStreamError(errors <-chan error) error {
	for err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

func parsePublicID(value, prefix string) (string, bool) {
	if !strings.HasPrefix(value, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(value, prefix)
	return id, id != ""
}

func publicChatSessionID(id string) string       { return "chat_" + id }
func publicChatMessageID(id string) string       { return "chatmsg_" + id }
func publicChatContextSourceID(id string) string { return "chatctx_" + id }
func publicChatRunID(id string) string           { return "chatrun_" + id }

func chatSessionResponse(session *models.ChatSession) gin.H {
	return gin.H{"id": publicChatSessionID(session.ID), "parent_transcription_id": "tr_" + session.ParentTranscriptionID, "title": session.Title, "provider": session.Provider, "model": session.Model, "system_prompt": session.SystemPrompt, "status": string(session.Status), "last_message_at": session.LastMessageAt, "created_at": session.CreatedAt, "updated_at": session.UpdatedAt}
}

func chatContextSourceResponse(source *models.ChatContextSource) gin.H {
	status := "active"
	if !source.Enabled {
		status = "disabled"
	} else if source.CompactionStatus != models.ChatContextCompactionStatusNone {
		status = string(source.CompactionStatus)
	}
	tokenEstimate := 0
	if source.CompactedSnapshot != nil && *source.CompactedSnapshot != "" {
		tokenEstimate = chatdomain.ApproxTokenEstimator{}.EstimateTokens(*source.CompactedSnapshot)
	} else if source.PlainTextSnapshot != nil && *source.PlainTextSnapshot != "" {
		tokenEstimate = chatdomain.ApproxTokenEstimator{}.EstimateTokens(*source.PlainTextSnapshot)
	}
	return gin.H{
		"id":                      publicChatContextSourceID(source.ID),
		"session_id":              publicChatSessionID(source.ChatSessionID),
		"transcription_id":        "tr_" + source.TranscriptionID,
		"kind":                    string(source.Kind),
		"enabled":                 source.Enabled,
		"status":                  status,
		"position":                source.Position,
		"compaction_status":       string(source.CompactionStatus),
		"has_plain_text_snapshot": source.PlainTextSnapshot != nil && *source.PlainTextSnapshot != "",
		"has_compacted_snapshot":  source.CompactedSnapshot != nil && *source.CompactedSnapshot != "",
		"snapshot_hash":           source.SnapshotHash,
		"source_version":          source.SourceVersion,
		"tokens_estimated":        tokenEstimate,
		"created_at":              source.CreatedAt,
		"updated_at":              source.UpdatedAt,
	}
}

func chatRunResponse(run *models.ChatGenerationRun) gin.H {
	return gin.H{"id": publicChatRunID(run.ID), "session_id": publicChatSessionID(run.ChatSessionID), "assistant_message_id": nullablePublicChatMessageID(run.AssistantMessageID), "status": string(run.Status), "provider": run.Provider, "model": run.Model, "context_window": run.ContextWindow, "context_window_source": run.ContextWindowSource, "context_tokens_estimated": run.ContextTokensEstimated, "created_at": run.CreatedAt, "updated_at": run.UpdatedAt}
}

func nullablePublicChatMessageID(id *string) any {
	if id == nil || *id == "" {
		return nil
	}
	return publicChatMessageID(*id)
}

func intPtr(value int) *int { return &value }

func chatMessageResponse(message *models.ChatMessage) gin.H {
	if message == nil {
		return gin.H{}
	}
	return gin.H{
		"id":                publicChatMessageID(message.ID),
		"session_id":        publicChatSessionID(message.ChatSessionID),
		"role":              string(message.Role),
		"content":           message.Content,
		"reasoning_content": message.ReasoningContent,
		"status":            string(message.Status),
		"provider":          message.Provider,
		"model":             message.Model,
		"run_id":            nullablePublicChatRunID(message.RunID),
		"prompt_tokens":     message.PromptTokens,
		"completion_tokens": message.CompletionTokens,
		"reasoning_tokens":  message.ReasoningTokens,
		"total_tokens":      message.TotalTokens,
		"created_at":        message.CreatedAt,
		"updated_at":        message.UpdatedAt,
	}
}

func nullablePublicChatRunID(id *string) any {
	if id == nil || *id == "" {
		return nil
	}
	return publicChatRunID(*id)
}

func chatUsageResponse(usage *llm.TokenUsage) gin.H {
	if usage == nil {
		return gin.H{}
	}
	return gin.H{
		"prompt_tokens":     usage.PromptTokens,
		"completion_tokens": usage.CompletionTokens,
		"reasoning_tokens":  usage.ReasoningTokens,
		"total_tokens":      usage.TotalTokens,
	}
}

func summarizeTitle(value string) string {
	words := strings.Fields(value)
	if len(words) > 8 {
		words = words[:8]
	}
	title := strings.Join(words, " ")
	if title == "" {
		return "Transcript chat"
	}
	return title
}
