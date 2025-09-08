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
	"strings"
	"testing"

	"scriberr/internal/api"
	"scriberr/internal/models"
	"scriberr/internal/queue"
	"scriberr/internal/transcription"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type APIHandlerTestSuite struct {
	suite.Suite
	helper             *TestHelper
	router             *gin.Engine
	handler            *api.Handler
	taskQueue          *queue.TaskQueue
	whisperXService    *transcription.WhisperXService
	quickTranscription *transcription.QuickTranscriptionService
}

func (suite *APIHandlerTestSuite) SetupSuite() {
	suite.helper = NewTestHelper(suite.T(), "api_handlers_test.db")

	// Initialize services
	suite.whisperXService = transcription.NewWhisperXService(suite.helper.Config)
	var err error
	suite.quickTranscription, err = transcription.NewQuickTranscriptionService(suite.helper.Config, suite.whisperXService)
	assert.NoError(suite.T(), err)

	suite.taskQueue = queue.NewTaskQueue(1, suite.whisperXService)
	suite.handler = api.NewHandler(suite.helper.Config, suite.helper.AuthService, suite.taskQueue, suite.whisperXService, suite.quickTranscription)

	// Set up router
	suite.router = api.SetupRoutes(suite.handler, suite.helper.AuthService)
}

func (suite *APIHandlerTestSuite) TearDownSuite() {
	suite.helper.Cleanup()
}

// Helper method to make authenticated requests
func (suite *APIHandlerTestSuite) makeAuthenticatedRequest(method, path string, body interface{}, useJWT bool) *httptest.ResponseRecorder {
	var req *http.Request
	var err error

	if body != nil {
		switch v := body.(type) {
		case string:
			req, err = http.NewRequest(method, path, strings.NewReader(v))
		case []byte:
			req, err = http.NewRequest(method, path, bytes.NewBuffer(v))
		case *bytes.Buffer:
			req, err = http.NewRequest(method, path, v)
		default:
			jsonBody, _ := json.Marshal(v)
			req, err = http.NewRequest(method, path, bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
		}
	} else {
		req, err = http.NewRequest(method, path, nil)
	}

	assert.NoError(suite.T(), err)

	// Add authentication
	if useJWT {
		req.Header.Set("Authorization", "Bearer "+suite.helper.TestToken)
	} else {
		req.Header.Set("X-API-Key", suite.helper.TestAPIKey)
	}

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

// Test health check endpoint
func (suite *APIHandlerTestSuite) TestHealthCheck() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", response["status"])
}

// Test user registration
func (suite *APIHandlerTestSuite) TestRegisterUser() {
	registerData := map[string]string{
		"username": "newuser123",
		"password": "newpassword123",
	}

	jsonData, _ := json.Marshal(registerData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	// Should return 400 because registration might be disabled or user already exists
	assert.True(suite.T(), w.Code == 200 || w.Code == 400 || w.Code == 409)
}

// Test user login
func (suite *APIHandlerTestSuite) TestLoginUser() {
	loginData := map[string]string{
		"username": suite.helper.TestUser.Username,
		"password": "testpassword123",
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
	assert.Equal(suite.T(), suite.helper.TestUser.Username, response.User.Username)
}

// Test getting registration status
func (suite *APIHandlerTestSuite) TestGetRegistrationStatus() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/auth/registration-status", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response, "registration_enabled")
}

// Test API key management
func (suite *APIHandlerTestSuite) TestAPIKeyManagement() {
	// List API keys (JWT required)
	w := suite.makeAuthenticatedRequest("GET", "/api/v1/api-keys/", nil, true)
	assert.Equal(suite.T(), 200, w.Code)

	var listResponse []models.APIKey
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	assert.NoError(suite.T(), err)

	// Should contain at least our test API key
	found := false
	for _, key := range listResponse {
		if key.Key == suite.helper.TestAPIKey {
			found = true
			break
		}
	}
	assert.True(suite.T(), found)

	// Create new API key (JWT required)
	createData := map[string]string{
		"name":        "Test Created Key",
		"description": "Key created during testing",
	}

	w = suite.makeAuthenticatedRequest("POST", "/api/v1/api-keys/", createData, true)
	assert.Equal(suite.T(), 200, w.Code)

	var createResponse models.APIKey
	err = json.Unmarshal(w.Body.Bytes(), &createResponse)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Test Created Key", createResponse.Name)
	assert.NotEmpty(suite.T(), createResponse.Key)

	// Delete the created API key
	w = suite.makeAuthenticatedRequest("DELETE", fmt.Sprintf("/api/v1/api-keys/%d", createResponse.ID), nil, true)
	assert.Equal(suite.T(), 200, w.Code)
}

// Test transcription job listing
func (suite *APIHandlerTestSuite) TestListTranscriptionJobs() {
	// Create a test job first
	testJob := suite.helper.CreateTestTranscriptionJob(suite.T(), "Test Job for Listing")

	w := suite.makeAuthenticatedRequest("GET", "/api/v1/transcription/list", nil, false)
	assert.Equal(suite.T(), 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "jobs")
	assert.Contains(suite.T(), response, "pagination")

	jobs := response["jobs"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(jobs), 1)

	// Check if our test job is in the list
	foundJob := false
	for _, job := range jobs {
		jobMap := job.(map[string]interface{})
		if jobMap["id"] == testJob.ID {
			foundJob = true
			break
		}
	}
	assert.True(suite.T(), foundJob)
}

// Test getting transcription job by ID
func (suite *APIHandlerTestSuite) TestGetTranscriptionJobByID() {
	testJob := suite.helper.CreateTestTranscriptionJob(suite.T(), "Test Job by ID")

	w := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/transcription/%s", testJob.ID), nil, false)
	assert.Equal(suite.T(), 200, w.Code)

	var response models.TranscriptionJob
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), testJob.ID, response.ID)
	assert.Equal(suite.T(), *testJob.Title, *response.Title)
}

