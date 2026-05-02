package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openRecordingRepositoryTestDB(t *testing.T) (*gorm.DB, RecordingRepository, models.User) {
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
	user := models.User{Username: "recording-repo-user", Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	return db, NewRecordingRepository(db), user
}

func TestRecordingRepositoryCreateListAndAddChunk(t *testing.T) {
	_, repo, user := openRecordingRepositoryTestDB(t)
	ctx := context.Background()

	session := &models.RecordingSession{
		UserID:   user.ID,
		Title:    stringPtr("Repo recording"),
		MimeType: "audio/webm;codecs=opus",
	}
	require.NoError(t, repo.CreateSession(ctx, session))

	require.NoError(t, repo.AddChunk(ctx, &models.RecordingChunk{
		UserID:     user.ID,
		SessionID:  session.ID,
		ChunkIndex: 0,
		Path:       "/tmp/chunk-0.webm",
		MimeType:   session.MimeType,
		SHA256:     stringPtr("abc"),
		SizeBytes:  42,
	}))

	got, err := repo.FindSessionForUser(ctx, user.ID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.ReceivedChunks)
	assert.Equal(t, int64(42), got.ReceivedBytes)

	chunks, err := repo.ListChunks(ctx, user.ID, session.ID)
	require.NoError(t, err)
	require.Len(t, chunks, 1)
	assert.Equal(t, 0, chunks[0].ChunkIndex)

	_, err = repo.FindChunk(ctx, user.ID, session.ID, 0)
	require.NoError(t, err)

	_, count, err := repo.ListSessionsForUser(ctx, user.ID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestRecordingRepositoryChunkUniqueAndStatusGuard(t *testing.T) {
	_, repo, user := openRecordingRepositoryTestDB(t)
	ctx := context.Background()
	session := &models.RecordingSession{UserID: user.ID, MimeType: "audio/webm"}
	require.NoError(t, repo.CreateSession(ctx, session))

	chunk := &models.RecordingChunk{UserID: user.ID, SessionID: session.ID, ChunkIndex: 0, Path: "/tmp/chunk.webm", MimeType: "audio/webm", SizeBytes: 1}
	require.NoError(t, repo.AddChunk(ctx, chunk))
	duplicate := &models.RecordingChunk{UserID: user.ID, SessionID: session.ID, ChunkIndex: 0, Path: "/tmp/chunk-retry.webm", MimeType: "audio/webm", SizeBytes: 1}
	require.Error(t, repo.AddChunk(ctx, duplicate))

	require.NoError(t, repo.MarkStopping(ctx, user.ID, session.ID, 0, nil, false, time.Now()))
	lateChunk := &models.RecordingChunk{UserID: user.ID, SessionID: session.ID, ChunkIndex: 1, Path: "/tmp/chunk-1.webm", MimeType: "audio/webm", SizeBytes: 1}
	require.Error(t, repo.AddChunk(ctx, lateChunk))
}

func TestRecordingRepositoryFinalizationLifecycle(t *testing.T) {
	db, repo, user := openRecordingRepositoryTestDB(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)
	session := &models.RecordingSession{UserID: user.ID, MimeType: "audio/webm"}
	require.NoError(t, repo.CreateSession(ctx, session))

	duration := int64(3000)
	require.NoError(t, repo.MarkStopping(ctx, user.ID, session.ID, 0, &duration, true, now))

	claimed, err := repo.ClaimNextFinalization(ctx, "worker-a", now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, session.ID, claimed.ID)
	assert.Equal(t, models.RecordingStatusFinalizing, claimed.Status)
	assert.Equal(t, "worker-a", *claimed.ClaimedBy)
	require.NotNil(t, claimed.ClaimExpiresAt)

	require.Error(t, repo.RenewFinalizationClaim(ctx, session.ID, "worker-b", now.Add(2*time.Minute)))
	require.NoError(t, repo.RenewFinalizationClaim(ctx, session.ID, "worker-a", now.Add(2*time.Minute)))

	file := models.TranscriptionJob{
		ID:             "file-internal",
		UserID:         user.ID,
		Title:          stringPtr("recorded file"),
		Status:         models.StatusUploaded,
		AudioPath:      "/tmp/final.webm",
		SourceFileName: "final.webm",
	}
	require.NoError(t, db.Create(&file).Error)
	fileID := file.ID
	transcription := models.TranscriptionJob{
		ID:             "tr-internal",
		UserID:         user.ID,
		Title:          stringPtr("recorded transcription"),
		Status:         models.StatusPending,
		AudioPath:      file.AudioPath,
		SourceFileName: file.SourceFileName,
		SourceFileHash: &fileID,
	}
	require.NoError(t, db.Create(&transcription).Error)

	require.NoError(t, repo.CompleteFinalization(ctx, session.ID, "worker-a", file.ID, &transcription.ID, now.Add(3*time.Second)))

	completed, err := repo.FindSessionForUser(ctx, user.ID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RecordingStatusReady, completed.Status)
	require.NotNil(t, completed.FileID)
	assert.Equal(t, "file-internal", *completed.FileID)
	require.NotNil(t, completed.TranscriptionID)
	assert.Equal(t, transcription.ID, *completed.TranscriptionID)
	assert.Nil(t, completed.ClaimedBy)
	assert.Nil(t, completed.ClaimExpiresAt)
}

func TestRecordingRepositoryRecoverAndExpire(t *testing.T) {
	_, repo, user := openRecordingRepositoryTestDB(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	expiredActive := &models.RecordingSession{UserID: user.ID, MimeType: "audio/webm", ExpiresAt: timePtr(now.Add(-time.Minute))}
	require.NoError(t, repo.CreateSession(ctx, expiredActive))
	freshActive := &models.RecordingSession{UserID: user.ID, MimeType: "audio/webm", ExpiresAt: timePtr(now.Add(time.Minute))}
	require.NoError(t, repo.CreateSession(ctx, freshActive))

	expired, err := repo.ExpireAbandonedSessions(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), expired)

	reloadedExpired, err := repo.FindSessionForUser(ctx, user.ID, expiredActive.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RecordingStatusExpired, reloadedExpired.Status)

	stopping := &models.RecordingSession{UserID: user.ID, MimeType: "audio/webm"}
	require.NoError(t, repo.CreateSession(ctx, stopping))
	require.NoError(t, repo.MarkStopping(ctx, user.ID, stopping.ID, 0, nil, false, now))
	claimed, err := repo.ClaimNextFinalization(ctx, "worker-a", now.Add(-time.Minute))
	require.NoError(t, err)

	recovered, err := repo.RecoverExpiredFinalizationClaims(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), recovered)
	reloaded, err := repo.FindSessionForUser(ctx, user.ID, claimed.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RecordingStatusStopping, reloaded.Status)
	assert.Nil(t, reloaded.ClaimedBy)
}

func TestRecordingRepositoryCancelTerminalBehavior(t *testing.T) {
	_, repo, user := openRecordingRepositoryTestDB(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	session := &models.RecordingSession{UserID: user.ID, MimeType: "audio/webm"}
	require.NoError(t, repo.CreateSession(ctx, session))
	require.NoError(t, repo.CancelSession(ctx, user.ID, session.ID, now))
	canceled, err := repo.FindSessionForUser(ctx, user.ID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RecordingStatusCanceled, canceled.Status)

	require.Error(t, repo.CancelSession(ctx, user.ID, session.ID, now))
}

func stringPtr(value string) *string {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}
