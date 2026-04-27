package repository

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openJobQueueTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "scriberr.db"))
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func createQueueTestUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
	user := models.User{Username: "queue-user-" + time.Now().Format("150405.000000000"), Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func createQueueTestJob(t *testing.T, db *gorm.DB, userID uint, id string, status models.JobStatus, createdAt time.Time) models.TranscriptionJob {
	t.Helper()
	title := id
	sourceID := "file-" + id
	job := models.TranscriptionJob{
		ID:             id,
		UserID:         userID,
		Title:          &title,
		Status:         status,
		AudioPath:      filepath.Join(t.TempDir(), id+".wav"),
		SourceFileName: id + ".wav",
		SourceFileHash: &sourceID,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	require.NoError(t, db.Create(&job).Error)
	return job
}

func TestJobRepositoryQueueSchemaIndexesExist(t *testing.T) {
	db := openJobQueueTestDB(t)

	assert.True(t, db.Migrator().HasColumn(&models.TranscriptionJob{}, "queued_at"))
	assert.True(t, db.Migrator().HasColumn(&models.TranscriptionJob{}, "started_at"))
	assert.True(t, db.Migrator().HasColumn(&models.TranscriptionJob{}, "failed_at"))
	assert.True(t, db.Migrator().HasColumn(&models.TranscriptionJob{}, "progress"))
	assert.True(t, db.Migrator().HasColumn(&models.TranscriptionJob{}, "progress_stage"))
	assert.True(t, db.Migrator().HasColumn(&models.TranscriptionJob{}, "claimed_by"))
	assert.True(t, db.Migrator().HasColumn(&models.TranscriptionJob{}, "claim_expires_at"))
	assert.True(t, db.Migrator().HasColumn(&models.TranscriptionJob{}, "engine_id"))
	assert.True(t, db.Migrator().HasIndex(&models.TranscriptionJob{}, "idx_transcriptions_queue_claim"))
	assert.True(t, db.Migrator().HasIndex(&models.TranscriptionJob{}, "idx_transcriptions_claim_expires_at"))
}

func TestJobRepositoryEnqueueAndClaimFIFO(t *testing.T) {
	db := openJobQueueTestDB(t)
	user := createQueueTestUser(t, db)
	repo := NewJobRepository(db)
	base := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
	newer := createQueueTestJob(t, db, user.ID, "job-newer", models.StatusUploaded, base.Add(2*time.Minute))
	older := createQueueTestJob(t, db, user.ID, "job-older", models.StatusUploaded, base)

	require.NoError(t, repo.EnqueueTranscription(context.Background(), newer.ID, base.Add(20*time.Second)))
	require.NoError(t, repo.EnqueueTranscription(context.Background(), older.ID, base.Add(10*time.Second)))

	claimed, err := repo.ClaimNextTranscription(context.Background(), "worker-a", base.Add(10*time.Minute))
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, older.ID, claimed.ID)
	assert.Equal(t, user.ID, claimed.UserID)
	assert.Equal(t, models.StatusProcessing, claimed.Status)
	assert.NotNil(t, claimed.StartedAt)
	assert.Equal(t, 0.05, claimed.Progress)
	assert.Equal(t, "preparing", claimed.ProgressStage)
	assert.Equal(t, "worker-a", *claimed.ClaimedBy)
	require.NotNil(t, claimed.ClaimExpiresAt)
	assert.Equal(t, base.Add(10*time.Minute).UnixNano(), claimed.ClaimExpiresAt.Truncate(time.Millisecond).UnixNano())

	var persistedOlder models.TranscriptionJob
	require.NoError(t, db.First(&persistedOlder, "id = ?", older.ID).Error)
	assert.Equal(t, models.StatusProcessing, persistedOlder.Status)
	assert.NotNil(t, persistedOlder.QueuedAt)

	claimed, err = repo.ClaimNextTranscription(context.Background(), "worker-b", base.Add(10*time.Minute))
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, newer.ID, claimed.ID)
}

func TestJobRepositoryClaimNextReturnsNotFoundWhenQueueEmpty(t *testing.T) {
	db := openJobQueueTestDB(t)
	repo := NewJobRepository(db)

	claimed, err := repo.ClaimNextTranscription(context.Background(), "worker-a", time.Now().Add(time.Minute))

	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.Nil(t, claimed)
}

