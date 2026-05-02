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
	FindTranscriptionByIDForUser(ctx context.Context, id string, userID uint) (*models.TranscriptionJob, error)
	FindLatestCompletedExecution(ctx context.Context, jobID string) (*models.TranscriptionJobExecution, error)
	ListWithParams(ctx context.Context, offset, limit int, sortBy, sortOrder, searchQuery string, updatedAfter *time.Time) ([]models.TranscriptionJob, int64, error)
	ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.TranscriptionJob, int64, error)
	EnqueueTranscription(ctx context.Context, jobID string, now time.Time) error
	ClaimNextTranscription(ctx context.Context, workerID string, leaseUntil time.Time) (*models.TranscriptionJob, error)
	RenewClaim(ctx context.Context, jobID, workerID string, leaseUntil time.Time) error
	RecoverOrphanedProcessing(ctx context.Context, now time.Time) (int64, error)
	UpdateProgress(ctx context.Context, jobID string, progress float64, stage string) error
	CompleteMediaImport(ctx context.Context, jobID, title, audioPath, sourceFileName string, durationMs *int64, completedAt time.Time) error
	FailMediaImport(ctx context.Context, jobID string, message string, failedAt time.Time) error
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
	UpdateLLMGeneratedTitle(ctx context.Context, transcriptionID string, recordingID string, title string, generatedAt time.Time) error
	UpdateLLMGeneratedDescription(ctx context.Context, transcriptionID string, recordingID string, summaryID string, description string, generatedAt time.Time) error
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

func (r *jobRepository) FindTranscriptionByIDForUser(ctx context.Context, id string, userID uint) (*models.TranscriptionJob, error) {
	var job models.TranscriptionJob
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND source_file_hash IS NOT NULL", id, userID).
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

func (r *jobRepository) CompleteMediaImport(ctx context.Context, jobID, title, audioPath, sourceFileName string, durationMs *int64, completedAt time.Time) error {
	updates := map[string]any{
		"status":             models.StatusUploaded,
		"title":              title,
		"source_file_path":   audioPath,
		"source_file_name":   sourceFileName,
		"source_duration_ms": durationMs,
		"completed_at":       completedAt,
		"failed_at":          nil,
		"progress":           1.0,
		"progress_stage":     "ready",
		"last_error":         nil,
	}
	result := r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).
		Where("id = ? AND status = ?", jobID, models.StatusProcessing).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *jobRepository) FailMediaImport(ctx context.Context, jobID string, message string, failedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).
		Where("id = ? AND status = ?", jobID, models.StatusProcessing).
		Updates(map[string]any{
			"status":         models.StatusFailed,
			"failed_at":      failedAt,
			"progress_stage": "failed",
			"last_error":     message,
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
				"status":           models.StatusStopped,
				"progress_stage":   "stopped",
				"claimed_by":       nil,
				"claim_expires_at": nil,
			}).Error; err != nil {
			return err
		}
		return updateLatestExecutionTerminal(tx, jobID, models.StatusStopped, map[string]any{
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

func (r *jobRepository) UpdateLLMGeneratedTitle(ctx context.Context, transcriptionID string, recordingID string, title string, generatedAt time.Time) error {
	ids := []string{transcriptionID}
	if recordingID != "" && recordingID != transcriptionID {
		ids = append(ids, recordingID)
	}
	return r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"title":                  title,
			"llm_title_generated":    true,
			"llm_title_generated_at": generatedAt,
		}).Error
}

func (r *jobRepository) UpdateLLMGeneratedDescription(ctx context.Context, transcriptionID string, recordingID string, summaryID string, description string, generatedAt time.Time) error {
	ids := []string{transcriptionID}
	if recordingID != "" && recordingID != transcriptionID {
		ids = append(ids, recordingID)
	}
	return r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"llm_description":                   description,
			"llm_description_generated_at":      generatedAt,
			"llm_description_source_summary_id": summaryID,
		}).Error
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

