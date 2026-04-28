package database

import (
	"path/filepath"
	"testing"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type legacyUserTable legacyUser

func (legacyUserTable) TableName() string { return "users" }

type legacyAPIKeyTable legacyAPIKey

func (legacyAPIKeyTable) TableName() string { return "api_keys" }

type legacyRefreshTokenTable legacyRefreshToken

func (legacyRefreshTokenTable) TableName() string { return "refresh_tokens" }

type legacyTranscriptionProfileTable legacyTranscriptionProfile

func (legacyTranscriptionProfileTable) TableName() string { return "transcription_profiles" }

type legacyTranscriptionJobTable legacyTranscriptionJob

func (legacyTranscriptionJobTable) TableName() string { return "transcription_jobs" }

type legacyTranscriptionExecutionTable legacyTranscriptionExecution

func (legacyTranscriptionExecutionTable) TableName() string { return "transcription_job_executions" }

type legacySpeakerMappingTable legacySpeakerMapping

func (legacySpeakerMappingTable) TableName() string { return "speaker_mappings" }

type legacySummaryTemplateTable legacySummaryTemplate

func (legacySummaryTemplateTable) TableName() string { return "summary_templates" }

type legacySummarySettingTable legacySummarySetting

func (legacySummarySettingTable) TableName() string { return "summary_settings" }

type legacySummaryTable legacySummary

func (legacySummaryTable) TableName() string { return "summaries" }

type legacyLLMConfigTable legacyLLMConfig

func (legacyLLMConfigTable) TableName() string { return "llm_configs" }

type legacyChatSessionTable legacyChatSession

func (legacyChatSessionTable) TableName() string { return "chat_sessions" }

type legacyChatMessageTable legacyChatMessage

func (legacyChatMessageTable) TableName() string { return "chat_messages" }

func TestFreshSchemaInitialization(t *testing.T) {
	db := openMigratedTestDB(t, "fresh.db")

	expectedTables := []string{
		"schema_migrations",
		"users",
		"api_keys",
		"refresh_tokens",
		"transcription_profiles",
		"transcriptions",
		"transcription_executions",
		"speaker_mappings",
		"summary_templates",
		"summaries",
		"transcript_annotations",
		"chat_sessions",
		"chat_messages",
		"llm_profiles",
	}
	for _, table := range expectedTables {
		assert.True(t, db.Migrator().HasTable(table), "expected table %s", table)
	}

	assert.Equal(t, latestSchemaVersion, schemaVersion(t, db))
	assert.Equal(t, "wal", pragmaString(t, db, "journal_mode"))
	assert.True(t, hasIndex(t, db, "speaker_mappings", "idx_speaker_mappings_unique"))
	assert.True(t, hasIndex(t, db, "transcription_executions", "idx_transcription_executions_unique"))
	assert.True(t, hasIndex(t, db, "transcription_profiles", "idx_transcription_profiles_user_default_unique"))
	assert.True(t, hasIndex(t, db, "summary_templates", "idx_summary_templates_user_default_unique"))
	assert.True(t, hasIndex(t, db, "llm_profiles", "idx_llm_profiles_user_default_unique"))
	assert.True(t, hasIndex(t, db, "transcript_annotations", "idx_transcript_annotations_user_transcription_created_at"))
	assert.True(t, hasIndex(t, db, "transcript_annotations", "idx_transcript_annotations_user_kind_updated_at"))
	assert.True(t, hasIndex(t, db, "transcript_annotations", "idx_transcript_annotations_transcription_time"))

	title := "Fresh transcription"
	job := models.TranscriptionJob{UserID: 1, Title: &title, Status: models.StatusUploaded, AudioPath: "/tmp/audio.wav"}
	require.NoError(t, db.Create(&job).Error)

	mapping1 := models.SpeakerMapping{UserID: job.UserID, TranscriptionJobID: job.ID, OriginalSpeaker: "SPEAKER_00", CustomName: "Alice"}
	require.NoError(t, db.Create(&mapping1).Error)
	mapping2 := models.SpeakerMapping{UserID: job.UserID, TranscriptionJobID: job.ID, OriginalSpeaker: "SPEAKER_00", CustomName: "Bob"}
	require.Error(t, db.Create(&mapping2).Error)

}

func TestTranscriptAnnotationSchemaValidationAndSoftDelete(t *testing.T) {
	db := openMigratedTestDB(t, "transcript-annotations.db")

	user := models.User{Username: "annotation-user", Password: "pw"}
	require.NoError(t, db.Create(&user).Error)

	title := "Annotated transcript"
	job := models.TranscriptionJob{UserID: user.ID, Title: &title, Status: models.StatusCompleted, AudioPath: "/tmp/audio.wav"}
	require.NoError(t, db.Create(&job).Error)

	content := "Follow up"
	color := "yellow"
	startWord := 2
	endWord := 5
	annotation := models.TranscriptAnnotation{
		UserID:          user.ID,
		TranscriptionID: job.ID,
		Kind:            models.AnnotationKindNote,
		Content:         &content,
		Color:           &color,
		Quote:           "important quote",
		AnchorStartMS:   1200,
		AnchorEndMS:     3400,
		AnchorStartWord: &startWord,
		AnchorEndWord:   &endWord,
	}
	require.NoError(t, db.Create(&annotation).Error)
	assert.NotEmpty(t, annotation.ID)
	assert.Equal(t, models.AnnotationStatusActive, annotation.Status)
	assert.Equal(t, "{}", annotation.MetadataJSON)

	invalidKind := models.TranscriptAnnotation{
		UserID:          user.ID,
		TranscriptionID: job.ID,
		Kind:            models.AnnotationKind("bookmark"),
		Quote:           "bad kind",
		AnchorStartMS:   100,
		AnchorEndMS:     200,
	}
	require.Error(t, db.Create(&invalidKind).Error)

	invalidRange := models.TranscriptAnnotation{
		UserID:          user.ID,
		TranscriptionID: job.ID,
		Kind:            models.AnnotationKindHighlight,
		Quote:           "bad range",
		AnchorStartMS:   500,
		AnchorEndMS:     100,
	}
	require.Error(t, db.Create(&invalidRange).Error)

	missingTranscription := models.TranscriptAnnotation{
		UserID:          user.ID,
		TranscriptionID: "missing",
		Kind:            models.AnnotationKindHighlight,
		Quote:           "missing parent",
		AnchorStartMS:   100,
		AnchorEndMS:     200,
	}
	require.Error(t, db.Create(&missingTranscription).Error)

	require.NoError(t, db.Delete(&annotation).Error)

	var visibleCount int64
	require.NoError(t, db.Model(&models.TranscriptAnnotation{}).Where("id = ?", annotation.ID).Count(&visibleCount).Error)
	assert.Zero(t, visibleCount)

	var storedCount int64
	require.NoError(t, db.Unscoped().Model(&models.TranscriptAnnotation{}).Where("id = ?", annotation.ID).Count(&storedCount).Error)
	assert.Equal(t, int64(1), storedCount)
}

func TestTranscriptAnnotationHardDeleteCascadesWithTranscription(t *testing.T) {
	db := openMigratedTestDB(t, "transcript-annotation-cascade.db")

	user := models.User{Username: "cascade-annotation-user", Password: "pw"}
	require.NoError(t, db.Create(&user).Error)

	title := "Cascade transcript"
	job := models.TranscriptionJob{UserID: user.ID, Title: &title, Status: models.StatusCompleted, AudioPath: "/tmp/audio.wav"}
	require.NoError(t, db.Create(&job).Error)

	annotation := models.TranscriptAnnotation{
		UserID:          user.ID,
		TranscriptionID: job.ID,
		Kind:            models.AnnotationKindHighlight,
		Quote:           "highlighted quote",
		AnchorStartMS:   1000,
		AnchorEndMS:     2000,
	}
	require.NoError(t, db.Create(&annotation).Error)

	require.NoError(t, db.Unscoped().Delete(&job).Error)

	var count int64
	require.NoError(t, db.Unscoped().Model(&models.TranscriptAnnotation{}).Where("id = ?", annotation.ID).Count(&count).Error)
	assert.Zero(t, count)
}

func TestCreateExecutionAssignsSequentialNumbers(t *testing.T) {
	db := openMigratedTestDB(t, "execution-sequence.db")

	user := models.User{Username: "execution-user", Password: "pw"}
	require.NoError(t, db.Create(&user).Error)

	title := "Execution job"
	job := models.TranscriptionJob{UserID: user.ID, Title: &title, Status: models.StatusUploaded, AudioPath: "/tmp/audio.wav"}
	require.NoError(t, db.Create(&job).Error)

	jobRepo := repository.NewJobRepository(db)

	execution1 := &models.TranscriptionJobExecution{
		TranscriptionJobID: job.ID,
		UserID:             user.ID,
		Status:             models.StatusProcessing,
		StartedAt:          time.Now(),
	}
	require.NoError(t, jobRepo.CreateExecution(t.Context(), execution1))
	assert.Equal(t, 1, execution1.ExecutionNumber)

	var persistedJob models.TranscriptionJob
	require.NoError(t, db.First(&persistedJob, "id = ?", job.ID).Error)
	assert.NotNil(t, persistedJob.LatestExecutionID)
	assert.Equal(t, execution1.ID, *persistedJob.LatestExecutionID)

	execution2 := &models.TranscriptionJobExecution{
		TranscriptionJobID: job.ID,
		UserID:             user.ID,
		Status:             models.StatusFailed,
		StartedAt:          time.Now(),
	}
	require.NoError(t, jobRepo.CreateExecution(t.Context(), execution2))
	assert.Equal(t, 2, execution2.ExecutionNumber)

	require.NoError(t, db.First(&persistedJob, "id = ?", job.ID).Error)
	assert.NotNil(t, persistedJob.LatestExecutionID)
	assert.Equal(t, execution2.ID, *persistedJob.LatestExecutionID)

	var executions []models.TranscriptionJobExecution
	require.NoError(t, db.Where("transcription_id = ?", job.ID).Order("execution_number ASC").Find(&executions).Error)
	require.Len(t, executions, 2)
	assert.Equal(t, 1, executions[0].ExecutionNumber)
	assert.Equal(t, 2, executions[1].ExecutionNumber)
}

func TestDefaultRecordsAreScopedPerUser(t *testing.T) {
	db := openUnmigratedTestDB(t, "default-records.db")

	userA := models.User{Username: "default-user-a", Password: "pw-a"}
	userB := models.User{Username: "default-user-b", Password: "pw-b"}
	require.NoError(t, db.Create(&userA).Error)
	require.NoError(t, db.Create(&userB).Error)

	base := time.Now().Truncate(time.Second)

	// Create duplicate per-user defaults for legacy cleanup to normalize.
	profileAFirst := models.TranscriptionProfile{ID: "profile-a-old", UserID: userA.ID, Name: "profile-old", IsDefault: true, CreatedAt: base, UpdatedAt: base}
	profileASecond := models.TranscriptionProfile{ID: "profile-a-new", UserID: userA.ID, Name: "profile-new", IsDefault: true, CreatedAt: base.Add(time.Minute), UpdatedAt: base.Add(time.Minute)}
	profileB := models.TranscriptionProfile{ID: "profile-b", UserID: userB.ID, Name: "profile-b", IsDefault: true, CreatedAt: base, UpdatedAt: base}
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&profileAFirst).Error)
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&profileASecond).Error)
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&profileB).Error)

	summaryTemplateAFirst := models.SummaryTemplate{ID: "template-a-old", UserID: userA.ID, Name: "template-old", Prompt: "prompt", IsDefault: true, Model: "gpt", CreatedAt: base, UpdatedAt: base}
	summaryTemplateASecond := models.SummaryTemplate{ID: "template-a-new", UserID: userA.ID, Name: "template-new", Prompt: "prompt", IsDefault: true, Model: "gpt", CreatedAt: base.Add(time.Minute), UpdatedAt: base.Add(time.Minute)}
	summaryTemplateB := models.SummaryTemplate{ID: "template-b", UserID: userB.ID, Name: "template-b", Prompt: "prompt", IsDefault: true, Model: "gpt", CreatedAt: base, UpdatedAt: base}
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&summaryTemplateAFirst).Error)
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&summaryTemplateASecond).Error)
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&summaryTemplateB).Error)

	llmAFirst := models.LLMConfig{ID: 1000, UserID: userA.ID, Provider: "provider-a", APIKey: ptr("k1"), IsDefault: true, CreatedAt: base, UpdatedAt: base}
	llmASecond := models.LLMConfig{ID: 1001, UserID: userA.ID, Provider: "provider-b", APIKey: ptr("k2"), IsDefault: true, CreatedAt: base.Add(time.Minute), UpdatedAt: base.Add(time.Minute)}
	llmB := models.LLMConfig{ID: 1002, UserID: userB.ID, Provider: "provider-c", APIKey: ptr("k3"), IsDefault: true, CreatedAt: base, UpdatedAt: base}
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&llmAFirst).Error)
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&llmASecond).Error)
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&llmB).Error)

	require.NoError(t, recordSchemaVersion(db, 1))
	require.NoError(t, Migrate(db))

	var profileA, profileBDef models.TranscriptionProfile
	require.NoError(t, db.Where("user_id = ? AND id = ?", userA.ID, "profile-a-new").First(&profileA).Error)
	require.NoError(t, db.Where("user_id = ?", userB.ID).Where("is_default = ?", true).First(&profileBDef).Error)
	assert.True(t, profileA.IsDefault)
	assert.True(t, profileBDef.IsDefault)

	var summaryCountA int64
	require.NoError(t, db.Model(&models.SummaryTemplate{}).Where("user_id = ? AND is_default = ?", userA.ID, true).Count(&summaryCountA).Error)
	assert.Equal(t, int64(1), summaryCountA)

	var llmCountA int64
	require.NoError(t, db.Model(&models.LLMConfig{}).Where("user_id = ? AND is_default = ?", userA.ID, true).Count(&llmCountA).Error)
	assert.Equal(t, int64(1), llmCountA)

	profileRepo := repository.NewProfileRepository(db)
	llmRepo := repository.NewLLMConfigRepository(db)

	userADefProfile, err := profileRepo.FindDefaultByUser(t.Context(), userA.ID)
	require.NoError(t, err)
	assert.Equal(t, "profile-a-new", userADefProfile.ID)

	userBDefProfile, err := profileRepo.FindDefaultByUser(t.Context(), userB.ID)
	require.NoError(t, err)
	assert.Equal(t, "profile-b", userBDefProfile.ID)

	var userADefTemplate models.SummaryTemplate
	require.NoError(t, db.Where("user_id = ? AND is_default = ?", userA.ID, true).First(&userADefTemplate).Error)
	assert.Equal(t, "template-a-new", userADefTemplate.ID)

	userAActiveLLM, err := llmRepo.GetActiveByUser(t.Context(), userA.ID)
	require.NoError(t, err)
	assert.Equal(t, "provider-b", userAActiveLLM.Provider)

	userBActiveLLM, err := llmRepo.GetActiveByUser(t.Context(), userB.ID)
	require.NoError(t, err)
	assert.Equal(t, "provider-c", userBActiveLLM.Provider)
}

