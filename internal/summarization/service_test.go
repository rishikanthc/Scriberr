package summarization

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/pkg/logger"

	"github.com/stretchr/testify/require"
	gormlogger "gorm.io/gorm/logger"
)

func TestPlainTranscriptTextJoinsSegmentsWithoutSpeakerOrTimestamps(t *testing.T) {
	text, err := plainTranscriptText(`{
		"text":"fallback text",
		"segments":[
			{"id":"seg_000001","start":0,"end":1,"speaker":"SPEAKER_00","text":"First idea."},
			{"id":"seg_000002","start":1,"end":2,"speaker":"SPEAKER_01","text":"Second idea."}
		],
		"words":[]
	}`)

	require.NoError(t, err)
	require.Equal(t, "First idea.\nSecond idea.", text)
	require.NotContains(t, text, "SPEAKER")
	require.NotContains(t, text, "0")
}

func TestFitTranscriptToContextTruncatesLongInput(t *testing.T) {
	transcript := strings.Repeat("word ", 4000)
	fitted, truncated := fitTranscriptToContext(transcript, 1200)

	require.True(t, truncated)
	require.Less(t, len(fitted), len(transcript))
	require.NotEmpty(t, fitted)
}

func TestEnqueueRequiresConfiguredProviderAndModels(t *testing.T) {
	logger.Init("silent")
	require.NoError(t, database.Initialize(filepath.Join(t.TempDir(), "scriberr.db")))
	database.DB.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	t.Cleanup(func() { _ = database.Close() })

	user := models.User{Username: "summary-user", Password: "hash"}
	require.NoError(t, database.DB.Create(&user).Error)
	transcript := `{"text":"Hello world.","segments":[{"id":"seg_000001","start":0,"end":1,"text":"Hello world."}],"words":[]}`
	job := models.TranscriptionJob{
		ID:             "job-summary",
		UserID:         user.ID,
		Status:         models.StatusCompleted,
		AudioPath:      "/tmp/audio.wav",
		SourceFileName: "audio.wav",
		SourceFileHash: stringPtr("source"),
		Transcript:     &transcript,
	}
	require.NoError(t, database.DB.Create(&job).Error)

	service := NewService(
		repository.NewSummaryRepository(database.DB),
		repository.NewLLMConfigRepository(database.DB),
		repository.NewJobRepository(database.DB),
		Config{},
	)

	require.NoError(t, service.EnqueueForTranscription(context.Background(), &job))
	var count int64
	require.NoError(t, database.DB.Model(&models.Summary{}).Count(&count).Error)
	require.Equal(t, int64(0), count)

	baseURL := "http://127.0.0.1:1234/v1"
	largeModel := "large"
	config := models.LLMConfig{
		UserID:     user.ID,
		Name:       "incomplete",
		Provider:   "openai_compatible",
		BaseURL:    &baseURL,
		IsDefault:  true,
		LargeModel: &largeModel,
	}
	require.NoError(t, database.DB.Create(&config).Error)

	require.NoError(t, service.EnqueueForTranscription(context.Background(), &job))
	require.NoError(t, database.DB.Model(&models.Summary{}).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

func stringPtr(value string) *string {
	return &value
}
