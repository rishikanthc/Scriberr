package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"scriberr/internal/auth"
	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/web"
	"scriberr/pkg/logger"
	"scriberr/pkg/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const requestIDKey = "request_id"

type Handler struct {
	config         *config.Config
	authService    *auth.AuthService
	readinessCheck func() error
}

type ErrorBody struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code      string  `json:"code"`
	Message   string  `json:"message"`
	Field     *string `json:"field,omitempty"`
	RequestID string  `json:"request_id"`
}

type registerRequest struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type changeUsernameRequest struct {
	NewUsername string `json:"new_username"`
	Password    string `json:"password"`
}

type createAPIKeyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateFileRequest struct {
	Title string `json:"title"`
}

func NewHandler(cfg *config.Config, authService *auth.AuthService, _ ...any) *Handler {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return &Handler{
		config:         cfg,
		authService:    authService,
		readinessCheck: database.HealthCheck,
	}
}

func SetupRoutes(handler *Handler, _ *auth.AuthService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	logger.SetGinOutput()

	router := gin.New()
	router.Use(recoveryMiddleware())
	router.Use(requestIDMiddleware())
	router.Use(logger.GinLogger())
	router.Use(middleware.CompressionMiddleware())
	router.Use(corsMiddleware(handler.config))

	router.GET("/health", handler.health)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", handler.health)
		v1.GET("/ready", handler.ready)

		authRoutes := v1.Group("/auth")
		{
			authRoutes.GET("/registration-status", handler.registrationStatus)
			authRoutes.POST("/register", handler.register)
			authRoutes.POST("/login", handler.login)
			authRoutes.POST("/refresh", handler.refresh)
			authRoutes.POST("/logout", handler.logout)

			protected := authRoutes.Group("")
			protected.Use(handler.jwtRequired())
			{
				protected.GET("/me", handler.me)
				protected.POST("/change-password", handler.changePassword)
				protected.POST("/change-username", handler.changeUsername)
			}
		}

		apiKeys := v1.Group("/api-keys")
		apiKeys.Use(handler.jwtRequired())
		{
			apiKeys.GET("", handler.listAPIKeys)
			apiKeys.POST("", handler.createAPIKey)
			apiKeys.DELETE("/:id", handler.deleteAPIKey)
		}

		files := v1.Group("/files")
		files.Use(handler.authRequired())
		{
			files.POST("", handler.uploadFile)
			files.GET("", handler.listFiles)
			files.GET("/:id", handler.getFile)
			files.PATCH("/:id", handler.updateFile)
			files.DELETE("/:id", handler.deleteFile)
			files.GET("/:id/audio", handler.streamFileAudio)
		}
		transcriptions := v1.Group("/transcriptions")
		transcriptions.Use(handler.authRequired())
		{
			transcriptions.POST("", handler.bindJSONPlaceholder("transcription create"))
			transcriptions.GET("", handler.notImplemented("transcription list"))
			transcriptions.GET("/:id", handler.notImplemented("transcription get"))
			transcriptions.PATCH("/:id", handler.bindJSONPlaceholder("transcription update"))
			transcriptions.DELETE("/:id", handler.notImplemented("transcription delete"))
			transcriptions.POST("/:idAction", handler.transcriptionCommand)
			transcriptions.GET("/:id/transcript", handler.notImplemented("transcript get"))
			transcriptions.GET("/:id/audio", handler.notImplemented("transcription audio stream"))
			transcriptions.GET("/:id/events", handler.notImplemented("transcription events"))
			transcriptions.GET("/:id/logs", handler.notImplemented("transcription logs"))
			transcriptions.GET("/:id/executions", handler.notImplemented("transcription executions"))
		}

		profiles := v1.Group("/profiles")
		profiles.Use(handler.authRequired())
		{
			profiles.GET("", handler.notImplemented("profile list"))
			profiles.POST("", handler.bindJSONPlaceholder("profile create"))
			profiles.GET("/:id", handler.notImplemented("profile get"))
			profiles.PATCH("/:id", handler.bindJSONPlaceholder("profile update"))
			profiles.DELETE("/:id", handler.notImplemented("profile delete"))
			profiles.POST("/:idAction", handler.profileCommand)
		}

		settings := v1.Group("/settings")
		settings.Use(handler.authRequired())
		{
			settings.GET("", handler.notImplemented("settings get"))
			settings.PATCH("", handler.bindJSONPlaceholder("settings update"))
		}

		v1.GET("/events", handler.authRequired(), handler.notImplemented("global events"))
		v1.GET("/models/transcription", handler.authRequired(), handler.notImplemented("transcription model capabilities"))
		v1.GET("/admin/queue", handler.authRequired(), handler.notImplemented("queue stats"))
	}

	web.SetupStaticRoutes(router, handler.authService)
	router.NoRoute(func(c *gin.Context) {
		if handler.handleCommandRoute(c) {
			return
		}
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "API endpoint not found", nil)
			return
		}
		cleanPath := strings.TrimPrefix(path.Clean(c.Request.URL.Path), "/")
		if strings.Contains(cleanPath, "..") {
			c.Status(http.StatusForbidden)
			return
		}
		if cleanPath != "" && strings.Contains(path.Base(cleanPath), ".") {
			web.GetStaticHandler().ServeHTTP(c.Writer, c.Request)
			return
		}
		indexHTML, err := web.GetIndexHTML()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error loading page")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})

	return router
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Set(requestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("API panic recovered", "request_id", requestID(c), "panic", recovered)
				writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
				c.Abort()
			}
		}()
		c.Next()
	}
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowOrigin := "*"
		if cfg != nil && cfg.IsProduction() && len(cfg.AllowedOrigins) > 0 {
			allowOrigin = ""
			for _, allowed := range cfg.AllowedOrigins {
				if origin == allowed {
					allowOrigin = origin
					break
				}
			}
		} else if origin != "" {
			allowOrigin = origin
		}
		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-API-Key, X-Request-ID, Idempotency-Key")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ready(c *gin.Context) {
	if h.readinessCheck != nil {
		if err := h.readinessCheck(); err != nil {
			writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "service is not ready", nil)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready", "database": "ok"})
}

