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

func openChatRepositoryTestDB(t *testing.T) *gorm.DB {
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

func createChatRepositoryFixture(t *testing.T, db *gorm.DB) (models.User, models.TranscriptionJob, models.TranscriptionJob) {
	t.Helper()
	user := models.User{Username: "chat-repo-user-" + time.Now().Format("150405.000000000"), Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	parentTranscript := `{"text":"hello","segments":[{"id":"s1","text":"hello","speaker":"SPEAKER_00"}]}`
	otherTranscript := `{"text":"followup","segments":[{"id":"s1","text":"followup","speaker":"SPEAKER_01"}]}`
	parentTitle := "parent transcript"
	parent := models.TranscriptionJob{
		UserID:     user.ID,
		Title:      &parentTitle,
		Status:     models.StatusCompleted,
		AudioPath:  filepath.Join(t.TempDir(), "parent.wav"),
		Transcript: &parentTranscript,
	}
	require.NoError(t, db.Create(&parent).Error)
	otherTitle := "other transcript"
	other := models.TranscriptionJob{
		UserID:     user.ID,
		Title:      &otherTitle,
		Status:     models.StatusCompleted,
		AudioPath:  filepath.Join(t.TempDir(), "other.wav"),
		Transcript: &otherTranscript,
	}
	require.NoError(t, db.Create(&other).Error)
	return user, parent, other
}

func TestChatRepositoryCreatesSessionAndParentSource(t *testing.T) {
	db := openChatRepositoryTestDB(t)
	user, parent, _ := createChatRepositoryFixture(t, db)
	repo := NewChatRepository(db)

	plain := "Speaker 1: hello"
	session := &models.ChatSession{
		UserID:                user.ID,
		ParentTranscriptionID: parent.ID,
		Title:                 "chat",
		Provider:              "openai_compatible",
		Model:                 "qwen3.5-4B",
	}
	source := &models.ChatContextSource{PlainTextSnapshot: &plain}
	require.NoError(t, repo.CreateSessionWithParentSource(context.Background(), session, source))

	found, err := repo.FindSessionForUser(context.Background(), user.ID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, parent.ID, found.ParentTranscriptionID)

	sources, err := repo.ListContextSources(context.Background(), user.ID, session.ID, true)
	require.NoError(t, err)
	require.Len(t, sources, 1)
	assert.Equal(t, models.ChatContextSourceKindParentTranscript, sources[0].Kind)
	assert.Equal(t, parent.ID, sources[0].TranscriptionID)
	require.NotNil(t, sources[0].SnapshotHash)
	assert.NotEmpty(t, *sources[0].SnapshotHash)
}

func TestChatRepositoryScopesContextSourcesByUserAndCompletedTranscript(t *testing.T) {
	db := openChatRepositoryTestDB(t)
	user, parent, other := createChatRepositoryFixture(t, db)
	otherUser := models.User{Username: "chat-repo-other", Password: "pw"}
	require.NoError(t, db.Create(&otherUser).Error)
	foreignTitle := "foreign transcript"
	foreign := models.TranscriptionJob{
		UserID:    otherUser.ID,
		Title:     &foreignTitle,
		Status:    models.StatusCompleted,
		AudioPath: filepath.Join(t.TempDir(), "foreign.wav"),
	}
	require.NoError(t, db.Create(&foreign).Error)
	pendingTitle := "pending transcript"
	pending := models.TranscriptionJob{
		UserID:    user.ID,
		Title:     &pendingTitle,
		Status:    models.StatusUploaded,
		AudioPath: filepath.Join(t.TempDir(), "pending.wav"),
	}
	require.NoError(t, db.Create(&pending).Error)

	repo := NewChatRepository(db)
	session := &models.ChatSession{
		UserID:                user.ID,
		ParentTranscriptionID: parent.ID,
		Title:                 "chat",
		Provider:              "openai_compatible",
		Model:                 "qwen3.5-4B",
	}
	require.NoError(t, repo.CreateSession(context.Background(), session))

	plain := "Speaker 1: followup"
	source, err := repo.UpsertContextSource(context.Background(), user.ID, session.ID, &models.ChatContextSource{
		TranscriptionID:   other.ID,
		Kind:              models.ChatContextSourceKindTranscript,
		PlainTextSnapshot: &plain,
	})
	require.NoError(t, err)
	assert.Equal(t, other.ID, source.TranscriptionID)
	assert.Equal(t, 0, source.Position)

	_, err = repo.UpsertContextSource(context.Background(), user.ID, session.ID, &models.ChatContextSource{
		TranscriptionID: foreign.ID,
		Kind:            models.ChatContextSourceKindTranscript,
	})
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	_, err = repo.UpsertContextSource(context.Background(), user.ID, session.ID, &models.ChatContextSource{
		TranscriptionID: pending.ID,
		Kind:            models.ChatContextSourceKindTranscript,
	})
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	require.ErrorIs(t, repo.SetContextSourceEnabled(context.Background(), otherUser.ID, session.ID, source.ID, false), gorm.ErrRecordNotFound)
	require.NoError(t, repo.SetContextSourceEnabled(context.Background(), user.ID, session.ID, source.ID, false))
	enabled, err := repo.ListContextSources(context.Background(), user.ID, session.ID, true)
	require.NoError(t, err)
	assert.Empty(t, enabled)
}

func TestChatRepositoryMessagesAndRunsAreScoped(t *testing.T) {
	db := openChatRepositoryTestDB(t)
	user, parent, _ := createChatRepositoryFixture(t, db)
	otherUser := models.User{Username: "chat-repo-message-other", Password: "pw"}
	require.NoError(t, db.Create(&otherUser).Error)
	repo := NewChatRepository(db)
	session := &models.ChatSession{
		UserID:                user.ID,
		ParentTranscriptionID: parent.ID,
		Title:                 "chat",
		Provider:              "openai_compatible",
		Model:                 "qwen3.5-4B",
	}
	require.NoError(t, repo.CreateSession(context.Background(), session))

	message := &models.ChatMessage{UserID: user.ID, ChatSessionID: session.ID, Role: models.ChatMessageRoleUser, Content: "question"}
	require.NoError(t, repo.CreateMessage(context.Background(), message))
	messages, count, err := repo.ListMessages(context.Background(), user.ID, session.ID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	require.Len(t, messages, 1)
	assert.Equal(t, message.ID, messages[0].ID)

	forged := &models.ChatMessage{UserID: otherUser.ID, ChatSessionID: session.ID, Role: models.ChatMessageRoleUser, Content: "forged"}
	require.ErrorIs(t, repo.CreateMessage(context.Background(), forged), gorm.ErrRecordNotFound)

	run := &models.ChatGenerationRun{
		UserID:              user.ID,
		ChatSessionID:       session.ID,
		Status:              models.ChatGenerationRunStatusPending,
		Provider:            "openai_compatible",
		Model:               "qwen3.5-4B",
		ContextWindow:       32768,
		ContextWindowSource: "provider",
	}
	require.NoError(t, repo.CreateGenerationRun(context.Background(), run))
	now := time.Now()
	require.NoError(t, repo.UpdateGenerationRunStatus(context.Background(), user.ID, run.ID, models.ChatGenerationRunStatusStreaming, now, nil))
	require.ErrorIs(t, repo.UpdateGenerationRunStatus(context.Background(), otherUser.ID, run.ID, models.ChatGenerationRunStatusCompleted, now, nil), gorm.ErrRecordNotFound)
}
