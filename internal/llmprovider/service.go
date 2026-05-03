package llmprovider

import (
	"context"
	"errors"
	"strings"

	"scriberr/internal/models"
	"scriberr/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrNotConfigured         = errors.New("LLM provider is not configured")
	ErrLargeModelUnavailable = errors.New("large model is not available")
	ErrSmallModelUnavailable = errors.New("small model is not available")
)

type ConnectionTester interface {
	TestLLMProviderConnection(ctx context.Context, rawBaseURL, apiKey string) (TestResult, error)
}

type TestResult struct {
	Provider string
	BaseURL  string
	Models   []string
}

type SaveRequest struct {
	BaseURL    string
	APIKey     string
	LargeModel string
	SmallModel string
}

type ConfigResult struct {
	Config          *models.LLMConfig
	Models          []string
	ConnectionError error
}

type Service struct {
	configs repository.LLMConfigRepository
	tester  ConnectionTester
}

func NewService(configs repository.LLMConfigRepository, tester ConnectionTester) *Service {
	return &Service{configs: configs, tester: tester}
}

func (s *Service) Get(ctx context.Context, userID uint) (ConfigResult, error) {
	config, err := s.configs.GetActiveByUser(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ConfigResult{}, ErrNotConfigured
	}
	if err != nil {
		return ConfigResult{}, err
	}
	baseURL := BaseURL(config)
	apiKey := ""
	if config.APIKey != nil {
		apiKey = strings.TrimSpace(*config.APIKey)
	}
	testResult, testErr := s.tester.TestLLMProviderConnection(ctx, baseURL, apiKey)
	if testErr != nil {
		return ConfigResult{Config: config, ConnectionError: testErr}, nil
	}
	config.Provider = testResult.Provider
	config.BaseURL = &testResult.BaseURL
	config.OpenAIBaseURL = &testResult.BaseURL
	return ConfigResult{Config: config, Models: testResult.Models}, nil
}

func (s *Service) Save(ctx context.Context, userID uint, req SaveRequest) (ConfigResult, error) {
	baseURL := strings.TrimSpace(req.BaseURL)
	apiKey := strings.TrimSpace(req.APIKey)
	largeModel := strings.TrimSpace(req.LargeModel)
	smallModel := strings.TrimSpace(req.SmallModel)

	existing, err := s.configs.GetActiveByUser(ctx, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return ConfigResult{}, err
	}
	effectiveAPIKey := apiKey
	if effectiveAPIKey == "" && existing != nil && existing.APIKey != nil {
		effectiveAPIKey = strings.TrimSpace(*existing.APIKey)
	}

	testResult, err := s.tester.TestLLMProviderConnection(ctx, baseURL, effectiveAPIKey)
	if err != nil {
		return ConfigResult{}, err
	}
	if largeModel != "" && !stringInSlice(testResult.Models, largeModel) {
		return ConfigResult{}, ErrLargeModelUnavailable
	}
	if smallModel != "" && !stringInSlice(testResult.Models, smallModel) {
		return ConfigResult{}, ErrSmallModelUnavailable
	}

	config := &models.LLMConfig{
		UserID:        userID,
		Name:          "Default LLM provider",
		Provider:      testResult.Provider,
		BaseURL:       stringPtr(testResult.BaseURL),
		OpenAIBaseURL: stringPtr(testResult.BaseURL),
		IsDefault:     true,
	}
	if effectiveAPIKey != "" {
		config.APIKey = stringPtr(effectiveAPIKey)
	}
	if largeModel != "" {
		config.LargeModel = stringPtr(largeModel)
	}
	if smallModel != "" {
		config.SmallModel = stringPtr(smallModel)
	}
	if err := s.configs.ReplaceActiveByUser(ctx, userID, config); err != nil {
		return ConfigResult{}, err
	}
	return ConfigResult{Config: config, Models: testResult.Models}, nil
}

func BaseURL(config *models.LLMConfig) string {
	if config.BaseURL != nil {
		return strings.TrimSpace(*config.BaseURL)
	}
	if config.OpenAIBaseURL != nil {
		return strings.TrimSpace(*config.OpenAIBaseURL)
	}
	return ""
}

func stringPtr(value string) *string {
	return &value
}

func stringInSlice(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
