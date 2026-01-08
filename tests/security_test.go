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
	"scriberr/internal/auth"
	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/processing"
	"scriberr/internal/queue"
	"scriberr/internal/repository"
	"scriberr/internal/service"
	"scriberr/internal/sse"
	"scriberr/internal/transcription"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SecurityTestSuite struct {
	suite.Suite
	router                    *gin.Engine
	config                    *config.Config
	authService               *auth.AuthService
	taskQueue                 *queue.TaskQueue
	unifiedProcessor          *transcription.UnifiedJobProcessor
	quickTranscriptionService *transcription.QuickTranscriptionService
	handler                   *api.Handler
}

func (suite *SecurityTestSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test configuration
	suite.config = &config.Config{
		Port:         "8080",
		Host:         "localhost",
		DatabasePath: "security_test.db",
		JWTSecret:    "test-secret",
		UploadDir:    "security_test_uploads",

		WhisperXEnv: "test_whisperx_env",
	}

	// Initialize test database
	if err := database.Initialize(suite.config.DatabasePath); err != nil {
		suite.T().Fatal("Failed to initialize test database:", err)
	}

	// Initialize services
	suite.authService = auth.NewAuthService(suite.config.JWTSecret)
	// Initialize repositories
	jobRepo := repository.NewJobRepository(database.DB)
	userRepo := repository.NewUserRepository(database.DB)
	apiKeyRepo := repository.NewAPIKeyRepository(database.DB)
	profileRepo := repository.NewProfileRepository(database.DB)
	llmConfigRepo := repository.NewLLMConfigRepository(database.DB)
	summaryRepo := repository.NewSummaryRepository(database.DB)
	chatRepo := repository.NewChatRepository(database.DB)
	noteRepo := repository.NewNoteRepository(database.DB)
	speakerMappingRepo := repository.NewSpeakerMappingRepository(database.DB)
	refreshTokenRepo := repository.NewRefreshTokenRepository(database.DB)

	// Initialize services
	userService := service.NewUserService(userRepo, suite.authService)
	fileService := service.NewFileService()

	// Initialize services
	suite.unifiedProcessor = transcription.NewUnifiedJobProcessor(jobRepo, suite.config.TempDir, suite.config.TranscriptsDir)
	var err error
	suite.quickTranscriptionService, err = transcription.NewQuickTranscriptionService(suite.config, suite.unifiedProcessor, jobRepo)
	if err != nil {
		suite.T().Fatal("Failed to initialize quick transcription service:", err)
	}
	suite.taskQueue = queue.NewTaskQueue(1, suite.unifiedProcessor, jobRepo)

	broadcaster := sse.NewBroadcaster()

	multiTrackProcessor := processing.NewMultiTrackProcessor(database.DB, jobRepo)

	suite.handler = api.NewHandler(
		suite.config,
		suite.authService,
		userService,
		fileService,
		jobRepo,
		apiKeyRepo,
		profileRepo,
		userRepo,
		llmConfigRepo,
		summaryRepo,
		chatRepo,
		noteRepo,
		speakerMappingRepo,
		refreshTokenRepo,
		suite.taskQueue,
		suite.unifiedProcessor,
		suite.quickTranscriptionService,
		multiTrackProcessor,
		broadcaster,
	)

	// Set up router
	suite.router = api.SetupRoutes(suite.handler, suite.authService)

	// Create upload directory
	os.MkdirAll(suite.config.UploadDir, 0755)
}

func (suite *SecurityTestSuite) TearDownSuite() {
	// Clean up test database
	database.Close()
	os.Remove(suite.config.DatabasePath)

	// Clean up upload directory
	os.RemoveAll(suite.config.UploadDir)
}

// Helper method to make requests without authentication
func (suite *SecurityTestSuite) makeUnauthenticatedRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
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

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