func (h *Handler) notImplemented(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		writeError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", feature+" is not implemented yet", nil)
	}
}

func (h *Handler) bindJSONPlaceholder(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && c.Request.ContentLength != 0 {
			var body map[string]any
			if err := c.ShouldBindJSON(&body); err != nil {
				writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON", nil)
				return
			}
		}
		writeError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", feature+" is not implemented yet", nil)
	}
}

func (h *Handler) registrationStatus(c *gin.Context) {
	var count int64
	if err := database.DB.Model(&models.User{}).Count(&count).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read registration status", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"registration_enabled": count == 0})
}

func (h *Handler) register(c *gin.Context) {
	var req registerRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.Username == "" || len(req.Username) < 3 || req.Password == "" || len(req.Password) < 8 || req.Password != req.ConfirmPassword {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "username and password are invalid", nil)
		return
	}

	var count int64
	if err := database.DB.Model(&models.User{}).Count(&count).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not register user", nil)
		return
	}
	if count > 0 {
		writeError(c, http.StatusConflict, "CONFLICT", "registration is already complete", nil)
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not register user", nil)
		return
	}
	user := models.User{Username: req.Username, Password: passwordHash}
	if err := database.DB.Create(&user).Error; err != nil {
		writeError(c, http.StatusConflict, "CONFLICT", "username is already in use", nil)
		return
	}
	h.writeTokenResponse(c, http.StatusOK, &user)
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "username and password are required", nil)
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid username or password", nil)
		return
	}
	if !auth.CheckPassword(req.Password, user.Password) {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid username or password", nil)
		return
	}
	h.writeTokenResponse(c, http.StatusOK, &user)
}

func (h *Handler) refresh(c *gin.Context) {
	var req refreshRequest
	if !bindJSON(c, &req) {
		return
	}
	refreshToken, err := h.findUsableRefreshToken(req.RefreshToken)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token", nil)
		return
	}

	now := time.Now()
	if err := database.DB.Model(&models.RefreshToken{}).Where("id = ?", refreshToken.ID).Update("revoked_at", &now).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not rotate refresh token", nil)
		return
	}

	var user models.User
	if err := database.DB.First(&user, refreshToken.UserID).Error; err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token", nil)
		return
	}
	h.writeTokenResponse(c, http.StatusOK, &user)
}

func (h *Handler) logout(c *gin.Context) {
	var req refreshRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.RefreshToken != "" {
		now := time.Now()
		_ = database.DB.Model(&models.RefreshToken{}).
			Where("token_hash = ? AND revoked_at IS NULL", sha256Hex(req.RefreshToken)).
			Update("revoked_at", &now).Error
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) me(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	c.JSON(http.StatusOK, userResponse(&user))
}

func (h *Handler) changePassword(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	var req changePasswordRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.NewPassword == "" || len(req.NewPassword) < 8 || req.NewPassword != req.ConfirmPassword {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "new password is invalid", nil)
		return
	}
	if !auth.CheckPassword(req.CurrentPassword, user.Password) {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "current password is invalid", nil)
		return
	}
	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not change password", nil)
		return
	}
	if err := database.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("password_hash", passwordHash).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not change password", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) changeUsername(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	var req changeUsernameRequest
	if !bindJSON(c, &req) {
		return
	}
	if len(req.NewUsername) < 3 || !auth.CheckPassword(req.Password, user.Password) {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "username change is not authorized", nil)
		return
	}
	if err := database.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("username", req.NewUsername).Error; err != nil {
		writeError(c, http.StatusConflict, "CONFLICT", "username is already in use", nil)
		return
	}
	user.Username = req.NewUsername
	c.JSON(http.StatusOK, userResponse(user))
}

