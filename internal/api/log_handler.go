package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// GetJobLogs returns the transcription logs for a specific job
// @Summary Get transcription logs
// @Description Get the raw transcription logs for a job
// @Tags transcription
// @Produce text/plain
// @Param id path string true "Job ID"
// @Success 200 {string} string "Log content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transcription/{id}/logs [get]
func (h *Handler) GetJobLogs(c *gin.Context) {
	jobID := c.Param("id")

	// Construct path to log file
	// Note: output directory is hardcoded to "data/transcripts" in UnifiedTranscriptionService
	logPath := filepath.Join("data", "transcripts", jobID, "transcription.log")

	// Check if file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Logs not found for this job"})
		return
	}

	// Read file content
	content, err := os.ReadFile(logPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read logs: %v", err)})
		return
	}

	// Return as plain text
	c.Data(http.StatusOK, "text/plain; charset=utf-8", content)
}
