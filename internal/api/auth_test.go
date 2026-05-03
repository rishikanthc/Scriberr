package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"scriberr/internal/account"
	"scriberr/internal/annotations"
	"scriberr/internal/auth"
	"scriberr/internal/automation"
	"scriberr/internal/config"
	"scriberr/internal/database"
	filesdomain "scriberr/internal/files"
	"scriberr/internal/llmprovider"
	"scriberr/internal/mediaimport"
	"scriberr/internal/models"
	profiledomain "scriberr/internal/profile"
	recordingdomain "scriberr/internal/recording"
	"scriberr/internal/repository"
	"scriberr/internal/tags"
	transcriptiondomain "scriberr/internal/transcription"
	"scriberr/pkg/logger"

	"github.com/stretchr/testify/require"
	gormlogger "gorm.io/gorm/logger"
)

type authTestServer struct {
	router    http.Handler
	auth      *auth.AuthService
	uploadDir string
	handler   *Handler
}

func newAuthTestServer(t *testing.T) *authTestServer {
	t.Helper()

	logger.Init("silent")
	require.NoError(t, database.Initialize(filepath.Join(t.TempDir(), "scriberr.db")))
	database.DB.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	t.Cleanup(func() { _ = database.Close() })

	authService := auth.NewAuthService("test-secret")
	uploadDir := filepath.Join(t.TempDir(), "uploads")
	cfg := &config.Config{
		Environment: "test",
		UploadDir:   uploadDir,
		Recordings: config.RecordingConfig{
			Dir:              filepath.Join(t.TempDir(), "recordings"),
			MaxChunkBytes:    8,
			MaxDuration:      time.Hour,
			SessionTTL:       time.Hour,
			AllowedMimeTypes: []string{"audio/webm;codecs=opus", "audio/webm"},
		},
	}
	jobRepo := repository.NewJobRepository(database.DB)
	profileRepo := repository.NewProfileRepository(database.DB)
	llmConfigRepo := repository.NewLLMConfigRepository(database.DB)
	accountService := account.NewService(
		repository.NewUserRepository(database.DB),
		repository.NewRefreshTokenRepository(database.DB),
		repository.NewAPIKeyRepository(database.DB),
		profileRepo,
		llmConfigRepo,
		authService,
	)
	profileService := profiledomain.NewService(profileRepo)
	llmProviderService := llmprovider.NewService(llmConfigRepo, LLMProviderConnectionTester{})
	fileService := filesdomain.NewService(jobRepo, filesdomain.Config{UploadDir: cfg.UploadDir})
	mediaImportService := mediaimport.NewService(mediaimport.ServiceOptions{
		Repository: jobRepo,
		UploadDir:  cfg.UploadDir,
	})
	transcriptionService := transcriptiondomain.NewService(jobRepo, profileRepo, nil)
	postFileAutomation := automation.NewService(jobRepo, repository.NewUserRepository(database.DB), profileRepo, llmConfigRepo, transcriptionService)
	fileService.SetReadyObserver(postFileAutomation)
	annotationService := annotations.NewService(repository.NewAnnotationRepository(database.DB), jobRepo)
	tagService := tags.NewService(repository.NewTagRepository(database.DB), jobRepo)
	recordingStorage, err := recordingdomain.NewStorage(cfg.Recordings.Dir)
	require.NoError(t, err)
	recordingService := recordingdomain.NewService(repository.NewRecordingRepository(database.DB), recordingStorage, recordingdomain.Config{
		MaxChunkBytes:    cfg.Recordings.MaxChunkBytes,
		MaxDuration:      cfg.Recordings.MaxDuration,
		SessionTTL:       cfg.Recordings.SessionTTL,
		AllowedMimeTypes: cfg.Recordings.AllowedMimeTypes,
	})
	handler := NewHandler(cfg, authService, HandlerDependencies{
		ReadinessCheck: func() error { return nil },
		Account:        accountService,
		Profiles:       profileService,
		LLMProvider:    llmProviderService,
		Files:          fileService,
		MediaImport:    mediaImportService,
		Annotations:    annotationService,
		Tags:           tagService,
		Recordings:     recordingService,
		Transcriptions: transcriptionService,
	})
	postFileAutomation.SetEventPublisher(handler)
	youtubeImporter := &fakeYouTubeImporter{block: make(chan struct{})}
	mediaImportService.SetImporter(youtubeImporter)
	t.Cleanup(func() {
		youtubeImporter.unblock()
		handler.asyncJobs.Wait()
	})

	return &authTestServer{router: SetupRoutes(handler, authService), auth: authService, uploadDir: uploadDir, handler: handler}
}

func (s *authTestServer) request(t *testing.T, method, path string, body any, token string, apiKey string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	var payload bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&payload).Encode(body))
	}
	req, err := http.NewRequest(method, path, &payload)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)

	var response map[string]any
	if recorder.Code != http.StatusNoContent {
		require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	}
	return recorder, response
}

func (s *authTestServer) rawRequest(t *testing.T, method, path string, body any, token string, apiKey string) (*httptest.ResponseRecorder, string) {
	t.Helper()

	var payload bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&payload).Encode(body))
	}
	req, err := http.NewRequest(method, path, &payload)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)
	return recorder, recorder.Body.String()
}