// Test getting job status
func (suite *APIHandlerTestSuite) TestGetJobStatus() {
	testJob := suite.helper.CreateTestTranscriptionJob(suite.T(), "Test Job Status")

	w := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/transcription/%s/status", testJob.ID), nil, false)
	assert.Equal(suite.T(), 200, w.Code)

	var response models.TranscriptionJob
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), testJob.ID, response.ID)
	assert.Equal(suite.T(), models.StatusPending, response.Status)
}

// Test updating transcription title
func (suite *APIHandlerTestSuite) TestUpdateTranscriptionTitle() {
	testJob := suite.helper.CreateTestTranscriptionJob(suite.T(), "Original Title")

	updateData := map[string]string{
		"title": "Updated Title",
	}

	w := suite.makeAuthenticatedRequest("PUT", fmt.Sprintf("/api/v1/transcription/%s/title", testJob.ID), updateData, false)
	assert.Equal(suite.T(), 200, w.Code)

	// Verify the title was updated
	w = suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/transcription/%s", testJob.ID), nil, false)
	assert.Equal(suite.T(), 200, w.Code)

	var response models.TranscriptionJob
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Title", *response.Title)
}

// Test deleting transcription job
func (suite *APIHandlerTestSuite) TestDeleteTranscriptionJob() {
	testJob := suite.helper.CreateTestTranscriptionJob(suite.T(), "Job to Delete")

	w := suite.makeAuthenticatedRequest("DELETE", fmt.Sprintf("/api/v1/transcription/%s", testJob.ID), nil, false)
	assert.Equal(suite.T(), 200, w.Code)

	// Verify the job was deleted
	w = suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/transcription/%s", testJob.ID), nil, false)
	assert.Equal(suite.T(), 404, w.Code)
}

// Test getting supported models
func (suite *APIHandlerTestSuite) TestGetSupportedModels() {
	w := suite.makeAuthenticatedRequest("GET", "/api/v1/transcription/models", nil, false)
	assert.Equal(suite.T(), 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "models")
	assert.Contains(suite.T(), response, "languages")

	models := response["models"].([]interface{})
	languages := response["languages"].([]interface{})

	assert.Greater(suite.T(), len(models), 0)
	assert.Greater(suite.T(), len(languages), 0)
}

// Test profile management
func (suite *APIHandlerTestSuite) TestProfileManagement() {
	// List profiles
	w := suite.makeAuthenticatedRequest("GET", "/api/v1/profiles/", nil, false)
	assert.Equal(suite.T(), 200, w.Code)

	// Create profile
	profileData := map[string]interface{}{
		"name":        "Test Profile",
		"description": "Test profile description",
		"parameters": map[string]interface{}{
			"model":      "base",
			"batch_size": 16,
			"device":     "auto",
		},
	}

	w = suite.makeAuthenticatedRequest("POST", "/api/v1/profiles/", profileData, false)
	assert.Equal(suite.T(), 200, w.Code)

	var createResponse models.TranscriptionProfile
	err := json.Unmarshal(w.Body.Bytes(), &createResponse)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Test Profile", createResponse.Name)

	// Get profile
	w = suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/profiles/%s", createResponse.ID), nil, false)
	assert.Equal(suite.T(), 200, w.Code)

	// Update profile
	updateData := map[string]interface{}{
		"name":        "Updated Profile",
		"description": "Updated description",
	}

	w = suite.makeAuthenticatedRequest("PUT", fmt.Sprintf("/api/v1/profiles/%s", createResponse.ID), updateData, false)
	assert.Equal(suite.T(), 200, w.Code)

	// Delete profile
	w = suite.makeAuthenticatedRequest("DELETE", fmt.Sprintf("/api/v1/profiles/%s", createResponse.ID), nil, false)
	assert.Equal(suite.T(), 200, w.Code)
}

