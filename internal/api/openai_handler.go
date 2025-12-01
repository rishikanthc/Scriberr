package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ValidateOpenAIKeyRequest represents the request to validate an OpenAI API key
type ValidateOpenAIKeyRequest struct {
	APIKey string `json:"api_key"`
}

// OpenAIModel represents a model returned by OpenAI API
type OpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// OpenAIModelListResponse represents the response from OpenAI models endpoint
type OpenAIModelListResponse struct {
	Object string        `json:"object"`
	Data   []OpenAIModel `json:"data"`
}

// ValidateOpenAIKey validates the API key and returns available models
// @Summary Validate OpenAI API Key
// @Description Validate the provided OpenAI API key and return available Whisper models
// @Tags config
// @Accept json
// @Produce json
// @Param request body ValidateOpenAIKeyRequest true "API Key"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/config/openai/validate [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) ValidateOpenAIKey(c *gin.Context) {
	var req ValidateOpenAIKeyRequest
	// If API key is not provided in request, try to use the one from config
	apiKey := req.APIKey
	if apiKey == "" {
		apiKey = h.config.OpenAIAPIKey
	}

	if apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API key is required (none provided and no server default)"})
		return
	}

	// Create request to OpenAI models endpoint
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	request, err := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	request.Header.Set("Authorization", "Bearer "+apiKey)

	response, err := client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to OpenAI: %v", err)})
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		return
	}

	if response.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("OpenAI API returned status: %d", response.StatusCode)})
		return
	}

	// Parse response
	var modelList OpenAIModelListResponse
	if err := json.NewDecoder(response.Body).Decode(&modelList); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse OpenAI response"})
		return
	}

	// Filter for whisper models
	var whisperModels []string
	for _, model := range modelList.Data {
		if model.ID == "whisper-1" || (len(model.ID) > 7 && model.ID[:7] == "whisper") {
			whisperModels = append(whisperModels, model.ID)
		}
	}

	// If no whisper models found (unlikely), default to whisper-1
	if len(whisperModels) == 0 {
		whisperModels = []string{"whisper-1"}
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":  true,
		"models": whisperModels,
	})
}
