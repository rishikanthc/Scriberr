package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"scriberr/internal/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

// UserRepository handles user-specific database operations
type UserRepository interface {
	Repository[models.User]
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	Count(ctx context.Context) (int64, error)
	CountWithAutoTranscription(ctx context.Context) (int64, error)
}

type userRepository struct {
	*BaseRepository[models.User]
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		BaseRepository: NewBaseRepository[models.User](db),
	}
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error
	return count, err
}

func (r *userRepository) CountWithAutoTranscription(ctx context.Context) (int64, error) {
	var users []models.User
	if err := r.db.WithContext(ctx).Find(&users).Error; err != nil {
		return 0, err
	}
	var count int64
	for _, user := range users {
		if user.AutoTranscriptionEnabled {
			count++
		}
	}
	return count, nil
}

// JobRepository handles transcription job operations
type JobRepository interface {
	Repository[models.TranscriptionJob]
	FindWithAssociations(ctx context.Context, id string) (*models.TranscriptionJob, error)
	FindLatestCompletedExecution(ctx context.Context, jobID string) (*models.TranscriptionJobExecution, error)
	ListWithParams(ctx context.Context, offset, limit int, sortBy, sortOrder, searchQuery string, updatedAfter *time.Time) ([]models.TranscriptionJob, int64, error)
	ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.TranscriptionJob, int64, error)
	EnqueueTranscription(ctx context.Context, jobID string, now time.Time) error
	ClaimNextTranscription(ctx context.Context, workerID string, leaseUntil time.Time) (*models.TranscriptionJob, error)
	RenewClaim(ctx context.Context, jobID, workerID string, leaseUntil time.Time) error
	RecoverOrphanedProcessing(ctx context.Context, now time.Time) (int64, error)
	UpdateProgress(ctx context.Context, jobID string, progress float64, stage string) error
	CompleteTranscription(ctx context.Context, jobID string, transcriptJSON string, outputPath *string, completedAt time.Time) error
	FailTranscription(ctx context.Context, jobID string, message string, failedAt time.Time) error
	CancelTranscription(ctx context.Context, jobID string, canceledAt time.Time) error
	ListExecutions(ctx context.Context, jobID string) ([]models.TranscriptionJobExecution, error)
	CountStatusesByUser(ctx context.Context, userID uint) (map[models.JobStatus]int64, error)
	UpdateTranscript(ctx context.Context, jobID string, transcript string) error
	CreateExecution(ctx context.Context, execution *models.TranscriptionJobExecution) error
	UpdateExecution(ctx context.Context, execution *models.TranscriptionJobExecution) error
	DeleteExecutionsByJobID(ctx context.Context, jobID string) error
	UpdateStatus(ctx context.Context, jobID string, status models.JobStatus) error
	UpdateError(ctx context.Context, jobID string, errorMsg string) error
	FindByStatus(ctx context.Context, status models.JobStatus) ([]models.TranscriptionJob, error)
	CountByStatus(ctx context.Context, status models.JobStatus) (int64, error)
	UpdateSummary(ctx context.Context, jobID string, summary string) error
}

type jobRepository struct {
	*BaseRepository[models.TranscriptionJob]
}

func NewJobRepository(db *gorm.DB) JobRepository {
	return &jobRepository{
		BaseRepository: NewBaseRepository[models.TranscriptionJob](db),
	}
}

func (r *jobRepository) FindWithAssociations(ctx context.Context, id string) (*models.TranscriptionJob, error) {
	var job models.TranscriptionJob
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *jobRepository) ListWithParams(ctx context.Context, offset, limit int, sortBy, sortOrder, searchQuery string, updatedAfter *time.Time) ([]models.TranscriptionJob, int64, error) {
	var jobs []models.TranscriptionJob
	var count int64

	db := r.db.WithContext(ctx).Model(&models.TranscriptionJob{})

	// Handle delta sync if updatedAfter provided
	if updatedAfter != nil {
		db = db.Unscoped().Where("updated_at > ?", *updatedAfter)
	}

	// Apply search filter
	if searchQuery != "" {
		search := "%" + searchQuery + "%"
		db = db.Where("title LIKE ? OR source_file_path LIKE ? OR source_file_name LIKE ?", search, search, search)
	}

	// Count total matching records
	if err := db.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	if sortBy != "" {
		if sortOrder == "" {
			sortOrder = "desc"
		}
		db = db.Order(sortBy + " " + sortOrder)
	} else {
		// Default sort
		db = db.Order("created_at desc")
	}

	// Apply pagination
	err := db.Offset(offset).Limit(limit).Find(&jobs).Error
	if err != nil {
		return nil, 0, err
	}

	return jobs, count, nil
}

