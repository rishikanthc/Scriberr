package api

import (
	"errors"
	"net/http"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handler) getTranscriptionSummary(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	var summary models.Summary
	err := database.DB.WithContext(c.Request.Context()).
		Where("transcription_id = ? AND user_id = ?", job.ID, job.UserID).
		Order("created_at DESC").
		First(&summary).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "summary not found", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load summary", nil)
		return
	}
	c.JSON(http.StatusOK, summaryResponse(&summary))
}

func summaryResponse(summary *models.Summary) gin.H {
	errorValue := any(nil)
	if summary.ErrorMessage != nil && *summary.ErrorMessage != "" {
		errorValue = sanitizePublicText(*summary.ErrorMessage)
	}
	return gin.H{
		"id":                   summary.ID,
		"transcription_id":     "tr_" + summary.TranscriptionID,
		"content":              summary.Content,
		"model":                summary.Model,
		"provider":             summary.Provider,
		"status":               summary.Status,
		"error":                errorValue,
		"transcript_truncated": summary.TranscriptTruncated,
		"context_window":       summary.ContextWindow,
		"input_characters":     summary.InputCharacters,
		"created_at":           summary.CreatedAt,
		"updated_at":           summary.UpdatedAt,
		"completed_at":         summary.CompletedAt,
	}
}

func (h *Handler) listTranscriptionSummaryWidgets(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	runs, err := repository.NewSummaryRepository(database.DB).ListSummaryWidgetRunsByTranscription(c.Request.Context(), job.ID, job.UserID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load summary widgets", nil)
		return
	}
	items := make([]gin.H, 0, len(runs))
	for i := range runs {
		items = append(items, summaryWidgetRunResponse(&runs[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}

func summaryWidgetRunResponse(run *models.SummaryWidgetRun) gin.H {
	errorValue := any(nil)
	if run.ErrorMessage != nil && *run.ErrorMessage != "" {
		errorValue = sanitizePublicText(*run.ErrorMessage)
	}
	return gin.H{
		"id":                run.ID,
		"summary_id":        run.SummaryID,
		"transcription_id":  "tr_" + run.TranscriptionID,
		"widget_id":         run.WidgetID,
		"widget_name":       run.WidgetName,
		"display_title":     run.DisplayTitle,
		"context_source":    run.ContextSource,
		"render_markdown":   run.RenderMarkdown,
		"model":             run.Model,
		"provider":          run.Provider,
		"status":            run.Status,
		"output":            run.Output,
		"error":             errorValue,
		"context_truncated": run.ContextTruncated,
		"context_window":    run.ContextWindow,
		"input_characters":  run.InputCharacters,
		"created_at":        run.CreatedAt,
		"updated_at":        run.UpdatedAt,
		"completed_at":      run.CompletedAt,
	}
}
