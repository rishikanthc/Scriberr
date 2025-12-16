package tests

import (
	"strings"
	"testing"
	"time"

	"scriberr/internal/auth"
	"scriberr/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AuthServiceTestSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *AuthServiceTestSuite) SetupSuite() {
	suite.helper = NewTestHelper(suite.T(), "auth_test.db")
}

func (suite *AuthServiceTestSuite) TearDownSuite() {
	suite.helper.Cleanup()
}

func (suite *AuthServiceTestSuite) SetupTest() {
	suite.helper.ResetDB(suite.T())
}

// Test JWT token generation
func (suite *AuthServiceTestSuite) TestGenerateToken() {
	user := &models.User{
		ID:       1,
		Username: "testuser",
	}

	token, err := suite.helper.AuthService.GenerateToken(user)

	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), token)
	assert.True(suite.T(), len(token) > 50, "JWT token should be reasonably long")
	assert.Equal(suite.T(), 3, len(strings.Split(token, ".")), "JWT should have 3 parts")
}

// Test JWT token validation with valid token
func (suite *AuthServiceTestSuite) TestValidateTokenValid() {
	user := &models.User{
		ID:       1,
		Username: "testuser",
	}

	token, err := suite.helper.AuthService.GenerateToken(user)
	assert.NoError(suite.T(), err)

	claims, err := suite.helper.AuthService.ValidateToken(token)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), claims)
	assert.Equal(suite.T(), user.ID, claims.UserID)
	assert.Equal(suite.T(), user.Username, claims.Username)
	assert.True(suite.T(), claims.ExpiresAt.After(time.Now()))
}

// Test JWT token validation with invalid token
func (suite *AuthServiceTestSuite) TestValidateTokenInvalid() {
	invalidTokens := []string{
		"invalid.token.here",
		"",
		"not-a-jwt-token",
		"header.payload", // Missing signature
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.invalid-signature",
	}

	for _, invalidToken := range invalidTokens {
		claims, err := suite.helper.AuthService.ValidateToken(invalidToken)

		assert.Error(suite.T(), err, "Token should be invalid: %s", invalidToken)
		assert.Nil(suite.T(), claims)
	}
}

// Test JWT token validation with expired token
func (suite *AuthServiceTestSuite) TestValidateTokenExpired() {
	// Create a custom auth service with short-lived tokens for testing
	authService := auth.NewAuthService("test-secret")

	// Manually create an expired token
	claims := &auth.Claims{
		UserID:   1,
		Username: "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)), // Issued 2 hours ago
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	assert.NoError(suite.T(), err)

	// Try to validate the expired token
	validClaims, err := authService.ValidateToken(tokenString)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), validClaims)
	assert.Contains(suite.T(), err.Error(), "expired")
}

// Test JWT token validation with wrong secret
func (suite *AuthServiceTestSuite) TestValidateTokenWrongSecret() {
	// Generate token with one secret
	authService1 := auth.NewAuthService("secret1")
	user := &models.User{ID: 1, Username: "testuser"}

	token, err := authService1.GenerateToken(user)
	assert.NoError(suite.T(), err)

	// Try to validate with different secret
	authService2 := auth.NewAuthService("secret2")
	claims, err := authService2.ValidateToken(token)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), claims)
}

// Test password hashing
func (suite *AuthServiceTestSuite) TestHashPassword() {
	passwords := []string{
		"simplepassword",
		"Complex123!@#",
		"",
		"very-long-password-with-many-characters-and-symbols-!@#$%^&*()",
		"unicode-パスワード-🔒",
	}

	for _, password := range passwords {
		hash, err := auth.HashPassword(password)

		assert.NoError(suite.T(), err, "Password hashing should succeed for: %s", password)
		assert.NotEmpty(suite.T(), hash)
		assert.NotEqual(suite.T(), password, hash, "Hash should not equal original password")
		assert.True(suite.T(), len(hash) > 50, "Hash should be reasonably long")
		assert.True(suite.T(), strings.HasPrefix(hash, "$2a$"), "Should use bcrypt format")
	}
}

// Test password verification
func (suite *AuthServiceTestSuite) TestCheckPassword() {
	testCases := []struct {
		password string
		correct  bool
	}{
		{"correctpassword", true},
		{"wrongpassword", false},
		{"", false},
		{"CORRECTPASSWORD", false},  // Case sensitive
		{"correct password", false}, // Different password
	}

	originalPassword := "correctpassword"
	hash, err := auth.HashPassword(originalPassword)
	assert.NoError(suite.T(), err)

	for _, tc := range testCases {
		result := auth.CheckPassword(tc.password, hash)
		assert.Equal(suite.T(), tc.correct, result,
			"Password check failed for password: %s (expected %v)", tc.password, tc.correct)
	}
}