func (r *jobRepository) ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.TranscriptionJob, int64, error) {
	var jobs []models.TranscriptionJob
	var count int64
	db := r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).Where("user_id = ?", userID)
	if err := db.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&jobs).Error; err != nil {
		return nil, 0, err
	}
	return jobs, count, nil
}

func (r *jobRepository) EnqueueTranscription(ctx context.Context, jobID string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":           models.StatusPending,
			"queued_at":        now,
			"started_at":       nil,
			"failed_at":        nil,
			"progress":         0,
			"progress_stage":   "queued",
			"claimed_by":       nil,
			"claim_expires_at": nil,
			"last_error":       nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *jobRepository) ClaimNextTranscription(ctx context.Context, workerID string, leaseUntil time.Time) (*models.TranscriptionJob, error) {
	var claimed models.TranscriptionJob
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidate models.TranscriptionJob
		result := tx.
			Where("status = ?", models.StatusPending).
			Order("queued_at ASC, created_at ASC, id ASC").
			Limit(1).
			Find(&candidate)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		now := time.Now()
		result = tx.Model(&models.TranscriptionJob{}).
			Where("id = ? AND status = ?", candidate.ID, models.StatusPending).
			Updates(map[string]any{
				"status":           models.StatusProcessing,
				"started_at":       now,
				"progress":         0.05,
				"progress_stage":   "preparing",
				"claimed_by":       workerID,
				"claim_expires_at": leaseUntil,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.First(&claimed, "id = ?", candidate.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &claimed, nil
}

func (r *jobRepository) RenewClaim(ctx context.Context, jobID, workerID string, leaseUntil time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).
		Where("id = ? AND status = ? AND claimed_by = ?", jobID, models.StatusProcessing, workerID).
		Update("claim_expires_at", leaseUntil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *jobRepository) RecoverOrphanedProcessing(ctx context.Context, now time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).
		Where("status = ?", models.StatusProcessing).
		Updates(map[string]any{
			"status":           models.StatusPending,
			"queued_at":        now,
			"progress_stage":   "recovered",
			"claimed_by":       nil,
			"claim_expires_at": nil,
		})
	return result.RowsAffected, result.Error
}

func (r *jobRepository) UpdateProgress(ctx context.Context, jobID string, progress float64, stage string) error {
	result := r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"progress":       progress,
			"progress_stage": stage,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *jobRepository) CompleteTranscription(ctx context.Context, jobID string, transcriptJSON string, outputPath *string, completedAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.TranscriptionJob{}).
			Where("id = ?", jobID).
			Updates(map[string]any{
				"status":           models.StatusCompleted,
				"transcript_text":  transcriptJSON,
				"output_json_path": outputPath,
				"completed_at":     completedAt,
				"failed_at":        nil,
				"progress":         1.0,
				"progress_stage":   "completed",
				"claimed_by":       nil,
				"claim_expires_at": nil,
				"last_error":       nil,
			}).Error; err != nil {
			return err
		}
		return updateLatestExecutionTerminal(tx, jobID, models.StatusCompleted, map[string]any{
			"completed_at":     completedAt,
			"failed_at":        nil,
			"error_message":    nil,
			"output_json_path": outputPath,
		})
	})
}

func (r *jobRepository) FailTranscription(ctx context.Context, jobID string, message string, failedAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.TranscriptionJob{}).
			Where("id = ?", jobID).
			Updates(map[string]any{
				"status":           models.StatusFailed,
				"failed_at":        failedAt,
				"progress_stage":   "failed",
				"claimed_by":       nil,
				"claim_expires_at": nil,
				"last_error":       message,
			}).Error; err != nil {
			return err
		}
		return updateLatestExecutionTerminal(tx, jobID, models.StatusFailed, map[string]any{
			"failed_at":     failedAt,
			"error_message": message,
		})
	})
}

func (r *jobRepository) CancelTranscription(ctx context.Context, jobID string, canceledAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.TranscriptionJob{}).
			Where("id = ?", jobID).
			Updates(map[string]any{
				"status":           models.StatusCanceled,
				"progress_stage":   "canceled",
				"claimed_by":       nil,
				"claim_expires_at": nil,
			}).Error; err != nil {
			return err
		}
		return updateLatestExecutionTerminal(tx, jobID, models.StatusCanceled, map[string]any{
			"completed_at": nil,
			"failed_at":    canceledAt,
		})
	})
}