// ChatRepository handles chat session, context source, message, and run persistence.
type ChatRepository interface {
	CreateSession(ctx context.Context, session *models.ChatSession) error
	CreateSessionWithParentSource(ctx context.Context, session *models.ChatSession, source *models.ChatContextSource) error
	FindSessionForUser(ctx context.Context, userID uint, sessionID string) (*models.ChatSession, error)
	ListSessionsForTranscription(ctx context.Context, userID uint, transcriptionID string, offset, limit int) ([]models.ChatSession, int64, error)
	UpdateSession(ctx context.Context, session *models.ChatSession) error
	DeleteSession(ctx context.Context, userID uint, sessionID string) error
	FindCompletedTranscriptionForUser(ctx context.Context, userID uint, transcriptionID string) (*models.TranscriptionJob, error)
	UpsertContextSource(ctx context.Context, userID uint, sessionID string, source *models.ChatContextSource) (*models.ChatContextSource, error)
	FindContextSourceForUser(ctx context.Context, userID uint, sessionID string, sourceID string) (*models.ChatContextSource, error)
	ListContextSources(ctx context.Context, userID uint, sessionID string, enabledOnly bool) ([]models.ChatContextSource, error)
	SetContextSourceEnabled(ctx context.Context, userID uint, sessionID string, sourceID string, enabled bool) error
	DeleteContextSource(ctx context.Context, userID uint, sessionID string, sourceID string) error
	UpdateContextSourceCompaction(ctx context.Context, userID uint, sessionID string, sourceID string, status models.ChatContextCompactionStatus, compactedSnapshot *string) error
	CreateMessage(ctx context.Context, message *models.ChatMessage) error
	UpdateMessage(ctx context.Context, message *models.ChatMessage) error
	ListMessages(ctx context.Context, userID uint, sessionID string, offset, limit int) ([]models.ChatMessage, int64, error)
	CreateGenerationRun(ctx context.Context, run *models.ChatGenerationRun) error
	FindGenerationRunForUser(ctx context.Context, userID uint, runID string) (*models.ChatGenerationRun, error)
	UpdateGenerationRunStatus(ctx context.Context, userID uint, runID string, status models.ChatGenerationRunStatus, at time.Time, errorMessage *string) error
	SaveContextSummary(ctx context.Context, summary *models.ChatContextSummary) error
	ListContextSummaries(ctx context.Context, userID uint, sessionID string, summaryType models.ChatContextSummaryType) ([]models.ChatContextSummary, error)
}

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) CreateSession(ctx context.Context, session *models.ChatSession) error {
	if session == nil {
		return fmt.Errorf("chat session is required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureCompletedTranscriptionForUser(tx, session.UserID, session.ParentTranscriptionID); err != nil {
			return err
		}
		return tx.Create(session).Error
	})
}

func (r *chatRepository) CreateSessionWithParentSource(ctx context.Context, session *models.ChatSession, source *models.ChatContextSource) error {
	if session == nil {
		return fmt.Errorf("chat session is required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureCompletedTranscriptionForUser(tx, session.UserID, session.ParentTranscriptionID); err != nil {
			return err
		}
		if err := tx.Create(session).Error; err != nil {
			return err
		}
		if source == nil {
			return nil
		}
		source.UserID = session.UserID
		source.ChatSessionID = session.ID
		source.TranscriptionID = session.ParentTranscriptionID
		source.Kind = models.ChatContextSourceKindParentTranscript
		if source.MetadataJSON == "" {
			source.MetadataJSON = "{}"
		}
		if err := applyContextSourceSnapshot(source); err != nil {
			return err
		}
		return tx.Create(source).Error
	})
}

func (r *chatRepository) FindSessionForUser(ctx context.Context, userID uint, sessionID string) (*models.ChatSession, error) {
	var session models.ChatSession
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *chatRepository) ListSessionsForTranscription(ctx context.Context, userID uint, transcriptionID string, offset, limit int) ([]models.ChatSession, int64, error) {
	var sessions []models.ChatSession
	var count int64
	query := r.db.WithContext(ctx).Model(&models.ChatSession{}).
		Where("user_id = ? AND parent_transcription_id = ?", userID, transcriptionID)
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("updated_at DESC").Offset(offset).Limit(limit).Find(&sessions).Error; err != nil {
		return nil, 0, err
	}
	return sessions, count, nil
}

