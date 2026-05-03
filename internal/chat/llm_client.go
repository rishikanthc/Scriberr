package chat

import (
	"fmt"
	"strings"

	"scriberr/internal/llm"
	"scriberr/internal/llmprovider"
	"scriberr/internal/models"
)

type LLMClientFactory func(*models.LLMConfig) (llm.Service, error)

func ClientForConfig(config *models.LLMConfig) (llm.Service, error) {
	baseURL := llmprovider.BaseURL(config)
	switch config.Provider {
	case "ollama":
		return llm.NewOllamaService(baseURL), nil
	case "openai", "openai_compatible":
		apiKey := ""
		if config.APIKey != nil {
			apiKey = strings.TrimSpace(*config.APIKey)
		}
		return llm.NewOpenAIService(apiKey, &baseURL), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider")
	}
}