func (r *jobRepository) ListExecutions(ctx context.Context, jobID string) ([]models.TranscriptionJobExecution, error) {
	var executions []models.TranscriptionJobExecution
	err := r.db.WithContext(ctx).
		Where("transcription_id = ?", jobID).
		Order("execution_number DESC").
		Find(&executions).Error
	return executions, err
}

func (r *jobRepository) CountStatusesByUser(ctx context.Context, userID uint) (map[models.JobStatus]int64, error) {
	type statusCount struct {
		Status models.JobStatus
		Count  int64
	}
	var rows []statusCount
	if err := r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).
		Select("status, count(*) as count").
		Where("user_id = ? AND source_file_hash IS NOT NULL", userID).
		Group("status").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := make(map[models.JobStatus]int64, len(rows))
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	return counts, nil
}

func updateLatestExecutionTerminal(tx *gorm.DB, jobID string, status models.JobStatus, updates map[string]any) error {
	var job models.TranscriptionJob
	if err := tx.Select("latest_execution_id").First(&job, "id = ?", jobID).Error; err != nil {
		return err
	}
	executionQuery := tx.Model(&models.TranscriptionJobExecution{}).Where("transcription_id = ?", jobID)
	if job.LatestExecutionID != nil && *job.LatestExecutionID != "" {
		executionQuery = executionQuery.Where("id = ?", *job.LatestExecutionID)
	} else {
		var latest models.TranscriptionJobExecution
		result := tx.Where("transcription_id = ?", jobID).Order("execution_number DESC").Limit(1).Find(&latest)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		executionQuery = executionQuery.Where("id = ?", latest.ID)
	}
	updates["status"] = status
	if err := executionQuery.Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

func (r *jobRepository) UpdateTranscript(ctx context.Context, jobID string, transcript string) error {
	return r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).
		Where("id = ?", jobID).
		Update("transcript_text", transcript).Error
}

func (r *jobRepository) CreateExecution(ctx context.Context, execution *models.TranscriptionJobExecution) error {
	const maxCreateExecutionRetries = 5
	var lastErr error
	for attempt := 0; attempt < maxCreateExecutionRetries; attempt++ {
		lastErr = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return createExecutionInTx(tx, execution)
		})
		if lastErr == nil {
			return nil
		}
		if !isExecutionNumberConflict(lastErr) {
			return lastErr
		}
	}
	return fmt.Errorf("unable to allocate execution number for transcription %s: %w", execution.TranscriptionJobID, lastErr)
}

func createExecutionInTx(tx *gorm.DB, execution *models.TranscriptionJobExecution) error {
	var job models.TranscriptionJob
	if err := tx.Select("id", "user_id").Where("id = ?", execution.TranscriptionJobID).First(&job).Error; err != nil {
		return err
	}

	execution.UserID = job.UserID

	var nextExecutionNumber int
	if err := tx.Model(&models.TranscriptionJobExecution{}).
		Where("transcription_id = ?", execution.TranscriptionJobID).
		Select("COALESCE(MAX(execution_number), 0) + 1").
		Scan(&nextExecutionNumber).Error; err != nil {
		return err
	}
	execution.ExecutionNumber = nextExecutionNumber

	if err := tx.Create(execution).Error; err != nil {
		return err
	}

	return tx.Model(&models.TranscriptionJob{}).
		Where("id = ?", execution.TranscriptionJobID).
		Update("latest_execution_id", execution.ID).Error
}

func isExecutionNumberConflict(err error) bool {
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "transcription_executions") && strings.Contains(errMsg, "execution_number")
}

func resolveLegacySingletonUserID(ctx context.Context, db *gorm.DB) (uint, error) {
	const scopeError = "legacy repository method requires explicit user-scoped method"
	var users []models.User
	if err := db.WithContext(ctx).
		Model(&models.User{}).
		Select("id").
		Order("id ASC").
		Limit(2).
		Find(&users).Error; err != nil {
		return 0, err
	}
	if len(users) == 0 {
		return 0, fmt.Errorf("%s: no users exist", scopeError)
	}
	if len(users) > 1 {
		return 0, fmt.Errorf("%s: multiple users exist", scopeError)
	}
	return users[0].ID, nil
}

