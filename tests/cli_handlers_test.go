package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scriberr/internal/api"
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

type CLIHandlerTestSuite struct {
	suite.Suite
	helper             *TestHelper
	router             *gin.Engine
	handler            *api.Handler
	taskQueue          *queue.TaskQueue
	unifiedProcessor   *transcription.UnifiedJobProcessor
	quickTranscription *transcription.QuickTranscriptionService
}

func (suite *CLIHandlerTestSuite) SetupSuite() {
	suite.helper = NewTestHelper(suite.T(), "cli_handlers_test.db")

	// Initialize repositories
	jobRepo := repository.NewJobRepository(suite.helper.DB)
	userRepo := repository.NewUserRepository(suite.helper.DB)
	apiKeyRepo := repository.NewAPIKeyRepository(suite.helper.DB)
	profileRepo := repository.NewProfileRepository(suite.helper.DB)
	llmConfigRepo := repository.NewLLMConfigRepository(suite.helper.DB)
	summaryRepo := repository.NewSummaryRepository(suite.helper.DB)
	chatRepo := repository.NewChatRepository(suite.helper.DB)
	noteRepo := repository.NewNoteRepository(suite.helper.DB)
	speakerMappingRepo := repository.NewSpeakerMappingRepository(suite.helper.DB)
	refreshTokenRepo := repository.NewRefreshTokenRepository(suite.helper.DB)

	// Initialize services
	userService := service.NewUserService(userRepo, suite.helper.AuthService)
	fileService := service.NewFileService()

	// Initialize services
	suite.unifiedProcessor = transcription.NewUnifiedJobProcessor(jobRepo, suite.helper.Config.TempDir, suite.helper.Config.TranscriptsDir)
	var err error
	suite.quickTranscription, err = transcription.NewQuickTranscriptionService(suite.helper.Config, suite.unifiedProcessor, jobRepo)
	assert.NoError(suite.T(), err)

	suite.taskQueue = queue.NewTaskQueue(1, suite.unifiedProcessor, jobRepo)

	broadcaster := sse.NewBroadcaster()

	multiTrackProcessor := processing.NewMultiTrackProcessor(suite.helper.DB, jobRepo)

	suite.handler = api.NewHandler(
		suite.helper.Config,
		suite.helper.AuthService,
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
		suite.quickTranscription,
		multiTrackProcessor,
		broadcaster,
	)

	// Set up router
	suite.router = api.SetupRoutes(suite.handler, suite.helper.AuthService)
}

func (suite *CLIHandlerTestSuite) TearDownSuite() {
	suite.helper.Cleanup()
}

func (suite *CLIHandlerTestSuite) SetupTest() {
	suite.helper.ResetDB(suite.T())
}

func (suite *CLIHandlerTestSuite) makeAuthenticatedRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var req *http.Request
	var err error

	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req, err = http.NewRequest(method, path, strings.NewReader(string(jsonBody)))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, path, nil)
	}

	assert.NoError(suite.T(), err)
	req.Header.Set("Authorization", "Bearer "+suite.helper.TestToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

func (suite *CLIHandlerTestSuite) TestAuthorizeCLI() {
	// Test GET /api/v1/auth/cli/authorize
	w := suite.makeAuthenticatedRequest("GET", "/api/v1/auth/cli/authorize", nil)
	assert.Equal(suite.T(), 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	userMap := response["user"].(map[string]interface{})
	assert.Equal(suite.T(), float64(suite.helper.TestUser.ID), userMap["id"])
	assert.Equal(suite.T(), suite.helper.TestUser.Username, userMap["username"])
}

func (suite *CLIHandlerTestSuite) TestConfirmCLIAuthorization() {
	// Test POST /api/v1/auth/cli/authorize
	body := map[string]string{
		"callback_url": "http://localhost:12345",
		"device_name":  "Test Device",
	}

	w := suite.makeAuthenticatedRequest("POST", "/api/v1/auth/cli/authorize", body)
	assert.Equal(suite.T(), 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	redirectURL := response["redirect_url"].(string)
	assert.Contains(suite.T(), redirectURL, "http://localhost:12345")
	assert.Contains(suite.T(), redirectURL, "token=")
	assert.Contains(suite.T(), redirectURL, "username="+suite.helper.TestUser.Username)
}

func (suite *CLIHandlerTestSuite) TestGetInstallScript() {
	// Test GET /api/v1/cli/install
	req, _ := http.NewRequest("GET", "/api/v1/cli/install", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
	assert.Equal(suite.T(), "text/x-shellscript", w.Header().Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(suite.T(), body, "#!/bin/bash")
	assert.Contains(suite.T(), body, "curl -sL")
}

func (suite *CLIHandlerTestSuite) TestDownloadCLIBinary() {
	// Create dummy binary file
	dummyDir := "bin/cli"
	os.MkdirAll(dummyDir, 0755)
	dummyFile := filepath.Join(dummyDir, "scriberr-linux-amd64")
	os.WriteFile(dummyFile, []byte("dummy binary content"), 0755)
	defer os.RemoveAll(dummyDir)

	// Test GET /api/v1/cli/download
	req, _ := http.NewRequest("GET", "/api/v1/cli/download?os=linux&arch=amd64", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 200, w.Code)
	assert.Equal(suite.T(), "dummy binary content", w.Body.String())
	assert.Contains(suite.T(), w.Header().Get("Content-Disposition"), "attachment")
}

func (suite *CLIHandlerTestSuite) TestDownloadCLIBinaryMissingParams() {
	req, _ := http.NewRequest("GET", "/api/v1/cli/download", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 400, w.Code)
}

func (suite *CLIHandlerTestSuite) TestDownloadCLIBinaryUnsupported() {
	req, _ := http.NewRequest("GET", "/api/v1/cli/download?os=unknown&arch=amd64", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), 400, w.Code)
}

func TestCLIHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(CLIHandlerTestSuite))
}
