package api

import (
	"errors"
	"net/http"
	"strings"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handler) listSummaryWidgets(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	widgets, err := repository.NewSummaryRepository(database.DB).ListSummaryWidgetsByUser(c.Request.Context(), userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list summary widgets", nil)
		return
	}
	items := make([]gin.H, 0, len(widgets))
	for i := range widgets {
		items = append(items, summaryWidgetResponse(&widgets[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}

func (h *Handler) createSummaryWidget(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var req summaryWidgetRequest
	if !bindJSON(c, &req) || !validateSummaryWidgetInput(c, req) {
		return
	}
	description := strings.TrimSpace(req.Description)
	widget := models.SummaryWidget{
		UserID:         userID,
		Name:           strings.TrimSpace(req.Name),
		Description:    stringPtrOrNil(description),
		AlwaysEnabled:  req.AlwaysEnabled,
		WhenToUse:      stringPtrOrNil(trimOptional(req.WhenToUse)),
		ContextSource:  strings.TrimSpace(req.ContextSource),
		Prompt:         strings.TrimSpace(req.Prompt),
		RenderMarkdown: req.RenderMarkdown,
		DisplayTitle:   strings.TrimSpace(req.DisplayTitle),
		Enabled:        true,
	}
	if req.Enabled != nil {
		widget.Enabled = *req.Enabled
	}
	repo := repository.NewSummaryRepository(database.DB)
	if duplicateSummaryWidgetName(c, repo, userID, "", widget.Name) {
		return
	}
	if err := repo.CreateSummaryWidget(c.Request.Context(), &widget); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create summary widget", nil)
		return
	}
	h.publishEvent("settings.updated", gin.H{"summary_widgets": true})
	c.JSON(http.StatusCreated, summaryWidgetResponse(&widget))
}

func (h *Handler) updateSummaryWidget(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	repo := repository.NewSummaryRepository(database.DB)
	widget, err := repo.FindSummaryWidgetByIDForUser(c.Request.Context(), c.Param("id"), userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "summary widget not found", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load summary widget", nil)
		return
	}
	var req summaryWidgetRequest
	if !bindJSON(c, &req) || !validateSummaryWidgetInput(c, req) {
		return
	}
	nextName := strings.TrimSpace(req.Name)
	if duplicateSummaryWidgetName(c, repo, userID, widget.ID, nextName) {
		return
	}
	description := strings.TrimSpace(req.Description)
	widget.Name = nextName
	widget.Description = stringPtrOrNil(description)
	widget.AlwaysEnabled = req.AlwaysEnabled
	widget.WhenToUse = stringPtrOrNil(trimOptional(req.WhenToUse))
	widget.ContextSource = strings.TrimSpace(req.ContextSource)
	widget.Prompt = strings.TrimSpace(req.Prompt)
	widget.RenderMarkdown = req.RenderMarkdown
	widget.DisplayTitle = strings.TrimSpace(req.DisplayTitle)
	if req.Enabled != nil {
		widget.Enabled = *req.Enabled
	}
	if err := repo.UpdateSummaryWidget(c.Request.Context(), widget); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update summary widget", nil)
		return
	}
	h.publishEvent("settings.updated", gin.H{"summary_widgets": true})
	c.JSON(http.StatusOK, summaryWidgetResponse(widget))
}

func (h *Handler) deleteSummaryWidget(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	repo := repository.NewSummaryRepository(database.DB)
	if _, err := repo.FindSummaryWidgetByIDForUser(c.Request.Context(), c.Param("id"), userID); errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "summary widget not found", nil)
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load summary widget", nil)
		return
	}
	if err := repo.DeleteSummaryWidget(c.Request.Context(), c.Param("id"), userID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete summary widget", nil)
		return
	}
	h.publishEvent("settings.updated", gin.H{"summary_widgets": true})
	c.Status(http.StatusNoContent)
}

func validateSummaryWidgetInput(c *gin.Context, req summaryWidgetRequest) bool {
	if strings.TrimSpace(req.Name) == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required", stringPtr("name"))
		return false
	}
	if len(strings.TrimSpace(req.Name)) > 120 {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is too long", stringPtr("name"))
		return false
	}
	if strings.TrimSpace(req.DisplayTitle) == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "display title is required", stringPtr("display_title"))
		return false
	}
	if len(strings.TrimSpace(req.DisplayTitle)) > 160 {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "display title is too long", stringPtr("display_title"))
		return false
	}
	if strings.TrimSpace(req.Prompt) == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "prompt is required", stringPtr("prompt"))
		return false
	}
	if len(strings.TrimSpace(req.Prompt)) > 12000 {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "prompt is too long", stringPtr("prompt"))
		return false
	}
	contextSource := strings.TrimSpace(req.ContextSource)
	if contextSource != "summary" && contextSource != "transcript" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "context source is invalid", stringPtr("context_source"))
		return false
	}
	if !req.AlwaysEnabled && strings.TrimSpace(trimOptional(req.WhenToUse)) == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "when to use is required for conditional widgets", stringPtr("when_to_use"))
		return false
	}
	return true
}

func duplicateSummaryWidgetName(c *gin.Context, repo repository.SummaryRepository, userID uint, exceptID string, name string) bool {
	widgets, err := repo.ListSummaryWidgetsByUser(c.Request.Context(), userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not validate summary widget", nil)
		return true
	}
	for _, widget := range widgets {
		if widget.ID != exceptID && strings.EqualFold(strings.TrimSpace(widget.Name), strings.TrimSpace(name)) {
			writeError(c, http.StatusConflict, "CONFLICT", "summary widget name already exists", stringPtr("name"))
			return true
		}
	}
	return false
}

func trimOptional(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func summaryWidgetResponse(widget *models.SummaryWidget) gin.H {
	description := ""
	if widget.Description != nil {
		description = *widget.Description
	}
	whenToUse := ""
	if widget.WhenToUse != nil {
		whenToUse = *widget.WhenToUse
	}
	return gin.H{
		"id":              widget.ID,
		"name":            widget.Name,
		"description":     description,
		"always_enabled":  widget.AlwaysEnabled,
		"when_to_use":     whenToUse,
		"context_source":  widget.ContextSource,
		"prompt":          widget.Prompt,
		"render_markdown": widget.RenderMarkdown,
		"display_title":   widget.DisplayTitle,
		"enabled":         widget.Enabled,
		"created_at":      widget.CreatedAt,
		"updated_at":      widget.UpdatedAt,
	}
}