func (r *jobRepository) UpdateExecution(ctx context.Context, execution *models.TranscriptionJobExecution) error {
	return r.db.WithContext(ctx).Save(execution).Error
}

func (r *jobRepository) DeleteExecutionsByJobID(ctx context.Context, jobID string) error {
	return r.db.WithContext(ctx).Where("transcription_id = ?", jobID).Delete(&models.TranscriptionJobExecution{}).Error
}

func (r *jobRepository) FindLatestCompletedExecution(ctx context.Context, jobID string) (*models.TranscriptionJobExecution, error) {
	var execution models.TranscriptionJobExecution
	err := r.db.WithContext(ctx).
		Where("transcription_id = ? AND status = ?", jobID, models.StatusCompleted).
		Order("created_at DESC").
		First(&execution).Error
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

func (r *jobRepository) UpdateStatus(ctx context.Context, jobID string, status models.JobStatus) error {
	return r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).Where("id = ?", jobID).Update("status", status).Error
}

func (r *jobRepository) UpdateError(ctx context.Context, jobID string, errorMsg string) error {
	return r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).Where("id = ?", jobID).Update("last_error", errorMsg).Error
}

func (r *jobRepository) FindByStatus(ctx context.Context, status models.JobStatus) ([]models.TranscriptionJob, error) {
	var jobs []models.TranscriptionJob
	err := r.db.WithContext(ctx).Where("status = ?", status).Find(&jobs).Error
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *jobRepository) CountByStatus(ctx context.Context, status models.JobStatus) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

func (r *jobRepository) UpdateSummary(ctx context.Context, jobID string, summary string) error {
	var job models.TranscriptionJob
	if err := r.db.WithContext(ctx).First(&job, "id = ?", jobID).Error; err != nil {
		return err
	}
	job.Summary = &summary
	return r.db.WithContext(ctx).Save(&job).Error
}

// APIKeyRepository handles API key operations
type APIKeyRepository interface {
	Repository[models.APIKey]
	FindByKey(ctx context.Context, key string) (*models.APIKey, error)
	// Deprecated: Legacy global access. Use ListActiveByUser instead.
	ListActive(ctx context.Context) ([]models.APIKey, error)
	ListActiveByUser(ctx context.Context, userID uint) ([]models.APIKey, error)
	FindByIDForUser(ctx context.Context, id, userID uint) (*models.APIKey, error)
	// Deprecated: Legacy global access. Use RevokeForUser instead.
	Revoke(ctx context.Context, id uint) error
	RevokeForUser(ctx context.Context, id, userID uint) error
}

type apiKeyRepository struct {
	*BaseRepository[models.APIKey]
}

func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{
		BaseRepository: NewBaseRepository[models.APIKey](db),
	}
}

func (r *apiKeyRepository) FindByKey(ctx context.Context, key string) (*models.APIKey, error) {
	var apiKey models.APIKey
	err := r.db.WithContext(ctx).
		Where("key_hash = ? AND revoked_at IS NULL", hashToken(key)).
		First(&apiKey).Error
	if err != nil {
		return nil, err
	}
	apiKey.Key = key
	return &apiKey, nil
}

func (r *apiKeyRepository) ListActive(ctx context.Context) ([]models.APIKey, error) {
	userID, err := resolveLegacySingletonUserID(ctx, r.db)
	if err != nil {
		return nil, err
	}
	return r.ListActiveByUser(ctx, userID)
}

func (r *apiKeyRepository) ListActiveByUser(ctx context.Context, userID uint) ([]models.APIKey, error) {
	var apiKeys []models.APIKey
	if err := r.db.WithContext(ctx).Where("user_id = ? AND revoked_at IS NULL", userID).Find(&apiKeys).Error; err != nil {
		return nil, err
	}
	return apiKeys, nil
}

func (r *apiKeyRepository) FindByIDForUser(ctx context.Context, id, userID uint) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&apiKey).Error; err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) Revoke(ctx context.Context, id uint) error {
	// Revoke is intentionally global for backward compatibility.
	// Prefer RevokeForUser with explicit user ID for all new call sites.
	userID, err := resolveLegacySingletonUserID(ctx, r.db)
	if err != nil {
		return err
	}
	return r.RevokeForUser(ctx, id, userID)
}