func (r *chatRepository) UpdateSession(ctx context.Context, session *models.ChatSession) error {
	if session == nil {
		return fmt.Errorf("chat session is required")
	}
	result := r.db.WithContext(ctx).Model(&models.ChatSession{}).
		Where("id = ? AND user_id = ?", session.ID, session.UserID).
		Updates(map[string]any{
			"title":         session.Title,
			"status":        session.Status,
			"system_prompt": session.SystemPrompt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *chatRepository) DeleteSession(ctx context.Context, userID uint, sessionID string) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", sessionID, userID).Delete(&models.ChatSession{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *chatRepository) FindCompletedTranscriptionForUser(ctx context.Context, userID uint, transcriptionID string) (*models.TranscriptionJob, error) {
	var job models.TranscriptionJob
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND status = ?", transcriptionID, userID, models.StatusCompleted).
		First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *chatRepository) UpsertContextSource(ctx context.Context, userID uint, sessionID string, source *models.ChatContextSource) (*models.ChatContextSource, error) {
	if source == nil {
		return nil, fmt.Errorf("chat context source is required")
	}
	var saved models.ChatContextSource
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if source.Kind == "" {
			source.Kind = models.ChatContextSourceKindTranscript
		}
		if source.CompactionStatus == "" {
			source.CompactionStatus = models.ChatContextCompactionStatusNone
		}
		session, err := findSessionForUserInTx(tx, userID, sessionID)
		if err != nil {
			return err
		}
		if source.Kind == models.ChatContextSourceKindParentTranscript && source.TranscriptionID != session.ParentTranscriptionID {
			return gorm.ErrRecordNotFound
		}
		if err := ensureCompletedTranscriptionForUser(tx, userID, source.TranscriptionID); err != nil {
			return err
		}

		var existing models.ChatContextSource
		result := tx.Where("user_id = ? AND chat_session_id = ? AND transcription_id = ? AND kind = ?", userID, sessionID, source.TranscriptionID, source.Kind).
			Limit(1).
			Find(&existing)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			position, err := nextContextSourcePosition(tx, sessionID)
			if err != nil {
				return err
			}
			source.UserID = userID
			source.ChatSessionID = sessionID
			source.Position = position
			source.Enabled = true
			if source.MetadataJSON == "" {
				source.MetadataJSON = "{}"
			}
			if err := applyContextSourceSnapshot(source); err != nil {
				return err
			}
			if err := tx.Create(source).Error; err != nil {
				return err
			}
			saved = *source
			return nil
		}

		updates := map[string]any{
			"enabled":            true,
			"metadata_json":      nonEmptyString(source.MetadataJSON, existing.MetadataJSON),
			"source_version":     source.SourceVersion,
			"compacted_snapshot": source.CompactedSnapshot,
			"compaction_status":  source.CompactionStatus,
		}
		if source.PlainTextSnapshot != nil {
			updates["plain_text_snapshot"] = source.PlainTextSnapshot
			updates["snapshot_hash"] = hashString(*source.PlainTextSnapshot)
		}
		if source.Position > 0 {
			updates["position"] = source.Position
		}
		if updates["compaction_status"] == "" {
			updates["compaction_status"] = models.ChatContextCompactionStatusNone
		}
		if err := tx.Model(&models.ChatContextSource{}).Where("id = ? AND user_id = ?", existing.ID, userID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&saved, "id = ?", existing.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

func (r *chatRepository) FindContextSourceForUser(ctx context.Context, userID uint, sessionID string, sourceID string) (*models.ChatContextSource, error) {
	var source models.ChatContextSource
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND chat_session_id = ?", sourceID, userID, sessionID).
		First(&source).Error; err != nil {
		return nil, err
	}
	return &source, nil
}

func (r *chatRepository) ListContextSources(ctx context.Context, userID uint, sessionID string, enabledOnly bool) ([]models.ChatContextSource, error) {
	query := r.db.WithContext(ctx).
		Preload("Transcription").
		Where("user_id = ? AND chat_session_id = ?", userID, sessionID)
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	var sources []models.ChatContextSource
	if err := query.Order("position ASC, created_at ASC, id ASC").Find(&sources).Error; err != nil {
		return nil, err
	}
	return sources, nil
}

func (r *chatRepository) SetContextSourceEnabled(ctx context.Context, userID uint, sessionID string, sourceID string, enabled bool) error {
	result := r.db.WithContext(ctx).Model(&models.ChatContextSource{}).
		Where("id = ? AND user_id = ? AND chat_session_id = ?", sourceID, userID, sessionID).
		Update("enabled", enabled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *chatRepository) DeleteContextSource(ctx context.Context, userID uint, sessionID string, sourceID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND chat_session_id = ?", sourceID, userID, sessionID).
		Delete(&models.ChatContextSource{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *chatRepository) UpdateContextSourceCompaction(ctx context.Context, userID uint, sessionID string, sourceID string, status models.ChatContextCompactionStatus, compactedSnapshot *string) error {
	updates := map[string]any{
		"compaction_status": status,
	}
	if compactedSnapshot != nil || status == models.ChatContextCompactionStatusNone {
		updates["compacted_snapshot"] = compactedSnapshot
	}
	result := r.db.WithContext(ctx).Model(&models.ChatContextSource{}).
		Where("id = ? AND user_id = ? AND chat_session_id = ?", sourceID, userID, sessionID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *chatRepository) CreateMessage(ctx context.Context, message *models.ChatMessage) error {
	if message == nil {
		return fmt.Errorf("chat message is required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := findSessionForUserInTx(tx, message.UserID, message.ChatSessionID); err != nil {
			return err
		}
		if err := tx.Create(message).Error; err != nil {
			return err
		}
		return tx.Model(&models.ChatSession{}).
			Where("id = ? AND user_id = ?", message.ChatSessionID, message.UserID).
			Update("last_message_at", message.CreatedAt).Error
	})
}

func (r *chatRepository) UpdateMessage(ctx context.Context, message *models.ChatMessage) error {
	if message == nil {
		return fmt.Errorf("chat message is required")
	}
	result := r.db.WithContext(ctx).Model(&models.ChatMessage{}).
		Where("id = ? AND user_id = ? AND chat_session_id = ?", message.ID, message.UserID, message.ChatSessionID).
		UpdateColumns(map[string]any{
			"content":           message.Content,
			"reasoning_content": message.ReasoningContent,
			"status":            message.Status,
			"provider":          message.Provider,
			"model_name":        message.Model,
			"run_id":            message.RunID,
			"prompt_tokens":     message.PromptTokens,
			"completion_tokens": message.CompletionTokens,
			"reasoning_tokens":  message.ReasoningTokens,
			"total_tokens":      message.TotalTokens,
			"metadata_json":     message.MetadataJSON,
			"updated_at":        time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *chatRepository) ListMessages(ctx context.Context, userID uint, sessionID string, offset, limit int) ([]models.ChatMessage, int64, error) {
	var messages []models.ChatMessage
	var count int64
	query := r.db.WithContext(ctx).Model(&models.ChatMessage{}).
		Where("user_id = ? AND chat_session_id = ?", userID, sessionID)
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at ASC, id ASC").Offset(offset).Limit(limit).Find(&messages).Error; err != nil {
		return nil, 0, err
	}
	return messages, count, nil
}

func (r *chatRepository) CreateGenerationRun(ctx context.Context, run *models.ChatGenerationRun) error {
	if run == nil {
		return fmt.Errorf("chat generation run is required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := findSessionForUserInTx(tx, run.UserID, run.ChatSessionID); err != nil {
			return err
		}
		return tx.Create(run).Error
	})
}

func (r *chatRepository) FindGenerationRunForUser(ctx context.Context, userID uint, runID string) (*models.ChatGenerationRun, error) {
	var run models.ChatGenerationRun
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", runID, userID).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *chatRepository) UpdateGenerationRunStatus(ctx context.Context, userID uint, runID string, status models.ChatGenerationRunStatus, at time.Time, errorMessage *string) error {
	updates := map[string]any{"status": status}
	switch status {
	case models.ChatGenerationRunStatusStreaming:
		updates["started_at"] = at
	case models.ChatGenerationRunStatusCompleted:
		updates["completed_at"] = at
		updates["error_message"] = nil
	case models.ChatGenerationRunStatusFailed:
		updates["failed_at"] = at
		updates["error_message"] = errorMessage
	case models.ChatGenerationRunStatusCanceled:
		updates["completed_at"] = at
		updates["error_message"] = errorMessage
	default:
	}
	result := r.db.WithContext(ctx).Model(&models.ChatGenerationRun{}).
		Where("id = ? AND user_id = ?", runID, userID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *chatRepository) SaveContextSummary(ctx context.Context, summary *models.ChatContextSummary) error {
	if summary == nil {
		return fmt.Errorf("chat context summary is required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := findSessionForUserInTx(tx, summary.UserID, summary.ChatSessionID); err != nil {
			return err
		}
		if summary.SourceTranscriptionID != nil && *summary.SourceTranscriptionID != "" {
			if err := ensureCompletedTranscriptionForUser(tx, summary.UserID, *summary.SourceTranscriptionID); err != nil {
				return err
			}
		}
		if summary.SourceMessageThroughID != nil && *summary.SourceMessageThroughID != "" {
			var message models.ChatMessage
			if err := tx.Select("id").
				Where("id = ? AND user_id = ? AND chat_session_id = ?", *summary.SourceMessageThroughID, summary.UserID, summary.ChatSessionID).
				First(&message).Error; err != nil {
				return err
			}
		}
		return tx.Create(summary).Error
	})
}

func (r *chatRepository) ListContextSummaries(ctx context.Context, userID uint, sessionID string, summaryType models.ChatContextSummaryType) ([]models.ChatContextSummary, error) {
	query := r.db.WithContext(ctx).
		Where("user_id = ? AND chat_session_id = ?", userID, sessionID)
	if summaryType != "" {
		query = query.Where("summary_type = ?", summaryType)
	}
	var summaries []models.ChatContextSummary
	if err := query.Order("created_at DESC, id DESC").Find(&summaries).Error; err != nil {
		return nil, err
	}
	return summaries, nil
}

func ensureCompletedTranscriptionForUser(tx *gorm.DB, userID uint, transcriptionID string) error {
	var job models.TranscriptionJob
	return tx.Select("id").
		Where("id = ? AND user_id = ? AND status = ?", transcriptionID, userID, models.StatusCompleted).
		First(&job).Error
}

func findSessionForUserInTx(tx *gorm.DB, userID uint, sessionID string) (*models.ChatSession, error) {
	var session models.ChatSession
	if err := tx.Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func nextContextSourcePosition(tx *gorm.DB, sessionID string) (int, error) {
	var maxPosition int
	if err := tx.Model(&models.ChatContextSource{}).
		Where("chat_session_id = ?", sessionID).
		Select("COALESCE(MAX(position), -1) + 1").
		Scan(&maxPosition).Error; err != nil {
		return 0, err
	}
	return maxPosition, nil
}

func applyContextSourceSnapshot(source *models.ChatContextSource) error {
	if source == nil || source.PlainTextSnapshot == nil {
		return nil
	}
	hash := hashString(*source.PlainTextSnapshot)
	source.SnapshotHash = &hash
	return nil
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func nonEmptyString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
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
	ListCompletedSummariesForTitleGeneration(ctx context.Context, limit int) ([]models.Summary, error)
	GetCompletedOutlineRun(ctx context.Context, summaryID string, transcriptionID string, userID uint) (*models.SummaryWidgetRun, error)
	GetLatestSummary(ctx context.Context, transcriptionID string) (*models.Summary, error)
	GetSummaryByID(ctx context.Context, id string) (*models.Summary, error)
	DeleteByTranscriptionID(ctx context.Context, transcriptionID string) error
	ListSummaryWidgetsByUser(ctx context.Context, userID uint) ([]models.SummaryWidget, error)
	ListEnabledSummaryWidgets(ctx context.Context, userID uint) ([]models.SummaryWidget, error)
	FindSummaryWidgetByIDForUser(ctx context.Context, id string, userID uint) (*models.SummaryWidget, error)
	CreateSummaryWidget(ctx context.Context, widget *models.SummaryWidget) error
	UpdateSummaryWidget(ctx context.Context, widget *models.SummaryWidget) error
	DeleteSummaryWidget(ctx context.Context, id string, userID uint) error
	EnqueueSummaryWidgetRuns(ctx context.Context, summary *models.Summary, widgets []models.SummaryWidget, model string, provider string) ([]models.SummaryWidgetRun, error)
	ClaimNextPendingSummaryWidgetRun(ctx context.Context, now time.Time) (*models.SummaryWidgetRun, error)
	CompleteSummaryWidgetRun(ctx context.Context, id string, output string, truncated bool, contextWindow int, inputCharacters int, completedAt time.Time) error
	FailSummaryWidgetRun(ctx context.Context, id string, message string, failedAt time.Time) error
	RecoverProcessingSummaryWidgetRuns(ctx context.Context) (int64, error)
	ListSummaryWidgetRunsByTranscription(ctx context.Context, transcriptionID string, userID uint) ([]models.SummaryWidgetRun, error)
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

func (r *summaryRepository) ListCompletedSummariesForTitleGeneration(ctx context.Context, limit int) ([]models.Summary, error) {
	if limit <= 0 {
		limit = 25
	}
	var summaries []models.Summary
	err := r.db.WithContext(ctx).
		Model(&models.Summary{}).
		Joins("JOIN transcriptions AS transcription_jobs ON transcription_jobs.id = summaries.transcription_id").
		Joins("LEFT JOIN transcriptions AS recording_jobs ON recording_jobs.id = transcription_jobs.source_file_hash").
		Where("summaries.status = ? AND TRIM(COALESCE(summaries.content, '')) <> ''", "completed").
		Where("COALESCE(recording_jobs.llm_title_generated, transcription_jobs.llm_title_generated) = ?", false).
		Order("COALESCE(summaries.completed_at, summaries.updated_at) DESC").
		Limit(limit).
		Find(&summaries).Error
	return summaries, err
}

func (r *summaryRepository) GetCompletedOutlineRun(ctx context.Context, summaryID string, transcriptionID string, userID uint) (*models.SummaryWidgetRun, error) {
	var run models.SummaryWidgetRun
	err := r.db.WithContext(ctx).
		Where("summary_id = ? AND transcription_id = ? AND user_id = ?", summaryID, transcriptionID, userID).
		Where("status = ? AND TRIM(COALESCE(output, '')) <> ''", "completed").
		Where("(LOWER(TRIM(display_title)) = ? OR LOWER(TRIM(widget_name)) = ?)", "outline", "outline").
		Order("COALESCE(completed_at, updated_at) DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *summaryRepository) GetLatestSummary(ctx context.Context, transcriptionID string) (*models.Summary, error) {
	var summary models.Summary
	err := r.db.WithContext(ctx).Where("transcription_id = ?", transcriptionID).Order("created_at DESC").First(&summary).Error
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

func (r *summaryRepository) GetSummaryByID(ctx context.Context, id string) (*models.Summary, error) {
	var summary models.Summary
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&summary).Error; err != nil {
		return nil, err
	}
	return &summary, nil
}

func (r *summaryRepository) DeleteByTranscriptionID(ctx context.Context, transcriptionID string) error {
	return r.db.WithContext(ctx).Where("transcription_id = ?", transcriptionID).Delete(&models.Summary{}).Error
}

func (r *summaryRepository) ListSummaryWidgetsByUser(ctx context.Context, userID uint) ([]models.SummaryWidget, error) {
	var widgets []models.SummaryWidget
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&widgets).Error
	return widgets, err
}

func (r *summaryRepository) ListEnabledSummaryWidgets(ctx context.Context, userID uint) ([]models.SummaryWidget, error) {
	var widgets []models.SummaryWidget
	err := r.db.WithContext(ctx).Where("user_id = ? AND enabled = ?", userID, true).Order("created_at ASC").Find(&widgets).Error
	return widgets, err
}

func (r *summaryRepository) FindSummaryWidgetByIDForUser(ctx context.Context, id string, userID uint) (*models.SummaryWidget, error) {
	var widget models.SummaryWidget
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&widget).Error; err != nil {
		return nil, err
	}
	return &widget, nil
}

func (r *summaryRepository) CreateSummaryWidget(ctx context.Context, widget *models.SummaryWidget) error {
	return r.db.WithContext(ctx).Create(widget).Error
}

func (r *summaryRepository) UpdateSummaryWidget(ctx context.Context, widget *models.SummaryWidget) error {
	return r.db.WithContext(ctx).Save(widget).Error
}

func (r *summaryRepository) DeleteSummaryWidget(ctx context.Context, id string, userID uint) error {
	return r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.SummaryWidget{}).Error
}

func (r *summaryRepository) EnqueueSummaryWidgetRuns(ctx context.Context, summary *models.Summary, widgets []models.SummaryWidget, model string, provider string) ([]models.SummaryWidgetRun, error) {
	if summary == nil || len(widgets) == 0 {
		return nil, nil
	}
	runs := make([]models.SummaryWidgetRun, 0, len(widgets))
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, widget := range widgets {
			var existing models.SummaryWidgetRun
			result := tx.Where("summary_id = ? AND widget_id = ? AND user_id = ?", summary.ID, widget.ID, summary.UserID).
				Limit(1).
				Find(&existing)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				runs = append(runs, existing)
				continue
			}
			run := models.SummaryWidgetRun{
				SummaryID:       summary.ID,
				TranscriptionID: summary.TranscriptionID,
				WidgetID:        widget.ID,
				UserID:          summary.UserID,
				WidgetName:      widget.Name,
				DisplayTitle:    widget.DisplayTitle,
				ContextSource:   widget.ContextSource,
				RenderMarkdown:  widget.RenderMarkdown,
				Model:           model,
				Provider:        provider,
				Status:          "pending",
				Output:          "",
			}
			if err := tx.Create(&run).Error; err != nil {
				return err
			}
			runs = append(runs, run)
		}
		return nil
	})
	return runs, err
}

func (r *summaryRepository) ClaimNextPendingSummaryWidgetRun(ctx context.Context, now time.Time) (*models.SummaryWidgetRun, error) {
	var run models.SummaryWidgetRun
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Preload("Widget").Where("status = ?", "pending").Order("created_at ASC").Limit(1).Find(&run)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		updateResult := tx.Model(&models.SummaryWidgetRun{}).
			Where("id = ? AND status = ?", run.ID, "pending").
			Updates(map[string]any{"status": "processing", "started_at": now})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		run.Status = "processing"
		run.StartedAt = &now
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *summaryRepository) CompleteSummaryWidgetRun(ctx context.Context, id string, output string, truncated bool, contextWindow int, inputCharacters int, completedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.SummaryWidgetRun{}).
		Where("id = ? AND status = ?", id, "processing").
		Updates(map[string]any{
			"output":            output,
			"status":            "completed",
			"context_truncated": truncated,
			"context_window":    contextWindow,
			"input_characters":  inputCharacters,
			"completed_at":      completedAt,
			"error_message":     nil,
		}).Error
}

func (r *summaryRepository) FailSummaryWidgetRun(ctx context.Context, id string, message string, failedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.SummaryWidgetRun{}).
		Where("id = ? AND status = ?", id, "processing").
		Updates(map[string]any{
			"status":        "failed",
			"error_message": message,
			"failed_at":     failedAt,
		}).Error
}