func (h *Handler) listAPIKeys(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	var keys []models.APIKey
	if err := database.DB.Where("user_id = ? AND revoked_at IS NULL", userID).Order("created_at DESC").Find(&keys).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list api keys", nil)
		return
	}
	items := make([]gin.H, 0, len(keys))
	for _, key := range keys {
		description := ""
		if key.Description != nil {
			description = *key.Description
		}
		items = append(items, gin.H{
			"id":           publicAPIKeyID(key.ID),
			"name":         key.Name,
			"description":  description,
			"key_preview":  keyPreview(key.KeyPrefix),
			"is_active":    key.RevokedAt == nil,
			"last_used_at": key.LastUsed,
			"created_at":   key.CreatedAt,
			"updated_at":   key.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}

func (h *Handler) createAPIKey(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	var req createAPIKeyRequest
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required", stringPtr("name"))
		return
	}
	rawKey := "sk_" + randomHex(32)
	description := req.Description
	key := models.APIKey{
		UserID:      userID,
		Name:        strings.TrimSpace(req.Name),
		Key:         rawKey,
		KeyPrefix:   rawKey[:8],
		KeyHash:     sha256Hex(rawKey),
		Description: &description,
	}
	if err := database.DB.Create(&key).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create api key", nil)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":          publicAPIKeyID(key.ID),
		"name":        key.Name,
		"description": req.Description,
		"key":         rawKey,
		"key_preview": keyPreview(key.KeyPrefix),
		"created_at":  key.CreatedAt,
	})
}

func (h *Handler) deleteAPIKey(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	id, ok := parseAPIKeyID(c.Param("id"))
	if !ok {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "api key not found", nil)
		return
	}
	now := time.Now()
	result := database.DB.Model(&models.APIKey{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", id, userID).
		Update("revoked_at", &now)
	if result.Error != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete api key", nil)
		return
	}
	if result.RowsAffected == 0 {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "api key not found", nil)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) uploadFile(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	header, err := c.FormFile("file")
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "file is required", stringPtr("file"))
		return
	}
	source, err := header.Open()
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "file could not be read", stringPtr("file"))
		return
	}
	defer source.Close()

	mimeType := mediaType(header.Header.Get("Content-Type"), header.Filename)
	kind := fileKind(mimeType)
	if kind == "" {
		writeError(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "unsupported media type", stringPtr("file"))
		return
	}

	uploadDir := h.config.UploadDir
	if uploadDir == "" {
		uploadDir = filepath.Join(os.TempDir(), "scriberr-uploads")
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not prepare file storage", nil)
		return
	}

	jobID := randomHex(16)
	filename := safeFilename(header.Filename)
	if filename == "" {
		filename = jobID
	}
	storedName := jobID + filepath.Ext(filename)
	storagePath := filepath.Join(uploadDir, storedName)
	destination, err := os.OpenFile(storagePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not store file", nil)
		return
	}
	if _, err := io.Copy(destination, source); err != nil {
		_ = destination.Close()
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not store file", nil)
		return
	}
	if err := destination.Close(); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not store file", nil)
		return
	}

	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		title = strings.TrimSuffix(filename, filepath.Ext(filename))
	}
	job := models.TranscriptionJob{
		ID:             jobID,
		UserID:         userID,
		Title:          &title,
		Status:         models.StatusUploaded,
		AudioPath:      storagePath,
		SourceFileName: filename,
	}
	if err := database.DB.Create(&job).Error; err != nil {
		_ = os.Remove(storagePath)
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create file record", nil)
		return
	}
	c.JSON(http.StatusCreated, fileResponse(&job, mimeType, kind))
}

func (h *Handler) listFiles(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	var jobs []models.TranscriptionJob
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&jobs).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list files", nil)
		return
	}
	items := make([]gin.H, 0, len(jobs))
	for i := range jobs {
		mimeType := mediaType("", jobs[i].SourceFileName)
		items = append(items, fileResponse(&jobs[i], mimeType, fileKind(mimeType)))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}