func (r *apiKeyRepository) RevokeForUser(ctx context.Context, id, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.APIKey{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("revoked_at", &now).Error
}

// ProfileRepository handles transcription profile operations
type ProfileRepository interface {
	Repository[models.TranscriptionProfile]
	// Deprecated: Legacy global access. Use FindDefaultByUser instead.
	FindDefault(ctx context.Context) (*models.TranscriptionProfile, error)
	// Deprecated: Legacy global access. Use FindByNameForUser instead.
	FindByName(ctx context.Context, name string) (*models.TranscriptionProfile, error)
	ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.TranscriptionProfile, int64, error)
	FindByIDForUser(ctx context.Context, id string, userID uint) (*models.TranscriptionProfile, error)
	FindDefaultByUser(ctx context.Context, userID uint) (*models.TranscriptionProfile, error)
	FindByNameForUser(ctx context.Context, userID uint, name string) (*models.TranscriptionProfile, error)
}

type profileRepository struct {
	*BaseRepository[models.TranscriptionProfile]
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{
		BaseRepository: NewBaseRepository[models.TranscriptionProfile](db),
	}
}

func (r *profileRepository) FindDefault(ctx context.Context) (*models.TranscriptionProfile, error) {
	userID, err := resolveLegacySingletonUserID(ctx, r.db)
	if err != nil {
		return nil, err
	}
	return r.FindDefaultByUser(ctx, userID)
}

func (r *profileRepository) ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.TranscriptionProfile, int64, error) {
	var profiles []models.TranscriptionProfile
	var count int64

	query := r.db.WithContext(ctx).Model(&models.TranscriptionProfile{}).Where("user_id = ?", userID)
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&profiles).Error; err != nil {
		return nil, 0, err
	}
	return profiles, count, nil
}

func (r *profileRepository) FindByIDForUser(ctx context.Context, id string, userID uint) (*models.TranscriptionProfile, error) {
	var profile models.TranscriptionProfile
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) FindDefaultByUser(ctx context.Context, userID uint) (*models.TranscriptionProfile, error) {
	var profile models.TranscriptionProfile
	if err := r.db.WithContext(ctx).Where("user_id = ? AND is_default = ?", userID, true).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) FindByName(ctx context.Context, name string) (*models.TranscriptionProfile, error) {
	userID, err := resolveLegacySingletonUserID(ctx, r.db)
	if err != nil {
		return nil, err
	}
	return r.FindByNameForUser(ctx, userID, name)
}

func (r *profileRepository) FindByNameForUser(ctx context.Context, userID uint, name string) (*models.TranscriptionProfile, error) {
	var profile models.TranscriptionProfile
	if err := r.db.WithContext(ctx).Where("user_id = ? AND name = ?", userID, name).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

// LLMConfigRepository handles LLM configuration operations
type LLMConfigRepository interface {
	Repository[models.LLMConfig]
	// Deprecated: Legacy global access. Use GetActiveByUser instead.
	GetActive(ctx context.Context) (*models.LLMConfig, error)
	GetActiveByUser(ctx context.Context, userID uint) (*models.LLMConfig, error)
}

type llmConfigRepository struct {
	*BaseRepository[models.LLMConfig]
}

func NewLLMConfigRepository(db *gorm.DB) LLMConfigRepository {
	return &llmConfigRepository{
		BaseRepository: NewBaseRepository[models.LLMConfig](db),
	}
}

func (r *llmConfigRepository) GetActive(ctx context.Context) (*models.LLMConfig, error) {
	userID, err := resolveLegacySingletonUserID(ctx, r.db)
	if err != nil {
		return nil, err
	}
	return r.GetActiveByUser(ctx, userID)
}

func (r *llmConfigRepository) GetActiveByUser(ctx context.Context, userID uint) (*models.LLMConfig, error) {
	var config models.LLMConfig
	if err := r.db.WithContext(ctx).Where("user_id = ? AND is_default = ?", userID, true).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// SummaryRepository handles summary templates and settings
type SummaryRepository interface {
	Repository[models.SummaryTemplate]
	// Deprecated: Legacy global access. Use GetSettingsByUser/SaveSettingsByUser instead.
	GetSettings(ctx context.Context) (*models.SummarySetting, error)
	// Deprecated: Legacy global access. Use GetSettingsByUser/SaveSettingsByUser instead.
	SaveSettings(ctx context.Context, settings *models.SummarySetting) error
	ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.SummaryTemplate, int64, error)
	FindByIDForUser(ctx context.Context, id string, userID uint) (*models.SummaryTemplate, error)
	GetSettingsByUser(ctx context.Context, userID uint) (*models.SummarySetting, error)
	SaveSettingsByUser(ctx context.Context, userID uint, settings *models.SummarySetting) error
	SaveSummary(ctx context.Context, summary *models.Summary) error
	EnqueueAutomaticSummary(ctx context.Context, transcriptionID string, userID uint, model string, provider string) (*models.Summary, bool, error)
	ClaimNextPendingSummary(ctx context.Context, now time.Time) (*models.Summary, error)
	CompleteSummary(ctx context.Context, id string, content string, truncated bool, contextWindow int, inputCharacters int, completedAt time.Time) error
	FailSummary(ctx context.Context, id string, message string, failedAt time.Time) error
	RecoverProcessingSummaries(ctx context.Context) (int64, error)
	GetLatestSummary(ctx context.Context, transcriptionID string) (*models.Summary, error)
	DeleteByTranscriptionID(ctx context.Context, transcriptionID string) error
}

type summaryRepository struct {
	*BaseRepository[models.SummaryTemplate]
}

func NewSummaryRepository(db *gorm.DB) SummaryRepository {
	return &summaryRepository{
		BaseRepository: NewBaseRepository[models.SummaryTemplate](db),
	}
}

func (r *summaryRepository) GetSettings(ctx context.Context) (*models.SummarySetting, error) {
	userID, err := resolveLegacySingletonUserID(ctx, r.db)
	if err != nil {
		return nil, err
	}
	return r.GetSettingsByUser(ctx, userID)
}

func (r *summaryRepository) SaveSettings(ctx context.Context, settings *models.SummarySetting) error {
	userID, err := resolveLegacySingletonUserID(ctx, r.db)
	if err != nil {
		return err
	}
	return r.SaveSettingsByUser(ctx, userID, settings)
}

func (r *summaryRepository) ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.SummaryTemplate, int64, error) {
	var templates []models.SummaryTemplate
	var count int64

	query := r.db.WithContext(ctx).Model(&models.SummaryTemplate{}).Where("user_id = ?", userID)
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&templates).Error; err != nil {
		return nil, 0, err
	}
	return templates, count, nil
}

