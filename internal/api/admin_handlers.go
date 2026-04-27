package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"scriberr/internal/database"
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
			"canceled":   stats.Canceled,
			"running":    stats.Running,
		})
		return
	}
	stats := gin.H{"queued": 0, "processing": 0, "completed": 0, "failed": 0, "canceled": 0, "running": 0}
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
		case models.StatusCanceled:
			stats["canceled"] = count.Count
		}
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) getTranscriptionExecutions(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	var executions []models.TranscriptionJobExecution
	if err := database.DB.Where("transcription_id = ?", job.ID).Order("execution_number DESC").Find(&executions).Error; err != nil {
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
	logText, err := logsForTranscription(c.Request.Context(), job.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read transcription logs", nil)
		return
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(sanitizePublicText(logText)))
}

func logsForTranscription(ctx context.Context, jobID string) (string, error) {
	var executions []models.TranscriptionJobExecution
	if err := database.DB.WithContext(ctx).Where("transcription_id = ?", jobID).Order("execution_number ASC").Find(&executions).Error; err != nil {
		return "", err
	}
	if len(executions) == 0 {
		return "No execution logs recorded.\n", nil
	}
	var out strings.Builder
	for _, execution := range executions {
		fmt.Fprintf(&out, "execution %d status=%s provider=%s model=%s started_at=%s\n",
			execution.ExecutionNumber,
			execution.Status,
			execution.Provider,
			execution.ModelName,
			execution.StartedAt.Format(time.RFC3339),
		)
		if execution.CompletedAt != nil {
			fmt.Fprintf(&out, "completed_at=%s\n", execution.CompletedAt.Format(time.RFC3339))
		}
		if execution.FailedAt != nil {
			fmt.Fprintf(&out, "failed_at=%s\n", execution.FailedAt.Format(time.RFC3339))
		}
		if execution.ErrorMessage != nil && *execution.ErrorMessage != "" {
			fmt.Fprintf(&out, "error=%s\n", *execution.ErrorMessage)
		}
		if execution.LogsPath != nil && *execution.LogsPath != "" {
			if data, err := os.ReadFile(*execution.LogsPath); err == nil {
				out.Write(data)
				if !strings.HasSuffix(out.String(), "\n") {
					out.WriteByte('\n')
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return "", err
			}
		}
	}
	return out.String(), nil
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