func TestSchemaUpgradeRunsVersionedBackfill(t *testing.T) {
	db := openUnmigratedTestDB(t, "schema-upgrade.db")

	userA := models.User{Username: "upgrade-a", Password: "pw-a"}
	userB := models.User{Username: "upgrade-b", Password: "pw-b"}
	require.NoError(t, db.Create(&userA).Error)
	require.NoError(t, db.Create(&userB).Error)

	base := time.Now().Truncate(time.Second)
	profileA := models.TranscriptionProfile{
		ID:        "upgrade-profile-a",
		UserID:    userA.ID,
		Name:      "profile-a",
		IsDefault: true,
		Parameters: models.WhisperXParams{
			Model:       "medium",
			ModelFamily: "whisper",
			Device:      "cpu",
			ComputeType: "float32",
		},
		CreatedAt: base,
		UpdatedAt: base,
	}
	profileB := models.TranscriptionProfile{
		ID:        "upgrade-profile-b",
		UserID:    userB.ID,
		Name:      "profile-b",
		IsDefault: true,
		Parameters: models.WhisperXParams{
			Model:       "large-v3",
			ModelFamily: "whisper",
			Device:      "cuda",
			ComputeType: "float16",
		},
		CreatedAt: base,
		UpdatedAt: base,
	}
	require.NoError(t, db.Create(&profileA).Error)
	require.NoError(t, db.Create(&profileB).Error)

	require.NoError(t, recordSchemaVersion(db, 1))
	require.NoError(t, db.Exec("UPDATE transcription_profiles SET config_json = ''").Error)

	require.NoError(t, Migrate(db))

	assert.Equal(t, latestSchemaVersion, schemaVersion(t, db))

	var reloadedA, reloadedB models.TranscriptionProfile
	require.NoError(t, db.First(&reloadedA, "id = ?", profileA.ID).Error)
	require.NoError(t, db.First(&reloadedB, "id = ?", profileB.ID).Error)
	assert.True(t, reloadedA.IsDefault)
	assert.True(t, reloadedB.IsDefault)
	assert.Equal(t, "medium", reloadedA.Parameters.Model)
	assert.Equal(t, "large-v3", reloadedB.Parameters.Model)
}