// Test password verification with invalid hash
func (suite *AuthServiceTestSuite) TestCheckPasswordInvalidHash() {
	invalidHashes := []string{
		"invalid-hash",
		"",
		"not-bcrypt-hash",
		"$2a$10$invalid", // Incomplete bcrypt hash
	}

	for _, invalidHash := range invalidHashes {
		result := auth.CheckPassword("anypassword", invalidHash)
		assert.False(suite.T(), result, "Should return false for invalid hash: %s", invalidHash)
	}
}

// Test hash consistency (same password should produce different hashes due to salt)
func (suite *AuthServiceTestSuite) TestHashConsistency() {
	password := "testpassword123"

	hash1, err1 := auth.HashPassword(password)
	hash2, err2 := auth.HashPassword(password)

	assert.NoError(suite.T(), err1)
	assert.NoError(suite.T(), err2)
	assert.NotEqual(suite.T(), hash1, hash2, "Same password should produce different hashes due to salt")

	// But both hashes should verify the original password
	assert.True(suite.T(), auth.CheckPassword(password, hash1))
	assert.True(suite.T(), auth.CheckPassword(password, hash2))
}

// Test token generation with different users
func (suite *AuthServiceTestSuite) TestTokenGenerationDifferentUsers() {
	users := []*models.User{
		{ID: 1, Username: "user1"},
		{ID: 2, Username: "user2"},
		{ID: 999, Username: "admin"},
	}

	tokens := make([]string, len(users))

	for i, user := range users {
		token, err := suite.helper.AuthService.GenerateToken(user)
		assert.NoError(suite.T(), err)
		tokens[i] = token

		// Validate each token
		claims, err := suite.helper.AuthService.ValidateToken(token)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), user.ID, claims.UserID)
		assert.Equal(suite.T(), user.Username, claims.Username)
	}

	// All tokens should be different
	for i := 0; i < len(tokens); i++ {
		for j := i + 1; j < len(tokens); j++ {
			assert.NotEqual(suite.T(), tokens[i], tokens[j],
				"Tokens for different users should be different")
		}
	}
}

// Test token expiry time
func (suite *AuthServiceTestSuite) TestTokenExpiryTime() {
	user := &models.User{ID: 1, Username: "testuser"}

	beforeGeneration := time.Now()
	token, err := suite.helper.AuthService.GenerateToken(user)
	afterGeneration := time.Now()

	assert.NoError(suite.T(), err)

	claims, err := suite.helper.AuthService.ValidateToken(token)
	assert.NoError(suite.T(), err)

	// Token should expire 24 hours from generation
	expectedExpiry := beforeGeneration.Add(24 * time.Hour)
	actualExpiry := claims.ExpiresAt.Time

	// Allow some tolerance for processing time
	assert.True(suite.T(), actualExpiry.After(expectedExpiry.Add(-1*time.Minute)))
	assert.True(suite.T(), actualExpiry.Before(afterGeneration.Add(24*time.Hour).Add(1*time.Minute)))

	// Issue time should be around now
	assert.True(suite.T(), claims.IssuedAt.Time.After(beforeGeneration.Add(-1*time.Minute)))
	assert.True(suite.T(), claims.IssuedAt.Time.Before(afterGeneration.Add(1*time.Minute)))
}

// Test long-lived token generation
func (suite *AuthServiceTestSuite) TestGenerateLongLivedToken() {
	user := &models.User{ID: 1, Username: "testuser"}

	beforeGeneration := time.Now()
	token, err := suite.helper.AuthService.GenerateLongLivedToken(user)
	afterGeneration := time.Now()

	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), token)

	claims, err := suite.helper.AuthService.ValidateToken(token)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, claims.UserID)

	// Token should expire 1 year from generation
	expectedExpiry := beforeGeneration.Add(365 * 24 * time.Hour)
	actualExpiry := claims.ExpiresAt.Time

	// Allow some tolerance
	assert.True(suite.T(), actualExpiry.After(expectedExpiry.Add(-1*time.Minute)))
	assert.True(suite.T(), actualExpiry.Before(afterGeneration.Add(365*24*time.Hour).Add(1*time.Minute)))
}

// Test new auth service creation
func (suite *AuthServiceTestSuite) TestNewAuthService() {
	secrets := []string{
		"simple-secret",
		"complex-secret-with-special-chars!@#$%^&*()",
		"",
		"very-long-secret-key-for-testing-purposes-with-many-characters",
	}

	for _, secret := range secrets {
		authService := auth.NewAuthService(secret)
		assert.NotNil(suite.T(), authService)

		// Test that the service works with a user
		user := &models.User{ID: 1, Username: "testuser"}
		token, err := authService.GenerateToken(user)
		assert.NoError(suite.T(), err)
		assert.NotEmpty(suite.T(), token)

		claims, err := authService.ValidateToken(token)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), user.ID, claims.UserID)
	}
}

func TestAuthServiceTestSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceTestSuite))
}