func TestAuthRegisterLoginRefreshMeLogout(t *testing.T) {
	s := newAuthTestServer(t)

	resp, body := s.request(t, http.MethodGet, "/api/v1/auth/registration-status", nil, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, true, body["registration_enabled"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":         "admin",
		"password":         "password123",
		"confirm_password": "password123",
	}, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.NotEmpty(t, body["access_token"])
	require.NotEmpty(t, body["refresh_token"])
	user := body["user"].(map[string]any)
	require.Equal(t, "user_self", user["id"])
	require.Equal(t, "admin", user["username"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/auth/registration-status", nil, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, false, body["registration_enabled"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "admin",
		"password": "password123",
	}, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	accessToken := body["access_token"].(string)
	refreshToken := body["refresh_token"].(string)

	resp, body = s.request(t, http.MethodGet, "/api/v1/auth/me", nil, accessToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "user_self", body["id"])
	require.Equal(t, "admin", body["username"])

	resp, body = s.request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]any{
		"refresh_token": refreshToken,
	}, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.NotEmpty(t, body["access_token"])
	rotatedRefresh := body["refresh_token"].(string)
	require.NotEqual(t, refreshToken, rotatedRefresh)

	resp, _ = s.request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]any{
		"refresh_token": refreshToken,
	}, "", "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)

	resp, body = s.request(t, http.MethodPost, "/api/v1/auth/logout", map[string]any{
		"refresh_token": rotatedRefresh,
	}, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, true, body["ok"])

	resp, _ = s.request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]any{
		"refresh_token": rotatedRefresh,
	}, "", "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestAuthValidationAndPasswordChanges(t *testing.T) {
	s := newAuthTestServer(t)

	resp, _ := s.request(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":         "ad",
		"password":         "password123",
		"confirm_password": "different",
	}, "", "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	resp, body := s.request(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":         "admin",
		"password":         "password123",
		"confirm_password": "password123",
	}, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	accessToken := body["access_token"].(string)

	resp, _ = s.request(t, http.MethodPost, "/api/v1/auth/change-password", map[string]any{
		"current_password": "wrong",
		"new_password":     "newpassword123",
		"confirm_password": "newpassword123",
	}, accessToken, "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)

	resp, body = s.request(t, http.MethodPost, "/api/v1/auth/change-password", map[string]any{
		"current_password": "password123",
		"new_password":     "newpassword123",
		"confirm_password": "newpassword123",
	}, accessToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, true, body["ok"])

	resp, _ = s.request(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "admin",
		"password": "password123",
	}, "", "")
	require.Equal(t, http.StatusUnauthorized, resp.Code)

	resp, body = s.request(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "admin",
		"password": "newpassword123",
	}, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	accessToken = body["access_token"].(string)

	resp, body = s.request(t, http.MethodPost, "/api/v1/auth/change-username", map[string]any{
		"new_username": "owner",
		"password":     "newpassword123",
	}, accessToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "user_self", body["id"])
	require.Equal(t, "owner", body["username"])
}

func TestAPIKeyCreateListDeleteAndRedaction(t *testing.T) {
	s := newAuthTestServer(t)

	resp, body := s.request(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":         "admin",
		"password":         "password123",
		"confirm_password": "password123",
	}, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	accessToken := body["access_token"].(string)

	resp, body = s.request(t, http.MethodPost, "/api/v1/api-keys", map[string]any{
		"name":        "CLI",
		"description": "Local scripts",
	}, accessToken, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	rawKey := body["key"].(string)
	require.NotEmpty(t, rawKey)
	require.Contains(t, rawKey, "sk_")
	keyID := body["id"].(string)

	var stored models.APIKey
	require.NoError(t, database.DB.First(&stored).Error)
	require.NotEqual(t, rawKey, stored.KeyHash)
	require.Equal(t, sha256String(rawKey), stored.KeyHash)

	resp, body = s.request(t, http.MethodGet, "/api/v1/api-keys", nil, accessToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	items := body["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	require.Equal(t, keyID, item["id"])
	require.NotContains(t, item, "key")
	require.NotContains(t, item, "key_hash")
	require.NotEmpty(t, item["key_preview"])

	resp, _ = s.request(t, http.MethodGet, "/api/v1/files", nil, "", rawKey)
	require.Equal(t, http.StatusOK, resp.Code)

	idNumber, err := strconv.Atoi(strings.TrimPrefix(keyID, "key_"))
	require.NoError(t, err)
	resp, _ = s.request(t, http.MethodDelete, "/api/v1/api-keys/"+strconv.Itoa(idNumber), nil, accessToken, "")
	require.Equal(t, http.StatusNoContent, resp.Code)

	resp, _ = s.request(t, http.MethodGet, "/api/v1/files", nil, "", rawKey)
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestAPIKeyManagementRequiresJWT(t *testing.T) {
	s := newAuthTestServer(t)

	resp, body := s.request(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":         "admin",
		"password":         "password123",
		"confirm_password": "password123",
	}, "", "")
	require.Equal(t, http.StatusOK, resp.Code)
	accessToken := body["access_token"].(string)

	resp, body = s.request(t, http.MethodPost, "/api/v1/api-keys", map[string]any{"name": "CLI"}, accessToken, "")
	require.Equal(t, http.StatusCreated, resp.Code)
	rawKey := body["key"].(string)

	resp, _ = s.request(t, http.MethodGet, "/api/v1/api-keys", nil, "", rawKey)
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func sha256String(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
