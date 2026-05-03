package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	chatdomain "scriberr/internal/chat"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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
	result, err := h.chat.ListProviderModels(c.Request.Context(), userID)
	if !h.writeChatProviderError(c, err) {
		return
	}
	items := make([]gin.H, 0, len(result.Models))
	for _, model := range result.Models {
		items = append(items, gin.H{
			"id":                    model.ID,
			"display_name":          model.DisplayName,
			"context_window":        model.ContextWindow,
			"context_window_source": model.ContextWindowSource,
			"supports_streaming":    model.SupportsStreaming,
			"supports_reasoning":    model.SupportsReasoning,
		})
	}
	c.JSON(http.StatusOK, gin.H{"provider": result.Provider, "configured": result.Configured, "models": items})
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
	if err := h.chat.EnsureModelAvailable(c.Request.Context(), config, model); !h.writeChatProviderError(c, err) {
		return
	}
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
	if err := h.chat.CreateSession(c.Request.Context(), session); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "parent transcription not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create chat session", nil)
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
	sessions, err := h.chat.ListSessions(c.Request.Context(), userID, parentID)
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
	if err := h.chat.UpdateSession(c.Request.Context(), session); err != nil {
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
	err := h.chat.DeleteSession(c.Request.Context(), userID, sessionID)
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
	messages, err := h.chat.ListMessages(c.Request.Context(), session.UserID, session.ID, 200)
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
	sources, err := h.chat.ListContextSources(c.Request.Context(), session.UserID, session.ID, false)
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
	source, err := h.chat.AddTranscriptSource(c.Request.Context(), session.UserID, session.ID, transcriptionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "transcription not found", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not add context transcript", nil)
		return
	}
	c.JSON(http.StatusCreated, chatContextSourceResponse(source))
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
		err := h.chat.SetContextSourceEnabled(c.Request.Context(), session.UserID, session.ID, sourceID, *req.Enabled)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "context source not found", nil)
			return
		}
		if err != nil {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update context source", nil)
			return
		}
	}
	source, err := h.chat.FindContextSource(c.Request.Context(), session.UserID, session.ID, sourceID)
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
	err := h.chat.DeleteContextSource(c.Request.Context(), session.UserID, session.ID, sourceID)
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
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "streaming is not supported", nil)
		return
	}
	events, err := h.chat.StreamMessage(c.Request.Context(), chatdomain.StreamMessageCommand{
		UserID:      session.UserID,
		SessionID:   session.ID,
		Content:     req.Content,
		Model:       req.Model,
		Temperature: req.Temperature,
	})
	if !h.writeChatProviderError(c, err) {
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	for event := range events {
		chatWriteSSE(c, flusher, event.Name, h.chatStreamPayload(event))
	}
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
	run, err := h.chat.FindGenerationRun(c.Request.Context(), userID, runID)
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
	if err := h.chat.UpdateGenerationRunStatus(c.Request.Context(), userID, run.ID, models.ChatGenerationRunStatusCanceled, time.Now(), &message); err != nil {
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
	messages, _ := h.chat.ListMessages(c.Request.Context(), session.UserID, session.ID, 1)
	title := session.Title
	if len(messages) > 0 && strings.TrimSpace(messages[0].Content) != "" {
		title = summarizeTitle(messages[0].Content)
		session.Title = title
		_ = h.chat.UpdateSession(c.Request.Context(), session)
	}
	c.JSON(http.StatusOK, gin.H{"id": publicChatSessionID(session.ID), "title": title})
}

func (h *Handler) activeLLMConfig(c *gin.Context, userID uint, write bool) (*models.LLMConfig, bool) {
	config, err := h.chat.ActiveLLMConfig(c.Request.Context(), userID)
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
	return config, true
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
	session, err := h.chat.GetSession(c.Request.Context(), userID, sessionID)
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

func (h *Handler) writeChatProviderError(c *gin.Context, err error) bool {
	if err == nil {
		return true
	}
	switch {
	case errors.Is(err, chatdomain.ErrEmptyMessage):
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "content is required", stringPtr("content"))
	case errors.Is(err, chatdomain.ErrModelUnavailable):
		writeError(c, http.StatusUnprocessableEntity, "MODEL_NOT_AVAILABLE", "model is not available from the configured provider", stringPtr("model"))
	case errors.Is(err, gorm.ErrRecordNotFound):
		writeError(c, http.StatusConflict, "LLM_PROVIDER_NOT_CONFIGURED", "Configure an LLM provider before starting chat.", nil)
	default:
		writeError(c, http.StatusServiceUnavailable, "LLM_PROVIDER_UNAVAILABLE", "LLM provider is not available.", nil)
	}
	return false
}

func (h *Handler) chatStreamPayload(event chatdomain.StreamEvent) gin.H {
	switch event.Name {
	case "chat.message.created":
		assistantMessageID := ""
		if event.AssistantMessage != nil {
			assistantMessageID = event.AssistantMessage.ID
		}
		return gin.H{
			"session_id":           publicChatSessionID(event.SessionID),
			"run_id":               publicChatRunID(event.RunID),
			"message_id":           publicChatMessageID(event.MessageID),
			"assistant_message_id": publicChatMessageID(assistantMessageID),
			"user_message":         chatMessageResponse(event.UserMessage),
			"assistant_message":    chatMessageResponse(event.AssistantMessage),
		}
	case "chat.delta.reasoning", "chat.delta.content":
		return chatRunPayload(event.SessionID, event.RunID, event.MessageID, gin.H{"delta": event.Delta})
	case "chat.run.failed":
		return chatRunPayload(event.SessionID, event.RunID, event.MessageID, gin.H{"error": event.Error})
	case "chat.run.completed":
		payload := gin.H{"status": string(event.Status), "assistant_message": chatMessageResponse(event.AssistantMessage)}
		if event.Usage != nil {
			payload["usage"] = chatUsageResponse(event.Usage)
		}
		return chatRunPayload(event.SessionID, event.RunID, event.MessageID, payload)
	default:
		return chatRunPayload(event.SessionID, event.RunID, event.MessageID, gin.H{"status": string(event.Status)})
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

func chatUsageResponse(usage *chatdomain.TokenUsage) gin.H {
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
