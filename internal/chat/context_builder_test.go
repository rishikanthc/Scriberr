package chat

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlainTranscriptTextIncludesSpeakerLabelsAndOmitsTiming(t *testing.T) {
	text, err := PlainTranscriptText(`{
		"text":"Hello there. General Kenobi. No speaker.",
		"segments":[
			{"id":"a","start":0,"end":1,"speaker":"SPEAKER_00","text":"Hello there."},
			{"id":"b","start":1,"end":2,"speaker":"SPEAKER_00","text":"General Kenobi."},
			{"id":"c","start":2,"end":3,"text":"No speaker."}
		],
		"words":[{"start":0,"end":0.5,"word":"Hello","speaker":"SPEAKER_00"}],
		"engine":{"provider":"local","transcription_model":"whisper"}
	}`)
	require.NoError(t, err)
	assert.Equal(t, "Speaker 1: Hello there. General Kenobi.\nNo speaker.", text)
	assert.NotContains(t, text, "start")
	assert.NotContains(t, text, "whisper")
}

func TestContextBuilderAddsSourcesAndBuildsBudgetedContext(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "scriberr.db"))
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	user := models.User{Username: "chat-builder-" + time.Now().Format("150405.000000000"), Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	parentText := `{"text":"parent text","segments":[{"id":"s1","speaker":"SPEAKER_00","text":"` + strings.Repeat("parent ", 80) + `"}]}`
	extraText := `{"text":"extra text","segments":[{"id":"s1","speaker":"SPEAKER_01","text":"` + strings.Repeat("extra ", 80) + `"}]}`
	parentTitle := "parent"
	parent := models.TranscriptionJob{UserID: user.ID, Title: &parentTitle, Status: models.StatusCompleted, AudioPath: "/tmp/parent.wav", Transcript: &parentText}
	require.NoError(t, db.Create(&parent).Error)
	extraTitle := "extra"
	extra := models.TranscriptionJob{UserID: user.ID, Title: &extraTitle, Status: models.StatusCompleted, AudioPath: "/tmp/extra.wav", Transcript: &extraText}
	require.NoError(t, db.Create(&extra).Error)

	repo := repository.NewChatRepository(db)
	session := &models.ChatSession{
		UserID:                user.ID,
		ParentTranscriptionID: parent.ID,
		Title:                 "chat",
		Provider:              "openai_compatible",
		Model:                 "qwen3.5-4B",
	}
	require.NoError(t, repo.CreateSession(context.Background(), session))

	builder := NewContextBuilder(repo, ApproxTokenEstimator{})
	parentMutation, err := builder.AddParentSource(context.Background(), user.ID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChatContextSourceKindParentTranscript, parentMutation.Source.Kind)
	extraMutation, err := builder.AddTranscriptSource(context.Background(), user.ID, session.ID, extra.ID, models.ChatContextSourceKindTranscript)
	require.NoError(t, err)
	assert.Equal(t, extra.ID, extraMutation.Source.TranscriptionID)

	built, err := builder.Build(context.Background(), user.ID, session.ID, BuildOptions{Budget: ContextBudget{
		ContextWindow:      120,
		ReservedResponse:   32,
		ReservedSystem:     8,
		ReservedChat:       8,
		SafetyMarginTokens: 8,
	}})
	require.NoError(t, err)
	require.Len(t, built.Sources, 2)
	assert.NotEmpty(t, built.Content)
	assert.LessOrEqual(t, built.TokensEstimated, 64)
	assert.True(t, built.Truncated)

	require.NoError(t, builder.SetSourceEnabled(context.Background(), user.ID, session.ID, extraMutation.Source.ID, false))
	built, err = builder.Build(context.Background(), user.ID, session.ID, BuildOptions{})
	require.NoError(t, err)
	require.Len(t, built.Sources, 1)
	assert.Contains(t, built.Content, "Speaker 1:")
}
