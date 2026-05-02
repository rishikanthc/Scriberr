package repository

import (
	"context"
	"time"

	"scriberr/internal/models"

	"gorm.io/gorm"
)

type RecordingRepository interface {
	CreateSession(ctx context.Context, session *models.RecordingSession) error
	FindSessionForUser(ctx context.Context, userID uint, sessionID string) (*models.RecordingSession, error)
	ListSessionsForUser(ctx context.Context, userID uint, offset, limit int) ([]models.RecordingSession, int64, error)
	AddChunk(ctx context.Context, chunk *models.RecordingChunk) error
	FindChunk(ctx context.Context, userID uint, sessionID string, chunkIndex int) (*models.RecordingChunk, error)
	ListChunks(ctx context.Context, userID uint, sessionID string) ([]models.RecordingChunk, error)
	MarkStopping(ctx context.Context, userID uint, sessionID string, finalIndex int, durationMs *int64, autoTranscribe bool, now time.Time) error
	ClaimNextFinalization(ctx context.Context, workerID string, leaseUntil time.Time) (*models.RecordingSession, error)
	RenewFinalizationClaim(ctx context.Context, sessionID, workerID string, leaseUntil time.Time) error
	MarkFinalizing(ctx context.Context, sessionID, workerID string, leaseUntil time.Time, now time.Time) error
	CompleteFinalization(ctx context.Context, sessionID, workerID string, fileID string, transcriptionID *string, completedAt time.Time) error
	FailFinalization(ctx context.Context, sessionID, workerID string, message string, failedAt time.Time) error
	CancelSession(ctx context.Context, userID uint, sessionID string, canceledAt time.Time) error
	ExpireAbandonedSessions(ctx context.Context, now time.Time) (int64, error)
	RecoverExpiredFinalizationClaims(ctx context.Context, now time.Time) (int64, error)
	ListArtifactCleanupCandidates(ctx context.Context, now time.Time, failedRetention time.Duration, limit int) ([]models.RecordingSession, error)
	MarkTemporaryArtifactsCleaned(ctx context.Context, sessionID string, cleanedAt time.Time) error
}

type RecordingHandoffRepository interface {
	CreateRecordingFileAndTranscription(ctx context.Context, file *models.TranscriptionJob, transcription *models.TranscriptionJob) error
}

type recordingRepository struct {
	db *gorm.DB
}

func NewRecordingRepository(db *gorm.DB) RecordingRepository {
	return &recordingRepository{db: db}
}

func (r *recordingRepository) CreateSession(ctx context.Context, session *models.RecordingSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *recordingRepository) FindSessionForUser(ctx context.Context, userID uint, sessionID string) (*models.RecordingSession, error) {
	var session models.RecordingSession
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *recordingRepository) ListSessionsForUser(ctx context.Context, userID uint, offset, limit int) ([]models.RecordingSession, int64, error) {
	var sessions []models.RecordingSession
	var count int64
	query := r.db.WithContext(ctx).Model(&models.RecordingSession{}).Where("user_id = ?", userID)
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&sessions).Error; err != nil {
		return nil, 0, err
	}
	return sessions, count, nil
}

func (r *recordingRepository) AddChunk(ctx context.Context, chunk *models.RecordingChunk) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var session models.RecordingSession
		if err := tx.Select("id", "user_id", "status").Where("id = ? AND user_id = ?", chunk.SessionID, chunk.UserID).First(&session).Error; err != nil {
			return err
		}
		if session.Status != models.RecordingStatusRecording {
			return gorm.ErrInvalidData
		}
		if err := tx.Create(chunk).Error; err != nil {
			return err
		}
		result := tx.Model(&models.RecordingSession{}).
			Where("id = ? AND user_id = ? AND status = ?", chunk.SessionID, chunk.UserID, models.RecordingStatusRecording).
			UpdateColumns(map[string]any{
				"received_chunks": gorm.Expr("received_chunks + ?", 1),
				"received_bytes":  gorm.Expr("received_bytes + ?", chunk.SizeBytes),
				"progress_stage":  "recording",
			})
		return rowsAffectedOrNotFound(result)
	})
}

