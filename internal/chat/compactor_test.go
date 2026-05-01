package chat

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCompactorCompactsOversizedTranscriptAndPersistsSummary(t *testing.T) {
	db := openCompactorTestDB(t)
	user, parent := createCompactorTranscript(t, db, "parent", strings.Repeat("parent transcript content ", 120))
	repo := repository.NewChatRepository(db)
	session := createCompactorSession(t, repo, user.ID, parent.ID)

	builder := NewContextBuilder(repo, ApproxTokenEstimator{})
	_, err := builder.AddParentSource(context.Background(), user.ID, session.ID)
	require.NoError(t, err)

	compactor := NewCompactor(repo, ApproxTokenEstimator{}, nil, CompactionConfig{})
	results, err := compactor.CompactOversizedTranscripts(context.Background(), user.ID, session.ID, ContextBudget{
		ContextWindow:      80,
		ReservedResponse:   16,
		ReservedSystem:     8,
		ReservedChat:       8,
		SafetyMarginTokens: 8,
	}, "openai_compatible", "large-context-model")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Compacted)
	assert.NotEmpty(t, results[0].SummaryID)
	assert.Less(t, results[0].OutputTokens, results[0].InputTokens)

	sources, err := repo.ListContextSources(context.Background(), user.ID, session.ID, true)
	require.NoError(t, err)
	require.Len(t, sources, 1)
	assert.Equal(t, models.ChatContextCompactionStatusCompacted, sources[0].CompactionStatus)
	require.NotNil(t, sources[0].CompactedSnapshot)
	assert.Contains(t, *sources[0].CompactedSnapshot, "Compacted context:")

	summaries, err := repo.ListContextSummaries(context.Background(), user.ID, session.ID, models.ChatContextSummaryTypeTranscript)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, parent.ID, *summaries[0].SourceTranscriptionID)
	assert.Equal(t, "large-context-model", summaries[0].Model)

	built, err := builder.Build(context.Background(), user.ID, session.ID, BuildOptions{})
	require.NoError(t, err)
	assert.Contains(t, built.Content, "Compacted context:")
}

func TestCompactorSummarizesOldMessagesWithoutTranscriptContext(t *testing.T) {
	db := openCompactorTestDB(t)
	user, parent := createCompactorTranscript(t, db, "parent", "TRANSCRIPT_SHOULD_NOT_APPEAR")
	repo := repository.NewChatRepository(db)
	session := createCompactorSession(t, repo, user.ID, parent.ID)
	builder := NewContextBuilder(repo, ApproxTokenEstimator{})
	_, err := builder.AddParentSource(context.Background(), user.ID, session.ID)
	require.NoError(t, err)

	messageIDs := make([]string, 0, 14)
	for i := 0; i < 14; i++ {
		role := models.ChatMessageRoleUser
		if i%2 == 1 {
			role = models.ChatMessageRoleAssistant
		}
		message := &models.ChatMessage{
			UserID:        user.ID,
			ChatSessionID: session.ID,
			Role:          role,
			Content:       strings.Repeat("message content ", 10),
		}
		require.NoError(t, repo.CreateMessage(context.Background(), message))
		messageIDs = append(messageIDs, message.ID)
	}

	compactor := NewCompactor(repo, ApproxTokenEstimator{}, nil, CompactionConfig{
		ThresholdRatio:      0.25,
		RecentMessageWindow: 4,
	})
	result, err := compactor.CompactSessionHistory(context.Background(), user.ID, session.ID, 200, "openai_compatible", "history-model")
	require.NoError(t, err)
	require.True(t, result.Compacted)
	assert.Equal(t, messageIDs[9], result.SourceMessageThroughID)
	assert.Equal(t, 4, result.RecentMessageCount)
	assert.NotEmpty(t, result.SummaryID)

	summaries, err := repo.ListContextSummaries(context.Background(), user.ID, session.ID, models.ChatContextSummaryTypeSession)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, messageIDs[9], *summaries[0].SourceMessageThroughID)
	assert.NotContains(t, summaries[0].Content, "TRANSCRIPT_SHOULD_NOT_APPEAR")
	assert.Contains(t, summaries[0].Content, "User:")
}

func TestCompactorSkipsSessionHistoryBelowThreshold(t *testing.T) {
	db := openCompactorTestDB(t)
	user, parent := createCompactorTranscript(t, db, "parent", "short transcript")
	repo := repository.NewChatRepository(db)
	session := createCompactorSession(t, repo, user.ID, parent.ID)
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.CreateMessage(context.Background(), &models.ChatMessage{
			UserID:        user.ID,
			ChatSessionID: session.ID,
			Role:          models.ChatMessageRoleUser,
			Content:       "short",
		}))
	}

	compactor := NewCompactor(repo, ApproxTokenEstimator{}, nil, CompactionConfig{RecentMessageWindow: 4})
	result, err := compactor.CompactSessionHistory(context.Background(), user.ID, session.ID, 4096, "openai_compatible", "history-model")
	require.NoError(t, err)
	assert.False(t, result.Compacted)

	summaries, err := repo.ListContextSummaries(context.Background(), user.ID, session.ID, models.ChatContextSummaryTypeSession)
	require.NoError(t, err)
	assert.Empty(t, summaries)
}

func openCompactorTestDB(t *testing.T) *gorm.DB {
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

func createCompactorTranscript(t *testing.T, db *gorm.DB, title string, content string) (models.User, models.TranscriptionJob) {
	t.Helper()
	user := models.User{Username: "chat-compactor-" + title + "-" + time.Now().Format("150405.000000000"), Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	transcript := `{"text":` + quoteJSON(content) + `,"segments":[{"id":"s1","speaker":"SPEAKER_00","text":` + quoteJSON(content) + `}]}`
	jobTitle := title
	job := models.TranscriptionJob{
		UserID:     user.ID,
		Title:      &jobTitle,
		Status:     models.StatusCompleted,
		AudioPath:  filepath.Join(t.TempDir(), title+".wav"),
		Transcript: &transcript,
	}
	require.NoError(t, db.Create(&job).Error)
	return user, job
}

func createCompactorSession(t *testing.T, repo repository.ChatRepository, userID uint, parentID string) *models.ChatSession {
	t.Helper()
	session := &models.ChatSession{
		UserID:                userID,
		ParentTranscriptionID: parentID,
		Title:                 "chat",
		Provider:              "openai_compatible",
		Model:                 "qwen3.5-4B",
	}
	require.NoError(t, repo.CreateSession(context.Background(), session))
	return session
}

func quoteJSON(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
