package api

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "scriberr/internal/database"
    "scriberr/internal/models"
)

type SummaryTemplateRequest struct {
    Name        string  `json:"name" binding:"required,min=1"`
    Description *string `json:"description"`
    Prompt      string  `json:"prompt" binding:"required,min=1"`
}

// ListSummaryTemplates returns all templates
func (h *Handler) ListSummaryTemplates(c *gin.Context) {
    var items []models.SummaryTemplate
    if err := database.DB.Order("created_at DESC").Find(&items).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch templates"})
        return
    }
    c.JSON(http.StatusOK, items)
}

// CreateSummaryTemplate creates a new template
func (h *Handler) CreateSummaryTemplate(c *gin.Context) {
    var req SummaryTemplateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    item := models.SummaryTemplate{
        Name:        req.Name,
        Description: req.Description,
        Prompt:      req.Prompt,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    if err := database.DB.Create(&item).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
        return
    }
    c.JSON(http.StatusCreated, item)
}

// GetSummaryTemplate fetches one by id
func (h *Handler) GetSummaryTemplate(c *gin.Context) {
    id := c.Param("id")
    var item models.SummaryTemplate
    if err := database.DB.Where("id = ?", id).First(&item).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch template"})
        return
    }
    c.JSON(http.StatusOK, item)
}

// UpdateSummaryTemplate updates an existing template
func (h *Handler) UpdateSummaryTemplate(c *gin.Context) {
    id := c.Param("id")
    var req SummaryTemplateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    var item models.SummaryTemplate
    if err := database.DB.Where("id = ?", id).First(&item).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch template"})
        return
    }
    item.Name = req.Name
    item.Description = req.Description
    item.Prompt = req.Prompt
    item.UpdatedAt = time.Now()
    if err := database.DB.Save(&item).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update template"})
        return
    }
    c.JSON(http.StatusOK, item)
}

// DeleteSummaryTemplate deletes a template
func (h *Handler) DeleteSummaryTemplate(c *gin.Context) {
    id := c.Param("id")
    if err := database.DB.Delete(&models.SummaryTemplate{}, "id = ?", id).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete template"})
        return
    }
    c.Status(http.StatusNoContent)
}

