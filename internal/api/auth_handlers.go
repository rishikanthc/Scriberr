package api

import (
	"errors"
	"net/http"
	"strings"

	"scriberr/internal/account"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) registrationStatus(c *gin.Context) {
	enabled, err := h.account.RegistrationEnabled(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read registration status", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"registration_enabled": enabled})
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
	response, err := h.account.Register(c.Request.Context(), req.Username, req.Password)
	if errors.Is(err, account.ErrRegistrationClosed) {
		writeError(c, http.StatusConflict, "CONFLICT", "registration is already complete", nil)
		return
	}
	if errors.Is(err, account.ErrUsernameInUse) {
		writeError(c, http.StatusConflict, "CONFLICT", "username is already in use", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not register user", nil)
		return
	}
	writeTokenResponse(c, http.StatusOK, response)
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
	response, err := h.account.Login(c.Request.Context(), req.Username, req.Password)
	if errors.Is(err, account.ErrInvalidCredentials) {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid username or password", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not login", nil)
		return
	}
	writeTokenResponse(c, http.StatusOK, response)
}
func (h *Handler) refresh(c *gin.Context) {
	var req refreshRequest
	if !bindJSON(c, &req) {
		return
	}
	response, err := h.account.Refresh(c.Request.Context(), req.RefreshToken)
	if errors.Is(err, account.ErrInvalidRefreshToken) {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not rotate refresh token", nil)
		return
	}
	writeTokenResponse(c, http.StatusOK, response)
}
func (h *Handler) logout(c *gin.Context) {
	var req refreshRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.account.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not logout", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (h *Handler) me(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	user, err := h.account.GetUser(c.Request.Context(), userID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	c.JSON(http.StatusOK, userResponse(user))
}
func (h *Handler) changePassword(c *gin.Context) {
	userID, ok := currentUserID(c)
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
	err := h.account.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword)
	if errors.Is(err, account.ErrInvalidCurrentPassword) {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "current password is invalid", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not change password", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (h *Handler) changeUsername(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token", nil)
		return
	}
	var req changeUsernameRequest
	if !bindJSON(c, &req) {
		return
	}
	if len(req.NewUsername) < 3 {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "username change is not authorized", nil)
		return
	}
	user, err := h.account.ChangeUsername(c.Request.Context(), userID, req.NewUsername, req.Password)
	if errors.Is(err, account.ErrInvalidCurrentPassword) {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "username change is not authorized", nil)
		return
	}
	if errors.Is(err, account.ErrUsernameInUse) {
		writeError(c, http.StatusConflict, "CONFLICT", "username is already in use", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not change username", nil)
		return
	}
	c.JSON(http.StatusOK, userResponse(user))
}
func writeTokenResponse(c *gin.Context, status int, response *account.TokenResponse) {
	c.JSON(status, gin.H{
		"access_token":  response.AccessToken,
		"refresh_token": response.RefreshToken,
		"user":          userResponse(response.User),
	})
}
func (h *Handler) currentUser(c *gin.Context) (*models.User, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		return nil, false
	}
	user, err := h.account.GetUser(c.Request.Context(), userID)
	if err != nil {
		return nil, false
	}
	return user, true
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
