package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"scriberr/internal/models"
)

type SummaryTemplateRequest struct {
	Name        string  `json:"name" binding:"required,min=1"`
	Description *string `json:"description"`
	Model       string  `json:"model" binding:"required,min=1"`
	Prompt      string  `json:"prompt" binding:"required,min=1"`
}

type SummarySettingsRequest struct {
	DefaultModel string `json:"default_model" binding:"required,min=1"`
}

type SummarySettingsResponse struct {
	DefaultModel string `json:"default_model"`
}

// ListSummaryTemplates returns all templates
// @Summary List summarization templates
// @Description Get all summarization templates
// @Tags summaries
// @Produce json
// @Success 200 {array} models.SummaryTemplate
// @Security ApiKeyAuth
// @Security BearerAuth
// @Security BearerAuth
// @Router /api/v1/summaries [get]
func (h *Handler) ListSummaryTemplates(c *gin.Context) {
	// TODO: Add pagination support
	items, _, err := h.summaryRepo.List(c.Request.Context(), 0, 1000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch templates"})
		return
	}
	c.JSON(http.StatusOK, items)
}

// CreateSummaryTemplate creates a new template
// @Summary Create summarization template
// @Description Create a new summarization template
// @Tags summaries
// @Accept json
// @Produce json
// @Param request body SummaryTemplateRequest true "Template payload"
// @Success 201 {object} models.SummaryTemplate
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Security BearerAuth
// @Security BearerAuth
// @Router /api/v1/summaries [post]
func (h *Handler) CreateSummaryTemplate(c *gin.Context) {
	var req SummaryTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := &models.SummaryTemplate{
		Name:        req.Name,
		Description: req.Description,
		Model:       req.Model,
		Prompt:      req.Prompt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := h.summaryRepo.Create(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

// GetSummaryTemplate fetches one by id
// @Summary Get summarization template
// @Description Get a summarization template by ID
// @Tags summaries
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} models.SummaryTemplate
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Security BearerAuth
// @Security BearerAuth
// @Router /api/v1/summaries/{id} [get]
func (h *Handler) GetSummaryTemplate(c *gin.Context) {
	id := c.Param("id")
	item, err := h.summaryRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// UpdateSummaryTemplate updates an existing template
// @Summary Update summarization template
// @Description Update a summarization template by ID
// @Tags summaries
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param request body SummaryTemplateRequest true "Template payload"
// @Success 200 {object} models.SummaryTemplate
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Security BearerAuth
// @Security BearerAuth
// @Router /api/v1/summaries/{id} [put]
func (h *Handler) UpdateSummaryTemplate(c *gin.Context) {
	id := c.Param("id")
	var req SummaryTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.summaryRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}
	item.Name = req.Name
	item.Description = req.Description
	item.Model = req.Model
	item.Prompt = req.Prompt
	item.UpdatedAt = time.Now()
	if err := h.summaryRepo.Update(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update template"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// DeleteSummaryTemplate deletes a template
// @Summary Delete summarization template
// @Description Delete a summarization template by ID
// @Tags summaries
// @Produce json
// @Param id path string true "Template ID"
// @Success 204 {string} string "No Content"
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /api/v1/summaries/{id} [delete]
func (h *Handler) DeleteSummaryTemplate(c *gin.Context) {
	id := c.Param("id")
	if err := h.summaryRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete template"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetSummarySettings returns the global summary settings (default model)
// @Summary Get summary settings
// @Description Get global summarization settings
// @Tags summaries
// @Produce json
// @Success 200 {object} SummarySettingsResponse
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /api/v1/summaries/settings [get]
func (h *Handler) GetSummarySettings(c *gin.Context) {
	s, err := h.summaryRepo.GetSettings(c.Request.Context())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, SummarySettingsResponse{DefaultModel: ""})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
		return
	}
	c.JSON(http.StatusOK, SummarySettingsResponse{DefaultModel: s.DefaultModel})
}

// SaveSummarySettings updates default model (creates row if absent)
// @Summary Save summary settings
// @Description Create or update global summarization settings
// @Tags summaries
// @Accept json
// @Produce json
// @Param request body SummarySettingsRequest true "Settings payload"
// @Success 200 {object} SummarySettingsResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /api/v1/summaries/settings [post]
func (h *Handler) SaveSummarySettings(c *gin.Context) {
	var req SummarySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s, err := h.summaryRepo.GetSettings(c.Request.Context())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			s = &models.SummarySetting{
				DefaultModel: req.DefaultModel,
				UpdatedAt:    time.Now(),
			}
			// We can't use Create from BaseRepository because it expects *T, but GetSettings returns *T.
			// BaseRepository.Create expects *T.
			// Actually BaseRepository[T] Create takes *T.
			// But here T is models.SummaryTemplate, NOT models.SummarySetting.
			// SummaryRepository handles SummaryTemplate.
			// But GetSettings returns SummarySetting.
			// So I can't use h.summaryRepo.Create(s) because s is SummarySetting, not SummaryTemplate.
			// I need to add SaveSettings to SummaryRepository which handles creation too.
			// I added SaveSettings(ctx, settings).
			if err := h.summaryRepo.SaveSettings(c.Request.Context(), s); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save settings"})
				return
			}
			c.JSON(http.StatusOK, SummarySettingsResponse{DefaultModel: s.DefaultModel})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save settings"})
		return
	}
	s.DefaultModel = req.DefaultModel
	s.UpdatedAt = time.Now()
	if err := h.summaryRepo.SaveSettings(c.Request.Context(), s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save settings"})
		return
	}
	c.JSON(http.StatusOK, SummarySettingsResponse{DefaultModel: s.DefaultModel})
}