func TestJobRepositoryConcurrentClaimsDoNotDuplicateJobs(t *testing.T) {
	db := openJobQueueTestDB(t)
	user := createQueueTestUser(t, db)
	repo := NewJobRepository(db)
	now := time.Now().Truncate(time.Millisecond)
	job := createQueueTestJob(t, db, user.ID, "job-only", models.StatusUploaded, now)
	require.NoError(t, repo.EnqueueTranscription(context.Background(), job.ID, now))

	var wg sync.WaitGroup
	claimedIDs := make(chan string, 2)
	errs := make(chan error, 2)
	for _, workerID := range []string{"worker-a", "worker-b"} {
		wg.Add(1)
		go func(workerID string) {
			defer wg.Done()
			claimed, err := repo.ClaimNextTranscription(context.Background(), workerID, now.Add(time.Minute))
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return
				}
				errs <- err
				return
			}
			if claimed != nil {
				claimedIDs <- claimed.ID
			}
		}(workerID)
	}
	wg.Wait()
	close(claimedIDs)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	var ids []string
	for id := range claimedIDs {
		ids = append(ids, id)
	}
	require.Len(t, ids, 1)
	assert.Equal(t, job.ID, ids[0])
}

func TestJobRepositoryRenewClaimRequiresOwningWorker(t *testing.T) {
	db := openJobQueueTestDB(t)
	user := createQueueTestUser(t, db)
	repo := NewJobRepository(db)
	now := time.Now().Truncate(time.Millisecond)
	job := createQueueTestJob(t, db, user.ID, "job-renew", models.StatusUploaded, now)
	require.NoError(t, repo.EnqueueTranscription(context.Background(), job.ID, now))
	claimed, err := repo.ClaimNextTranscription(context.Background(), "worker-a", now.Add(time.Minute))
	require.NoError(t, err)

	err = repo.RenewClaim(context.Background(), claimed.ID, "worker-b", now.Add(2*time.Minute))
	require.Error(t, err)

	require.NoError(t, repo.RenewClaim(context.Background(), claimed.ID, "worker-a", now.Add(3*time.Minute)))
	var persisted models.TranscriptionJob
	require.NoError(t, db.First(&persisted, "id = ?", claimed.ID).Error)
	require.NotNil(t, persisted.ClaimExpiresAt)
	assert.Equal(t, now.Add(3*time.Minute).UnixNano(), persisted.ClaimExpiresAt.Truncate(time.Millisecond).UnixNano())
}

func TestJobRepositoryRecoverOrphanedProcessingJobs(t *testing.T) {
	db := openJobQueueTestDB(t)
	user := createQueueTestUser(t, db)
	repo := NewJobRepository(db)
	now := time.Now().Truncate(time.Millisecond)
	job := createQueueTestJob(t, db, user.ID, "job-recover", models.StatusProcessing, now)
	workerID := "old-worker"
	claimExpiry := now.Add(time.Hour)
	require.NoError(t, db.Model(&models.TranscriptionJob{}).Where("id = ?", job.ID).Updates(map[string]any{
		"progress":         0.5,
		"progress_stage":   "transcribing",
		"claimed_by":       workerID,
		"claim_expires_at": claimExpiry,
	}).Error)

	recovered, err := repo.RecoverOrphanedProcessing(context.Background(), now.Add(time.Second))
	require.NoError(t, err)
	assert.Equal(t, int64(1), recovered)

	var persisted models.TranscriptionJob
	require.NoError(t, db.First(&persisted, "id = ?", job.ID).Error)
	assert.Equal(t, models.StatusPending, persisted.Status)
	assert.Equal(t, "recovered", persisted.ProgressStage)
	assert.Nil(t, persisted.ClaimedBy)
	assert.Nil(t, persisted.ClaimExpiresAt)
	require.NotNil(t, persisted.QueuedAt)
}

