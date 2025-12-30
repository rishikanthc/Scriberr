package tests

import (
	"os"
	"strings"
	"testing"

	"scriberr/internal/auth"
	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"

	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"scriberr/internal/llm"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

		WhisperXEnv: "test_whisperx_env",
	}

	// Initialize test database
	if err := database.Initialize(cfg.DatabasePath); err != nil {
		t.Fatal("Failed to initialize test database:", err)
	}

	// Store the database instance in our helper
	testDB := database.DB

	// Create upload directory
	_ = os.MkdirAll(cfg.UploadDir, 0755)

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

// ResetDB cleans all tables in the database to ensure a clean state for each test
func (h *TestHelper) ResetDB(t *testing.T) {
	// List of models to clean
	modelsToClean := []interface{}{
		&models.Note{},
		&models.ChatSession{},
		&models.TranscriptionJobExecution{}, // Assuming this exists based on MockJobRepository
		&models.TranscriptionJob{},
		&models.TranscriptionProfile{},
		&models.SummaryTemplate{},
		&models.LLMConfig{},
		&models.APIKey{},
		&models.User{},
	}

	for _, model := range modelsToClean {
		// specific check to see if table exists before trying to delete
		if h.DB.Migrator().HasTable(model) {
			if err := h.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(model).Error; err != nil {
				// Ignore errors if table doesn't exist or other non-critical issues during cleanup
				// But log it just in case
				t.Logf("Failed to clean table for model %T: %v", model, err)
			}
		}
	}

	// Re-create test credentials as they are deleted by the cleanup
	h.createTestCredentials(t)
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

// MockJobRepository is a mock implementation of JobRepository
type MockJobRepository struct {
	mock.Mock
}

func (m *MockJobRepository) Create(ctx context.Context, entity *models.TranscriptionJob) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

func (m *MockJobRepository) FindByID(ctx context.Context, id interface{}) (*models.TranscriptionJob, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TranscriptionJob), args.Error(1)
}

func (m *MockJobRepository) Update(ctx context.Context, entity *models.TranscriptionJob) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

func (m *MockJobRepository) Delete(ctx context.Context, id interface{}) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockJobRepository) List(ctx context.Context, offset, limit int) ([]models.TranscriptionJob, int64, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]models.TranscriptionJob), args.Get(1).(int64), args.Error(2)
}

func (m *MockJobRepository) FindWithAssociations(ctx context.Context, id string) (*models.TranscriptionJob, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TranscriptionJob), args.Error(1)
}

func (m *MockJobRepository) ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.TranscriptionJob, int64, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]models.TranscriptionJob), args.Get(1).(int64), args.Error(2)
}

func (m *MockJobRepository) UpdateTranscript(ctx context.Context, jobID string, transcript string) error {
	args := m.Called(ctx, jobID, transcript)
	return args.Error(0)
}

func (m *MockJobRepository) CreateExecution(ctx context.Context, execution *models.TranscriptionJobExecution) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}

func (m *MockJobRepository) UpdateExecution(ctx context.Context, execution *models.TranscriptionJobExecution) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}

func (m *MockJobRepository) DeleteExecutionsByJobID(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockJobRepository) DeleteMultiTrackFilesByJobID(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockJobRepository) ListWithParams(ctx context.Context, offset, limit int, sortBy, sortOrder, searchQuery string, updatedAfter *time.Time) ([]models.TranscriptionJob, int64, error) {
	args := m.Called(ctx, offset, limit, sortBy, sortOrder, searchQuery, updatedAfter)
	return args.Get(0).([]models.TranscriptionJob), args.Get(1).(int64), args.Error(2)
}

func (m *MockJobRepository) FindActiveTrackJobs(ctx context.Context, parentJobID string) ([]models.TranscriptionJob, error) {
	args := m.Called(ctx, parentJobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TranscriptionJob), args.Error(1)
}

func (m *MockJobRepository) FindLatestCompletedExecution(ctx context.Context, jobID string) (*models.TranscriptionJobExecution, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TranscriptionJobExecution), args.Error(1)
}

func (m *MockJobRepository) UpdateStatus(ctx context.Context, jobID string, status models.JobStatus) error {
	args := m.Called(ctx, jobID, status)
	return args.Error(0)
}

func (m *MockJobRepository) UpdateError(ctx context.Context, jobID string, errorMsg string) error {
	args := m.Called(ctx, jobID, errorMsg)
	return args.Error(0)
}

func (m *MockJobRepository) FindByStatus(ctx context.Context, status models.JobStatus) ([]models.TranscriptionJob, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TranscriptionJob), args.Error(1)
}

func (m *MockJobRepository) CountByStatus(ctx context.Context, status models.JobStatus) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockJobRepository) UpdateSummary(ctx context.Context, jobID string, summary string) error {
	args := m.Called(ctx, jobID, summary)
	return args.Error(0)
}

// NewMockOpenAIServer creates a new mock OpenAI server for testing
func NewMockOpenAIServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			handleModelsRequest(w, r)
		case "/chat/completions":
			handleChatCompletionRequest(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func handleModelsRequest(w http.ResponseWriter, r *http.Request) {
	// Return mock models response
	response := llm.ModelsResponse{
		Data: []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}{
			{ID: "gpt-3.5-turbo", Object: "model", Created: 1677610602, OwnedBy: "openai"},
			{ID: "gpt-4", Object: "model", Created: 1687882411, OwnedBy: "openai"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func handleChatCompletionRequest(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var chatReq llm.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid request"}}`))
		return
	}

	if chatReq.Stream {
		handleStreamingResponse(w, chatReq)
	} else {
		handleNonStreamingResponse(w, chatReq)
	}
}

func handleNonStreamingResponse(w http.ResponseWriter, chatReq llm.ChatRequest) {
	response := llm.ChatResponse{
		ID:      "chatcmpl-test123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   chatReq.Model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: "This is a test response from the mock OpenAI service.",
				},
				FinishReason: "stop",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func handleStreamingResponse(w http.ResponseWriter, chatReq llm.ChatRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	tokens := []string{"This", " is", " a", " test", " streaming", " response", "."}

	for _, token := range tokens {
		response := llm.ChatStreamResponse{
			ID:      "chatcmpl-stream123",
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   chatReq.Model,
			Choices: []struct {
				Index int `json:"index"`
				Delta struct {
					Role    string `json:"role,omitempty"`
					Content string `json:"content,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Delta: struct {
						Role    string `json:"role,omitempty"`
						Content string `json:"content,omitempty"`
					}{
						Content: token,
					},
				},
			},
		}

		data, _ := json.Marshal(response)
		_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
		w.(http.Flusher).Flush()
		time.Sleep(10 * time.Millisecond)
	}

	_, _ = w.Write([]byte("data: [DONE]\n\n"))
	w.(http.Flusher).Flush()
}
