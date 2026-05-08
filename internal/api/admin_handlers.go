package api

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	admindomain "scriberr/internal/admin"
	"scriberr/internal/models"
	transcriptiondomain "scriberr/internal/transcription"
	"scriberr/internal/transcription/asrcontract"
	"scriberr/internal/transcription/scheduler"
	"scriberr/internal/transcription/worker"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listTranscriptionModels(c *gin.Context) {
	if h.modelRegistry != nil {
		models, err := h.modelRegistry.Models(c.Request.Context())
		if err != nil {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list transcription models", nil)
			return
		}
		transcriptionModels := make([]asrcontract.ModelCard, 0, len(models))
		for _, model := range models {
			if model.Supports(asrcontract.CapabilityTranscription) {
				transcriptionModels = append(transcriptionModels, model)
			}
		}
		c.JSON(http.StatusOK, gin.H{"items": sanitizeModelCards(transcriptionModels)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": []gin.H{}})
}
func (h *Handler) queueStats(c *gin.Context) {
	if h.queueService != nil {
		stats, err := h.queueService.AdminStats(c.Request.Context())
		if err != nil {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read queue stats", nil)
			return
		}
		c.JSON(http.StatusOK, adminQueueStatsResponse(stats))
		return
	}
	stats, err := h.transcriptions.AdminStats(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read queue stats", nil)
		return
	}
	c.JSON(http.StatusOK, transcriptionAdminQueueStatsResponse(stats))
}

func (h *Handler) userQueueStats(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.queueService != nil {
		stats, err := h.queueService.Stats(c.Request.Context(), userID)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read queue stats", nil)
			return
		}
		c.JSON(http.StatusOK, queueStatsResponse(stats))
		return
	}
	stats, err := h.transcriptions.Stats(c.Request.Context(), userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read queue stats", nil)
		return
	}
	c.JSON(http.StatusOK, transcriptionQueueStatsResponse(stats))
}

func queueStatsResponse(stats worker.QueueStats) gin.H {
	return gin.H{
		"queued":     stats.Queued,
		"processing": stats.Processing,
		"completed":  stats.Completed,
		"failed":     stats.Failed,
		"stopped":    stats.Canceled,
		"canceled":   stats.Canceled,
		"running":    stats.Running,
	}
}

func adminQueueStatsResponse(stats worker.AdminQueueStats) gin.H {
	body := queueStatsResponse(stats.QueueStats)
	items := make([]gin.H, 0, len(stats.ByUser))
	for _, item := range stats.ByUser {
		items = append(items, gin.H{
			"user_id":    "user_" + strconv.FormatUint(uint64(item.UserID), 10),
			"username":   item.Username,
			"queued":     item.Queued,
			"processing": item.Processing,
			"completed":  item.Completed,
			"failed":     item.Failed,
			"stopped":    item.Canceled,
			"canceled":   item.Canceled,
			"running":    item.Running,
		})
	}
	body["by_user"] = items
	return body
}

func transcriptionQueueStatsResponse(stats transcriptiondomain.Stats) gin.H {
	return gin.H{
		"queued":     stats.Queued,
		"processing": stats.Processing,
		"completed":  stats.Completed,
		"failed":     stats.Failed,
		"stopped":    stats.Canceled,
		"canceled":   stats.Canceled,
		"running":    stats.Running,
	}
}

func transcriptionAdminQueueStatsResponse(stats transcriptiondomain.AdminStats) gin.H {
	body := transcriptionQueueStatsResponse(stats.Stats)
	items := make([]gin.H, 0, len(stats.ByUser))
	for _, item := range stats.ByUser {
		items = append(items, gin.H{
			"user_id":    "user_" + strconv.FormatUint(uint64(item.UserID), 10),
			"username":   item.Username,
			"queued":     item.Queued,
			"processing": item.Processing,
			"completed":  item.Completed,
			"failed":     item.Failed,
			"stopped":    item.Canceled,
			"canceled":   item.Canceled,
			"running":    item.Running,
		})
	}
	body["by_user"] = items
	return body
}

func (h *Handler) getAdminQueueScheduler(c *gin.Context) {
	actorID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.admin == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "admin service is not configured", nil)
		return
	}
	config, err := h.admin.GetSchedulerConfig(c.Request.Context(), actorID)
	if !writeAdminSchedulerError(c, err, "could not load scheduler config") {
		return
	}
	c.JSON(http.StatusOK, adminSchedulerResponse(config))
}

