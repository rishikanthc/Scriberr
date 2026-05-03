package api

import (
	"errors"
	"net/http"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/summarization"

	"github.com/gin-gonic/gin"
)

func (h *Handler) getTranscriptionSummary(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	summary, err := h.summaries.LatestForTranscription(c.Request.Context(), job.UserID, job.ID)
	if errors.Is(err, summarization.ErrNotFound) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "summary not found", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load summary", nil)
		return
	}
	c.JSON(http.StatusOK, summaryResponse(summary))
}

type SummaryResponse struct {
	ID                  string     `json:"id"`
	TranscriptionID     string     `json:"transcription_id"`
	Content             string     `json:"content"`
	Model               string     `json:"model"`
	Provider            string     `json:"provider"`
	Status              string     `json:"status"`
	Error               any        `json:"error"`
	TranscriptTruncated bool       `json:"transcript_truncated"`
	ContextWindow       int        `json:"context_window"`
	InputCharacters     int        `json:"input_characters"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	CompletedAt         *time.Time `json:"completed_at"`
}

type SummaryWidgetRunResponse struct {
	ID               string     `json:"id"`
	SummaryID        string     `json:"summary_id"`
	TranscriptionID  string     `json:"transcription_id"`
	WidgetID         string     `json:"widget_id"`
	WidgetName       string     `json:"widget_name"`
	DisplayTitle     string     `json:"display_title"`
	ContextSource    string     `json:"context_source"`
	RenderMarkdown   bool       `json:"render_markdown"`
	Model            string     `json:"model"`
	Provider         string     `json:"provider"`
	Status           string     `json:"status"`
	Output           string     `json:"output"`
	Error            any        `json:"error"`
	ContextTruncated bool       `json:"context_truncated"`
	ContextWindow    int        `json:"context_window"`
	InputCharacters  int        `json:"input_characters"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CompletedAt      *time.Time `json:"completed_at"`
}

func summaryResponse(summary *models.Summary) SummaryResponse {
	errorValue := any(nil)
	if summary.ErrorMessage != nil && *summary.ErrorMessage != "" {
		errorValue = sanitizePublicText(*summary.ErrorMessage)
	}
	return SummaryResponse{
		ID:                  summary.ID,
		TranscriptionID:     "tr_" + summary.TranscriptionID,
		Content:             summary.Content,
		Model:               summary.Model,
		Provider:            summary.Provider,
		Status:              summary.Status,
		Error:               errorValue,
		TranscriptTruncated: summary.TranscriptTruncated,
		ContextWindow:       summary.ContextWindow,
		InputCharacters:     summary.InputCharacters,
		CreatedAt:           summary.CreatedAt,
		UpdatedAt:           summary.UpdatedAt,
		CompletedAt:         summary.CompletedAt,
	}
}

func (h *Handler) listTranscriptionSummaryWidgets(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	runs, err := h.summaries.ListWidgetRunsForTranscription(c.Request.Context(), job.UserID, job.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load summary widgets", nil)
		return
	}
	items := make([]SummaryWidgetRunResponse, 0, len(runs))
	for i := range runs {
		items = append(items, summaryWidgetRunResponse(&runs[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}

func summaryWidgetRunResponse(run *models.SummaryWidgetRun) SummaryWidgetRunResponse {
	errorValue := any(nil)
	if run.ErrorMessage != nil && *run.ErrorMessage != "" {
		errorValue = sanitizePublicText(*run.ErrorMessage)
	}
	return SummaryWidgetRunResponse{
		ID:               run.ID,
		SummaryID:        run.SummaryID,
		TranscriptionID:  "tr_" + run.TranscriptionID,
		WidgetID:         run.WidgetID,
		WidgetName:       run.WidgetName,
		DisplayTitle:     run.DisplayTitle,
		ContextSource:    run.ContextSource,
		RenderMarkdown:   run.RenderMarkdown,
		Model:            run.Model,
		Provider:         run.Provider,
		Status:           run.Status,
		Output:           run.Output,
		Error:            errorValue,
		ContextTruncated: run.ContextTruncated,
		ContextWindow:    run.ContextWindow,
		InputCharacters:  run.InputCharacters,
		CreatedAt:        run.CreatedAt,
		UpdatedAt:        run.UpdatedAt,
		CompletedAt:      run.CompletedAt,
	}
}
