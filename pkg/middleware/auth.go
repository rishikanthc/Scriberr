package middleware

import (
	"net/http"
	"strings"

	"scriberr/internal/auth"
	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware handles both API key and JWT authentication
func AuthMiddleware(authService *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key first
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			if validateAPIKey(apiKey) {
				c.Set("auth_type", "api_key")
				c.Set("api_key", apiKey)
				c.Next()
				return
			}
		}

		// Check for JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("auth_type", "jwt")
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// validateAPIKey validates an API key against the database
func validateAPIKey(key string) bool {
	var apiKey models.APIKey
	result := database.DB.Where("key = ? AND is_active = ?", key, true).First(&apiKey)
	return result.Error == nil
}

// APIKeyOnlyMiddleware only allows API key authentication
func APIKeyOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		if !validateAPIKey(apiKey) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		c.Set("auth_type", "api_key")
		c.Set("api_key", apiKey)
		c.Next()
	}
}

// JWTOnlyMiddleware only allows JWT authentication
func JWTOnlyMiddleware(authService *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("auth_type", "jwt")
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}