// Helper method to create multipart form data without auth
func (suite *SecurityTestSuite) makeMultipartRequest(path string, fields map[string]string, filename string) *httptest.ResponseRecorder {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	for key, value := range fields {
		writer.WriteField(key, value)
	}

	// Add file if filename provided
	if filename != "" {
		// Create a dummy file
		tmpFile, err := os.CreateTemp("", "security_test_*.mp3")
		assert.NoError(suite.T(), err)
		tmpFile.WriteString("dummy audio data for security testing")
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		file, err := os.Open(tmpFile.Name())
		assert.NoError(suite.T(), err)
		defer file.Close()

		part, err := writer.CreateFormFile("audio", filename)
		assert.NoError(suite.T(), err)
		io.Copy(part, file)
	}

	writer.Close()

	req, err := http.NewRequest("POST", path, body)
	assert.NoError(suite.T(), err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

// Test JWT-only endpoints (account management)
func (suite *SecurityTestSuite) TestAccountManagementEndpointsUnauthorized() {
	testCases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{"POST", "/api/v1/auth/change-password", map[string]string{"old_password": "old", "new_password": "new"}},
		{"POST", "/api/v1/auth/change-username", map[string]string{"new_username": "newuser"}},
	}

	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			w := suite.makeUnauthenticatedRequest(tc.method, tc.path, tc.body)
			assert.Equal(t, 401, w.Code, "Should return 401 Unauthorized for unauthenticated request to %s %s", tc.method, tc.path)
		})
	}
}

// Test API key management endpoints (JWT-only)
func (suite *SecurityTestSuite) TestAPIKeyManagementEndpointsUnauthorized() {
	testCases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{"GET", "/api/v1/api-keys/", nil},
		{"POST", "/api/v1/api-keys/", map[string]string{"name": "test key"}},
		{"DELETE", "/api/v1/api-keys/123", nil},
	}

	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			w := suite.makeUnauthenticatedRequest(tc.method, tc.path, tc.body)
			assert.Equal(t, 401, w.Code, "Should return 401 Unauthorized for unauthenticated request to %s %s", tc.method, tc.path)
		})
	}
}

// Test transcription endpoints
func (suite *SecurityTestSuite) TestTranscriptionEndpointsUnauthorized() {
	testCases := []struct {
		method      string
		path        string
		body        interface{}
		isMultipart bool
	}{
		{"POST", "/api/v1/transcription/upload", nil, true},
		{"POST", "/api/v1/transcription/youtube", map[string]string{"url": "https://youtube.com/watch?v=123"}, false},
		{"POST", "/api/v1/transcription/submit", nil, true},
		{"POST", "/api/v1/transcription/test-id/start", nil, false},
		{"POST", "/api/v1/transcription/test-id/kill", nil, false},
		{"GET", "/api/v1/transcription/test-id/status", nil, false},
		{"GET", "/api/v1/transcription/test-id/transcript", nil, false},
		{"GET", "/api/v1/transcription/test-id/audio", nil, false},
		{"PUT", "/api/v1/transcription/test-id/title", map[string]string{"title": "New Title"}, false},
		{"GET", "/api/v1/transcription/test-id/summary", nil, false},
		{"GET", "/api/v1/transcription/test-id", nil, false},
		{"DELETE", "/api/v1/transcription/test-id", nil, false},
		{"GET", "/api/v1/transcription/list", nil, false},
		{"GET", "/api/v1/transcription/models", nil, false},
		{"GET", "/api/v1/transcription/test-id/notes", nil, false},
		{"POST", "/api/v1/transcription/test-id/notes", map[string]string{"content": "Test note"}, false},
		{"POST", "/api/v1/transcription/quick", nil, true},
		{"GET", "/api/v1/transcription/quick/test-id", nil, false},
	}

	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			var w *httptest.ResponseRecorder

			if tc.isMultipart && tc.method == "POST" {
				fields := map[string]string{
					"title": "Test Audio",
					"model": "base",
				}
				if strings.Contains(tc.path, "quick") {
					w = suite.makeMultipartRequest(tc.path, fields, "test.mp3")
				} else {
					w = suite.makeMultipartRequest(tc.path, fields, "test.mp3")
				}
			} else {
				w = suite.makeUnauthenticatedRequest(tc.method, tc.path, tc.body)
			}

			assert.Equal(t, 401, w.Code, "Should return 401 Unauthorized for unauthenticated request to %s %s", tc.method, tc.path)
		})
	}
}