func (h *Handler) getFile(c *gin.Context) {
	job, ok := h.fileByPublicID(c)
	if !ok {
		return
	}
	mimeType := mediaType("", job.SourceFileName)
	c.JSON(http.StatusOK, fileResponse(job, mimeType, fileKind(mimeType)))
}

func (h *Handler) updateFile(c *gin.Context) {
	job, ok := h.fileByPublicID(c)
	if !ok {
		return
	}
	var req updateFileRequest
	if !bindJSON(c, &req) {
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "title is required", stringPtr("title"))
		return
	}
	if err := database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", job.ID).Update("title", title).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update file", nil)
		return
	}
	job.Title = &title
	mimeType := mediaType("", job.SourceFileName)
	c.JSON(http.StatusOK, fileResponse(job, mimeType, fileKind(mimeType)))
}

func (h *Handler) deleteFile(c *gin.Context) {
	job, ok := h.fileByPublicID(c)
	if !ok {
		return
	}
	if err := database.DB.Delete(&models.TranscriptionJob{}, "id = ?", job.ID).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete file", nil)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) streamFileAudio(c *gin.Context) {
	job, ok := h.fileByPublicID(c)
	if !ok {
		return
	}
	file, err := os.Open(job.AudioPath)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file audio not found", nil)
		return
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file audio not found", nil)
		return
	}
	mimeType := mediaType("", job.SourceFileName)
	c.Header("Content-Type", mimeType)
	c.Header("Accept-Ranges", "bytes")
	http.ServeContent(c.Writer, c.Request, job.SourceFileName, stat.ModTime(), file)
}

func (h *Handler) fileByPublicID(c *gin.Context) (*models.TranscriptionJob, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return nil, false
	}
	id := strings.TrimPrefix(c.Param("id"), "file_")
	if id == "" || id == c.Param("id") {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file not found", nil)
		return nil, false
	}
	var job models.TranscriptionJob
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&job).Error; err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "file not found", nil)
		return nil, false
	}
	return &job, true
}

func (h *Handler) transcriptionCommand(c *gin.Context) {
	action := c.Param("idAction")
	switch {
	case strings.HasSuffix(action, ":cancel"):
		writeError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "transcription cancel is not implemented yet", nil)
	case strings.HasSuffix(action, ":retry"):
		writeError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "transcription retry is not implemented yet", nil)
	default:
		writeError(c, http.StatusNotFound, "NOT_FOUND", "API endpoint not found", nil)
	}
}

func (h *Handler) profileCommand(c *gin.Context) {
	if strings.HasSuffix(c.Param("idAction"), ":set-default") {
		writeError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "profile set default is not implemented yet", nil)
		return
	}
	writeError(c, http.StatusNotFound, "NOT_FOUND", "API endpoint not found", nil)
}

func (h *Handler) handleCommandRoute(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}
	switch c.Request.URL.Path {
	case "/api/v1/files:import-youtube":
		if !h.requireAuthForNoRoute(c) {
			return true
		}
		h.bindJSONPlaceholder("youtube import")(c)
		return true
	case "/api/v1/transcriptions:submit":
		if !h.requireAuthForNoRoute(c) {
			return true
		}
		h.notImplemented("transcription submit")(c)
		return true
	default:
		return false
	}
}

func (h *Handler) requireAuthForNoRoute(c *gin.Context) bool {
	if h.authenticateAPIKey(c) || h.authenticateJWT(c) {
		return true
	}
	writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
	return false
}

func (h *Handler) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.authenticateAPIKey(c) || h.authenticateJWT(c) {
			c.Next()
			return
		}
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		c.Abort()
	}
}

func (h *Handler) jwtRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.authenticateJWT(c) {
			c.Next()
			return
		}
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		c.Abort()
	}
}

func (h *Handler) authenticateJWT(c *gin.Context) bool {
	if h.authService == nil {
		return false
	}
	token := bearerToken(c.GetHeader("Authorization"))
	if token == "" {
		if cookie, err := c.Cookie("scriberr_access_token"); err == nil {
			token = cookie
		}
	}
	if token == "" {
		return false
	}
	claims, err := h.authService.ValidateToken(token)
	if err != nil {
		return false
	}
	c.Set("auth_type", "jwt")
	c.Set("user_id", claims.UserID)
	c.Set("username", claims.Username)
	return true
}