// Test notes management
func (suite *APIHandlerTestSuite) TestNotesManagement() {
	// Create a transcription job first
	testJob := suite.helper.CreateTestTranscriptionJob(suite.T(), "Job for Notes")

	// Create note
	noteData := map[string]interface{}{
		"start_word_index": 0,
		"end_word_index":   5,
		"start_time":       0.0,
		"end_time":         2.5,
		"quote":            "Test quote text",
		"content":          "Test note content",
	}

	w := suite.makeAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/transcription/%s/notes", testJob.ID), noteData, false)
	assert.Equal(suite.T(), 200, w.Code)

	var createResponse models.Note
	err := json.Unmarshal(w.Body.Bytes(), &createResponse)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Test note content", createResponse.Content)

	// List notes for transcription
	w = suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/transcription/%s/notes", testJob.ID), nil, false)
	assert.Equal(suite.T(), 200, w.Code)

	var listResponse []models.Note
	err = json.Unmarshal(w.Body.Bytes(), &listResponse)
	assert.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(listResponse), 1)

	// Update note
	updateData := map[string]string{
		"content": "Updated note content",
	}

	w = suite.makeAuthenticatedRequest("PUT", fmt.Sprintf("/api/v1/notes/%s", createResponse.ID), updateData, false)
	assert.Equal(suite.T(), 200, w.Code)

	// Get updated note
	w = suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/notes/%s", createResponse.ID), nil, false)
	assert.Equal(suite.T(), 200, w.Code)

	var updatedNote models.Note
	err = json.Unmarshal(w.Body.Bytes(), &updatedNote)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated note content", updatedNote.Content)

	// Delete note
	w = suite.makeAuthenticatedRequest("DELETE", fmt.Sprintf("/api/v1/notes/%s", createResponse.ID), nil, false)
	assert.Equal(suite.T(), 200, w.Code)
}

// Test queue stats
func (suite *APIHandlerTestSuite) TestGetQueueStats() {
	w := suite.makeAuthenticatedRequest("GET", "/api/v1/admin/queue/stats", nil, false)
	assert.Equal(suite.T(), 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "queue_size")
	assert.Contains(suite.T(), response, "workers")
	assert.Contains(suite.T(), response, "pending_jobs")
	assert.Contains(suite.T(), response, "processing_jobs")
	assert.Contains(suite.T(), response, "completed_jobs")
	assert.Contains(suite.T(), response, "failed_jobs")
}

// Test multipart file upload (transcription submit)
func (suite *APIHandlerTestSuite) TestTranscriptionSubmit() {
	// Create a dummy audio file
	tmpFile, err := os.CreateTemp("", "test_audio_*.mp3")
	assert.NoError(suite.T(), err)
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("dummy audio data for API handler testing")
	tmpFile.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add audio file
	file, err := os.Open(tmpFile.Name())
	assert.NoError(suite.T(), err)
	defer file.Close()

	part, err := writer.CreateFormFile("audio", "test.mp3")
	assert.NoError(suite.T(), err)
	io.Copy(part, file)

	// Add form fields
	writer.WriteField("title", "API Handler Test Audio")
	writer.WriteField("model", "base")
	writer.WriteField("diarization", "false")

	writer.Close()

	req, _ := http.NewRequest("POST", "/api/v1/transcription/submit", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", suite.helper.TestAPIKey)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)

	var response models.TranscriptionJob
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), response.ID)
	assert.Equal(suite.T(), "API Handler Test Audio", *response.Title)
	assert.Equal(suite.T(), models.StatusPending, response.Status)
}

// Test error responses for non-existent resources
func (suite *APIHandlerTestSuite) TestNotFoundErrors() {
	endpoints := []string{
		"/api/v1/transcription/nonexistent-job",
		"/api/v1/transcription/nonexistent-job/status",
		"/api/v1/transcription/nonexistent-job/transcript",
		"/api/v1/profiles/nonexistent-profile",
		"/api/v1/notes/nonexistent-note",
	}

	for _, endpoint := range endpoints {
		w := suite.makeAuthenticatedRequest("GET", endpoint, nil, false)
		assert.Equal(suite.T(), 404, w.Code, "Endpoint %s should return 404", endpoint)
	}
}

// Test invalid request data
func (suite *APIHandlerTestSuite) TestInvalidRequestData() {
	// Test invalid JSON for login
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 400, w.Code)

	// Test missing required fields
	emptyLogin := map[string]string{}
	w = suite.makeAuthenticatedRequest("POST", "/api/v1/auth/login", emptyLogin, false)
	assert.True(suite.T(), w.Code >= 400, "Should return error for empty login data")
}

// Test logout
func (suite *APIHandlerTestSuite) TestLogout() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/logout", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
}

func TestAPIHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(APIHandlerTestSuite))
}
