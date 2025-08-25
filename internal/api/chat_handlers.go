package api

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "time"

	"scriberr/internal/database"
	"scriberr/internal/llm"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	ID              string                 `json:"id"`
	TranscriptionID string                 `json:"transcription_id"`
	Title           string                 `json:"title"`
	Model           string                 `json:"model"`
	Provider        string                 `json:"provider"`
	IsActive        bool                   `json:"is_active"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	MessageCount    int                    `json:"message_count"`
	LastActivityAt  *time.Time             `json:"last_activity_at,omitempty"`
	LastMessage     *ChatMessageResponse   `json:"last_message,omitempty"`
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

// getOpenAIService creates an OpenAI service from the active LLM config
func (h *Handler) getOpenAIService() (*llm.OpenAIService, error) {
	var llmConfig models.LLMConfig
	if err := database.DB.Where("is_active = ? AND provider = ?", true, "openai").First(&llmConfig).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no active OpenAI configuration found")
		}
		return nil, fmt.Errorf("failed to get LLM config: %w", err)
	}

	if llmConfig.APIKey == nil || *llmConfig.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}

	return llm.NewOpenAIService(*llmConfig.APIKey), nil
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
func (h *Handler) GetChatModels(c *gin.Context) {
	openaiService, err := h.getOpenAIService()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := openaiService.GetModels(ctx)
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
func (h *Handler) CreateChatSession(c *gin.Context) {
	var req ChatCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify transcription exists and has completed transcript
	var transcription models.TranscriptionJob
	if err := database.DB.Where("id = ?", req.TranscriptionID).First(&transcription).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transcription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transcription"})
		return
	}

	if transcription.Status != models.StatusCompleted || transcription.Transcript == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transcription must be completed to create a chat session"})
		return
	}

	// Verify OpenAI service is available
	_, err := h.getOpenAIService()
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
	chatSession := models.ChatSession{
		JobID:           req.TranscriptionID, // Use same ID for JobID as TranscriptionID
		TranscriptionID: req.TranscriptionID,
		Title:           title,
		Model:           req.Model,
		Provider:        "openai",
		MessageCount:    0,
		LastActivityAt:  &now,
		IsActive:        true,
	}

	if err := database.DB.Create(&chatSession).Error; err != nil {
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
func (h *Handler) GetChatSessions(c *gin.Context) {
	transcriptionID := c.Param("transcription_id")
	if transcriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transcription ID is required"})
		return
	}

	var sessions []models.ChatSession
	if err := database.DB.Where("transcription_id = ?", transcriptionID).
		Order("updated_at DESC").Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat sessions"})
		return
	}

	var responses []ChatSessionResponse
	for _, session := range sessions {
		// Count messages
		var messageCount int64
		database.DB.Model(&models.ChatMessage{}).Where("chat_session_id = ?", session.ID).Count(&messageCount)

		// Get last message
		var lastMessage models.ChatMessage
		var lastMessageResponse *ChatMessageResponse
		if err := database.DB.Where("chat_session_id = ?", session.ID).
			Order("created_at DESC").First(&lastMessage).Error; err == nil {
			lastMessageResponse = &ChatMessageResponse{
				ID:        lastMessage.ID,
				Role:      lastMessage.Role,
				Content:   lastMessage.Content,
				CreatedAt: lastMessage.CreatedAt,
			}
		}

		responses = append(responses, ChatSessionResponse{
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
			LastMessage:     lastMessageResponse,
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
func (h *Handler) GetChatSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	var session models.ChatSession
	if err := database.DB.Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat session"})
		return
	}

	var messages []models.ChatMessage
	if err := database.DB.Where("chat_session_id = ?", sessionID).
		Order("created_at ASC").Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}

	var messageResponses []ChatMessageResponse
	for _, msg := range messages {
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
	var session models.ChatSession
	if err := database.DB.Preload("Transcription").Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat session"})
		return
	}

	// Get OpenAI service
	openaiService, err := h.getOpenAIService()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save user message
	userMessage := models.ChatMessage{
		SessionID:     sessionID,
		ChatSessionID: sessionID,
		Role:          "user",
		Content:       req.Content,
	}

	if err := database.DB.Create(&userMessage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save message"})
		return
	}

	// Check if this is the first user message and update session title
	var messageCount int64
	database.DB.Model(&models.ChatMessage{}).Where("chat_session_id = ? AND role = ?", sessionID, "user").Count(&messageCount)
	if messageCount == 1 {
		// Generate a title based on the first message
		title := generateChatTitle(req.Content)
		database.DB.Model(&session).Update("title", title)
	}

	// Get conversation history
	var messages []models.ChatMessage
	database.DB.Where("chat_session_id = ?", sessionID).Order("created_at ASC").Find(&messages)

	// Build OpenAI messages including transcript context
	var openaiMessages []llm.ChatMessage
	
	// Add system message with transcript context
	if session.Transcription.Transcript != nil {
		systemContent := fmt.Sprintf("You are a helpful assistant analyzing this transcript. Please answer questions and provide insights based on the following transcript:\n\n%s", *session.Transcription.Transcript)
		openaiMessages = append(openaiMessages, llm.ChatMessage{
			Role:    "system",
			Content: systemContent,
		})
	}

	// Add conversation history
	for _, msg := range messages {
		openaiMessages = append(openaiMessages, llm.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Set up streaming response
	c.Header("Content-Type", "text/plain")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Stream the response
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	contentChan, errorChan := openaiService.ChatCompletionStream(ctx, session.Model, openaiMessages, 0.7)

	var assistantResponse strings.Builder
	for {
		select {
		case content, ok := <-contentChan:
			if !ok {
				// Channel closed, save complete response and return
				if assistantResponse.Len() > 0 {
					assistantMessage := models.ChatMessage{
						SessionID:     sessionID,
						ChatSessionID: sessionID,
						Role:          "assistant",
						Content:       assistantResponse.String(),
					}
					database.DB.Create(&assistantMessage)

					// Update session updated_at, message count, and last activity
					now := time.Now()
					database.DB.Model(&session).Updates(map[string]interface{}{
						"updated_at": now,
						"last_activity_at": now,
						"message_count": gorm.Expr("message_count + ?", 2), // +2 for user + assistant message
					})
				}
				return
			}
			
			// Write content to response
			c.Writer.WriteString(content)
			c.Writer.Flush()
			assistantResponse.WriteString(content)

		case err := <-errorChan:
			if err != nil {
				c.Writer.WriteString("\nError: " + err.Error())
				c.Writer.Flush()
				return
			}

		case <-ctx.Done():
			c.Writer.WriteString("\nRequest timeout")
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

	var session models.ChatSession
	if err := database.DB.Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat session"})
		return
	}

	session.Title = req.Title
	if err := database.DB.Save(&session).Error; err != nil {
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
func (h *Handler) DeleteChatSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	// Delete messages first (due to foreign key constraint)
	if err := database.DB.Where("chat_session_id = ?", sessionID).Delete(&models.ChatMessage{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete messages"})
		return
	}

	// Delete session
	result := database.DB.Where("id = ?", sessionID).Delete(&models.ChatSession{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chat session"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat session not found"})
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
func (h *Handler) AutoGenerateChatTitle(c *gin.Context) {
    sessionID := c.Param("session_id")
    if sessionID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
        return
    }

    // Load session
    var session models.ChatSession
    if err := database.DB.Where("id = ?", sessionID).First(&session).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "Chat session not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat session"})
        return
    }

    // Determine if title appears user-unset (default or derived from first user message)
    isDefaultTitle := strings.EqualFold(strings.TrimSpace(session.Title), "New Chat Session")

    // Load first user message to compare against simple generator
    var firstUser models.ChatMessage
    _ = database.DB.Where("chat_session_id = ? AND role = ?", sessionID, "user").Order("created_at ASC").First(&firstUser).Error
    if firstUser.ID != 0 {
        simple := generateChatTitle(firstUser.Content)
        if strings.EqualFold(strings.TrimSpace(session.Title), strings.TrimSpace(simple)) {
            isDefaultTitle = true
        }
    }

    if !isDefaultTitle {
        // Respect user-edited titles; return current session response
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
        return
    }

    // Fetch recent messages (first user + first assistant ideally)
    var msgs []models.ChatMessage
    if err := database.DB.Where("chat_session_id = ?", sessionID).Order("created_at ASC").Limit(6).Find(&msgs).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
        return
    }
    if len(msgs) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough conversation to generate a title"})
        return
    }

    // Prepare LLM messages
    prompt := `You are an expert at titling conversations concisely.