func (r *summaryRepository) RecoverProcessingSummaryWidgetRuns(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.SummaryWidgetRun{}).
		Where("status = ?", "processing").
		Update("status", "pending")
	return result.RowsAffected, result.Error
}

func (r *summaryRepository) ListSummaryWidgetRunsByTranscription(ctx context.Context, transcriptionID string, userID uint) ([]models.SummaryWidgetRun, error) {
	var runs []models.SummaryWidgetRun
	err := r.db.WithContext(ctx).
		Where("transcription_id = ? AND user_id = ?", transcriptionID, userID).
		Order("created_at ASC").
		Find(&runs).Error
	return runs, err
}

// AnnotationRepository handles transcript highlights and notes.
type AnnotationRepository interface {
	CreateAnnotation(ctx context.Context, annotation *models.TranscriptAnnotation) error
	CreateAnnotationWithEntry(ctx context.Context, annotation *models.TranscriptAnnotation, entry *models.TranscriptAnnotationEntry) error
	FindAnnotationForUser(ctx context.Context, userID uint, transcriptionID string, annotationID string) (*models.TranscriptAnnotation, error)
	ListAnnotationsForTranscription(ctx context.Context, userID uint, transcriptionID string, kind *models.AnnotationKind, updatedAfter *time.Time, offset int, limit int) ([]models.TranscriptAnnotation, int64, error)
	UpdateAnnotation(ctx context.Context, annotation *models.TranscriptAnnotation) error
	UpdateAnnotationStatus(ctx context.Context, userID uint, transcriptionID string, annotationID string, status string) error
	SoftDeleteAnnotation(ctx context.Context, userID uint, transcriptionID string, annotationID string) error
	CreateAnnotationEntry(ctx context.Context, entry *models.TranscriptAnnotationEntry) error
	FindAnnotationEntryForUser(ctx context.Context, userID uint, annotationID string, entryID string) (*models.TranscriptAnnotationEntry, error)
	UpdateAnnotationEntry(ctx context.Context, entry *models.TranscriptAnnotationEntry) error
	SoftDeleteAnnotationEntry(ctx context.Context, userID uint, annotationID string, entryID string) error
}

