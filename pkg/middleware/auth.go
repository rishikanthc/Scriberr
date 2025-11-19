package middleware

import (
    "net/http"
    "strings"
    "time"

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
        if key, ok := validateAPIKey(apiKey); ok {
            c.Set("auth_type", "api_key")
            c.Set("api_key", apiKey)
            if key.UserID != 0 {
                c.Set("user_id", key.UserID)
            }
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

// validateAPIKey validates an API key against the database and updates last used timestamp
func validateAPIKey(key string) (*models.APIKey, bool) {
    var apiKey models.APIKey
    result := database.DB.Where("key = ? AND is_active = ?", key, true).First(&apiKey)
    if result.Error != nil {
        return nil, false
    }

    // Update last used timestamp
    now := time.Now()
    apiKey.LastUsed = &now
    database.DB.Save(&apiKey)

    return &apiKey, true
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

    key, ok := validateAPIKey(apiKey)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
        c.Abort()
        return
    }

    c.Set("auth_type", "api_key")
    c.Set("api_key", apiKey)
    if key != nil && key.UserID != 0 {
        c.Set("user_id", key.UserID)
    }
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

// AdminOnlyMiddleware ensures the authenticated JWT user has admin privileges
func AdminOnlyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Require a user_id set by JWTOnlyMiddleware
        v, ok := c.Get("user_id")
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
            c.Abort()
            return
        }

        var uid uint
        switch id := v.(type) {
        case uint:
            uid = id
        case int:
            if id < 0 {
                c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
                c.Abort()
                return
            }
            uid = uint(id)
        case int64:
            if id < 0 {
                c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
                c.Abort()
                return
            }
            uid = uint(id)
        default:
            c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
            c.Abort()
            return
        }

        var user models.User
        if err := database.DB.Select("id, is_admin").First(&user, uid).Error; err != nil {
            c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
            c.Abort()
            return
        }
        if !user.IsAdmin {
            c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
            c.Abort()
            return
        }
        c.Next()
    }
}
