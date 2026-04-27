package api

import (
	"errors"
	"net/http"

	"scriberr/internal/database"
	"scriberr/internal/models"

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