type annotationRepository struct {
	db *gorm.DB
}

func NewAnnotationRepository(db *gorm.DB) AnnotationRepository {
	return &annotationRepository{db: db}
}

func (r *annotationRepository) CreateAnnotation(ctx context.Context, annotation *models.TranscriptAnnotation) error {
	return r.db.WithContext(ctx).Create(annotation).Error
}

func (r *annotationRepository) CreateAnnotationWithEntry(ctx context.Context, annotation *models.TranscriptAnnotation, entry *models.TranscriptAnnotationEntry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(annotation).Error; err != nil {
			return err
		}
		entry.AnnotationID = annotation.ID
		entry.UserID = annotation.UserID
		return tx.Create(entry).Error
	})
}

func (r *annotationRepository) FindAnnotationForUser(ctx context.Context, userID uint, transcriptionID string, annotationID string) (*models.TranscriptAnnotation, error) {
	var annotation models.TranscriptAnnotation
	if err := r.db.WithContext(ctx).
		Preload("Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC, id ASC")
		}).
		Where("id = ? AND user_id = ? AND transcription_id = ?", annotationID, userID, transcriptionID).
		First(&annotation).Error; err != nil {
		return nil, err
	}
	return &annotation, nil
}

func (r *annotationRepository) ListAnnotationsForTranscription(ctx context.Context, userID uint, transcriptionID string, kind *models.AnnotationKind, updatedAfter *time.Time, offset int, limit int) ([]models.TranscriptAnnotation, int64, error) {
	var annotations []models.TranscriptAnnotation
	var count int64
	query := r.db.WithContext(ctx).
		Model(&models.TranscriptAnnotation{}).
		Where("user_id = ? AND transcription_id = ?", userID, transcriptionID)
	if kind != nil {
		query = query.Where("kind = ?", *kind)
	}
	if updatedAfter != nil {
		query = query.Where("updated_at > ?", *updatedAfter)
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := query.
		Preload("Entries", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC, id ASC")
		}).
		Order("created_at DESC, id DESC").
		Offset(offset).
		Limit(limit).
		Find(&annotations).Error; err != nil {
		return nil, 0, err
	}
	return annotations, count, nil
}