func TestSchemaUpgradePreservesUpdatedAt(t *testing.T) {
	db := openUnmigratedTestDB(t, "schema-upgrade-updated-at.db")

	originalUpdatedAt := time.Date(2024, 12, 31, 12, 0, 0, 0, time.UTC)
	user := models.User{
		ID:        77,
		Username:  "preserve-updated-at",
		Password:  "pw",
		UpdatedAt: originalUpdatedAt,
	}
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&user).Error)
	require.NoError(t, recordSchemaVersion(db, 1))
	require.NoError(t, db.Exec("UPDATE users SET settings_json = '' WHERE id = ?", user.ID).Error)

	require.NoError(t, Migrate(db))

	var reloaded models.User
	require.NoError(t, db.First(&reloaded, "id = ?", user.ID).Error)
	assert.True(t, reloaded.UpdatedAt.Equal(originalUpdatedAt))
}

func TestSchemaUpgradeDoesNotInventCompletionOrFailureTimestamps(t *testing.T) {
	db := openMigratedTestDB(t, "schema-upgrade-no-invented-timestamps.db")

	title := "legacy-completed-without-timestamp"
	job := models.TranscriptionJob{
		ID:        "job-no-completed-at",
		UserID:    1,
		Title:     &title,
		Status:    models.StatusCompleted,
		AudioPath: "/tmp/audio.wav",
	}
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&job).Error)
	require.NoError(t, db.Exec("UPDATE transcriptions SET completed_at = NULL, metadata_json = '' WHERE id = ?", job.ID).Error)

	execution := models.TranscriptionJobExecution{
		ID:                 "exec-no-failed-at",
		TranscriptionJobID: job.ID,
		UserID:             1,
		ExecutionNumber:    1,
		Status:             models.StatusFailed,
		StartedAt:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&execution).Error)
	require.NoError(t, db.Exec("UPDATE transcription_executions SET failed_at = NULL, request_json = '', config_json = '' WHERE id = ?", execution.ID).Error)

	require.NoError(t, db.Exec("DELETE FROM schema_migrations").Error)
	require.NoError(t, recordSchemaVersion(db, 1))
	require.NoError(t, Migrate(db))

	var reloadedJob models.TranscriptionJob
	require.NoError(t, db.First(&reloadedJob, "id = ?", job.ID).Error)
	assert.Nil(t, reloadedJob.CompletedAt)

	var reloadedExec models.TranscriptionJobExecution
	require.NoError(t, db.First(&reloadedExec, "id = ?", execution.ID).Error)
	assert.Nil(t, reloadedExec.FailedAt)
}

