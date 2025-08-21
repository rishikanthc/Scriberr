package tests

import (
	"testing"

	"scriberr/internal/auth"
	"scriberr/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"
	
	hash, err := auth.HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestCheckPassword(t *testing.T) {
	password := "testpassword123"
	hash, _ := auth.HashPassword(password)
	
	// Test correct password
	assert.True(t, auth.CheckPassword(password, hash))
	
	// Test incorrect password
	assert.False(t, auth.CheckPassword("wrongpassword", hash))
}

func TestAuthService_GenerateToken(t *testing.T) {
	authService := auth.NewAuthService("test-secret")
	
	user := &models.User{
		ID:       1,
		Username: "testuser",
	}
	
	token, err := authService.GenerateToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestAuthService_ValidateToken(t *testing.T) {
	authService := auth.NewAuthService("test-secret")
	
	user := &models.User{
		ID:       1,
		Username: "testuser",
	}
	
	// Generate a token
	token, err := authService.GenerateToken(user)
	assert.NoError(t, err)
	
	// Validate the token
	claims, err := authService.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
}

func TestAuthService_ValidateInvalidToken(t *testing.T) {
	authService := auth.NewAuthService("test-secret")
	
	// Test invalid token
	claims, err := authService.ValidateToken("invalid-token")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestAuthService_ValidateTokenWithWrongSecret(t *testing.T) {
	authService1 := auth.NewAuthService("secret1")
	authService2 := auth.NewAuthService("secret2")
	
	user := &models.User{
		ID:       1,
		Username: "testuser",
	}
	
	// Generate token with first service
	token, err := authService1.GenerateToken(user)
	assert.NoError(t, err)
	
	// Try to validate with second service (different secret)
	claims, err := authService2.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}