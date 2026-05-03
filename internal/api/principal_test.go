package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCurrentPrincipalExtractsJWTAndAPIKeyMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("auth_type", "jwt")
	c.Set("user_id", uint(7))
	c.Set("username", "owner")
	c.Set("role", "admin")

	principal, ok := handler.currentPrincipal(c)
	require.True(t, ok)
	require.Equal(t, uint(7), principal.UserID)
	require.Equal(t, "owner", principal.Username)
	require.Equal(t, "admin", principal.Role)
	require.Equal(t, "jwt", principal.AuthType)
	require.Nil(t, principal.APIKeyID)

	c.Set("auth_type", "api_key")
	c.Set("api_key_id", uint(12))
	c.Set("role", "")

	principal, ok = handler.currentPrincipal(c)
	require.True(t, ok)
	require.Equal(t, "api_key", principal.AuthType)
	require.Empty(t, principal.Role)
	require.NotNil(t, principal.APIKeyID)
	require.Equal(t, uint(12), *principal.APIKeyID)
}
