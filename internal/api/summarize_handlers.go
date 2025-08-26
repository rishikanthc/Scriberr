package api

import (
    "bufio"
    "context"
    "net/http"
    "time"

    "scriberr/internal/database"
    "scriberr/internal/llm"
    "scriberr/internal/models"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

type SummarizeRequest struct {
    Model           string  `json:"model" binding:"required"`
    Content         string  `json:"content" binding:"required"`
    TranscriptionID string  `json:"transcription_id" binding:"required"`
    TemplateID      *string `json:"template_id,omitempty"`
}

// Summarize streams LLM output for a given content prompt
func (h *Handler) Summarize(c *gin.Context) {
    var req SummarizeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    openaiService, err := h.getOpenAIService()
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Prepare chat messages: simple single-user message with full content
    messages := []llm.ChatMessage{{Role: "user", Content: req.Content}}

    // Stream response
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")

    ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
    defer cancel()

    contentChan, errChan := openaiService.ChatCompletionStream(ctx, req.Model, messages, 0.0)
    flusher, _ := c.Writer.(http.Flusher)
    writer := bufio.NewWriter(c.Writer)

    finalText := ""
    for {
        select {
        case chunk, ok := <-contentChan:
            if !ok {
                writer.Flush()
                if flusher != nil {
                    flusher.Flush()
                }
                // Persist summary once streaming completes
                if req.TranscriptionID != "" && finalText != "" {
                    sum := models.Summary{
                        TranscriptionID: req.TranscriptionID,
                        TemplateID:      req.TemplateID,
                        Model:           req.Model,
                        Content:         finalText,
                    }
                    _ = database.DB.Create(&sum).Error
                }
                return
            }
            finalText += chunk
            writer.WriteString(chunk)
            writer.Flush()
            if flusher != nil {
                flusher.Flush()
            }
        case err := <-errChan:
            if err != nil {
                // Best-effort error signal
                c.Writer.Write([]byte("\n"))
                writer.Flush()
                if flusher != nil {
                    flusher.Flush()
                }
            }
            return
        case <-ctx.Done():
            return
        }
    }
}

// GetSummaryForTranscription returns the latest summary for a transcription
func (h *Handler) GetSummaryForTranscription(c *gin.Context) {
    tid := c.Param("id")
    if tid == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Transcription ID required"})
        return
    }
    var s models.Summary
    if err := database.DB.Where("transcription_id = ?", tid).Order("created_at DESC").First(&s).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "Summary not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch summary"})
        return
    }
    c.JSON(http.StatusOK, s)
}