func (h *Handler) updateAdminQueueScheduler(c *gin.Context) {
	actorID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.admin == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "admin service is not configured", nil)
		return
	}
	var req adminSchedulerRequest
	if !bindJSON(c, &req) {
		return
	}
	config, err := h.admin.UpdateSchedulerConfig(c.Request.Context(), actorID, scheduler.Config{
		Policy:               scheduler.Policy(req.Policy),
		MaxConcurrentPerUser: req.MaxConcurrentPerUser,
	})
	if !writeAdminSchedulerError(c, err, "could not update scheduler config") {
		return
	}
	c.JSON(http.StatusOK, adminSchedulerResponse(config))
}

func (h *Handler) listAdminUsers(c *gin.Context) {
	actorID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.admin == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "admin service is not configured", nil)
		return
	}
	users, total, err := h.admin.ListUsers(c.Request.Context(), actorID, 0, 100)
	if !writeAdminUserError(c, err, "could not list users") {
		return
	}
	items := make([]gin.H, 0, len(users))
	for i := range users {
		items = append(items, adminUserResponse(&users[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "next_cursor": nil})
}

func (h *Handler) createAdminUser(c *gin.Context) {
	actorID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return
	}
	if h.admin == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "admin service is not configured", nil)
		return
	}
	var req adminCreateUserRequest
	if !bindJSON(c, &req) {
		return
	}
	user, err := h.admin.CreateUser(c.Request.Context(), actorID, admindomain.CreateUserCommand{
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Role:        req.Role,
		Password:    req.Password,
	})
	if !writeAdminUserError(c, err, "could not create user") {
		return
	}
	c.JSON(http.StatusCreated, adminUserResponse(user))
}

func (h *Handler) getAdminUser(c *gin.Context) {
	actorID, targetID, ok := h.adminUserIdentity(c)
	if !ok {
		return
	}
	if h.admin == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "admin service is not configured", nil)
		return
	}
	user, err := h.admin.GetUser(c.Request.Context(), actorID, targetID)
	if !writeAdminUserError(c, err, "could not load user") {
		return
	}
	c.JSON(http.StatusOK, adminUserResponse(user))
}

func (h *Handler) updateAdminUser(c *gin.Context) {
	actorID, targetID, ok := h.adminUserIdentity(c)
	if !ok {
		return
	}
	if h.admin == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "admin service is not configured", nil)
		return
	}
	var req adminUpdateUserRequest
	if !bindJSON(c, &req) {
		return
	}
	user, err := h.admin.UpdateUser(c.Request.Context(), actorID, targetID, admindomain.UpdateUserCommand{
		Email:            req.Email,
		DisplayName:      req.DisplayName,
		Role:             req.Role,
		Status:           req.Status,
		ClearEmail:       req.Email != nil && *req.Email == "",
		ClearDisplayName: req.DisplayName != nil && *req.DisplayName == "",
	})
	if !writeAdminUserError(c, err, "could not update user") {
		return
	}
	c.JSON(http.StatusOK, adminUserResponse(user))
}

func (h *Handler) resetAdminUserPassword(c *gin.Context) {
	actorID, targetID, ok := h.adminUserIdentity(c)
	if !ok {
		return
	}
	if h.admin == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "admin service is not configured", nil)
		return
	}
	var req adminResetPasswordRequest
	if !bindJSON(c, &req) {
		return
	}
	err := h.admin.ResetPassword(c.Request.Context(), actorID, targetID, req.Password)
	if !writeAdminUserError(c, err, "could not reset password") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) disableAdminUser(c *gin.Context) {
	actorID, targetID, ok := h.adminUserIdentity(c)
	if !ok {
		return
	}
	if h.admin == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "admin service is not configured", nil)
		return
	}
	user, err := h.admin.DisableUser(c.Request.Context(), actorID, targetID)
	if !writeAdminUserError(c, err, "could not disable user") {
		return
	}
	c.JSON(http.StatusOK, adminUserResponse(user))
}

func (h *Handler) enableAdminUser(c *gin.Context) {
	actorID, targetID, ok := h.adminUserIdentity(c)
	if !ok {
		return
	}
	if h.admin == nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "admin service is not configured", nil)
		return
	}
	user, err := h.admin.EnableUser(c.Request.Context(), actorID, targetID)
	if !writeAdminUserError(c, err, "could not enable user") {
		return
	}
	c.JSON(http.StatusOK, adminUserResponse(user))
}

func (h *Handler) adminUserIdentity(c *gin.Context) (uint, uint, bool) {
	actorID, ok := currentUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication", nil)
		return 0, 0, false
	}
	targetID, ok := parseAdminUserID(c.Param("user_id"))
	if !ok {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "user not found", nil)
		return 0, 0, false
	}
	return actorID, targetID, true
}

