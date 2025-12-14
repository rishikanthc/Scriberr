package api

import (
	"fmt"
	"net/http"
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
	logPath := filepath.Join(h.config.TranscriptsDir, jobID, "transcription.log")

	// Check if file exists
	exists, err := h.fileService.FileExists(logPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to check logs: %v", err)})
		return
	}
	if !exists {
		// Return graceful empty response instead of 404
		c.JSON(http.StatusOK, gin.H{
			"job_id":    jobID,
			"available": false,
			"content":   "",
			"message":   "No logs available for this job",
		})
		return
	}

	// Read file content
	content, err := h.fileService.ReadFile(logPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read logs: %v", err)})
		return
	}

	// Return as JSON with content for consistency
	c.JSON(http.StatusOK, gin.H{
		"job_id":    jobID,
		"available": true,
		"content":   string(content),
	})
}