func (r *summaryRepository) FindByIDForUser(ctx context.Context, id string, userID uint) (*models.SummaryTemplate, error) {
	var template models.SummaryTemplate
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&template).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *summaryRepository) GetSettingsByUser(ctx context.Context, userID uint) (*models.SummarySetting, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return &models.SummarySetting{DefaultModel: user.SummaryDefaultModel}, nil
}

func (r *summaryRepository) SaveSettingsByUser(ctx context.Context, userID uint, settings *models.SummarySetting) error {
	var user models.User
	if err := r.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	user.SummaryDefaultModel = settings.DefaultModel
	return r.db.WithContext(ctx).Save(&user).Error
}

func (r *summaryRepository) SaveSummary(ctx context.Context, summary *models.Summary) error {
	return r.db.WithContext(ctx).Create(summary).Error
}

func (r *summaryRepository) EnqueueAutomaticSummary(ctx context.Context, transcriptionID string, userID uint, model string, provider string) (*models.Summary, bool, error) {
	var existing models.Summary
	result := r.db.WithContext(ctx).
		Where("transcription_id = ? AND user_id = ? AND status IN ?", transcriptionID, userID, []string{"pending", "processing", "completed"}).
		Order("created_at DESC").
		Limit(1).
		Find(&existing)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected > 0 {
		return &existing, false, nil
	}
	summary := &models.Summary{
		TranscriptionID: transcriptionID,
		UserID:          userID,
		Content:         "",
		Model:           model,
		Provider:        provider,
		Status:          "pending",
	}
	if err := r.db.WithContext(ctx).Create(summary).Error; err != nil {
		return nil, false, err
	}
	return summary, true, nil
}

func (r *summaryRepository) ClaimNextPendingSummary(ctx context.Context, now time.Time) (*models.Summary, error) {
	var summary models.Summary
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("status = ?", "pending").Order("created_at ASC").Limit(1).Find(&summary)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		updateResult := tx.Model(&models.Summary{}).
			Where("id = ? AND status = ?", summary.ID, "pending").
			Updates(map[string]any{"status": "processing", "started_at": now})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		summary.Status = "processing"
		summary.StartedAt = &now
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

func (r *summaryRepository) CompleteSummary(ctx context.Context, id string, content string, truncated bool, contextWindow int, inputCharacters int, completedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.Summary{}).
		Where("id = ? AND status = ?", id, "processing").
		Updates(map[string]any{
			"content":              content,
			"status":               "completed",
			"transcript_truncated": truncated,
			"context_window":       contextWindow,
			"input_characters":     inputCharacters,
			"completed_at":         completedAt,
			"error_message":        nil,
		}).Error
}

