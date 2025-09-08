package tests

import (
	"os"
	"strings"
	"testing"

	"scriberr/internal/auth"
	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestHelper provides common test setup and utilities
type TestHelper struct {
	Config      *config.Config
	AuthService *auth.AuthService
	TestAPIKey  string
	TestUser    *models.User
	TestToken   string
	DB          *gorm.DB // Store our own DB instance
}

// NewTestHelper creates a new test helper with isolated database
func NewTestHelper(t *testing.T, dbName string) *TestHelper {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create unique test config
	cfg := &config.Config{
		Port:         "8080",
		Host:         "localhost",
		DatabasePath: dbName,
		JWTSecret:    "test-secret-key-for-unit-tests",
		UploadDir:    "test_uploads_" + dbName,
		UVPath:       "uv",
		WhisperXEnv:  "test_whisperx_env",
	}

	// Initialize test database
	if err := database.Initialize(cfg.DatabasePath); err != nil {
		t.Fatal("Failed to initialize test database:", err)
	}

	// Store the database instance in our helper
	testDB := database.DB

	// Create upload directory
	os.MkdirAll(cfg.UploadDir, 0755)

	// Initialize auth service
	authService := auth.NewAuthService(cfg.JWTSecret)

	helper := &TestHelper{
		Config:      cfg,
		AuthService: authService,
		DB:          testDB,
	}

	// Create test API key and user
	helper.createTestCredentials(t)

	return helper
}

// GetDB returns the current database instance
func (h *TestHelper) GetDB() *gorm.DB {
	return h.DB
}

// Cleanup removes test database and upload directory
func (h *TestHelper) Cleanup() {
	database.Close()
	os.Remove(h.Config.DatabasePath)
	os.RemoveAll(h.Config.UploadDir)
}

// createTestCredentials creates a test user and API key for testing
func (h *TestHelper) createTestCredentials(t *testing.T) {
	// Create test user
	hashedPassword, err := auth.HashPassword("testpassword123")
	assert.NoError(t, err)

	user := models.User{
		Username: "testuser",
		Password: hashedPassword,
	}

	result := h.DB.Create(&user)
	assert.NoError(t, result.Error)
	h.TestUser = &user

	// Generate JWT token
	token, err := h.AuthService.GenerateToken(&user)
	assert.NoError(t, err)
	h.TestToken = token

	// Create test API key
	apiKey := models.APIKey{
		Key:      "test-api-key-" + strings.ReplaceAll(t.Name(), "/", "_"),
		Name:     "Test API Key for " + strings.ReplaceAll(t.Name(), "/", "_"),
		IsActive: true,
	}

	result = h.DB.Create(&apiKey)
	assert.NoError(t, result.Error)
	h.TestAPIKey = apiKey.Key
}

// CreateTestTranscriptionJob creates a test transcription job
func (h *TestHelper) CreateTestTranscriptionJob(t *testing.T, title string) *models.TranscriptionJob {
	// Let GORM assign a unique UUID via model hook to avoid ID collisions
	job := &models.TranscriptionJob{
		Title:     &title,
		Status:    models.StatusPending,
		AudioPath: "test/path/audio.mp3",
		Parameters: models.WhisperXParams{
			Model:       "base",
			BatchSize:   16,
			ComputeType: "float16",
			Device:      "auto",
		},
	}

	result := h.DB.Create(job)
	assert.NoError(t, result.Error)
	return job
}

// CreateTestProfile creates a test transcription profile
func (h *TestHelper) CreateTestProfile(t *testing.T, name string, isDefault bool) *models.TranscriptionProfile {
	profile := &models.TranscriptionProfile{
		ID:          "test-profile-" + strings.ReplaceAll(t.Name(), "/", "_"),
		Name:        name,
		Description: stringPtr("Test profile description"),
		IsDefault:   isDefault,
		Parameters: models.WhisperXParams{
			Model:       "small",
			BatchSize:   8,
			ComputeType: "float32",
			Device:      "cpu",
		},
	}

	result := h.DB.Create(profile)
	assert.NoError(t, result.Error)
	return profile
}

// CreateTestNote creates a test note for a transcription
func (h *TestHelper) CreateTestNote(t *testing.T, transcriptionID string) *models.Note {
	note := &models.Note{
		ID:              "test-note-" + strings.ReplaceAll(t.Name(), "/", "_"),
		TranscriptionID: transcriptionID,
		StartWordIndex:  0,
		EndWordIndex:    5,
		StartTime:       0.0,
		EndTime:         2.5,
		Quote:           "Test quote text",
		Content:         "Test note content",
	}

	result := h.DB.Create(note)
	assert.NoError(t, result.Error)
	return note
}

// CreateTestSummaryTemplate creates a test summary template
func (h *TestHelper) CreateTestSummaryTemplate(t *testing.T, name string) *models.SummaryTemplate {
	template := &models.SummaryTemplate{
		ID:          "test-template-" + strings.ReplaceAll(t.Name(), "/", "_"),
		Name:        name,
		Description: stringPtr("Test template description"),
		Model:       "gpt-4",
		Prompt:      "Summarize this: {{content}}",
	}

	result := h.DB.Create(template)
	assert.NoError(t, result.Error)
	return template
}

// CreateTestChatSession creates a test chat session
func (h *TestHelper) CreateTestChatSession(t *testing.T, transcriptionID string) *models.ChatSession {
	session := &models.ChatSession{
		ID:              "test-chat-session-" + strings.ReplaceAll(t.Name(), "/", "_"),
		JobID:           transcriptionID,
		TranscriptionID: transcriptionID,
		Title:           "Test Chat Session",
		Model:           "gpt-4",
		Provider:        "openai",
		MessageCount:    0,
		IsActive:        true,
	}

	result := h.DB.Create(session)
	assert.NoError(t, result.Error)
	return session
}

// CreateTestLLMConfig creates a test LLM configuration
func (h *TestHelper) CreateTestLLMConfig(t *testing.T, provider string) *models.LLMConfig {
	config := &models.LLMConfig{
		Provider: provider,
		BaseURL:  stringPtr("https://api.test.com"),
		APIKey:   stringPtr("test-llm-api-key"),
		IsActive: true,
	}

	result := h.DB.Create(config)
	assert.NoError(t, result.Error)
	return config
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// Helper function to create float pointer
func floatPtr(f float64) *float64 {
	return &f
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
