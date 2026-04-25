package api

import (
	"net/http"
	"strings"
	"time"

	"scriberr/internal/auth"
	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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