func TestDetectLegacySchemaWithLegacySameNameTables(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy-same-name.db")
	createLegacyDatabase(t, dbPath, false)

	db, err := Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeDB(db)) })

	require.NoError(t, db.Exec("DROP TABLE transcription_jobs").Error)
	require.NoError(t, db.Exec("DROP TABLE transcription_job_executions").Error)
	require.NoError(t, db.Exec("DROP TABLE llm_configs").Error)
	require.NoError(t, db.Exec("DROP TABLE summary_settings").Error)

	legacy, err := detectLegacySchema(db)
	require.NoError(t, err)
	assert.True(t, legacy)
}

func TestExtractNormalizedWherePredicateAcceptsEquivalentPartialIndexPredicates(t *testing.T) {
	cases := []struct {
		name string
		sql  string
	}{
		{
			name: "quoted identifier",
			sql:  `CREATE UNIQUE INDEX idx_profiles_default ON transcription_profiles(user_id) WHERE ("is_default" = 1)`,
		},
		{
			name: "true literal",
			sql:  `CREATE UNIQUE INDEX idx_profiles_default ON transcription_profiles(user_id) WHERE is_default = TRUE`,
		},
		{
			name: "newline before where",
			sql:  "CREATE UNIQUE INDEX idx_profiles_default ON transcription_profiles(user_id)\nWHERE is_default = 1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, "is_default=1", extractNormalizedWherePredicate(tc.sql))
		})
	}
}