Given the following early conversation messages between a user and an assistant, produce a short, specific title (max 8 words) in Title Case.
- No quotes, no trailing punctuation, no emojis.
- Avoid generic words like "Chat", "Conversation", or model names.
- Capture the main topic or task.
Return ONLY the title string.`

    var chatMsgs []llm.ChatMessage
    chatMsgs = append(chatMsgs, llm.ChatMessage{Role: "system", Content: prompt})
    // Include up to first 2-4 messages for context
    maxCtx := len(msgs)
    if maxCtx > 4 {
        maxCtx = 4
    }
    for i := 0; i < maxCtx; i++ {
        role := msgs[i].Role
        if role != "user" && role != "assistant" {
            role = "user"
        }
        chatMsgs = append(chatMsgs, llm.ChatMessage{Role: role, Content: msgs[i].Content})
    }

    // Use configured OpenAI service
    openaiService, err := h.getOpenAIService()
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    resp, err := openaiService.ChatCompletion(ctx, session.Model, chatMsgs, 0.2)
    if err != nil || resp == nil || len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate title"})
        return
    }

    title := strings.TrimSpace(resp.Choices[0].Message.Content)
    // Sanitize: enforce max length and single line
    title = strings.ReplaceAll(title, "\n", " ")
    title = strings.ReplaceAll(title, "\r", " ")
    if len(title) > 60 {
        title = title[:57] + "..."
    }

    // Update session title
    if err := database.DB.Model(&session).Update("title", title).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update title"})
        return
    }

    // Reload to return response
    var updated models.ChatSession
    if err := database.DB.Where("id = ?", sessionID).First(&updated).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load updated session"})
        return
    }

    c.JSON(http.StatusOK, ChatSessionResponse{
        ID:              updated.ID,
        TranscriptionID: updated.TranscriptionID,
        Title:           updated.Title,
        Model:           updated.Model,
        Provider:        updated.Provider,
        IsActive:        updated.IsActive,
        CreatedAt:       updated.CreatedAt,
        UpdatedAt:       updated.UpdatedAt,
        MessageCount:    updated.MessageCount,
        LastActivityAt:  updated.LastActivityAt,
    })
}
