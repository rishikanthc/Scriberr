package api

import (
	"net/http"
	"strings"
	"time"

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/pkg/logger"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

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
func (h *Handler) handleCommandRoute(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}
	switch c.Request.URL.Path {
	case "/api/v1/files:import-youtube":
		if !h.requireAuthForNoRoute(c) {
			return true
		}
		h.runIdempotent(c, h.importYouTube)
		return true
	case "/api/v1/transcriptions:submit":
		if !h.requireAuthForNoRoute(c) {
			return true
		}
		h.runIdempotent(c, h.submitTranscription)
		return true
	default:
		if sessionID, ok := parseChatMessageStreamPath(c.Request.URL.Path); ok {
			if !h.requireAuthForNoRoute(c) {
				return true
			}
			h.runIdempotent(c, func(c *gin.Context) { h.streamChatMessage(c, sessionID) })
			return true
		}
		if runID, ok := parseChatRunCancelPath(c.Request.URL.Path); ok {
			if !h.requireAuthForNoRoute(c) {
				return true
			}
			h.cancelChatRun(c, runID)
			return true
		}
		if publicID, command, ok := parseTranscriptionCommandPath(c.Request.URL.Path); ok {
			if !h.requireAuthForNoRoute(c) {
				return true
			}
			h.runIdempotent(c, func(c *gin.Context) {
				switch command {
				case "stop", "cancel":
					h.cancelTranscription(c, publicID)
				case "retry":
					h.retryTranscription(c, publicID)
				default:
					writeError(c, http.StatusNotFound, "NOT_FOUND", "API endpoint not found", nil)
				}
			})
			return true
		}
		return false
	}
}

func parseChatMessageStreamPath(requestPath string) (string, bool) {
	trimmed := strings.TrimPrefix(requestPath, "/api/v1/chat/sessions/")
	if trimmed == requestPath || trimmed == "" {
		return "", false
	}
	sessionID, suffix, ok := strings.Cut(trimmed, "/messages:stream")
	if !ok || suffix != "" || sessionID == "" {
		return "", false
	}
	return sessionID, true
}

func parseChatRunCancelPath(requestPath string) (string, bool) {
	trimmed := strings.TrimPrefix(requestPath, "/api/v1/chat/runs/")
	if trimmed == requestPath || trimmed == "" {
		return "", false
	}
	runID, command, ok := strings.Cut(trimmed, ":")
	if !ok || command != "cancel" || runID == "" {
		return "", false
	}
	return runID, true
}

func parseTranscriptionCommandPath(requestPath string) (string, string, bool) {
	trimmed := strings.TrimPrefix(requestPath, "/api/v1/transcriptions/")
	if trimmed == requestPath || trimmed == "" || strings.Contains(trimmed, "/") {
		return "", "", false
	}
	publicID, command, ok := strings.Cut(trimmed, ":")
	if !ok || publicID == "" || command == "" {
		return "", "", false
	}
	switch command {
	case "stop", "cancel", "retry":
		return publicID, command, true
	default:
		return "", "", false
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