func TestLegacyMigrationPreservesData(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	createLegacyDatabase(t, dbPath, true)

	db, err := Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeDB(db)) })

	require.NoError(t, Migrate(db))
	require.NoError(t, Migrate(db))

	assert.True(t, db.Migrator().HasTable("transcriptions"))
	assert.True(t, db.Migrator().HasTable("transcription_jobs"), "legacy table should be preserved")
	assert.True(t, db.Migrator().HasTable("legacy_users"), "same-name legacy table should be archived")
	assert.Equal(t, latestSchemaVersion, schemaVersion(t, db))

	var user models.User
	require.NoError(t, db.First(&user, "id = ?", 7).Error)
	assert.Equal(t, "legacy-admin", user.Username)
	assert.True(t, user.AutoTranscriptionEnabled)
	require.NotNil(t, user.DefaultProfileID)
	assert.Equal(t, "profile-1", *user.DefaultProfileID)
	assert.Equal(t, "gpt-4o-mini", user.SummaryDefaultModel)

	var transcription models.TranscriptionJob
	require.NoError(t, db.First(&transcription, "id = ?", "job-1").Error)
	assert.Equal(t, models.StatusPending, transcription.Status)
	assert.Equal(t, "/legacy/audio.wav", transcription.AudioPath)
	require.NotNil(t, transcription.Transcript)
	assert.Contains(t, *transcription.Transcript, "hello world")
	require.NotNil(t, transcription.Summary)
	assert.Equal(t, "legacy summary cache", *transcription.Summary)
	assert.Equal(t, "medium", transcription.Parameters.Model)
	require.NotNil(t, transcription.LatestExecutionID)

	var execution models.TranscriptionJobExecution
	require.NoError(t, db.First(&execution, "id = ?", *transcription.LatestExecutionID).Error)
	assert.Equal(t, "job-1", execution.TranscriptionJobID)
	assert.Equal(t, 1, execution.ExecutionNumber)
	assert.Equal(t, models.StatusCompleted, execution.Status)
	assert.Equal(t, "medium", execution.ActualParameters.Model)
	var mappings []models.SpeakerMapping
	require.NoError(t, db.Where("transcription_id = ?", "job-1").Find(&mappings).Error)
	require.Len(t, mappings, 1)
	assert.Equal(t, "Latest Alice", mappings[0].CustomName)

	var template models.SummaryTemplate
	require.NoError(t, db.First(&template, "id = ?", "template-1").Error)
	assert.Equal(t, "gpt-4o", template.Model)
	assert.True(t, template.IncludeSpeakerInfo)

	var summary models.Summary
	require.NoError(t, db.First(&summary, "id = ?", "summary-1").Error)
	assert.Equal(t, "job-1", summary.TranscriptionID)
	assert.Equal(t, "gpt-4o", summary.Model)
	assert.Equal(t, "completed", summary.Status)

	var chatSession models.ChatSession
	require.NoError(t, db.First(&chatSession, "id = ?", "chat-1").Error)
	assert.Equal(t, "job-1", chatSession.TranscriptionID)
	assert.Equal(t, 2, chatSession.MessageCount)
	assert.True(t, chatSession.IsActive)

	var chatMessages []models.ChatMessage
	require.NoError(t, db.Where("chat_session_id = ?", "chat-1").Order("id ASC").Find(&chatMessages).Error)
	require.Len(t, chatMessages, 2)
	assert.Equal(t, "chat-1", chatMessages[0].SessionID)
	require.NotNil(t, chatMessages[1].TokensUsed)
	assert.Equal(t, 42, *chatMessages[1].TokensUsed)

	keyRepo := repository.NewAPIKeyRepository(db)
	migratedKey, err := keyRepo.FindByKey(t.Context(), "legacy-api-key-secret")
	require.NoError(t, err)
	assert.Equal(t, "legacy-a", migratedKey.KeyPrefix)
	assert.True(t, migratedKey.IsActive)
	assert.Equal(t, "legacy description", derefString(migratedKey.Description))

	tokenRepo := repository.NewRefreshTokenRepository(db)
	migratedToken, err := tokenRepo.FindByHash(t.Context(), "legacy-token-hash")
	require.NoError(t, err)
	assert.True(t, migratedToken.Revoked)

	llmRepo := repository.NewLLMConfigRepository(db)
	llmConfig, err := llmRepo.GetActive(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "openai", llmConfig.Provider)
	require.NotNil(t, llmConfig.APIKey)
	assert.Equal(t, "openai-secret", *llmConfig.APIKey)
	require.NotNil(t, llmConfig.OpenAIBaseURL)
	assert.Equal(t, "https://openai.example", *llmConfig.OpenAIBaseURL)
}

func TestLegacyMigrationOnEmptyDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy-empty.db")
	createLegacyDatabase(t, dbPath, false)

	db, err := Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeDB(db)) })

	require.NoError(t, Migrate(db))

	assert.Equal(t, latestSchemaVersion, schemaVersion(t, db))
	var count int64
	require.NoError(t, db.Model(&models.User{}).Count(&count).Error)
	assert.Zero(t, count, "empty legacy DB should not invent a user")
}

func openMigratedTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), name)
	db, err := Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeDB(db)) })
	require.NoError(t, Migrate(db))
	return db
}

func openUnmigratedTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), name)
	db, err := Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeDB(db)) })
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.TranscriptionProfile{},
		&models.SummaryTemplate{},
		&models.LLMConfig{},
	))
	return db
}

func schemaVersion(t *testing.T, db *gorm.DB) int {
	t.Helper()
	var migration schemaMigration
	require.NoError(t, db.Order("version DESC").First(&migration).Error)
	return migration.Version
}

func pragmaString(t *testing.T, db *gorm.DB, pragma string) string {
	t.Helper()
	var value string
	require.NoError(t, db.Raw("PRAGMA "+pragma).Scan(&value).Error)
	return value
}

func hasIndex(t *testing.T, db *gorm.DB, tableName, indexName string) bool {
	t.Helper()
	type row struct {
		Name string `gorm:"column:name"`
	}
	var rows []row
	require.NoError(t, db.Raw("PRAGMA index_list('"+tableName+"')").Scan(&rows).Error)
	for _, row := range rows {
		if row.Name == indexName {
			return true
		}
	}
	return false
}

