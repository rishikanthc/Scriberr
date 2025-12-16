package tests

import (
	"encoding/json"
	"net/http"

	"scriberr/internal/api"
	"scriberr/internal/auth"
	"scriberr/internal/models"

	"github.com/stretchr/testify/assert"
)

func (suite *APIHandlerTestSuite) TestChangePassword() {
	// Setup user is done in helper (user: testuser, pass: testpassword123)

	req := api.ChangePasswordRequest{
		CurrentPassword: "testpassword123",
		NewPassword:     "newpassword456",
		ConfirmPassword: "newpassword456",
	}

	resp := suite.makeAuthenticatedRequest("POST", "/api/v1/auth/change-password", req, true)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Verify password change by trying to login with new password
	var user models.User
	suite.helper.DB.First(&user, "username = ?", "testuser")
	valid := auth.CheckPassword("newpassword456", user.Password)
	assert.True(suite.T(), valid)

	// Verify old password fails
	valid = auth.CheckPassword("testpassword123", user.Password)
	assert.False(suite.T(), valid)
}

func (suite *APIHandlerTestSuite) TestChangeUsername() {
	// Setup
	req := api.ChangeUsernameRequest{
		NewUsername: "updateduser",
		Password:    "testpassword123", // Need to use current password (might be changed if run after TestChangePassword, but ResetDB handles that)
	}
	// Note: SetupTest calls ResetDB which recreates credentials with "testpassword123"

	resp := suite.makeAuthenticatedRequest("POST", "/api/v1/auth/change-username", req, true)
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	// Verify username change
	var user models.User
	err := suite.helper.DB.First(&user, "username = ?", "updateduser").Error
	assert.NoError(suite.T(), err)

	// Verify old username gone
	var count int64
	suite.helper.DB.Model(&models.User{}).Where("username = ?", "testuser").Count(&count)
	assert.Equal(suite.T(), int64(0), count)

	// Check response message
	var respMsg map[string]string
	json.Unmarshal(resp.Body.Bytes(), &respMsg)
	assert.Equal(suite.T(), "Username changed successfully", respMsg["message"])
}