func TestJobRepositoryProgressAndTerminalTransitionsUpdateLatestExecution(t *testing.T) {
	db := openJobQueueTestDB(t)
	user := createQueueTestUser(t, db)
	repo := NewJobRepository(db)
	now := time.Now().Truncate(time.Millisecond)
	job := createQueueTestJob(t, db, user.ID, "job-terminal", models.StatusUploaded, now)
	require.NoError(t, repo.EnqueueTranscription(context.Background(), job.ID, now))
	claimed, err := repo.ClaimNextTranscription(context.Background(), "worker-a", now.Add(time.Minute))
	require.NoError(t, err)

	execution := &models.TranscriptionJobExecution{
		TranscriptionJobID: claimed.ID,
		UserID:             user.ID,
		Status:             models.StatusProcessing,
		Provider:           "local",
		ModelName:          "whisper-base",
		StartedAt:          now,
	}
	require.NoError(t, repo.CreateExecution(context.Background(), execution))

	require.NoError(t, repo.UpdateProgress(context.Background(), claimed.ID, 0.42, "transcribing"))
	var progressed models.TranscriptionJob
	require.NoError(t, db.First(&progressed, "id = ?", claimed.ID).Error)
	assert.Equal(t, 0.42, progressed.Progress)
	assert.Equal(t, "transcribing", progressed.ProgressStage)

	outputPath := "/internal/transcripts/job-terminal/transcript.json"
	require.NoError(t, repo.CompleteTranscription(context.Background(), claimed.ID, `{"text":"done"}`, &outputPath, now.Add(5*time.Second)))

	var completed models.TranscriptionJob
	require.NoError(t, db.First(&completed, "id = ?", claimed.ID).Error)
	assert.Equal(t, models.StatusCompleted, completed.Status)
	assert.Equal(t, 1.0, completed.Progress)
	assert.Equal(t, "completed", completed.ProgressStage)
	assert.Nil(t, completed.ClaimedBy)
	assert.Nil(t, completed.ClaimExpiresAt)
	assert.NotNil(t, completed.CompletedAt)
	assert.NotNil(t, completed.OutputJSONPath)
	assert.Equal(t, outputPath, *completed.OutputJSONPath)

	var completedExecution models.TranscriptionJobExecution
	require.NoError(t, db.First(&completedExecution, "id = ?", execution.ID).Error)
	assert.Equal(t, models.StatusCompleted, completedExecution.Status)
	assert.NotNil(t, completedExecution.CompletedAt)
	assert.Equal(t, outputPath, *completedExecution.OutputJSONPath)

	executions, err := repo.ListExecutions(context.Background(), claimed.ID)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	assert.Equal(t, execution.ID, executions[0].ID)
}

func TestJobRepositoryFailAndCancelTranscription(t *testing.T) {
	db := openJobQueueTestDB(t)
	user := createQueueTestUser(t, db)
	repo := NewJobRepository(db)
	now := time.Now().Truncate(time.Millisecond)

	failing := createQueueTestJob(t, db, user.ID, "job-fail", models.StatusUploaded, now)
	require.NoError(t, repo.EnqueueTranscription(context.Background(), failing.ID, now))
	claimed, err := repo.ClaimNextTranscription(context.Background(), "worker-a", now.Add(time.Minute))
	require.NoError(t, err)
	execution := &models.TranscriptionJobExecution{TranscriptionJobID: claimed.ID, UserID: user.ID, Status: models.StatusProcessing, StartedAt: now}
	require.NoError(t, repo.CreateExecution(context.Background(), execution))
	require.NoError(t, repo.FailTranscription(context.Background(), claimed.ID, "model unavailable", now.Add(time.Second)))

	var failed models.TranscriptionJob
	require.NoError(t, db.First(&failed, "id = ?", claimed.ID).Error)
	assert.Equal(t, models.StatusFailed, failed.Status)
	assert.Equal(t, "failed", failed.ProgressStage)
	assert.NotNil(t, failed.FailedAt)
	assert.Equal(t, "model unavailable", *failed.ErrorMessage)

	var failedExecution models.TranscriptionJobExecution
	require.NoError(t, db.First(&failedExecution, "id = ?", execution.ID).Error)
	assert.Equal(t, models.StatusFailed, failedExecution.Status)
	assert.Equal(t, "model unavailable", *failedExecution.ErrorMessage)

	canceling := createQueueTestJob(t, db, user.ID, "job-cancel", models.StatusUploaded, now)
	require.NoError(t, repo.EnqueueTranscription(context.Background(), canceling.ID, now))
	require.NoError(t, repo.CancelTranscription(context.Background(), canceling.ID, now.Add(2*time.Second)))
	var stopped models.TranscriptionJob
	require.NoError(t, db.First(&stopped, "id = ?", canceling.ID).Error)
	assert.Equal(t, models.StatusStopped, stopped.Status)
	assert.Equal(t, "stopped", stopped.ProgressStage)
	assert.Nil(t, stopped.ClaimedBy)
	assert.Nil(t, stopped.ClaimExpiresAt)
}
