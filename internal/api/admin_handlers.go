package api

import (
	"net/http"

	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listTranscriptionModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"items": []gin.H{
			{
				"id":           "base",
				"name":         "Whisper base",
				"provider":     "local",
				"capabilities": []string{"transcription"},
			},
		},
	})
}
func (h *Handler) queueStats(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	stats := gin.H{"queued": 0, "processing": 0, "completed": 0, "failed": 0}
	type statusCount struct {
		Status models.JobStatus
		Count  int64
	}
	var counts []statusCount
	if err := database.DB.Model(&models.TranscriptionJob{}).
		Select("status, count(*) as count").
		Where("user_id = ? AND source_file_hash IS NOT NULL", userID).
		Group("status").
		Find(&counts).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read queue stats", nil)
		return
	}
	for _, count := range counts {
		switch count.Status {
		case models.StatusPending:
			stats["queued"] = count.Count
		case models.StatusProcessing:
			stats["processing"] = count.Count
		case models.StatusCompleted:
			stats["completed"] = count.Count
		case models.StatusFailed:
			stats["failed"] = count.Count
		}
	}
	c.JSON(http.StatusOK, stats)
}
