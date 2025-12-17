package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"scriberr/internal/llm"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	StylePrompt = "\n\nINSTRUCTIONS: Return your answer as a raw markdown string. \n1. Use LaTeX for equations (e.g., $E=mc^2$). \n2. Do NOT use code block fences (```) around the entire response. \n3. Do NOT include any meta-comments (e.g., \"Here is the markdown...\"). \n4. Just provide the raw content."
	RoleUser    = "user"
)

// ChatCreateRequest represents a request to create a new chat session
type ChatCreateRequest struct {
	TranscriptionID string `json:"transcription_id" binding:"required"`
	Model           string `json:"model" binding:"required"`
	Title           string `json:"title,omitempty"`
}

// ChatMessageRequest represents a request to send a message
type ChatMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// ChatSessionResponse represents a chat session response
type ChatSessionResponse struct {
	ID              string               `json:"id"`
	TranscriptionID string               `json:"transcription_id"`
	Title           string               `json:"title"`
	Model           string               `json:"model"`
	Provider        string               `json:"provider"`
	IsActive        bool                 `json:"is_active"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	MessageCount    int                  `json:"message_count"`
	LastActivityAt  *time.Time           `json:"last_activity_at,omitempty"`
	LastMessage     *ChatMessageResponse `json:"last_message,omitempty"`
}

// ChatMessageResponse represents a chat message response
type ChatMessageResponse struct {
	ID        uint      `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatModelsResponse represents the available chat models
type ChatModelsResponse struct {
	Models []string `json:"models"`
}

// ChatSessionWithMessages represents a chat session with messages
type ChatSessionWithMessages struct {
	ChatSessionResponse
	Messages []ChatMessageResponse `json:"messages"`
}

type Transcript struct {
	Segments []Segment `json:"segments"`
}

type Segment struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Text    string  `json:"text"`
	Speaker string  `json:"speaker"`
}

// getLLMService returns a provider-agnostic LLM service based on active config
func (h *Handler) getLLMService(ctx context.Context) (llm.Service, string, error) {
	cfg, err := h.llmConfigRepo.GetActive(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, "", fmt.Errorf("no active LLM configuration found")
		}
		return nil, "", fmt.Errorf("failed to get LLM config: %w", err)
	}
	switch strings.ToLower(cfg.Provider) {
	case "openai":
		if cfg.APIKey == nil || *cfg.APIKey == "" {
			return nil, cfg.Provider, fmt.Errorf("OpenAI API key not configured")
		}
		return llm.NewOpenAIService(*cfg.APIKey, cfg.OpenAIBaseURL), cfg.Provider, nil
	case "ollama":
		if cfg.BaseURL == nil || *cfg.BaseURL == "" {
			return nil, cfg.Provider, fmt.Errorf("Ollama base URL not configured")
		}
		return llm.NewOllamaService(*cfg.BaseURL), cfg.Provider, nil
	default:
		return nil, cfg.Provider, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}

// @Summary Get available chat models
// @Description Get list of available OpenAI chat models
// @Tags chat
// @Produce json
// @Success 200 {object} ChatModelsResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/chat/models [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetChatModels(c *gin.Context) {
	svc, _, err := h.getLLMService(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := svc.GetModels(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch models: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ChatModelsResponse{Models: models})
}

// @Summary Create a new chat session
// @Description Create a new chat session for a transcription
// @Tags chat
// @Accept json
// @Produce json
// @Param request body ChatCreateRequest true "Chat session creation request"
// @Success 201 {object} ChatSessionResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/chat/sessions [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) CreateChatSession(c *gin.Context) {
	var req ChatCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify transcription exists and has completed transcript
	transcription, err := h.jobRepo.FindByID(c.Request.Context(), req.TranscriptionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transcription not found"})
		return
	}

	if transcription.Status != models.StatusCompleted || transcription.Transcript == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transcription must be completed to create a chat session"})
		return
	}

	// Verify LLM service is available
	_, _, err = h.getLLMService(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create chat session
	title := req.Title
	if title == "" {
		title = "New Chat Session"
	}

	now := time.Now()
	chatSession := &models.ChatSession{
		JobID:           req.TranscriptionID, // Use same ID for JobID as TranscriptionID
		TranscriptionID: req.TranscriptionID,
		Title:           title,
		Model:           req.Model,
		Provider:        "openai",
		MessageCount:    0,
		LastActivityAt:  &now,
		IsActive:        true,
	}

	if err := h.chatRepo.Create(c.Request.Context(), chatSession); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat session"})
		return
	}

	response := ChatSessionResponse{
		ID:              chatSession.ID,
		TranscriptionID: chatSession.TranscriptionID,
		Title:           chatSession.Title,
		Model:           chatSession.Model,
		Provider:        chatSession.Provider,
		IsActive:        chatSession.IsActive,
		CreatedAt:       chatSession.CreatedAt,
		UpdatedAt:       chatSession.UpdatedAt,
		MessageCount:    chatSession.MessageCount,
		LastActivityAt:  chatSession.LastActivityAt,
	}

	c.JSON(http.StatusCreated, response)
}