func (h *Handler) authenticateAPIKey(c *gin.Context) bool {
	key := strings.TrimSpace(c.GetHeader("X-API-Key"))
	if key == "" || database.DB == nil {
		return false
	}

	var apiKey models.APIKey
	if err := database.DB.Where("key_hash = ? AND revoked_at IS NULL", sha256Hex(key)).First(&apiKey).Error; err != nil {
		return false
	}
	now := time.Now()
	apiKey.LastUsed = &now
	_ = database.DB.Save(&apiKey).Error

	c.Set("auth_type", "api_key")
	c.Set("user_id", apiKey.UserID)
	c.Set("api_key_id", apiKey.ID)
	return true
}

func (h *Handler) writeTokenResponse(c *gin.Context, status int, user *models.User) {
	accessToken, err := h.authService.GenerateToken(user)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not issue access token", nil)
		return
	}
	refreshToken := "rt_" + randomHex(32)
	stored := models.RefreshToken{
		UserID:    user.ID,
		Hashed:    sha256Hex(refreshToken),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := database.DB.Create(&stored).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not issue refresh token", nil)
		return
	}
	c.JSON(status, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          userResponse(user),
	})
}

func (h *Handler) findUsableRefreshToken(raw string) (*models.RefreshToken, error) {
	if raw == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var refreshToken models.RefreshToken
	err := database.DB.
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", sha256Hex(raw), time.Now()).
		First(&refreshToken).Error
	if err != nil {
		return nil, err
	}
	return &refreshToken, nil
}

func (h *Handler) currentUser(c *gin.Context) (*models.User, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		return nil, false
	}
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, false
	}
	return &user, true
}

func currentUserID(c *gin.Context) (uint, bool) {
	value, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case uint:
		return typed, true
	case int:
		return uint(typed), typed > 0
	case float64:
		return uint(typed), typed > 0
	default:
		return 0, false
	}
}

func bearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func writeError(c *gin.Context, status int, code, message string, field *string) {
	if c.Writer.Written() {
		return
	}
	c.JSON(status, ErrorBody{Error: APIError{
		Code:      code,
		Message:   message,
		Field:     field,
		RequestID: requestID(c),
	}})
}

func requestID(c *gin.Context) string {
	if value, ok := c.Get(requestIDKey); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}

func newRequestID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "req_fallback"
	}
	return "req_" + hex.EncodeToString(b[:])
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func bindJSON(c *gin.Context, dest any) bool {
	if err := c.ShouldBindJSON(dest); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON", nil)
		return false
	}
	return true
}

func userResponse(user *models.User) gin.H {
	return gin.H{
		"id":       "user_self",
		"username": user.Username,
	}
}

func publicAPIKeyID(id uint) string {
	return fmt.Sprintf("key_%d", id)
}

func parseAPIKeyID(raw string) (uint, bool) {
	trimmed := strings.TrimPrefix(raw, "key_")
	id, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

func randomHex(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func keyPreview(prefix string) string {
	if prefix == "" {
		return "sk_..."
	}
	if len(prefix) > 4 {
		return prefix[:4] + "..." + prefix[len(prefix)-4:]
	}
	return prefix + "..."
}

func stringPtr(value string) *string {
	return &value
}

func fileResponse(job *models.TranscriptionJob, mimeType, kind string) gin.H {
	title := ""
	if job.Title != nil {
		title = *job.Title
	}
	size := int64(0)
	if stat, err := os.Stat(job.AudioPath); err == nil {
		size = stat.Size()
	}
	status := "uploaded"
	if job.Status == models.StatusUploaded {
		status = "ready"
	}
	durationSeconds := any(nil)
	if job.SourceDurationMs != nil {
		durationSeconds = float64(*job.SourceDurationMs) / 1000
	}
	return gin.H{
		"id":               "file_" + job.ID,
		"title":            title,
		"kind":             kind,
		"status":           status,
		"mime_type":        mimeType,
		"size_bytes":       size,
		"duration_seconds": durationSeconds,
		"created_at":       job.CreatedAt,
		"updated_at":       job.UpdatedAt,
	}
}

func mediaType(headerValue, filename string) string {
	cleanHeader := strings.ToLower(strings.TrimSpace(strings.Split(headerValue, ";")[0]))
	if strings.HasPrefix(cleanHeader, "audio/") || strings.HasPrefix(cleanHeader, "video/") {
		return cleanHeader
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".wav":
		return "audio/wav"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a":
		return "audio/mp4"
	case ".flac":
		return "audio/flac"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	default:
		return cleanHeader
	}
}

func fileKind(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	default:
		return ""
	}
}

func safeFilename(filename string) string {
	base := filepath.Base(strings.TrimSpace(filename))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return strings.NewReplacer("/", "_", "\\", "_", "\x00", "").Replace(base)
}