// Test profile endpoints
func (suite *SecurityTestSuite) TestProfileEndpointsUnauthorized() {
	testCases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{"GET", "/api/v1/profiles/", nil},
		{"POST", "/api/v1/profiles/", map[string]interface{}{
			"name": "Test Profile",
			"parameters": map[string]interface{}{
				"model":      "base",
				"batch_size": 16,
				"device":     "auto",
			},
		}},
		{"GET", "/api/v1/profiles/123", nil},
		{"PUT", "/api/v1/profiles/123", map[string]interface{}{
			"name": "Updated Profile",
		}},
		{"DELETE", "/api/v1/profiles/123", nil},
		{"POST", "/api/v1/profiles/123/set-default", nil},
	}

	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			w := suite.makeUnauthenticatedRequest(tc.method, tc.path, tc.body)
			assert.Equal(t, 401, w.Code, "Should return 401 Unauthorized for unauthenticated request to %s %s", tc.method, tc.path)
		})
	}
}

// Test admin endpoints
func (suite *SecurityTestSuite) TestAdminEndpointsUnauthorized() {
	testCases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{"GET", "/api/v1/admin/queue/stats", nil},
	}

	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			w := suite.makeUnauthenticatedRequest(tc.method, tc.path, tc.body)
			assert.Equal(t, 401, w.Code, "Should return 401 Unauthorized for unauthenticated request to %s %s", tc.method, tc.path)
		})
	}
}

// Test LLM configuration endpoints
func (suite *SecurityTestSuite) TestLLMConfigEndpointsUnauthorized() {
	testCases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{"GET", "/api/v1/llm/config", nil},
		{"POST", "/api/v1/llm/config", map[string]interface{}{
			"provider": "openai",
			"api_key":  "test-key",
		}},
	}

	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			w := suite.makeUnauthenticatedRequest(tc.method, tc.path, tc.body)
			assert.Equal(t, 401, w.Code, "Should return 401 Unauthorized for unauthenticated request to %s %s", tc.method, tc.path)
		})
	}
}

// Test summary template endpoints
func (suite *SecurityTestSuite) TestSummaryTemplateEndpointsUnauthorized() {
	testCases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{"GET", "/api/v1/summaries/", nil},
		{"POST", "/api/v1/summaries/", map[string]string{
			"name":     "Test Template",
			"template": "Summarize this: {{content}}",
		}},
		{"GET", "/api/v1/summaries/123", nil},
		{"PUT", "/api/v1/summaries/123", map[string]string{
			"name": "Updated Template",
		}},
		{"DELETE", "/api/v1/summaries/123", nil},
		{"GET", "/api/v1/summaries/settings", nil},
		{"POST", "/api/v1/summaries/settings", map[string]interface{}{
			"auto_summarize": true,
		}},
	}

	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			w := suite.makeUnauthenticatedRequest(tc.method, tc.path, tc.body)
			assert.Equal(t, 401, w.Code, "Should return 401 Unauthorized for unauthenticated request to %s %s", tc.method, tc.path)
		})
	}
}

// Test chat endpoints
func (suite *SecurityTestSuite) TestChatEndpointsUnauthorized() {
	testCases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{"GET", "/api/v1/chat/models", nil},
		{"POST", "/api/v1/chat/sessions", map[string]interface{}{
			"transcription_id": "test-id",
			"title":            "Test Session",
		}},
		{"GET", "/api/v1/chat/transcriptions/test-id/sessions", nil},
		{"GET", "/api/v1/chat/sessions/session-123", nil},
		{"POST", "/api/v1/chat/sessions/session-123/messages", map[string]string{
			"content": "Hello",
		}},
		{"PUT", "/api/v1/chat/sessions/session-123/title", map[string]string{
			"title": "New Title",
		}},
		{"POST", "/api/v1/chat/sessions/session-123/title/auto", nil},
		{"DELETE", "/api/v1/chat/sessions/session-123", nil},
	}

	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			w := suite.makeUnauthenticatedRequest(tc.method, tc.path, tc.body)
			assert.Equal(t, 401, w.Code, "Should return 401 Unauthorized for unauthenticated request to %s %s", tc.method, tc.path)
		})
	}
}

// Test notes endpoints
func (suite *SecurityTestSuite) TestNotesEndpointsUnauthorized() {
	testCases := []struct {
		method string
		path   string
		body   interface{}
	}{
		{"GET", "/api/v1/notes/note-123", nil},
		{"PUT", "/api/v1/notes/note-123", map[string]string{
			"content": "Updated note content",
		}},
		{"DELETE", "/api/v1/notes/note-123", nil},
	}

	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			w := suite.makeUnauthenticatedRequest(tc.method, tc.path, tc.body)
			assert.Equal(t, 401, w.Code, "Should return 401 Unauthorized for unauthenticated request to %s %s", tc.method, tc.path)
		})
	}
}

