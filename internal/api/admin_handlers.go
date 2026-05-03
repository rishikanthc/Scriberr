package api

import (
	"net/http"
	"regexp"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/transcription/engineprovider"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listTranscriptionModels(c *gin.Context) {
	if h.modelRegistry != nil {
		capabilities, err := h.modelRegistry.Capabilities(c.Request.Context())
		if err != nil {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list transcription models", nil)
			return
		}
		items := make([]gin.H, 0, len(capabilities))
		for _, capability := range capabilities {
			items = append(items, modelCapabilityResponse(capability))
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items": []gin.H{
			{
				"id":           engineprovider.DefaultTranscriptionModel,
				"name":         "Whisper Base",
				"provider":     "local",
				"installed":    false,
				"default":      true,
				"capabilities": []string{"transcription", "word_timestamps"},
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
	if h.queueService != nil {
		stats, err := h.queueService.Stats(c.Request.Context(), userID)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read queue stats", nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"queued":     stats.Queued,
			"processing": stats.Processing,
			"completed":  stats.Completed,
			"failed":     stats.Failed,
			"stopped":    stats.Canceled,
			"canceled":   stats.Canceled,
			"running":    stats.Running,
		})
		return
	}
	stats, err := h.transcriptions.Stats(c.Request.Context(), userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read queue stats", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"queued":     stats.Queued,
		"processing": stats.Processing,
		"completed":  stats.Completed,
		"failed":     stats.Failed,
		"stopped":    stats.Canceled,
		"canceled":   stats.Canceled,
		"running":    stats.Running,
	})
}

func (h *Handler) getTranscriptionExecutions(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	executions, err := h.transcriptions.ListExecutions(c.Request.Context(), job.UserID, job.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list transcription executions", nil)
		return
	}
	items := make([]gin.H, 0, len(executions))
	for i := range executions {
		items = append(items, executionResponse(&executions[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}

func (h *Handler) getTranscriptionLogs(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	logText, err := h.transcriptions.Logs(c.Request.Context(), job.UserID, job.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read transcription logs", nil)
		return
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(sanitizePublicText(logText)))
}

func modelCapabilityResponse(capability engineprovider.ModelCapability) gin.H {
	return gin.H{
		"id":           capability.ID,
		"name":         capability.Name,
		"provider":     capability.Provider,
		"installed":    capability.Installed,
		"default":      capability.Default,
		"capabilities": capability.Capabilities,
	}
}

func executionResponse(execution *models.TranscriptionJobExecution) gin.H {
	errorValue := any(nil)
	if execution.ErrorMessage != nil && *execution.ErrorMessage != "" {
		errorValue = sanitizePublicText(*execution.ErrorMessage)
	}
	return gin.H{
		"id":                     "exec_" + execution.ID,
		"transcription_id":       "tr_" + execution.TranscriptionJobID,
		"status":                 string(execution.Status),
		"provider":               execution.Provider,
		"model":                  execution.ModelName,
		"started_at":             execution.StartedAt,
		"completed_at":           execution.CompletedAt,
		"failed_at":              execution.FailedAt,
		"processing_duration_ms": executionDurationMS(execution),
		"error":                  errorValue,
	}
}

func executionDurationMS(execution *models.TranscriptionJobExecution) any {
	var end time.Time
	switch {
	case execution.CompletedAt != nil:
		end = *execution.CompletedAt
	case execution.FailedAt != nil:
		end = *execution.FailedAt
	default:
		return nil
	}
	if execution.StartedAt.IsZero() || end.Before(execution.StartedAt) {
		return nil
	}
	return end.Sub(execution.StartedAt).Milliseconds()
}

var (
	publicAbsolutePathPattern = regexp.MustCompile(`(?:[A-Za-z]:\\|/)[^\s:;,'")]+`)
	publicTokenPattern        = regexp.MustCompile(`(?i)\b([A-Za-z0-9_]*(?:token|api_key|apikey)[A-Za-z0-9_]*)=[^\s]+`)
)

func sanitizePublicText(value string) string {
	value = publicAbsolutePathPattern.ReplaceAllString(value, "[redacted-path]")
	return publicTokenPattern.ReplaceAllString(value, "$1=[redacted]")
}