func parseAdminUserID(raw string) (uint, bool) {
	trimmed := regexp.MustCompile(`^user_`).ReplaceAllString(raw, "")
	id, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

func adminUserResponse(user *models.User) gin.H {
	email := any(nil)
	if user.Email != nil {
		email = *user.Email
	}
	displayName := any(nil)
	if user.DisplayName != nil {
		displayName = *user.DisplayName
	}
	return gin.H{
		"id":                  "user_" + strconv.FormatUint(uint64(user.ID), 10),
		"username":            user.Username,
		"email":               email,
		"display_name":        displayName,
		"role":                user.Role,
		"status":              user.Status,
		"last_login_at":       user.LastLoginAt,
		"password_changed_at": user.PasswordChangedAt,
		"created_at":          user.CreatedAt,
		"updated_at":          user.UpdatedAt,
	}
}

func adminSchedulerResponse(config scheduler.Config) gin.H {
	return gin.H{
		"policy":                  string(config.Policy),
		"max_concurrent_per_user": config.MaxConcurrentPerUser,
	}
}

func writeAdminSchedulerError(c *gin.Context, err error, fallback string) bool {
	if err == nil {
		return true
	}
	switch {
	case errors.Is(err, admindomain.ErrForbidden):
		writeError(c, http.StatusForbidden, "FORBIDDEN", "admin access is required", nil)
	case errors.Is(err, admindomain.ErrInvalidScheduler):
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "scheduler config is invalid", stringPtr("policy"))
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", fallback, nil)
	}
	return false
}

func writeAdminUserError(c *gin.Context, err error, fallback string) bool {
	if err == nil {
		return true
	}
	switch {
	case errors.Is(err, admindomain.ErrForbidden):
		writeError(c, http.StatusForbidden, "FORBIDDEN", "admin access is required", nil)
	case errors.Is(err, admindomain.ErrUserNotFound):
		writeError(c, http.StatusNotFound, "NOT_FOUND", "user not found", nil)
	case errors.Is(err, admindomain.ErrUsernameInUse):
		writeError(c, http.StatusConflict, "CONFLICT", "username is already in use", stringPtr("username"))
	case errors.Is(err, admindomain.ErrInvalidUser):
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "user fields are invalid", nil)
	case errors.Is(err, admindomain.ErrLastActiveAdmin):
		writeError(c, http.StatusConflict, "CONFLICT", "last active admin cannot be disabled or demoted", nil)
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", fallback, nil)
	}
	return false
}

func (h *Handler) getTranscriptionExecutions(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	executions, err := h.transcriptions.ListExecutions(c.Request.Context(), job.UserID, job.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list transcription executions", nil)
		return
	}
	items := make([]gin.H, 0, len(executions))
	for i := range executions {
		items = append(items, executionResponse(&executions[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "next_cursor": nil})
}

func (h *Handler) getTranscriptionLogs(c *gin.Context) {
	job, ok := h.transcriptionByPublicID(c, c.Param("id"))
	if !ok {
		return
	}
	logText, err := h.transcriptions.Logs(c.Request.Context(), job.UserID, job.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read transcription logs", nil)
		return
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(sanitizePublicText(logText)))
}

func executionResponse(execution *models.TranscriptionJobExecution) gin.H {
	errorValue := any(nil)
	if execution.ErrorMessage != nil && *execution.ErrorMessage != "" {
		errorValue = sanitizePublicText(*execution.ErrorMessage)
	}
	return gin.H{
		"id":                     "exec_" + execution.ID,
		"transcription_id":       "tr_" + execution.TranscriptionJobID,
		"status":                 string(execution.Status),
		"provider":               execution.Provider,
		"model":                  execution.ModelName,
		"started_at":             execution.StartedAt,
		"completed_at":           execution.CompletedAt,
		"failed_at":              execution.FailedAt,
		"processing_duration_ms": executionDurationMS(execution),
		"error":                  errorValue,
	}
}

func executionDurationMS(execution *models.TranscriptionJobExecution) any {
	var end time.Time
	switch {
	case execution.CompletedAt != nil:
		end = *execution.CompletedAt
	case execution.FailedAt != nil:
		end = *execution.FailedAt
	default:
		return nil
	}
	if execution.StartedAt.IsZero() || end.Before(execution.StartedAt) {
		return nil
	}
	return end.Sub(execution.StartedAt).Milliseconds()
}

var (
	publicAbsolutePathPattern = regexp.MustCompile(`(?:[A-Za-z]:\\|/)[^\s:;,'")]+`)
	publicTokenPattern        = regexp.MustCompile(`(?i)\b([A-Za-z0-9_]*(?:token|api_key|apikey)[A-Za-z0-9_]*)=[^\s]+`)
)

func sanitizePublicText(value string) string {
	value = publicAbsolutePathPattern.ReplaceAllString(value, "[redacted-path]")
	return publicTokenPattern.ReplaceAllString(value, "$1=[redacted]")
}
