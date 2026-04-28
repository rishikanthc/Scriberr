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

func openAnnotationRepositoryTestDB(t *testing.T) *gorm.DB {
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

func createAnnotationRepositoryFixture(t *testing.T, db *gorm.DB) (models.User, models.TranscriptionJob) {
	t.Helper()
	user := models.User{Username: "annotation-repo-user-" + time.Now().Format("150405.000000000"), Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	title := "annotation repo transcript"
	sourceID := "source-file"
	job := models.TranscriptionJob{
		ID:             "annotation-repo-job",
		UserID:         user.ID,
		Title:          &title,
		Status:         models.StatusCompleted,
		AudioPath:      filepath.Join(t.TempDir(), "audio.wav"),
		SourceFileName: "audio.wav",
		SourceFileHash: &sourceID,
	}
	require.NoError(t, db.Create(&job).Error)
	return user, job
}

func TestAnnotationRepositoryScopesByUserTranscriptionAndKind(t *testing.T) {
	db := openAnnotationRepositoryTestDB(t)
	user, job := createAnnotationRepositoryFixture(t, db)
	otherUser := models.User{Username: "annotation-repo-other", Password: "pw"}
	require.NoError(t, db.Create(&otherUser).Error)

	repo := NewAnnotationRepository(db)
	noteContent := "remember this"
	note := &models.TranscriptAnnotation{
		UserID:          user.ID,
		TranscriptionID: job.ID,
		Kind:            models.AnnotationKindNote,
		Content:         &noteContent,
		Quote:           "quoted note",
		AnchorStartMS:   100,
		AnchorEndMS:     200,
	}
	highlight := &models.TranscriptAnnotation{
		UserID:          user.ID,
		TranscriptionID: job.ID,
		Kind:            models.AnnotationKindHighlight,
		Quote:           "quoted highlight",
		AnchorStartMS:   300,
		AnchorEndMS:     400,
	}
	otherUserHighlight := &models.TranscriptAnnotation{
		UserID:          otherUser.ID,
		TranscriptionID: job.ID,
		Kind:            models.AnnotationKindHighlight,
		Quote:           "other user quote",
		AnchorStartMS:   500,
		AnchorEndMS:     600,
	}
	require.NoError(t, repo.CreateAnnotation(context.Background(), note))
	require.NoError(t, repo.CreateAnnotation(context.Background(), highlight))
	require.NoError(t, repo.CreateAnnotation(context.Background(), otherUserHighlight))

	kind := models.AnnotationKindHighlight
	items, count, err := repo.ListAnnotationsForTranscription(context.Background(), user.ID, job.ID, &kind, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	require.Len(t, items, 1)
	assert.Equal(t, highlight.ID, items[0].ID)

	found, err := repo.FindAnnotationForUser(context.Background(), user.ID, job.ID, note.ID)
	require.NoError(t, err)
	assert.Equal(t, note.ID, found.ID)

	_, err = repo.FindAnnotationForUser(context.Background(), otherUser.ID, job.ID, note.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestAnnotationRepositoryUpdateAndSoftDeleteAreScoped(t *testing.T) {
	db := openAnnotationRepositoryTestDB(t)
	user, job := createAnnotationRepositoryFixture(t, db)
	otherUser := models.User{Username: "annotation-repo-delete-other", Password: "pw"}
	require.NoError(t, db.Create(&otherUser).Error)

	repo := NewAnnotationRepository(db)
	annotation := &models.TranscriptAnnotation{
		UserID:          user.ID,
		TranscriptionID: job.ID,
		Kind:            models.AnnotationKindHighlight,
		Quote:           "original",
		AnchorStartMS:   100,
		AnchorEndMS:     200,
	}
	require.NoError(t, repo.CreateAnnotation(context.Background(), annotation))

	annotation.Quote = "updated"
	annotation.AnchorStartMS = 150
	require.NoError(t, repo.UpdateAnnotation(context.Background(), annotation))

	found, err := repo.FindAnnotationForUser(context.Background(), user.ID, job.ID, annotation.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated", found.Quote)
	assert.Equal(t, int64(150), found.AnchorStartMS)

	require.ErrorIs(t, repo.SoftDeleteAnnotation(context.Background(), otherUser.ID, job.ID, annotation.ID), gorm.ErrRecordNotFound)
	require.NoError(t, repo.SoftDeleteAnnotation(context.Background(), user.ID, job.ID, annotation.ID))

	_, err = repo.FindAnnotationForUser(context.Background(), user.ID, job.ID, annotation.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
