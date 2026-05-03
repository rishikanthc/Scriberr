package summarization

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"scriberr/internal/database"
	"scriberr/internal/llm"
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

func TestSelectorJSONPayloadAcceptsFencedJSON(t *testing.T) {
	payload := selectorJSONPayload("```json\n{\"widget_names\":[\"Outline\"]}\n```")

	require.Equal(t, `{"widget_names":["Outline"]}`, payload)
}

func TestSelectorJSONPayloadExtractsJSONFromExplanatoryText(t *testing.T) {
	payload := selectorJSONPayload("Here is the result:\n{\"widget_names\":[\"Action Items\"]}\nThanks")

	require.Equal(t, `{"widget_names":["Action Items"]}`, payload)
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

func TestGenerateTitleForSummaryRenamesRecordingAndPublishesFileEvent(t *testing.T) {
	logger.Init("silent")
	require.NoError(t, database.Initialize(filepath.Join(t.TempDir(), "scriberr.db")))
	database.DB.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	t.Cleanup(func() { _ = database.Close() })

	user, job := createTitleGenerationFixture(t, "job-title")
	baseURL := "http://127.0.0.1:1234/v1"
	smallModel := "small"
	require.NoError(t, database.DB.Create(&models.LLMConfig{
		UserID: user.ID, Name: "configured", Provider: "openai_compatible", BaseURL: &baseURL, IsDefault: true, SmallModel: &smallModel,
	}).Error)

	events := &recordingSummaryEvents{}
	service := NewService(repository.NewSummaryRepository(database.DB), repository.NewLLMConfigRepository(database.DB), repository.NewJobRepository(database.DB), Config{})
	service.SetEventPublisher(events)
	service.clientFor = func(*models.LLMConfig) (llm.Service, error) {
		return &fakeTitleLLM{content: "Building Better Home Theater Systems Today"}, nil
	}

	summary := &models.Summary{ID: "sum-title", UserID: user.ID, TranscriptionID: job.ID, Status: "completed"}
	require.NoError(t, service.generateTitleForSummary(context.Background(), summary, "A summary about home theater surround sound setup."))

	var updated models.TranscriptionJob
	require.NoError(t, database.DB.First(&updated, "id = ?", job.ID).Error)
	require.NotNil(t, updated.Title)
	require.Equal(t, "Building Better Home Theater Systems Today", *updated.Title)
	require.True(t, updated.LLMTitleGenerated)
	require.NotNil(t, updated.LLMTitleGeneratedAt)
	var recording models.TranscriptionJob
	require.NoError(t, database.DB.First(&recording, "id = ?", *job.SourceFileHash).Error)
	require.NotNil(t, recording.Title)
	require.Equal(t, "Building Better Home Theater Systems Today", *recording.Title)
	require.True(t, recording.LLMTitleGenerated)
	require.NotNil(t, recording.LLMTitleGeneratedAt)
	require.Len(t, events.fileEvents, 1)
	require.Equal(t, "file.updated", events.fileEvents[0].name)
	require.Equal(t, "file_"+recording.ID, events.fileEvents[0].payload["id"])
	require.Equal(t, *recording.Title, events.fileEvents[0].payload["title"])
}

func TestGenerateTitleForSummarySkipsAlreadyRenamedRecording(t *testing.T) {
	logger.Init("silent")
	require.NoError(t, database.Initialize(filepath.Join(t.TempDir(), "scriberr.db")))
	database.DB.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	t.Cleanup(func() { _ = database.Close() })

	user, job := createTitleGenerationFixture(t, "job-title-skip")
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", *job.SourceFileHash).Update("llm_title_generated", true).Error)

	service := NewService(repository.NewSummaryRepository(database.DB), repository.NewLLMConfigRepository(database.DB), repository.NewJobRepository(database.DB), Config{})
	called := false
	service.clientFor = func(*models.LLMConfig) (llm.Service, error) {
		called = true
		return &fakeTitleLLM{content: "Should Not Run"}, nil
	}

	summary := &models.Summary{ID: "sum-title-skip", UserID: user.ID, TranscriptionID: job.ID, Status: "completed"}
	require.NoError(t, service.generateTitleForSummary(context.Background(), summary, "A summary."))
	require.False(t, called)
}

func TestGenerateTitleForSummarySkipsWhenAutoRenameDisabled(t *testing.T) {
	logger.Init("silent")
	require.NoError(t, database.Initialize(filepath.Join(t.TempDir(), "scriberr.db")))
	database.DB.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	t.Cleanup(func() { _ = database.Close() })

	user, job := createTitleGenerationFixture(t, "job-title-disabled")
	baseURL := "http://127.0.0.1:1234/v1"
	smallModel := "small"
	require.NoError(t, database.DB.Create(&models.LLMConfig{
		UserID: user.ID, Name: "configured", Provider: "openai_compatible", BaseURL: &baseURL, IsDefault: true, SmallModel: &smallModel,
	}).Error)

	service := NewService(repository.NewSummaryRepository(database.DB), repository.NewLLMConfigRepository(database.DB), repository.NewJobRepository(database.DB), Config{})
	service.SetUserSettingsReader(repository.NewUserRepository(database.DB))
	called := false
	service.clientFor = func(*models.LLMConfig) (llm.Service, error) {
		called = true
		return &fakeTitleLLM{content: "Should Not Be Used"}, nil
	}

	summary := &models.Summary{ID: "sum-title-disabled", UserID: user.ID, TranscriptionID: job.ID, Status: "completed"}
	require.NoError(t, service.generateTitleForSummary(context.Background(), summary, "A summary about home theater surround sound setup."))

	var updated models.TranscriptionJob
	require.NoError(t, database.DB.First(&updated, "id = ?", job.ID).Error)
	require.NotNil(t, updated.Title)
	require.Equal(t, "Original title", *updated.Title)
	require.False(t, updated.LLMTitleGenerated)
	require.False(t, called)
}

func TestGenerateTitleForSummaryKeepsFaithfulCurrentTitleAndMarksProcessed(t *testing.T) {
	logger.Init("silent")
	require.NoError(t, database.Initialize(filepath.Join(t.TempDir(), "scriberr.db")))
	database.DB.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	t.Cleanup(func() { _ = database.Close() })

	user, job := createTitleGenerationFixture(t, "job-title-keep")
	currentTitle := "Surround Sound Setup Guide"
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", job.ID).Update("title", currentTitle).Error)
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", *job.SourceFileHash).Update("title", currentTitle).Error)
	baseURL := "http://127.0.0.1:1234/v1"
	largeModel := "large"
	smallModel := "small"
	require.NoError(t, database.DB.Create(&models.LLMConfig{
		UserID: user.ID, Name: "configured", Provider: "openai_compatible", BaseURL: &baseURL, IsDefault: true, LargeModel: &largeModel, SmallModel: &smallModel,
	}).Error)

	events := &recordingSummaryEvents{}
	fake := &fakeTitleLLM{content: currentTitle}
	service := NewService(repository.NewSummaryRepository(database.DB), repository.NewLLMConfigRepository(database.DB), repository.NewJobRepository(database.DB), Config{})
	service.SetEventPublisher(events)
	service.clientFor = func(*models.LLMConfig) (llm.Service, error) {
		return fake, nil
	}

	summary := &models.Summary{ID: "sum-title-keep", UserID: user.ID, TranscriptionID: job.ID, Status: "completed"}
	require.NoError(t, service.generateTitleForSummary(context.Background(), summary, "A summary about surround sound setup."))

	var updated models.TranscriptionJob
	require.NoError(t, database.DB.First(&updated, "id = ?", job.ID).Error)
	require.NotNil(t, updated.Title)
	require.Equal(t, currentTitle, *updated.Title)
	require.True(t, updated.LLMTitleGenerated)
	require.NotNil(t, updated.LLMTitleGeneratedAt)
	var recording models.TranscriptionJob
	require.NoError(t, database.DB.First(&recording, "id = ?", *job.SourceFileHash).Error)
	require.NotNil(t, recording.Title)
	require.Equal(t, currentTitle, *recording.Title)
	require.True(t, recording.LLMTitleGenerated)
	require.NotNil(t, recording.LLMTitleGeneratedAt)
	require.Empty(t, events.fileEvents)
	require.Len(t, fake.messages, 1)
	require.Contains(t, fake.messages[0].Content, "Current title:\n"+currentTitle)
	require.Contains(t, fake.messages[0].Content, "If the current title already faithfully describes the audio")
}

func TestGenerateDescriptionForSummaryPersistsRecordingAndPublishesFileEvent(t *testing.T) {
	logger.Init("silent")
	require.NoError(t, database.Initialize(filepath.Join(t.TempDir(), "scriberr.db")))
	database.DB.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	t.Cleanup(func() { _ = database.Close() })

	user, job := createTitleGenerationFixture(t, "job-description")
	secretTranscript := `{"text":"SECRET TRANSCRIPT SHOULD NOT BE SENT","segments":[{"id":"seg_000001","start":0,"end":1,"text":"SECRET TRANSCRIPT SHOULD NOT BE SENT"}],"words":[]}`
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", job.ID).Update("transcript_text", secretTranscript).Error)
	baseURL := "http://127.0.0.1:1234/v1"
	smallModel := "small"
	require.NoError(t, database.DB.Create(&models.LLMConfig{
		UserID: user.ID, Name: "configured", Provider: "openai_compatible", BaseURL: &baseURL, IsDefault: true, SmallModel: &smallModel,
	}).Error)
	summary := createCompletedSummaryWithOutline(t, user.ID, job.ID, "sum-description", "A roundtable on compact home theater systems and setup tradeoffs.", "Setup goals\nSpeaker placement\nRoom calibration")

	events := &recordingSummaryEvents{}
	fake := &fakeTitleLLM{content: "Compact home theater setup and calibration.\nPractical tradeoffs for speakers and room layout."}
	service := NewService(repository.NewSummaryRepository(database.DB), repository.NewLLMConfigRepository(database.DB), repository.NewJobRepository(database.DB), Config{})
	service.SetEventPublisher(events)
	service.clientFor = func(*models.LLMConfig) (llm.Service, error) {
		return fake, nil
	}

	require.NoError(t, service.generateDescriptionForSummary(context.Background(), summary))

	want := "Compact home theater setup and calibration.\nPractical tradeoffs for speakers and room layout."
	var recording models.TranscriptionJob
	require.NoError(t, database.DB.First(&recording, "id = ?", *job.SourceFileHash).Error)
	require.NotNil(t, recording.LLMDescription)
	require.Equal(t, want, *recording.LLMDescription)
	require.NotNil(t, recording.LLMDescriptionAt)
	require.NotNil(t, recording.LLMDescriptionSourceSummaryID)
	require.Equal(t, summary.ID, *recording.LLMDescriptionSourceSummaryID)

	var transcription models.TranscriptionJob
	require.NoError(t, database.DB.First(&transcription, "id = ?", job.ID).Error)
	require.NotNil(t, transcription.LLMDescription)
	require.Equal(t, want, *transcription.LLMDescription)

	require.Len(t, events.fileEvents, 1)
	require.Equal(t, "file.updated", events.fileEvents[0].name)
	require.Equal(t, "file_"+recording.ID, events.fileEvents[0].payload["id"])
	require.Equal(t, want, events.fileEvents[0].payload["description"])
	require.Len(t, fake.messages, 1)
	require.Contains(t, fake.messages[0].Content, "Summary:\n"+summary.Content)
	require.Contains(t, fake.messages[0].Content, "Outline:\nSetup goals")
	require.NotContains(t, fake.messages[0].Content, "SECRET TRANSCRIPT")
}

func TestGenerateDescriptionForSummarySkipsWhenProviderOrOutlineMissing(t *testing.T) {
	logger.Init("silent")
	require.NoError(t, database.Initialize(filepath.Join(t.TempDir(), "scriberr.db")))
	database.DB.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	t.Cleanup(func() { _ = database.Close() })

	user, job := createTitleGenerationFixture(t, "job-description-skip")
	summary := createCompletedSummaryWithOutline(t, user.ID, job.ID, "sum-description-skip", "A summary.", "An outline.")

	service := NewService(repository.NewSummaryRepository(database.DB), repository.NewLLMConfigRepository(database.DB), repository.NewJobRepository(database.DB), Config{})
	called := false
	service.clientFor = func(*models.LLMConfig) (llm.Service, error) {
		called = true
		return &fakeTitleLLM{content: "Should not\nRun"}, nil
	}
	require.NoError(t, service.generateDescriptionForSummary(context.Background(), summary))
	require.False(t, called)

	baseURL := "http://127.0.0.1:1234/v1"
	smallModel := "small"
	require.NoError(t, database.DB.Create(&models.LLMConfig{
		UserID: user.ID, Name: "configured", Provider: "openai_compatible", BaseURL: &baseURL, IsDefault: true, SmallModel: &smallModel,
	}).Error)
	noOutline := &models.Summary{ID: "sum-no-outline", UserID: user.ID, TranscriptionID: job.ID, Status: "completed", Content: "A summary without outline."}
	called = false
	require.NoError(t, service.generateDescriptionForSummary(context.Background(), noOutline))
	require.False(t, called)
}

func TestGenerateDescriptionForSummaryRejectsInvalidOutput(t *testing.T) {
	logger.Init("silent")
	require.NoError(t, database.Initialize(filepath.Join(t.TempDir(), "scriberr.db")))
	database.DB.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	t.Cleanup(func() { _ = database.Close() })

	user, job := createTitleGenerationFixture(t, "job-description-invalid")
	baseURL := "http://127.0.0.1:1234/v1"
	smallModel := "small"
	require.NoError(t, database.DB.Create(&models.LLMConfig{
		UserID: user.ID, Name: "configured", Provider: "openai_compatible", BaseURL: &baseURL, IsDefault: true, SmallModel: &smallModel,
	}).Error)
	summary := createCompletedSummaryWithOutline(t, user.ID, job.ID, "sum-description-invalid", "A summary.", "An outline.")

	events := &recordingSummaryEvents{}
	service := NewService(repository.NewSummaryRepository(database.DB), repository.NewLLMConfigRepository(database.DB), repository.NewJobRepository(database.DB), Config{})
	service.SetEventPublisher(events)
	service.clientFor = func(*models.LLMConfig) (llm.Service, error) {
		return &fakeTitleLLM{content: "Only one line"}, nil
	}

	require.NoError(t, service.generateDescriptionForSummary(context.Background(), summary))

	var recording models.TranscriptionJob
	require.NoError(t, database.DB.First(&recording, "id = ?", *job.SourceFileHash).Error)
	require.Nil(t, recording.LLMDescription)
	require.Empty(t, events.fileEvents)
}

func TestSanitizeGeneratedDescriptionRequiresTwoPlainLines(t *testing.T) {
	require.Equal(t, "Line one.\nLine two.", sanitizeGeneratedDescription("  Line one.  \n\n Line two. "))
	require.Equal(t, "", sanitizeGeneratedDescription("Only one line."))
	require.Equal(t, "", sanitizeGeneratedDescription("One.\nTwo.\nThree."))
	require.Equal(t, "", sanitizeGeneratedDescription("- One.\n- Two."))
	require.Equal(t, "", sanitizeGeneratedDescription("Audio recording\nSpecific second line."))
}

func TestSanitizeGeneratedTitleEnforcesSevenWordsAndRejectsGenericTitles(t *testing.T) {
	require.Equal(t, "One Two Three Four Five Six Seven", sanitizeGeneratedTitle(`"One Two Three Four Five Six Seven Eight Nine."`))
	require.Equal(t, "", sanitizeGeneratedTitle("Audio Recording"))
	require.Equal(t, "A Specific Useful Title", sanitizeGeneratedTitle("  “A Specific Useful Title!”  "))
}

func createCompletedSummaryWithOutline(t *testing.T, userID uint, transcriptionID string, summaryID string, content string, outline string) *models.Summary {
	t.Helper()
	summary := &models.Summary{
		ID:              summaryID,
		UserID:          userID,
		TranscriptionID: transcriptionID,
		Status:          "completed",
		Content:         content,
	}
	require.NoError(t, database.DB.Create(summary).Error)
	widget := models.SummaryWidget{
		ID:            summaryID + "-outline-widget",
		UserID:        userID,
		Name:          "Outline",
		DisplayTitle:  "Outline",
		ContextSource: "summary",
		Prompt:        "Create an outline.",
		Enabled:       true,
		AlwaysEnabled: true,
	}
	require.NoError(t, database.DB.Create(&widget).Error)
	run := models.SummaryWidgetRun{
		ID:              summaryID + "-outline-run",
		SummaryID:       summary.ID,
		TranscriptionID: transcriptionID,
		WidgetID:        widget.ID,
		UserID:          userID,
		WidgetName:      widget.Name,
		DisplayTitle:    widget.DisplayTitle,
		ContextSource:   widget.ContextSource,
		Model:           "small",
		Provider:        "openai_compatible",
		Status:          "completed",
		Output:          outline,
	}
	require.NoError(t, database.DB.Create(&run).Error)
	return summary
}

func createTitleGenerationFixture(t *testing.T, id string) (models.User, models.TranscriptionJob) {
	t.Helper()
	user := models.User{Username: id + "-user", Password: "hash"}
	require.NoError(t, database.DB.Create(&user).Error)
	title := "Original title"
	transcript := `{"text":"Hello world.","segments":[{"id":"seg_000001","start":0,"end":1,"text":"Hello world."}],"words":[]}`
	sourceID := id + "-source"
	source := models.TranscriptionJob{
		ID:             sourceID,
		UserID:         user.ID,
		Title:          &title,
		Status:         models.StatusUploaded,
		AudioPath:      "/tmp/audio.wav",
		SourceFileName: "audio.wav",
	}
	require.NoError(t, database.DB.Create(&source).Error)
	job := models.TranscriptionJob{
		ID:             id,
		UserID:         user.ID,
		Title:          &title,
		Status:         models.StatusCompleted,
		AudioPath:      "/tmp/audio.wav",
		SourceFileName: "audio.wav",
		SourceFileHash: stringPtr(sourceID),
		Transcript:     &transcript,
	}
	require.NoError(t, database.DB.Create(&job).Error)
	return user, job
}

type recordingSummaryEvents struct {
	fileEvents []recordedFileEvent
}

type recordedFileEvent struct {
	name    string
	payload map[string]any
}

func (r *recordingSummaryEvents) PublishSummaryStatus(context.Context, StatusEvent) {}

func (r *recordingSummaryEvents) PublishFileEvent(_ context.Context, name string, payload map[string]any) {
	r.fileEvents = append(r.fileEvents, recordedFileEvent{name: name, payload: payload})
}

type fakeTitleLLM struct {
	content  string
	messages []llm.ChatMessage
}

func (f *fakeTitleLLM) GetModels(context.Context) ([]string, error) { return []string{"small"}, nil }

func (f *fakeTitleLLM) ChatCompletion(_ context.Context, _ string, messages []llm.ChatMessage, _ float64) (*llm.ChatResponse, error) {
	f.messages = append(f.messages, messages...)
	return chatResponse(f.content), nil
}

func (f *fakeTitleLLM) ChatCompletionStream(context.Context, string, []llm.ChatMessage, float64) (<-chan string, <-chan error) {
	return nil, nil
}

func (f *fakeTitleLLM) ChatCompletionStreamEvents(context.Context, string, []llm.ChatMessage, float64) (<-chan llm.StreamEvent, <-chan error) {
	return nil, nil
}

func (f *fakeTitleLLM) GetContextWindow(context.Context, string) (int, error) { return 4096, nil }

func chatResponse(content string) *llm.ChatResponse {
	response := &llm.ChatResponse{}
	response.Choices = append(response.Choices, struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	}{})
	response.Choices[0].Message.Role = "assistant"
	response.Choices[0].Message.Content = content
	return response
}

func stringPtr(value string) *string {
	return &value
}
