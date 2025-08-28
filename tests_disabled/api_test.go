package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"scriberr/internal/api"
	"scriberr/internal/auth"
	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/queue"
	"scriberr/internal/transcription"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type APITestSuite struct {
	suite.Suite
	router   *gin.Engine
	config   *config.Config
	authService *auth.AuthService
	taskQueue   *queue.TaskQueue
	whisperXService *transcription.WhisperXService
	handler  *api.Handler
	apiKey   string
	token    string
}

func (suite *APITestSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test configuration
	suite.config = &config.Config{
		Port:          "8080",
		Host:          "localhost",
		DatabasePath:  "test.db",
		JWTSecret:     "test-secret",
		UploadDir:     "test_uploads",
		PythonPath:    "python3",
		UVPath:        "uv",
		WhisperXEnv:   "test_whisperx_env",
		DefaultAPIKey: "test-api-key",
	}

	// Initialize test database
	if err := database.Initialize(suite.config.DatabasePath); err != nil {
		suite.T().Fatal("Failed to initialize test database:", err)
	}

	// Initialize services
	suite.authService = auth.NewAuthService(suite.config.JWTSecret)
	suite.whisperXService = transcription.NewWhisperXService(suite.config)
	suite.taskQueue = queue.NewTaskQueue(1, suite.whisperXService)
	suite.handler = api.NewHandler(suite.config, suite.authService, suite.taskQueue, suite.whisperXService)

	// Set up router
	suite.router = api.SetupRoutes(suite.handler, suite.authService)

	// Create test data
	suite.setupTestData()

	// Create upload directory
	os.MkdirAll(suite.config.UploadDir, 0755)
}

func (suite *APITestSuite) TearDownSuite() {
	// Clean up test database
	database.Close()
	os.Remove(suite.config.DatabasePath)
	
	// Clean up upload directory
	os.RemoveAll(suite.config.UploadDir)
}

func (suite *APITestSuite) setupTestData() {
	// Create test user
	hashedPassword, _ := auth.HashPassword("testpass")
	user := models.User{
		Username: "testuser",
		Password: hashedPassword,
	}
	database.DB.Create(&user)

	// Generate JWT token
	token, _ := suite.authService.GenerateToken(&user)
	suite.token = token

	// Create test API key
	apiKey := models.APIKey{
		Key:      "test-api-key-123",
		Name:     "Test API Key",
		IsActive: true,
	}
	database.DB.Create(&apiKey)
	suite.apiKey = apiKey.Key
}

func (suite *APITestSuite) TestHealthCheck() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", response["status"])
}

func (suite *APITestSuite) TestLogin() {
	loginData := map[string]string{
		"username": "testuser",
		"password": "testpass",
	}
	
	jsonData, _ := json.Marshal(loginData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
	
	var response api.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), response.Token)
	assert.Equal(suite.T(), "testuser", response.User.Username)
}

func (suite *APITestSuite) TestLoginInvalidCredentials() {
	loginData := map[string]string{
		"username": "testuser",
		"password": "wrongpass",
	}
	
	jsonData, _ := json.Marshal(loginData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 401, w.Code)
}

func (suite *APITestSuite) TestSubmitJobWithAPIKey() {
	// Create test audio file
	audioFile := suite.createTestAudioFile()
	defer os.Remove(audioFile)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	// Add audio file
	file, err := os.Open(audioFile)
	assert.NoError(suite.T(), err)
	defer file.Close()
	
	part, err := writer.CreateFormFile("audio", "test.mp3")
	assert.NoError(suite.T(), err)
	io.Copy(part, file)
	
	// Add other form fields
	writer.WriteField("title", "Test Audio")
	writer.WriteField("model", "base")
	writer.WriteField("diarization", "false")
	
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/transcription/submit", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", suite.apiKey)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
	
	var response models.TranscriptionJob
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), response.ID)
	assert.Equal(suite.T(), "Test Audio", *response.Title)
	assert.Equal(suite.T(), models.StatusPending, response.Status)
}

func (suite *APITestSuite) TestSubmitJobWithJWT() {
	audioFile := suite.createTestAudioFile()
	defer os.Remove(audioFile)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	file, err := os.Open(audioFile)
	assert.NoError(suite.T(), err)
	defer file.Close()
	
	part, err := writer.CreateFormFile("audio", "test.mp3")
	assert.NoError(suite.T(), err)
	io.Copy(part, file)
	
	writer.WriteField("title", "JWT Test Audio")
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/transcription/submit", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+suite.token)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
}