func (r *recordingRepository) FindChunk(ctx context.Context, userID uint, sessionID string, chunkIndex int) (*models.RecordingChunk, error) {
	var chunk models.RecordingChunk
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND session_id = ? AND chunk_index = ?", userID, sessionID, chunkIndex).
		First(&chunk).Error; err != nil {
		return nil, err
	}
	return &chunk, nil
}

func (r *recordingRepository) ListChunks(ctx context.Context, userID uint, sessionID string) ([]models.RecordingChunk, error) {
	var chunks []models.RecordingChunk
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Order("chunk_index ASC").
		Find(&chunks).Error
	return chunks, err
}

func (r *recordingRepository) MarkStopping(ctx context.Context, userID uint, sessionID string, finalIndex int, durationMs *int64, autoTranscribe bool, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.RecordingSession{}).
		Where("id = ? AND user_id = ? AND status IN ?", sessionID, userID, []models.RecordingStatus{models.RecordingStatusRecording, models.RecordingStatusFailed}).
		UpdateColumns(map[string]any{
			"status":               models.RecordingStatusStopping,
			"expected_final_index": finalIndex,
			"duration_ms":          durationMs,
			"auto_transcribe":      autoTranscribe,
			"stopped_at":           now,
			"finalize_queued_at":   now,
			"finalize_started_at":  nil,
			"failed_at":            nil,
			"last_error":           nil,
			"progress":             0.75,
			"progress_stage":       "queued_for_finalization",
			"claimed_by":           nil,
			"claim_expires_at":     nil,
		})
	return rowsAffectedOrNotFound(result)
}

func (r *recordingRepository) ClaimNextFinalization(ctx context.Context, workerID string, leaseUntil time.Time) (*models.RecordingSession, error) {
	var claimed models.RecordingSession
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidate models.RecordingSession
		result := tx.
			Where("status = ?", models.RecordingStatusStopping).
			Order("finalize_queued_at ASC, created_at ASC, id ASC").
			Limit(1).
			Find(&candidate)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		now := time.Now()
		result = tx.Model(&models.RecordingSession{}).
			Where("id = ? AND status = ?", candidate.ID, models.RecordingStatusStopping).
			UpdateColumns(map[string]any{
				"status":              models.RecordingStatusFinalizing,
				"finalize_started_at": now,
				"progress":            0.80,
				"progress_stage":      "finalizing",
				"claimed_by":          workerID,
				"claim_expires_at":    leaseUntil,
			})
		if err := rowsAffectedOrNotFound(result); err != nil {
			return err
		}
		return tx.First(&claimed, "id = ?", candidate.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &claimed, nil
}

func (r *recordingRepository) RenewFinalizationClaim(ctx context.Context, sessionID, workerID string, leaseUntil time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.RecordingSession{}).
		Where("id = ? AND status = ? AND claimed_by = ?", sessionID, models.RecordingStatusFinalizing, workerID).
		UpdateColumn("claim_expires_at", leaseUntil)
	return rowsAffectedOrNotFound(result)
}

func (r *recordingRepository) MarkFinalizing(ctx context.Context, sessionID, workerID string, leaseUntil time.Time, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.RecordingSession{}).
		Where("id = ? AND status = ?", sessionID, models.RecordingStatusStopping).
		UpdateColumns(map[string]any{
			"status":              models.RecordingStatusFinalizing,
			"finalize_started_at": now,
			"progress":            0.80,
			"progress_stage":      "finalizing",
			"claimed_by":          workerID,
			"claim_expires_at":    leaseUntil,
		})
	return rowsAffectedOrNotFound(result)
}