func createLegacyDatabase(t *testing.T, dbPath string, withData bool) {
	t.Helper()
	db, err := Open(dbPath)
	require.NoError(t, err)
	defer func() { require.NoError(t, closeDB(db)) }()

	require.NoError(t, db.AutoMigrate(
		&legacyUserTable{},
		&legacyAPIKeyTable{},
		&legacyRefreshTokenTable{},
		&legacyTranscriptionProfileTable{},
		&legacyTranscriptionJobTable{},
		&legacyTranscriptionExecutionTable{},
		&legacySpeakerMappingTable{},
		&legacySummaryTemplateTable{},
		&legacySummarySettingTable{},
		&legacySummaryTable{},
		&legacyLLMConfigTable{},
		&legacyChatSessionTable{},
		&legacyChatMessageTable{},
	))

	if !withData {
		return
	}

	defaultProfileID := "profile-1"
	now := time.Now().UTC().Truncate(time.Second)
	completedAt := now.Add(10 * time.Minute)
	processingDuration := int64(600000)
	transcriptJSON := `{"text":"hello world","segments":[{"start":0,"end":1,"text":"hello world"}]}`
	title := "Legacy job"
	profileDescription := "legacy profile"
	summaryDescription := "legacy summary template"
	openAIBaseURL := "https://openai.example"
	openAIKey := "openai-secret"
	lastUsed := now.Add(2 * time.Hour)
	errorMessage := "old error"

	user := legacyUser{ID: 7, Username: "legacy-admin", Password: "hashed", DefaultProfileID: &defaultProfileID, AutoTranscriptionEnabled: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Table("users").Create(&user).Error)

	profile := legacyTranscriptionProfile{ID: "profile-1", Name: "Legacy Profile", Description: &profileDescription, IsDefault: true, Parameters: models.WhisperXParams{Model: "medium", ModelFamily: "whisper", Device: "cpu", ComputeType: "float32", Diarize: true}, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Table("transcription_profiles").Create(&profile).Error)

	job := legacyTranscriptionJob{ID: "job-1", Title: &title, Status: "pending", AudioPath: "/legacy/audio.wav", Transcript: &transcriptJSON, Diarization: true, Summary: ptr("legacy summary cache"), ErrorMessage: &errorMessage, CreatedAt: now, UpdatedAt: completedAt, Parameters: models.WhisperXParams{Model: "medium", ModelFamily: "whisper", Device: "cpu", ComputeType: "float32", Diarize: true}}
	require.NoError(t, db.Table("transcription_jobs").Create(&job).Error)

	execution := legacyTranscriptionExecution{ID: "exec-1", TranscriptionJobID: "job-1", StartedAt: now, CompletedAt: &completedAt, ProcessingDuration: &processingDuration, ActualParameters: job.Parameters, Status: "completed", CreatedAt: completedAt, UpdatedAt: completedAt}
	require.NoError(t, db.Table("transcription_job_executions").Create(&execution).Error)

	olderMapping := legacySpeakerMapping{ID: 1, TranscriptionJobID: "job-1", OriginalSpeaker: "SPEAKER_00", CustomName: "Old Alice", CreatedAt: now, UpdatedAt: now}
	newerMapping := legacySpeakerMapping{ID: 2, TranscriptionJobID: "job-1", OriginalSpeaker: "SPEAKER_00", CustomName: "Latest Alice", CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute)}
	require.NoError(t, db.Table("speaker_mappings").Create(&olderMapping).Error)
	require.NoError(t, db.Table("speaker_mappings").Create(&newerMapping).Error)

	summaryTemplate := legacySummaryTemplate{ID: "template-1", Name: "Legacy Summary", Description: &summaryDescription, Model: "gpt-4o", Prompt: "Summarize", IncludeSpeakerInfo: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Table("summary_templates").Create(&summaryTemplate).Error)
	require.NoError(t, db.Table("summary_settings").Create(&legacySummarySetting{ID: 1, DefaultModel: "gpt-4o-mini", UpdatedAt: now}).Error)
	require.NoError(t, db.Table("summaries").Create(&legacySummary{ID: "summary-1", TranscriptionID: "job-1", TemplateID: &summaryTemplate.ID, Model: "gpt-4o", Content: "summary body", CreatedAt: completedAt, UpdatedAt: completedAt}).Error)

	llmConfig := legacyLLMConfig{ID: 3, Provider: "openai", OpenAIBaseURL: &openAIBaseURL, APIKey: &openAIKey, IsActive: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Table("llm_configs").Create(&llmConfig).Error)

	session := legacyChatSession{ID: "chat-1", JobID: "job-1", TranscriptionID: "job-1", Title: "Legacy chat", Model: "gpt-4o-mini", Provider: "openai", MessageCount: 2, LastActivityAt: &completedAt, IsActive: true, CreatedAt: now, UpdatedAt: completedAt}
	require.NoError(t, db.Table("chat_sessions").Create(&session).Error)
	require.NoError(t, db.Table("chat_messages").Create(&legacyChatMessage{ID: 1, SessionID: "chat-1", ChatSessionID: "chat-1", Role: "user", Content: "Hello", CreatedAt: now}).Error)
	require.NoError(t, db.Table("chat_messages").Create(&legacyChatMessage{ID: 2, SessionID: "chat-1", ChatSessionID: "chat-1", Role: "assistant", Content: "Hi", TokensUsed: ptrInt(42), CreatedAt: completedAt}).Error)

	require.NoError(t, db.Table("api_keys").Create(&legacyAPIKey{ID: 9, Key: "legacy-api-key-secret", Name: "Legacy API key", Description: ptr("legacy description"), IsActive: true, LastUsed: &lastUsed, CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Table("refresh_tokens").Create(&legacyRefreshToken{ID: 11, UserID: 7, Hashed: "legacy-token-hash", ExpiresAt: now.Add(24 * time.Hour), Revoked: true, CreatedAt: now, UpdatedAt: completedAt}).Error)
}

func ptr[T any](v T) *T { return &v }
func ptrInt(v int) *int { return &v }

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