// @Summary Get chat sessions for a transcription
// @Description Get all chat sessions for a specific transcription
// @Tags chat
// @Produce json
// @Param transcription_id path string true "Transcription ID"
// @Success 200 {array} ChatSessionResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/chat/transcriptions/{transcription_id}/sessions [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetChatSessions(c *gin.Context) {
	transcriptionID := c.Param("transcription_id")
	if transcriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transcription ID is required"})
		return
	}

	sessions, err := h.chatRepo.ListByJob(c.Request.Context(), transcriptionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat sessions"})
		return
	}

	// Extract session IDs for batch queries
	sessionIDs := make([]string, len(sessions))
	for i, session := range sessions {
		sessionIDs[i] = session.ID
	}

	// Batch query for message counts - eliminates N+1 problem
	messageCountMap, _ := h.chatRepo.GetMessageCountsBySessionIDs(c.Request.Context(), sessionIDs)

	// Batch query for last messages - eliminates N+1 problem
	lastMsgsMap, _ := h.chatRepo.GetLastMessagesBySessionIDs(c.Request.Context(), sessionIDs)

	// Create last message response lookup map
	lastMessageMap := make(map[string]*ChatMessageResponse)
	for sessionID, msg := range lastMsgsMap {
		lastMessageMap[sessionID] = &ChatMessageResponse{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		}
	}

	var responses []ChatSessionResponse
	for _, session := range sessions {
		responses = append(responses, ChatSessionResponse{
			ID:              session.ID,
			TranscriptionID: session.TranscriptionID,
			Title:           session.Title,
			Model:           session.Model,
			Provider:        session.Provider,
			IsActive:        session.IsActive,
			CreatedAt:       session.CreatedAt,
			UpdatedAt:       session.UpdatedAt,
			MessageCount:    int(messageCountMap[session.ID]), // Use batch-loaded count
			LastActivityAt:  session.LastActivityAt,
			LastMessage:     lastMessageMap[session.ID], // Use batch-loaded last message
		})
	}

	c.JSON(http.StatusOK, responses)
}

// @Summary Get a chat session with messages
// @Description Get a specific chat session with all its messages
// @Tags chat
// @Produce json
// @Param session_id path string true "Chat Session ID"
// @Success 200 {object} ChatSessionWithMessages
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/chat/sessions/{session_id} [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetChatSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	session, err := h.chatRepo.GetSessionWithMessages(c.Request.Context(), sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat session"})
		return
	}

	var messageResponses []ChatMessageResponse
	for _, msg := range session.Messages {
		messageResponses = append(messageResponses, ChatMessageResponse{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		})
	}

	response := ChatSessionWithMessages{
		ChatSessionResponse: ChatSessionResponse{
			ID:              session.ID,
			TranscriptionID: session.TranscriptionID,
			Title:           session.Title,
			Model:           session.Model,
			Provider:        session.Provider,
			IsActive:        session.IsActive,
			CreatedAt:       session.CreatedAt,
			UpdatedAt:       session.UpdatedAt,
			MessageCount:    len(messageResponses),
			LastActivityAt:  session.LastActivityAt,
		},
		Messages: messageResponses,
	}

	c.JSON(http.StatusOK, response)
}

