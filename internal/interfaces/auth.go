package interfaces

import (
	"scriberr/internal/auth"
	"scriberr/internal/models"
)

// AuthServiceInterface defines the contract for authentication services.
// This allows handlers to depend on an interface rather than a concrete type.
type AuthServiceInterface interface {
	// GenerateToken generates a JWT token for a user (24h expiry)
	GenerateToken(user *models.User) (string, error)

	// GenerateLongLivedToken generates a JWT token for a user (1 year expiry)
	GenerateLongLivedToken(user *models.User) (string, error)

	// ValidateToken validates a JWT token and returns claims
	ValidateToken(tokenString string) (*auth.Claims, error)
}