func (r *summaryRepository) FailSummary(ctx context.Context, id string, message string, failedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.Summary{}).
		Where("id = ? AND status = ?", id, "processing").
		Updates(map[string]any{
			"status":        "failed",
			"error_message": message,
			"failed_at":     failedAt,
		}).Error
}

func (r *summaryRepository) RecoverProcessingSummaries(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Summary{}).
		Where("status = ?", "processing").
		Update("status", "pending")
	return result.RowsAffected, result.Error
}

func (r *summaryRepository) GetLatestSummary(ctx context.Context, transcriptionID string) (*models.Summary, error) {
	var summary models.Summary
	err := r.db.WithContext(ctx).Where("transcription_id = ?", transcriptionID).Order("created_at DESC").First(&summary).Error
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

func (r *summaryRepository) DeleteByTranscriptionID(ctx context.Context, transcriptionID string) error {
	return r.db.WithContext(ctx).Where("transcription_id = ?", transcriptionID).Delete(&models.Summary{}).Error
}

// ChatRepository handles chat sessions and messages
type ChatRepository interface {
	Repository[models.ChatSession]
	GetSessionWithMessages(ctx context.Context, id string) (*models.ChatSession, error)
	GetSessionWithTranscription(ctx context.Context, id string) (*models.ChatSession, error)
	AddMessage(ctx context.Context, message *models.ChatMessage) error
	ListByJob(ctx context.Context, jobID string) ([]models.ChatSession, error)
	DeleteSession(ctx context.Context, id string) error
	GetMessages(ctx context.Context, sessionID string, limit int) ([]models.ChatMessage, error)
	DeleteByJobID(ctx context.Context, jobID string) error
	GetMessageCountsBySessionIDs(ctx context.Context, sessionIDs []string) (map[string]int64, error)
	GetLastMessagesBySessionIDs(ctx context.Context, sessionIDs []string) (map[string]*models.ChatMessage, error)
}

type chatRepository struct {
	*BaseRepository[models.ChatSession]
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{
		BaseRepository: NewBaseRepository[models.ChatSession](db),
	}
}

func (r *chatRepository) GetSessionWithMessages(ctx context.Context, id string) (*models.ChatSession, error) {
	var session models.ChatSession
	err := r.db.WithContext(ctx).Preload("Messages").Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *chatRepository) GetSessionWithTranscription(ctx context.Context, id string) (*models.ChatSession, error) {
	var session models.ChatSession
	err := r.db.WithContext(ctx).Preload("Transcription").Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *chatRepository) AddMessage(ctx context.Context, message *models.ChatMessage) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *chatRepository) ListByJob(ctx context.Context, jobID string) ([]models.ChatSession, error) {
	var sessions []models.ChatSession
	err := r.db.WithContext(ctx).Where("transcription_id = ?", jobID).Order("created_at DESC").Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *chatRepository) DeleteSession(ctx context.Context, id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete messages first
		if err := tx.Where("chat_session_id = ?", id).Delete(&models.ChatMessage{}).Error; err != nil {
			return err
		}
		// Delete session
		return tx.Delete(&models.ChatSession{}, "id = ?", id).Error
	})
}

func (r *chatRepository) DeleteByJobID(ctx context.Context, jobID string) error {
	// Find all sessions for this job
	var sessions []models.ChatSession
	if err := r.db.WithContext(ctx).Where("transcription_id = ?", jobID).Find(&sessions).Error; err != nil {
		return err
	}

	// Delete each session (which deletes messages)
	for _, session := range sessions {
		if err := r.DeleteSession(ctx, session.ID); err != nil {
			return err
		}
	}
	return nil
}