func (r *annotationRepository) UpdateAnnotation(ctx context.Context, annotation *models.TranscriptAnnotation) error {
	result := r.db.WithContext(ctx).
		Model(annotation).
		Where("user_id = ? AND transcription_id = ?", annotation.UserID, annotation.TranscriptionID).
		Select(
			"Content",
			"Color",
			"Quote",
			"AnchorStartMS",
			"AnchorEndMS",
			"AnchorStartWord",
			"AnchorEndWord",
			"AnchorStartChar",
			"AnchorEndChar",
			"AnchorTextHash",
			"Status",
			"MetadataJSON",
		).
		Updates(annotation)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *annotationRepository) UpdateAnnotationStatus(ctx context.Context, userID uint, transcriptionID string, annotationID string, status string) error {
	result := r.db.WithContext(ctx).
		Session(&gorm.Session{SkipHooks: true}).
		Model(&models.TranscriptAnnotation{}).
		Where("id = ? AND user_id = ? AND transcription_id = ?", annotationID, userID, transcriptionID).
		Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *annotationRepository) SoftDeleteAnnotation(ctx context.Context, userID uint, transcriptionID string, annotationID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND transcription_id = ?", annotationID, userID, transcriptionID).
		Delete(&models.TranscriptAnnotation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *annotationRepository) CreateAnnotationEntry(ctx context.Context, entry *models.TranscriptAnnotationEntry) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *annotationRepository) FindAnnotationEntryForUser(ctx context.Context, userID uint, annotationID string, entryID string) (*models.TranscriptAnnotationEntry, error) {
	var entry models.TranscriptAnnotationEntry
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND annotation_id = ?", entryID, userID, annotationID).
		First(&entry).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *annotationRepository) UpdateAnnotationEntry(ctx context.Context, entry *models.TranscriptAnnotationEntry) error {
	result := r.db.WithContext(ctx).
		Model(entry).
		Where("id = ? AND user_id = ? AND annotation_id = ?", entry.ID, entry.UserID, entry.AnnotationID).
		Select("Content").
		Updates(entry)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *annotationRepository) SoftDeleteAnnotationEntry(ctx context.Context, userID uint, annotationID string, entryID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND annotation_id = ?", entryID, userID, annotationID).
		Delete(&models.TranscriptAnnotationEntry{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// TagRepository handles audio tag persistence and tag assignments.
type TagRepository interface {
	CreateTag(ctx context.Context, tag *models.AudioTag) error
	FindTagForUser(ctx context.Context, userID uint, tagID string) (*models.AudioTag, error)
	FindTagForUserByNormalizedName(ctx context.Context, userID uint, normalizedName string) (*models.AudioTag, error)
	ListTagsForUser(ctx context.Context, userID uint, search string, offset int, limit int) ([]models.AudioTag, int64, error)
	UpdateTag(ctx context.Context, tag *models.AudioTag) error
	SoftDeleteTag(ctx context.Context, userID uint, tagID string) error
	ListTagsForTranscription(ctx context.Context, userID uint, transcriptionID string) ([]models.AudioTag, error)
	ReplaceTagsForTranscription(ctx context.Context, userID uint, transcriptionID string, tagIDs []string) error
	AddTagToTranscription(ctx context.Context, userID uint, transcriptionID string, tagID string) error
	RemoveTagFromTranscription(ctx context.Context, userID uint, transcriptionID string, tagID string) error
	ListTranscriptionIDsByTags(ctx context.Context, userID uint, tagIDs []string, matchAll bool) ([]string, error)
}

type tagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

func (r *tagRepository) CreateTag(ctx context.Context, tag *models.AudioTag) error {
	return r.db.WithContext(ctx).Create(tag).Error
}

func (r *tagRepository) FindTagForUser(ctx context.Context, userID uint, tagID string) (*models.AudioTag, error) {
	var tag models.AudioTag
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", tagID, userID).
		First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *tagRepository) FindTagForUserByNormalizedName(ctx context.Context, userID uint, normalizedName string) (*models.AudioTag, error) {
	var tag models.AudioTag
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND normalized_name = ?", userID, normalizedName).
		First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *tagRepository) ListTagsForUser(ctx context.Context, userID uint, search string, offset int, limit int) ([]models.AudioTag, int64, error) {
	var tags []models.AudioTag
	var count int64
	query := r.db.WithContext(ctx).Model(&models.AudioTag{}).Where("user_id = ?", userID)
	if search != "" {
		searchLike := "%" + search + "%"
		query = query.Where("name LIKE ? OR normalized_name LIKE ?", searchLike, searchLike)
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("normalized_name ASC, id ASC").Offset(offset).Limit(limit).Find(&tags).Error; err != nil {
		return nil, 0, err
	}
	return tags, count, nil
}

func (r *tagRepository) UpdateTag(ctx context.Context, tag *models.AudioTag) error {
	result := r.db.WithContext(ctx).
		Model(tag).
		Where("id = ? AND user_id = ?", tag.ID, tag.UserID).
		Select("Name", "NormalizedName", "Color", "Description", "MetadataJSON").
		Updates(tag)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *tagRepository) SoftDeleteTag(ctx context.Context, userID uint, tagID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ? AND user_id = ?", tagID, userID).Delete(&models.AudioTag{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("user_id = ? AND tag_id = ?", userID, tagID).Delete(&models.AudioTagAssignment{}).Error
	})
}

func (r *tagRepository) ListTagsForTranscription(ctx context.Context, userID uint, transcriptionID string) ([]models.AudioTag, error) {
	var tags []models.AudioTag
	err := r.db.WithContext(ctx).
		Table("audio_tags").
		Select("audio_tags.*").
		Joins("JOIN audio_tag_assignments ON audio_tag_assignments.tag_id = audio_tags.id").
		Where("audio_tags.user_id = ? AND audio_tag_assignments.user_id = ? AND audio_tag_assignments.transcription_id = ?", userID, userID, transcriptionID).
		Where("audio_tags.deleted_at IS NULL AND audio_tag_assignments.deleted_at IS NULL").
		Order("audio_tags.normalized_name ASC, audio_tags.id ASC").
		Find(&tags).Error
	return tags, err
}

func (r *tagRepository) ReplaceTagsForTranscription(ctx context.Context, userID uint, transcriptionID string, tagIDs []string) error {
	desired := make(map[string]struct{}, len(tagIDs))
	for _, tagID := range tagIDs {
		desired[tagID] = struct{}{}
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var assignments []models.AudioTagAssignment
		if err := tx.Where("user_id = ? AND transcription_id = ?", userID, transcriptionID).Find(&assignments).Error; err != nil {
			return err
		}
		active := make(map[string]struct{}, len(assignments))
		for _, assignment := range assignments {
			if _, ok := desired[assignment.TagID]; !ok {
				if err := tx.Where("id = ? AND user_id = ?", assignment.ID, userID).Delete(&models.AudioTagAssignment{}).Error; err != nil {
					return err
				}
				continue
			}
			active[assignment.TagID] = struct{}{}
		}
		for _, tagID := range tagIDs {
			if _, ok := active[tagID]; ok {
				continue
			}
			if err := tx.Create(&models.AudioTagAssignment{
				UserID:          userID,
				TagID:           tagID,
				TranscriptionID: transcriptionID,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *tagRepository) AddTagToTranscription(ctx context.Context, userID uint, transcriptionID string, tagID string) error {
	var existing models.AudioTagAssignment
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND transcription_id = ? AND tag_id = ?", userID, transcriptionID, tagID).
		First(&existing)
	if result.Error == nil {
		return nil
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	return r.db.WithContext(ctx).Create(&models.AudioTagAssignment{
		UserID:          userID,
		TagID:           tagID,
		TranscriptionID: transcriptionID,
	}).Error
}

func (r *tagRepository) RemoveTagFromTranscription(ctx context.Context, userID uint, transcriptionID string, tagID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND transcription_id = ? AND tag_id = ?", userID, transcriptionID, tagID).
		Delete(&models.AudioTagAssignment{}).Error
}

func (r *tagRepository) ListTranscriptionIDsByTags(ctx context.Context, userID uint, tagIDs []string, matchAll bool) ([]string, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}
	type row struct {
		TranscriptionID string `gorm:"column:transcription_id"`
	}
	var rows []row
	query := r.db.WithContext(ctx).
		Model(&models.AudioTagAssignment{}).
		Select("transcription_id").
		Where("user_id = ? AND tag_id IN ?", userID, tagIDs).
		Group("transcription_id")
	if matchAll {
		query = query.Having("COUNT(DISTINCT tag_id) = ?", len(tagIDs))
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TranscriptionID)
	}
	return ids, nil
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
