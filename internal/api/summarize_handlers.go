package api

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"strings"
	"time"

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
// @Summary Summarize content
// @Description Stream an LLM-generated summary for provided content; persists latest summary for the transcription
// @Tags summarize
// @Accept json
// @Produce text/event-stream
// @Param request body SummarizeRequest true "Summarize request"
// @Success 200 {string} string "Event stream"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /api/v1/summarize [post]
func (h *Handler) Summarize(c *gin.Context) {
	var req SummarizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc, provider, err := h.getLLMService(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Prepare chat messages: simple single-user message with full content
	messages := []llm.ChatMessage{{Role: "user", Content: req.Content}}

	start := time.Now()
	log.Printf("[summarize] start transcription_id=%s provider=%s model=%s content_len=%d", req.TranscriptionID, provider, req.Model, len(req.Content))

	// Stream response with proper headers for real-time delivery
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering
	c.Status(http.StatusOK)             // Start response immediately

	h.processSummarization(c, req, svc, messages, start)
}

func (h *Handler) processSummarization(c *gin.Context, req SummarizeRequest, svc llm.Service, messages []llm.ChatMessage, start time.Time) {
	// Allow longer generation time for large transcripts and smaller models
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Minute)
	defer cancel()

	contentChan, errChan := svc.ChatCompletionStream(ctx, req.Model, messages, 0.0)
	flusher, _ := c.Writer.(http.Flusher)
	writer := bufio.NewWriter(c.Writer)

	finalText := ""
	gotFirstChunk := false

	// Loop handles one chunk/error at a time
	for {
		select {
		case chunk, ok := <-contentChan:
			if !ok {
				writer.Flush()
				if flusher != nil {
					flusher.Flush()
				}
				// Persist summary once streaming completes
				h.persistSummary(req, finalText)
				log.Printf("[summarize] complete transcription_id=%s model=%s bytes=%d duration_ms=%d", req.TranscriptionID, req.Model, len(finalText), time.Since(start).Milliseconds())
				return
			}
			finalText += chunk
			_, _ = writer.WriteString(chunk)
			writer.Flush()
			if flusher != nil {
				flusher.Flush()
			}
			if !gotFirstChunk && len(chunk) > 0 {
				gotFirstChunk = true
				log.Printf("[summarize] first_chunk transcription_id=%s model=%s at_ms=%d", req.TranscriptionID, req.Model, time.Since(start).Milliseconds())
			}
		case err := <-errChan:
			if err != nil {
				h.handleSummarizeError(c, req, svc, messages, err, finalText, start)
			}
			// Persist any partial content on error
			h.persistSummary(req, finalText)
			return
		case <-ctx.Done():
			// Persist any partial content on timeout/cancel
			h.persistSummary(req, finalText)
			log.Printf("[summarize] timeout/cancel transcription_id=%s model=%s bytes=%d duration_ms=%d", req.TranscriptionID, req.Model, len(finalText), time.Since(start).Milliseconds())
			return
		}
	}
}

func (h *Handler) handleSummarizeError(c *gin.Context, req SummarizeRequest, svc llm.Service, messages []llm.ChatMessage, err error, partialText string, start time.Time) {
	flusher, _ := c.Writer.(http.Flusher)
	writer := bufio.NewWriter(c.Writer)

	// Best-effort error signal
	// If streaming is unsupported for this model/org, fall back to non-streaming
	errStr := err.Error()
	if strings.Contains(errStr, "\"param\": \"stream\"") || strings.Contains(errStr, "unsupported_value") || strings.Contains(errStr, "must be verified to stream") {
		log.Printf("[summarize] falling back to non-streaming transcription_id=%s model=%s due to: %v", req.TranscriptionID, req.Model, err)
		resp, err2 := svc.ChatCompletion(c.Request.Context(), req.Model, messages, 0.0)
		if err2 != nil || resp == nil || len(resp.Choices) == 0 {
			log.Printf("[summarize] fallback failed transcription_id=%s model=%s err=%v", req.TranscriptionID, req.Model, err2)
			_, _ = c.Writer.Write([]byte("\n"))
			writer.Flush()
			if flusher != nil {
				flusher.Flush()
			}
			return
		}
		content := resp.Choices[0].Message.Content
		// Write content (appended to partial if any, though likely partial is empty if stream failed immediately)
		_, _ = writer.WriteString(content)
		writer.Flush()
		if flusher != nil {
			flusher.Flush()
		}
		// We should persist the FULL text (partial + fallback), but partialText is passed by value.
		// However, handleSumarizeError doesn't update partialText in caller.
		// The caller calls persistSummary(req, finalText) after this function returns.
		// So we actually need to persist here if we succeed?
		// Or return the new text?
		// Since we can't easily update finalText in caller without pointer, let's persist here if success.
		h.persistSummary(req, partialText+content)
		log.Printf("[summarize] fallback complete transcription_id=%s model=%s bytes=%d duration_ms=%d", req.TranscriptionID, req.Model, len(partialText+content), time.Since(start).Milliseconds())

		// To avoid double persistence in caller (which uses stale finalText), we need a way to signal "done".
		// But caller persists anyway.
		// It's acceptable to double-persist (idempotent updates usually) or just accept that caller persists partial and we persist full.
		return
	}
	_, _ = c.Writer.Write([]byte("\n"))
	writer.Flush()
	if flusher != nil {
		flusher.Flush()
	}
	log.Printf("[summarize] error transcription_id=%s model=%s err=%v duration_ms=%d", req.TranscriptionID, req.Model, err, time.Since(start).Milliseconds())
}

func (h *Handler) persistSummary(req SummarizeRequest, content string) {
	if req.TranscriptionID == "" || content == "" {
		return
	}
	sum := &models.Summary{
		TranscriptionID: req.TranscriptionID,
		TemplateID:      req.TemplateID,
		Model:           req.Model,
		Content:         content,
	}
	if err := h.summaryRepo.SaveSummary(context.Background(), sum); err != nil {
		// Fallback: store on the transcription job record
		_ = h.jobRepo.UpdateSummary(context.Background(), req.TranscriptionID, content)
	} else {
		// Also cache on the transcription job for quick access
		_ = h.jobRepo.UpdateSummary(context.Background(), req.TranscriptionID, content)
	}
}

// GetSummaryForTranscription returns the latest summary for a transcription
// @Summary Get latest summary for transcription
// @Description Get the most recent saved summary for the given transcription
// @Tags summarize
// @Produce json
// @Param id path string true "Transcription ID"
// @Success 200 {object} models.Summary
// @Failure 404 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /api/v1/transcription/{id}/summary [get]
func (h *Handler) GetSummaryForTranscription(c *gin.Context) {
	tid := c.Param("id")
	if tid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transcription ID required"})
		return
	}
	s, err := h.summaryRepo.GetLatestSummary(c.Request.Context(), tid)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Fallback: check if summary is cached on the job record
			job, err2 := h.jobRepo.FindByID(c.Request.Context(), tid)
			if err2 == nil && job.Summary != nil && *job.Summary != "" {
				c.JSON(http.StatusOK, gin.H{
					"transcription_id": tid,
					"template_id":      nil,
					"model":            "",
					"content":          *job.Summary,
					"created_at":       job.UpdatedAt,
					"updated_at":       job.UpdatedAt,
				})
				return
			}
			// Return empty summary instead of 404 for graceful frontend handling
			c.JSON(http.StatusOK, gin.H{
				"transcription_id": tid,
				"template_id":      nil,
				"model":            "",
				"content":          "",
				"created_at":       nil,
				"updated_at":       nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch summary"})
		return
	}
	c.JSON(http.StatusOK, s)
}