func (r *chatRepository) GetMessages(ctx context.Context, sessionID string, limit int) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	query := r.db.WithContext(ctx).Where("chat_session_id = ?", sessionID).Order("created_at ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *chatRepository) GetMessageCountsBySessionIDs(ctx context.Context, sessionIDs []string) (map[string]int64, error) {
	if len(sessionIDs) == 0 {
		return make(map[string]int64), nil
	}

	type MessageCount struct {
		SessionID string `gorm:"column:session_id"`
		Count     int64  `gorm:"column:count"`
	}
	var counts []MessageCount

	err := r.db.WithContext(ctx).Model(&models.ChatMessage{}).
		Select("chat_session_id as session_id, COUNT(*) as count").
		Where("chat_session_id IN ?", sessionIDs).
		Group("chat_session_id").
		Scan(&counts).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64)
	for _, c := range counts {
		result[c.SessionID] = c.Count
	}
	return result, nil
}

func (r *chatRepository) GetLastMessagesBySessionIDs(ctx context.Context, sessionIDs []string) (map[string]*models.ChatMessage, error) {
	if len(sessionIDs) == 0 {
		return make(map[string]*models.ChatMessage), nil
	}

	var lastMessages []models.ChatMessage
	err := r.db.WithContext(ctx).Where(`id IN (
		SELECT id FROM chat_messages cm1
		WHERE cm1.chat_session_id IN ? 
		AND cm1.created_at = (
			SELECT MAX(cm2.created_at) 
			FROM chat_messages cm2 
			WHERE cm2.chat_session_id = cm1.chat_session_id
		)
	)`, sessionIDs).Find(&lastMessages).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]*models.ChatMessage)
	for i := range lastMessages {
		result[lastMessages[i].ChatSessionID] = &lastMessages[i]
	}
	return result, nil
}

// NoteRepository handles notes
type NoteRepository interface {
	Repository[models.Note]
	ListByJob(ctx context.Context, jobID string) ([]models.Note, error)
	DeleteByTranscriptionID(ctx context.Context, transcriptionID string) error
}

type noteRepository struct {
	*BaseRepository[models.Note]
}

func NewNoteRepository(db *gorm.DB) NoteRepository {
	return &noteRepository{
		BaseRepository: NewBaseRepository[models.Note](db),
	}
}

func (r *noteRepository) ListByJob(ctx context.Context, jobID string) ([]models.Note, error) {
	var notes []models.Note
	err := r.db.WithContext(ctx).Where("transcription_id = ?", jobID).Order("created_at DESC").Find(&notes).Error
	if err != nil {
		return nil, err
	}
	return notes, nil
}

func (r *noteRepository) DeleteByTranscriptionID(ctx context.Context, transcriptionID string) error {
	return r.db.WithContext(ctx).Where("transcription_id = ?", transcriptionID).Delete(&models.Note{}).Error
}

// SpeakerMappingRepository handles speaker mappings
type SpeakerMappingRepository interface {
	Repository[models.SpeakerMapping]
	ListByJob(ctx context.Context, jobID string) ([]models.SpeakerMapping, error)
	UpdateMappings(ctx context.Context, jobID string, mappings []models.SpeakerMapping) error
	DeleteByJobID(ctx context.Context, jobID string) error
}

type speakerMappingRepository struct {
	*BaseRepository[models.SpeakerMapping]
}

func NewSpeakerMappingRepository(db *gorm.DB) SpeakerMappingRepository {
	return &speakerMappingRepository{
		BaseRepository: NewBaseRepository[models.SpeakerMapping](db),
	}
}

func (r *speakerMappingRepository) ListByJob(ctx context.Context, jobID string) ([]models.SpeakerMapping, error) {
	var mappings []models.SpeakerMapping
	err := r.db.WithContext(ctx).Where("transcription_id = ?", jobID).Find(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *speakerMappingRepository) DeleteByJobID(ctx context.Context, jobID string) error {
	return r.db.WithContext(ctx).Where("transcription_id = ?", jobID).Delete(&models.SpeakerMapping{}).Error
}

func (r *speakerMappingRepository) UpdateMappings(ctx context.Context, jobID string, mappings []models.SpeakerMapping) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing mappings for this job
		if err := tx.Where("transcription_id = ?", jobID).Delete(&models.SpeakerMapping{}).Error; err != nil {
			return err
		}

		// Create new mappings
		if len(mappings) > 0 {
			if err := tx.Create(&mappings).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// RefreshTokenRepository handles refresh token operations
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error)
	Revoke(ctx context.Context, id uint) error
	RevokeByHash(ctx context.Context, hash string) error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *refreshTokenRepository) FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).Where("id = ?", id).Update("revoked_at", &now).Error
}

func (r *refreshTokenRepository) RevokeByHash(ctx context.Context, hash string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).Where("token_hash = ?", hash).Update("revoked_at", &now).Error
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