// Test summarize endpoint
func (suite *SecurityTestSuite) TestSummarizeEndpointUnauthorized() {
	w := suite.makeUnauthenticatedRequest("POST", "/api/v1/summarize/", map[string]interface{}{
		"transcription_id": "test-id",
		"template_id":      "template-123",
	})
	assert.Equal(suite.T(), 401, w.Code, "Should return 401 Unauthorized for unauthenticated request to POST /api/v1/summarize/")
}

// Test that public endpoints still work without authentication
func (suite *SecurityTestSuite) TestPublicEndpointsAccessible() {
	publicEndpoints := []struct {
		method       string
		path         string
		allowedCodes []int // codes that are acceptable (anything except 401)
	}{
		{"GET", "/health", []int{200}},
		{"GET", "/swagger/index.html", []int{200, 301, 302, 404}}, // swagger might redirect or not exist
		{"GET", "/api/v1/auth/registration-status", []int{200}},
		{"POST", "/api/v1/auth/register", []int{200, 400, 409}}, // 400 for validation errors, 409 for user exists
		{"POST", "/api/v1/auth/login", []int{200, 400, 401}},    // 401 for invalid creds is OK for login endpoint
		{"POST", "/api/v1/auth/logout", []int{200}},
	}

	for _, endpoint := range publicEndpoints {
		suite.T().Run(fmt.Sprintf("%s %s should be accessible", endpoint.method, endpoint.path), func(t *testing.T) {
			var body interface{}
			if endpoint.method == "POST" && strings.Contains(endpoint.path, "register") {
				body = map[string]string{
					"username": "newtestuser",
					"password": "testpass",
				}
			} else if endpoint.method == "POST" && strings.Contains(endpoint.path, "login") {
				body = map[string]string{
					"username": "nonexistentuser",
					"password": "wrongpass",
				}
			}

			w := suite.makeUnauthenticatedRequest(endpoint.method, endpoint.path, body)

			// Check if response code is in allowed codes list
			codeAllowed := false
			for _, allowedCode := range endpoint.allowedCodes {
				if w.Code == allowedCode {
					codeAllowed = true
					break
				}
			}

			assert.True(t, codeAllowed, "Public endpoint %s %s returned %d, expected one of %v", endpoint.method, endpoint.path, w.Code, endpoint.allowedCodes)
		})
	}
}

// Test with invalid/malformed authorization headers
func (suite *SecurityTestSuite) TestMalformedAuthHeaders() {
	testEndpoint := "/api/v1/transcription/list"

	malformedHeaders := []struct {
		name   string
		header string
		value  string
	}{
		{"Invalid Bearer format", "Authorization", "InvalidBearer token123"},
		{"Empty Bearer token", "Authorization", "Bearer "},
		{"Invalid JWT token", "Authorization", "Bearer invalid.jwt.token"},
		{"Empty API key", "X-API-Key", ""},
		{"Malformed API key", "X-API-Key", "malformed-key-123"},
	}

	for _, tc := range malformedHeaders {
		suite.T().Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", testEndpoint, nil)
			req.Header.Set(tc.header, tc.value)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(t, 401, w.Code, "Should return 401 for malformed auth header: %s", tc.name)
		})
	}
}

// Test CORS preflight requests don't bypass authentication
func (suite *SecurityTestSuite) TestCORSPreflightDoesNotBypassAuth() {
	protectedEndpoint := "/api/v1/transcription/list"

	// OPTIONS request should return 204 (handled by CORS middleware)
	req, _ := http.NewRequest("OPTIONS", protectedEndpoint, nil)
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Origin", "https://evil.example.com")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 204, w.Code, "OPTIONS request should return 204")

	// But actual GET request should still require authentication
	w2 := suite.makeUnauthenticatedRequest("GET", protectedEndpoint, nil)
	assert.Equal(suite.T(), 401, w2.Code, "GET request after CORS preflight should still require authentication")
}

// Test security headers are properly set
func (suite *SecurityTestSuite) TestSecurityHeaders() {
	w := suite.makeUnauthenticatedRequest("GET", "/health", nil)

	// Check CORS headers are present
	assert.NotEmpty(suite.T(), w.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(suite.T(), w.Header().Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(suite.T(), w.Header().Get("Access-Control-Allow-Headers"))
}

func TestSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}
