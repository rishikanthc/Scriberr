package api

import (
	"errors"
	"net/http"
	"strings"

	"scriberr/internal/llmprovider"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) getLLMProvider(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}

	result, err := h.llmProvider.Get(c.Request.Context(), user.ID)
	if errors.Is(err, llmprovider.ErrNotConfigured) {
		c.JSON(http.StatusOK, gin.H{
			"configured":  false,
			"provider":    "openai_compatible",
			"base_url":    "",
			"has_api_key": false,
			"key_preview": nil,
			"model_count": 0,
			"models":      []string{},
			"large_model": nil,
			"small_model": nil,
		})
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load LLM provider", nil)
		return
	}
	if result.ConnectionError != nil {
		response := llmProviderResponse(result.Config, nil)
		response["connection_error"] = result.ConnectionError.Error()
		c.JSON(http.StatusOK, response)
		return
	}
	c.JSON(http.StatusOK, llmProviderResponse(result.Config, result.Models))
}

func (h *Handler) updateLLMProvider(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}

	var req updateLLMProviderRequest
	if !bindJSON(c, &req) {
		return
	}

	if strings.TrimSpace(req.BaseURL) == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "base_url is required", stringPtr("base_url"))
		return
	}
	result, err := h.llmProvider.Save(c.Request.Context(), user.ID, llmprovider.SaveRequest{
		BaseURL:    req.BaseURL,
		APIKey:     req.APIKey,
		LargeModel: req.LargeModel,
		SmallModel: req.SmallModel,
	})
	if errors.Is(err, llmprovider.ErrLargeModelUnavailable) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "large_model is not available from this provider", stringPtr("large_model"))
		return
	}
	if errors.Is(err, llmprovider.ErrSmallModelUnavailable) {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "small_model is not available from this provider", stringPtr("small_model"))
		return
	}
	if err != nil {
		writeError(c, http.StatusUnprocessableEntity, "PROVIDER_CONNECTION_FAILED", err.Error(), stringPtr("base_url"))
		return
	}

	response := llmProviderResponse(result.Config, result.Models)
	h.publishEventForUser("settings.updated", gin.H{"llm_provider_configured": true}, user.ID)
	c.JSON(http.StatusOK, response)
}

func llmProviderResponse(config *models.LLMConfig, models []string) gin.H {
	baseURL := llmprovider.BaseURL(config)
	hasKey := config.APIKey != nil && strings.TrimSpace(*config.APIKey) != ""
	var keyPreviewValue any
	if hasKey {
		keyPreviewValue = keyPreview(*config.APIKey)
	}
	if models == nil {
		models = []string{}
	}
	return gin.H{
		"configured":  true,
		"provider":    config.Provider,
		"base_url":    baseURL,
		"has_api_key": hasKey,
		"key_preview": keyPreviewValue,
		"model_count": len(models),
		"models":      models,
		"large_model": config.LargeModel,
		"small_model": config.SmallModel,
		"updated_at":  config.UpdatedAt,
	}
}

func llmProviderBaseURL(config *models.LLMConfig) string {
	return llmprovider.BaseURL(config)
}

func stringInSlice(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