func (r *recordingRepository) CompleteFinalization(ctx context.Context, sessionID, workerID string, fileID string, transcriptionID *string, completedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.RecordingSession{}).
		Where("id = ? AND status = ? AND claimed_by = ?", sessionID, models.RecordingStatusFinalizing, workerID).
		UpdateColumns(map[string]any{
			"status":           models.RecordingStatusReady,
			"file_id":          fileID,
			"transcription_id": transcriptionID,
			"completed_at":     completedAt,
			"failed_at":        nil,
			"last_error":       nil,
			"progress":         1.0,
			"progress_stage":   "ready",
			"claimed_by":       nil,
			"claim_expires_at": nil,
		})
	return rowsAffectedOrNotFound(result)
}

func (r *recordingRepository) FailFinalization(ctx context.Context, sessionID, workerID string, message string, failedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.RecordingSession{}).
		Where("id = ? AND status = ? AND claimed_by = ?", sessionID, models.RecordingStatusFinalizing, workerID).
		UpdateColumns(map[string]any{
			"status":           models.RecordingStatusFailed,
			"failed_at":        failedAt,
			"last_error":       message,
			"progress_stage":   "failed",
			"claimed_by":       nil,
			"claim_expires_at": nil,
		})
	return rowsAffectedOrNotFound(result)
}

func (r *recordingRepository) CancelSession(ctx context.Context, userID uint, sessionID string, canceledAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.RecordingSession{}).
		Where("id = ? AND user_id = ? AND status NOT IN ?", sessionID, userID, []models.RecordingStatus{models.RecordingStatusReady, models.RecordingStatusCanceled, models.RecordingStatusExpired}).
		UpdateColumns(map[string]any{
			"status":           models.RecordingStatusCanceled,
			"completed_at":     canceledAt,
			"progress_stage":   "canceled",
			"claimed_by":       nil,
			"claim_expires_at": nil,
		})
	return rowsAffectedOrNotFound(result)
}

func (r *recordingRepository) ExpireAbandonedSessions(ctx context.Context, now time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.RecordingSession{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at <= ?", models.RecordingStatusRecording, now).
		UpdateColumns(map[string]any{
			"status":           models.RecordingStatusExpired,
			"completed_at":     now,
			"progress_stage":   "expired",
			"claimed_by":       nil,
			"claim_expires_at": nil,
		})
	return result.RowsAffected, result.Error
}

func (r *recordingRepository) RecoverExpiredFinalizationClaims(ctx context.Context, now time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.RecordingSession{}).
		Where("status = ? AND claim_expires_at IS NOT NULL AND claim_expires_at <= ?", models.RecordingStatusFinalizing, now).
		UpdateColumns(map[string]any{
			"status":             models.RecordingStatusStopping,
			"finalize_queued_at": now,
			"progress_stage":     "recovered",
			"claimed_by":         nil,
			"claim_expires_at":   nil,
		})
	return result.RowsAffected, result.Error
}

func (r *recordingRepository) ListArtifactCleanupCandidates(ctx context.Context, now time.Time, failedRetention time.Duration, limit int) ([]models.RecordingSession, error) {
	if limit <= 0 {
		limit = 100
	}
	failedBefore := now.Add(-failedRetention)
	var sessions []models.RecordingSession
	err := r.db.WithContext(ctx).
		Where("temporary_artifacts_cleaned_at IS NULL").
		Where(
			r.db.Where("status IN ?", []models.RecordingStatus{
				models.RecordingStatusReady,
				models.RecordingStatusCanceled,
				models.RecordingStatusExpired,
			}).Or("status = ? AND failed_at IS NOT NULL AND failed_at <= ?", models.RecordingStatusFailed, failedBefore),
		).
		Order("updated_at ASC, id ASC").
		Limit(limit).
		Find(&sessions).Error
	return sessions, err
}

func (r *recordingRepository) MarkTemporaryArtifactsCleaned(ctx context.Context, sessionID string, cleanedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.RecordingSession{}).
		Where("id = ? AND temporary_artifacts_cleaned_at IS NULL", sessionID).
		UpdateColumn("temporary_artifacts_cleaned_at", cleanedAt)
	return rowsAffectedOrNotFound(result)
}

func rowsAffectedOrNotFound(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