// format time from transcription json as 00:00:00
func formatTime(seconds float64) string {
	s := int(math.Round(seconds))

	hours := s / 3600
	minutes := (s % 3600) / 60
	secs := s % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// @Summary Send a message to a chat session
// @Description Send a message to a chat session and get streaming response
// @Tags chat
// @Accept json
// @Produce text/plain
// @Param session_id path string true "Chat Session ID"
// @Param message body ChatMessageRequest true "Message content"
// @Success 200 {string} string "Streaming response"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/chat/sessions/{session_id}/messages [post]
// @Security ApiKeyAuth
// @Security BearerAuth
//
//nolint:gocyclo // Streaming chat logic is inherently complex
func (h *Handler) SendChatMessage(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	var req ChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get chat session
	session, err := h.chatRepo.GetSessionWithTranscription(c.Request.Context(), sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat session"})
		return
	}

	// Get LLM service
	svc, _, err := h.getLLMService(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save user message
	userMessage := &models.ChatMessage{
		SessionID:     sessionID,
		ChatSessionID: sessionID,
		Role:          RoleUser,
		Content:       req.Content,
	}

	if err := h.chatRepo.AddMessage(c.Request.Context(), userMessage); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save message"})
		return
	}

	// Check if this is the first user message and update session title
	messages, err := h.chatRepo.GetMessages(c.Request.Context(), sessionID, 0)
	if err == nil {
		userMsgCount := 0
		for _, m := range messages {
			if m.Role == RoleUser {
				userMsgCount++
			}
		}
		if userMsgCount == 1 {
			// Generate a title based on the first message
			title := generateChatTitle(req.Content)
			session.Title = title
			_ = h.chatRepo.Update(c.Request.Context(), session)
		}
	}

	// Get context window
	contextWindow, err := svc.GetContextWindow(c.Request.Context(), session.Model)
	if err != nil {
		fmt.Printf("Failed to get context window for model %s: %v. Using default 4096.\n", session.Model, err)
		contextWindow = 4096
	}

	// Build OpenAI messages including transcript context
	var openaiMessages []llm.ChatMessage
	var currentTokenCount int
	var transcriptContext string

	// Fallback: If transcript wasn't loaded via Preload, fetch it directly from the job repository
	if session.Transcription.Transcript == nil || *session.Transcription.Transcript == "" {
		fmt.Printf("Debug: Transcript not loaded via Preload for session %s (TranscriptionID: %s), fetching directly...\n", sessionID, session.TranscriptionID)
		job, jobErr := h.jobRepo.FindByID(c.Request.Context(), session.TranscriptionID)
		if jobErr == nil && job != nil && job.Transcript != nil && *job.Transcript != "" {
			session.Transcription.Transcript = job.Transcript
			fmt.Printf("Debug: Direct fetch succeeded, transcript length: %d\n", len(*job.Transcript))
		} else {
			fmt.Printf("Debug: Direct fetch failed or transcript empty. Error: %v\n", jobErr)
		}
	}

	// Add system message with transcript context
	if session.Transcription.Transcript != nil && *session.Transcription.Transcript != "" {
		transcript := *session.Transcription.Transcript
		fmt.Printf("Debug: Transcript found for session %s. Length: %d\n", sessionID, len(transcript))

		// Parse transcript json segments and build string with format: [SPEAKER_01] [00:00:17 - 00:00:19] Nej, det var tråkigt att höra.
		var t Transcript
		if err := json.Unmarshal([]byte(transcript), &t); err != nil {
			fmt.Printf("Error parsing transcript JSON for session %s: %v\n", sessionID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse transcript data"})
			return
		}

		fmt.Printf("Debug: Parsed %d segments from transcript\n", len(t.Segments))

		var sb strings.Builder

		// Get speaker mappings
		mappings, err := h.speakerMappingRepo.ListByJob(c.Request.Context(), session.TranscriptionID)
		speakerMap := make(map[string]string)
		if err == nil {
			for _, m := range mappings {
				speakerMap[m.OriginalSpeaker] = m.CustomName
			}
		} else {
			fmt.Printf("Failed to get speaker mappings for job %s: %v\n", session.TranscriptionID, err)
		}

		for _, seg := range t.Segments {
			start := formatTime(seg.Start)
			end := formatTime(seg.End)

			speakerName := seg.Speaker
			if customName, ok := speakerMap[speakerName]; ok {
				speakerName = customName
			}

			fmt.Fprintf(&sb, "[%s] [%s - %s] %s\n",
				speakerName,
				start,
				end,
				strings.TrimSpace(seg.Text),
			)
		}

		cleanTranscript := sb.String()
		fmt.Printf("Debug: Clean transcript length: %d\n", len(cleanTranscript))

		// Build transcript context - will be prepended to first user message for better model compatibility
		transcriptContext = fmt.Sprintf("You are analyzing the following transcript. Use this transcript to answer questions:\n\n---TRANSCRIPT START---\n%s\n---TRANSCRIPT END---\n\n", cleanTranscript)

		fmt.Printf("Injecting transcript of length %d into chat context for session %s\n", len(transcriptContext), sessionID)

		// Check if transcript itself exceeds context (leaving some room for response)
		// Estimate 1 token ~= 4 chars
		transcriptTokens := len(transcriptContext) / 4
		if transcriptTokens > contextWindow-500 { // Leave 500 tokens for response/history
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Transcript is too long for this model's context window (estimated %d tokens, limit %d). Please use a model with a larger context window.", transcriptTokens, contextWindow)})
			return
		}
		currentTokenCount += transcriptTokens
	} else {
		fmt.Printf("Warning: Transcript is nil or empty for chat session %s. Transcription ID: %s\n", sessionID, session.TranscriptionID)
		if session.Transcription.ID == "" {
			fmt.Println("Warning: Session Transcription relation seems missing or empty")
		}
	}

	// Add conversation history with transcript context prepended to first user message
	for i, msg := range messages {
		msgContent := msg.Content
		// Prepend transcript context to the first user message for better model compatibility
		// (Some models like Qwen3 don't properly handle system messages)
		if i == 0 && msg.Role == RoleUser && transcriptContext != "" {
			msgContent = transcriptContext + "User question: " + msg.Content
			fmt.Printf("Debug: Prepended transcript to first user message\n")
		}
		msgTokens := len(msgContent) / 4

		// Inject style prompt for user messages (in-memory only, not saved to DB)
		finalContent := msgContent
		if msg.Role == RoleUser {
			finalContent += StylePrompt
		}

		openaiMessages = append(openaiMessages, llm.ChatMessage{
			Role:    msg.Role,
			Content: finalContent,
		})
		currentTokenCount += msgTokens
	}

	// Intelligent context trimming: if context exceeds limit, remove oldest messages
	// Keep the first message (with transcript context) and trim from the middle
	trimmedCount := 0
	for currentTokenCount > contextWindow && len(openaiMessages) > 2 {
		// Remove the second message (oldest after the context-bearing first message)
		removed := openaiMessages[1]
		removedTokens := len(removed.Content) / 4
		openaiMessages = append(openaiMessages[:1], openaiMessages[2:]...)
		currentTokenCount -= removedTokens
		trimmedCount++
		fmt.Printf("Debug: Trimmed message to fit context. Removed %d tokens, new count: %d/%d\n", removedTokens, currentTokenCount, contextWindow)
	}

	if trimmedCount > 0 {
		fmt.Printf("Debug: Trimmed %d messages to fit context window\n", trimmedCount)
	}

	// Final check - if still over limit after trimming all possible messages, return error
	if currentTokenCount > contextWindow {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Transcript alone exceeds model context limit (%d tokens > %d). Please use a model with larger context window.", currentTokenCount, contextWindow)})
		return
	}

	// Set up streaming response with context info headers
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// CORS headers for streaming
	origin := c.Request.Header.Get("Origin")
	allowOrigin := "*"
	if h.config.IsProduction() && len(h.config.AllowedOrigins) > 0 {
		allowOrigin = ""
		for _, allowed := range h.config.AllowedOrigins {
			if origin == allowed {
				allowOrigin = origin
				break
			}
		}
	} else if origin != "" {
		allowOrigin = origin
	}
	if allowOrigin != "" {
		c.Header("Access-Control-Allow-Origin", allowOrigin)
		c.Header("Access-Control-Allow-Credentials", "true")
	}
	c.Header("Access-Control-Expose-Headers", "X-Context-Used, X-Context-Limit, X-Messages-Trimmed")
	c.Header("X-Context-Used", fmt.Sprintf("%d", currentTokenCount))
	c.Header("X-Context-Limit", fmt.Sprintf("%d", contextWindow))
	c.Header("X-Messages-Trimmed", fmt.Sprintf("%d", trimmedCount))
	c.Status(http.StatusOK) // Start the response immediately

	// Stream the response
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	// Use model defaults: do not set temperature explicitly
	contentChan, errorChan := svc.ChatCompletionStream(ctx, session.Model, openaiMessages, 0.0)

	var assistantResponse strings.Builder
	for {
		select {
		case content, ok := <-contentChan:
			if !ok {
				// Channel closed, save complete response and return
				if assistantResponse.Len() > 0 {
					assistantMessage := &models.ChatMessage{
						SessionID:     sessionID,
						ChatSessionID: sessionID,
						Role:          "assistant",
						Content:       assistantResponse.String(),
					}
					_ = h.chatRepo.AddMessage(context.Background(), assistantMessage)

					// Update session updated_at, message count, and last activity
					now := time.Now()
					session.UpdatedAt = now
					session.LastActivityAt = &now
					session.MessageCount += 2 // +2 for user + assistant message
					_ = h.chatRepo.Update(context.Background(), session)
				}
				return
			}

			// Write content to response
			_, _ = c.Writer.WriteString(content)
			c.Writer.Flush()
			assistantResponse.WriteString(content)

		case err := <-errorChan:
			if err != nil {
				// If streaming is not supported for this model/org, fall back to non-streaming
				errStr := err.Error()
				if strings.Contains(errStr, "\"param\": \"stream\"") || strings.Contains(errStr, "unsupported_value") || strings.Contains(errStr, "must be verified to stream") {
					resp, err2 := svc.ChatCompletion(ctx, session.Model, openaiMessages, 0.0)
					if err2 != nil || resp == nil || len(resp.Choices) == 0 {
						_, _ = c.Writer.WriteString("\nError: " + err2.Error())
						c.Writer.Flush()
						return
					}
					content := resp.Choices[0].Message.Content
					_, _ = c.Writer.WriteString(content)
					c.Writer.Flush()
					assistantResponse.WriteString(content)

					if assistantResponse.Len() > 0 {
						assistantMessage := &models.ChatMessage{
							SessionID:     sessionID,
							ChatSessionID: sessionID,
							Role:          "assistant",
							Content:       assistantResponse.String(),
						}
						_ = h.chatRepo.AddMessage(context.Background(), assistantMessage)

						// Update session updated_at, message count, and last activity
						now := time.Now()
						session.UpdatedAt = now
						session.LastActivityAt = &now
						session.MessageCount += 2 // +2 for user + assistant message
						_ = h.chatRepo.Update(context.Background(), session)
					}
					return
				}

				// Otherwise, return the error to the client
				_, _ = c.Writer.WriteString("\nError: " + err.Error())
				c.Writer.Flush()
				return
			}

		case <-ctx.Done():
			_, _ = c.Writer.WriteString("\nRequest timeout")
			c.Writer.Flush()
			return
		}
	}
}

// @Summary Update chat session title
// @Description Update the title of a chat session
// @Tags chat
// @Accept json
// @Produce json
// @Param session_id path string true "Chat Session ID"
// @Param request body map[string]string true "Title update request"
// @Success 200 {object} ChatSessionResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/chat/sessions/{session_id}/title [put]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) UpdateChatSessionTitle(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	var req struct {
		Title string `json:"title" binding:"required,min=1,max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.chatRepo.FindByID(c.Request.Context(), sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat session"})
		return
	}

	session.Title = req.Title
	if err := h.chatRepo.Update(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update title"})
		return
	}

	response := ChatSessionResponse{
		ID:              session.ID,
		TranscriptionID: session.TranscriptionID,
		Title:           session.Title,
		Model:           session.Model,
		Provider:        session.Provider,
		IsActive:        session.IsActive,
		CreatedAt:       session.CreatedAt,
		UpdatedAt:       session.UpdatedAt,
		MessageCount:    session.MessageCount,
		LastActivityAt:  session.LastActivityAt,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Delete a chat session
// @Description Delete a chat session and all its messages
// @Tags chat
// @Produce json
// @Param session_id path string true "Chat Session ID"
// @Success 204
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/chat/sessions/{session_id} [delete]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) DeleteChatSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	if err := h.chatRepo.DeleteSession(c.Request.Context(), sessionID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chat session"})
		return
	}

	c.Status(http.StatusNoContent)
}

// generateChatTitle generates a title based on the first user message
func generateChatTitle(message string) string {
	// Truncate to reasonable length and clean up
	title := strings.TrimSpace(message)
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	// Remove newlines and replace with spaces
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.ReplaceAll(title, "\r", " ")

	// Replace multiple spaces with single space
	for strings.Contains(title, "  ") {
		title = strings.ReplaceAll(title, "  ", " ")
	}

	return title
}

// AutoGenerateChatTitle generates a session title using the configured LLM based on conversation history
// @Summary Auto-generate chat session title
// @Description Uses the configured LLM to summarize the first exchange into a concise title. Only updates if the current title appears default/user-unset.
// @Tags chat
// @Produce json
// @Param session_id path string true "Chat Session ID"
// @Success 200 {object} ChatSessionResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/chat/sessions/{session_id}/title/auto [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) AutoGenerateChatTitle(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	session, err := h.chatRepo.FindByID(c.Request.Context(), sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat session"})
		return
	}

	if defaultTitle, err := h.isDefaultTitle(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check title status"})
		return
	} else if !defaultTitle {
		h.respondWithSession(c, session)
		return
	}

	recentMsgs, err := h.chatRepo.GetMessages(c.Request.Context(), sessionID, 6)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}
	if len(recentMsgs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough conversation to generate a title"})
		return
	}

	svc, _, err := h.getLLMService(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	title, err := h.generateTitleFromLLM(c.Request.Context(), svc, session.Model, recentMsgs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate title"})
		return
	}

	session.Title = title
	if err := h.chatRepo.Update(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update title"})
		return
	}

	// Reload to return response
	updated, err := h.chatRepo.FindByID(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load updated session"})
		return
	}
	h.respondWithSession(c, updated)
}

func (h *Handler) isDefaultTitle(ctx context.Context, session *models.ChatSession) (bool, error) {
	isDefault := strings.EqualFold(strings.TrimSpace(session.Title), "New Chat Session")
	if isDefault {
		return true, nil
	}

	// Check if title matches simple heuristic from first message
	msgs, err := h.chatRepo.GetMessages(ctx, session.ID, 0)
	if err != nil {
		return false, err
	}
	if len(msgs) > 0 {
		for _, m := range msgs {
			if m.Role == RoleUser {
				simple := generateChatTitle(m.Content)
				if strings.EqualFold(strings.TrimSpace(session.Title), strings.TrimSpace(simple)) {
					return true, nil
				}
				break
			}
		}
	}
	return false, nil
}

func (h *Handler) generateTitleFromLLM(ctx context.Context, svc llm.Service, model string, msgs []models.ChatMessage) (string, error) {
	prompt := `You are an expert at creating concise, meaningful titles for conversations. Based on the conversation below, generate a short, descriptive title (3-8 words) that captures the main topic or purpose.

Guidelines:
- Use Title Case formatting (Every Important Word Capitalized)
- Be specific and descriptive, not generic
- Focus on the core subject matter or task being discussed
- Avoid generic terms like "Chat", "Discussion", "Question", "Conversation"
- Avoid mentioning AI, Assistant, or model names
- No quotation marks, brackets, or punctuation at the end
- No emojis or special characters
- Make it something a user would easily recognize and remember

Examples of good titles:
- "Python Data Analysis Tutorial"
- "Marketing Strategy Planning Session"
- "JavaScript Debugging Help"
- "Recipe for Chocolate Cake"
- "React Component Architecture"

Return only the title, nothing else.`

	var chatMsgs []llm.ChatMessage
	chatMsgs = append(chatMsgs, llm.ChatMessage{Role: "system", Content: prompt})

	for _, msg := range msgs {
		role := msg.Role
		if role != "user" && role != "assistant" {
			role = "user"
		}
		// Truncate very long messages
		content := msg.Content
		if len(content) > 500 {
			content = content[:497] + "..."
		}
		chatMsgs = append(chatMsgs, llm.ChatMessage{Role: role, Content: content})
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := svc.ChatCompletion(timeoutCtx, model, chatMsgs, 0.0)
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty response from LLM")
	}

	title := strings.TrimSpace(resp.Choices[0].Message.Content)
	// Strip wrapping quotes/backticks
	title = strings.Trim(title, "'\"`")

	// Sanitize
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.ReplaceAll(title, "\r", " ")
	if len(title) > 60 {
		title = title[:57] + "..."
	}
	return title, nil
}

func (h *Handler) respondWithSession(c *gin.Context, session *models.ChatSession) {
	c.JSON(http.StatusOK, ChatSessionResponse{
		ID:              session.ID,
		TranscriptionID: session.TranscriptionID,
		Title:           session.Title,
		Model:           session.Model,
		Provider:        session.Provider,
		IsActive:        session.IsActive,
		CreatedAt:       session.CreatedAt,
		UpdatedAt:       session.UpdatedAt,
		MessageCount:    session.MessageCount,
		LastActivityAt:  session.LastActivityAt,
	})
}