func (suite *APITestSuite) TestSubmitJobUnauthorized() {
	audioFile := suite.createTestAudioFile()
	defer os.Remove(audioFile)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	file, err := os.Open(audioFile)
	assert.NoError(suite.T(), err)
	defer file.Close()
	
	part, err := writer.CreateFormFile("audio", "test.mp3")
	assert.NoError(suite.T(), err)
	io.Copy(part, file)
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/transcription/submit", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// No authentication header
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 401, w.Code)
}

func (suite *APITestSuite) TestGetJobStatus() {
	// Create test job
	job := suite.createTestJob()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/transcription/%s/status", job.ID), nil)
	req.Header.Set("X-API-Key", suite.apiKey)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
	
	var response models.TranscriptionJob
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), job.ID, response.ID)
}

func (suite *APITestSuite) TestGetJobStatusNotFound() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/transcription/nonexistent/status", nil)
	req.Header.Set("X-API-Key", suite.apiKey)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 404, w.Code)
}

func (suite *APITestSuite) TestGetJobByID() {
	job := suite.createTestJob()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/transcription/%s", job.ID), nil)
	req.Header.Set("X-API-Key", suite.apiKey)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
	
	var response models.TranscriptionJob
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), job.ID, response.ID)
}

func (suite *APITestSuite) TestListJobs() {
	// Create multiple test jobs
	job1 := suite.createTestJob()
	job2 := suite.createTestJob()
	_ = job1
	_ = job2

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/transcription/list", nil)
	req.Header.Set("X-API-Key", suite.apiKey)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	
	jobs := response["jobs"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(jobs), 2)
	
	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(suite.T(), float64(1), pagination["page"])
}

func (suite *APITestSuite) TestListJobsWithPagination() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/transcription/list?page=1&limit=5", nil)
	req.Header.Set("X-API-Key", suite.apiKey)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
}

func (suite *APITestSuite) TestGetTranscriptNotCompleted() {
	job := suite.createTestJob()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/transcription/%s/transcript", job.ID), nil)
	req.Header.Set("X-API-Key", suite.apiKey)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 400, w.Code)
}

func (suite *APITestSuite) TestGetTranscriptCompleted() {
	job := suite.createTestJobWithTranscript()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/transcription/%s/transcript", job.ID), nil)
	req.Header.Set("X-API-Key", suite.apiKey)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), job.ID, response["job_id"])
	assert.NotNil(suite.T(), response["transcript"])
}

func (suite *APITestSuite) TestGetSupportedModels() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/transcription/models", nil)
	req.Header.Set("X-API-Key", suite.apiKey)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	
	models := response["models"].([]interface{})
	languages := response["languages"].([]interface{})
	
	assert.Greater(suite.T(), len(models), 0)
	assert.Greater(suite.T(), len(languages), 0)
}

func (suite *APITestSuite) TestGetQueueStats() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/queue/stats", nil)
	req.Header.Set("X-API-Key", suite.apiKey)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	
	assert.Contains(suite.T(), response, "queue_size")
	assert.Contains(suite.T(), response, "workers")
	assert.Contains(suite.T(), response, "pending_jobs")
}

// Helper methods

func (suite *APITestSuite) createTestAudioFile() string {
	// Create a dummy MP3 file for testing
	tmpFile, err := os.CreateTemp("", "test_audio_*.mp3")
	assert.NoError(suite.T(), err)
	
	// Write some dummy data
	tmpFile.WriteString("dummy mp3 data for testing")
	tmpFile.Close()
	
	return tmpFile.Name()
}

func (suite *APITestSuite) createTestJob() *models.TranscriptionJob {
	job := &models.TranscriptionJob{
		ID:        "test-job-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Title:     stringPtr("Test Job"),
		Status:    models.StatusPending,
		AudioPath: "test/path/audio.mp3",
		Parameters: models.WhisperXParams{
			Model:       "base",
			BatchSize:   16,
			ComputeType: "float16",
			Device:      "auto",
		},
	}
	
	database.DB.Create(job)
	return job
}

func (suite *APITestSuite) createTestJobWithTranscript() *models.TranscriptionJob {
	transcript := `{"segments": [{"start": 0.0, "end": 5.0, "text": "This is a test transcript."}], "language": "en"}`
	
	job := &models.TranscriptionJob{
		ID:         "test-job-with-transcript-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Title:      stringPtr("Test Job with Transcript"),
		Status:     models.StatusCompleted,
		AudioPath:  "test/path/audio.mp3",
		Transcript: &transcript,
		Parameters: models.WhisperXParams{
			Model:       "base",
			BatchSize:   16,
			ComputeType: "float16",
			Device:      "auto",
		},
	}
	
	database.DB.Create(job)
	return job
}

func stringPtr(s string) *string {
	return &s
}

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}