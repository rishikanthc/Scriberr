package database

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"scriberr/internal/auth"
	"scriberr/internal/models"

	"gorm.io/gorm"
)

const legacyPrefix = "legacy_"

type legacyUser struct {
	ID                       uint
	Username                 string
	Password                 string
	DefaultProfileID         *string
	AutoTranscriptionEnabled bool
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type legacyAPIKey struct {
	ID          uint
	Key         string
	Name        string
	Description *string
	IsActive    bool
	LastUsed    *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type legacyRefreshToken struct {
	ID        uint
	UserID    uint
	Hashed    string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type legacyTranscriptionProfile struct {
	ID          string
	Name        string
	Description *string
	IsDefault   bool
	Parameters  models.WhisperXParams `gorm:"embedded"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type legacyTranscriptionJob struct {
	ID           string
	Title        *string
	Status       string
	AudioPath    string
	Transcript   *string
	Diarization  bool
	Summary      *string
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt
	Parameters   models.WhisperXParams `gorm:"embedded"`
}

type legacyTranscriptionExecution struct {
	ID                 string
	TranscriptionJobID string
	StartedAt          time.Time
	CompletedAt        *time.Time
	ProcessingDuration *int64
	ActualParameters   models.WhisperXParams `gorm:"embedded;embeddedPrefix:actual_"`
	Status             string
	ErrorMessage       *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type legacySpeakerMapping struct {
	ID                 uint
	TranscriptionJobID string
	OriginalSpeaker    string
	CustomName         string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type legacySummaryTemplate struct {
	ID                 string
	Name               string
	Description        *string
	Model              string
	Prompt             string
	IncludeSpeakerInfo bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type legacySummarySetting struct {
	ID           uint
	DefaultModel string
	UpdatedAt    time.Time
}

type legacySummary struct {
	ID              string
	TranscriptionID string
	TemplateID      *string
	Model           string
	Content         string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type legacyLLMConfig struct {
	ID            uint
	Provider      string
	BaseURL       *string
	OpenAIBaseURL *string
	APIKey        *string
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type legacyChatSession struct {
	ID              string
	JobID           string
	TranscriptionID string
	Title           string
	Model           string
	Provider        string
	SystemContext   *string
	MessageCount    int
	LastActivityAt  *time.Time
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type legacyChatMessage struct {
	ID            uint
	SessionID     string
	ChatSessionID string
	Role          string
	Content       string
	TokensUsed    *int
	CreatedAt     time.Time
}

func migrateLegacy(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := archiveConflictingLegacyTables(tx); err != nil {
			return err
		}
		if err := createTargetSchema(tx); err != nil {
			return err
		}

		userID, err := migrateUsers(tx)
		if err != nil {
			return err
		}
		if userID == 0 {
			hasData, err := legacyHasOwnedData(tx)
			if err != nil {
				return err
			}
			if hasData {
				userID, err = createMigrationUser(tx)
				if err != nil {
					return err
				}
			}
		}

		if err := migrateProfiles(tx, userID); err != nil {
			return err
		}
		if err := migrateTranscriptions(tx, userID); err != nil {
			return err
		}
		if err := migrateExecutions(tx, userID); err != nil {
			return err
		}
		if err := migrateSpeakerMappings(tx, userID); err != nil {
			return err
		}
		if err := migrateSummaryTemplates(tx, userID); err != nil {
			return err
		}
		if err := migrateSummaries(tx, userID); err != nil {
			return err
		}
		if err := migrateLLMProfiles(tx, userID); err != nil {
			return err
		}
		if err := migrateChatSessions(tx, userID); err != nil {
			return err
		}
		if err := migrateChatMessages(tx, userID); err != nil {
			return err
		}
		if err := migrateAPIKeys(tx, userID); err != nil {
			return err
		}
		if err := migrateRefreshTokens(tx, userID); err != nil {
			return err
		}
		if err := migrateSummarySettings(tx, userID); err != nil {
			return err
		}
		if err := ensureSingleDefaultPerUser(tx); err != nil {
			return err
		}
		if err := updateLatestExecutions(tx); err != nil {
			return err
		}
		return recordSchemaVersion(tx, latestSchemaVersion)
	})
}

func archiveConflictingLegacyTables(tx *gorm.DB) error {
	sameNameTables := []string{
		"users",
		"api_keys",
		"refresh_tokens",
		"transcription_profiles",
		"speaker_mappings",
		"summary_templates",
		"summaries",
		"chat_sessions",
		"chat_messages",
	}
	for _, table := range sameNameTables {
		if !tx.Migrator().HasTable(table) {
			continue
		}
		archived := legacyPrefix + table
		if tx.Migrator().HasTable(archived) {
			continue
		}
		if err := tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", table, archived)).Error; err != nil {
			return fmt.Errorf("archive legacy table %s: %w", table, err)
		}
	}
	return nil
}

func migrateUsers(tx *gorm.DB) (uint, error) {
	source := legacyPrefix + "users"
	if !tx.Migrator().HasTable(source) {
		return 0, nil
	}
	var users []legacyUser
	if err := tx.Table(source).Order("id ASC").Find(&users).Error; err != nil {
		return 0, fmt.Errorf("load legacy users: %w", err)
	}
	var defaultUserID uint
	for _, legacyUser := range users {
		user := models.User{
			ID:                       legacyUser.ID,
			Username:                 legacyUser.Username,
			Password:                 legacyUser.Password,
			Role:                     "admin",
			CreatedAt:                legacyUser.CreatedAt,
			UpdatedAt:                legacyUser.UpdatedAt,
			DefaultProfileID:         legacyUser.DefaultProfileID,
			AutoTranscriptionEnabled: legacyUser.AutoTranscriptionEnabled,
		}
		if err := tx.Create(&user).Error; err != nil {
			return 0, fmt.Errorf("create migrated user %d: %w", legacyUser.ID, err)
		}
		if defaultUserID == 0 {
			defaultUserID = user.ID
		}
	}
	return defaultUserID, nil
}

func legacyHasOwnedData(tx *gorm.DB) (bool, error) {
	tables := []string{
		"transcription_jobs",
		legacyPrefix + "transcription_profiles",
		legacyPrefix + "summary_templates",
		legacyPrefix + "chat_sessions",
		"llm_configs",
		legacyPrefix + "api_keys",
		legacyPrefix + "refresh_tokens",
	}
	for _, table := range tables {
		if !tx.Migrator().HasTable(table) {
			continue
		}
		var count int64
		if err := tx.Table(table).Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	return false, nil
}

func createMigrationUser(tx *gorm.DB) (uint, error) {
	password, err := auth.HashPassword("migration-placeholder-password")
	if err != nil {
		return 0, fmt.Errorf("hash migration user password: %w", err)
	}
	user := models.User{
		Username: "migrated-user",
		Password: password,
		Role:     "admin",
	}
	if err := tx.Create(&user).Error; err != nil {
		return 0, fmt.Errorf("create migration user: %w", err)
	}
	return user.ID, nil
}

func migrateProfiles(tx *gorm.DB, userID uint) error {
	source := legacyPrefix + "transcription_profiles"
	if !tx.Migrator().HasTable(source) {
		return nil
	}
	var profiles []legacyTranscriptionProfile
	if err := tx.Table(source).Order("created_at ASC").Find(&profiles).Error; err != nil {
		return fmt.Errorf("load legacy profiles: %w", err)
	}
	for _, legacyProfile := range profiles {
		profile := models.TranscriptionProfile{
			ID:          legacyProfile.ID,
			UserID:      userID,
			Name:        legacyProfile.Name,
			Description: legacyProfile.Description,
			IsDefault:   legacyProfile.IsDefault,
			Parameters:  legacyProfile.Parameters,
			CreatedAt:   legacyProfile.CreatedAt,
			UpdatedAt:   legacyProfile.UpdatedAt,
		}
		if err := tx.Create(&profile).Error; err != nil {
			return fmt.Errorf("create migrated profile %s: %w", legacyProfile.ID, err)
		}
	}
	return nil
}

func migrateTranscriptions(tx *gorm.DB, userID uint) error {
	if !tx.Migrator().HasTable("transcription_jobs") {
		return nil
	}
	var jobs []legacyTranscriptionJob
	if err := tx.Table("transcription_jobs").Order("created_at ASC").Find(&jobs).Error; err != nil {
		return fmt.Errorf("load legacy transcriptions: %w", err)
	}
	for _, legacyJob := range jobs {
		status := mapLegacyStatus(legacyJob.Status)
		job := models.TranscriptionJob{
			ID:           legacyJob.ID,
			UserID:       userID,
			Title:        legacyJob.Title,
			Status:       status,
			AudioPath:    legacyJob.AudioPath,
			Transcript:   legacyJob.Transcript,
			ErrorMessage: legacyJob.ErrorMessage,
			CreatedAt:    legacyJob.CreatedAt,
			UpdatedAt:    legacyJob.UpdatedAt,
			DeletedAt:    legacyJob.DeletedAt,
			Diarization:  legacyJob.Diarization,
			Summary:      legacyJob.Summary,
			Parameters:   legacyJob.Parameters,
		}
		if status == models.StatusCompleted {
			completedAt := legacyJob.UpdatedAt
			job.CompletedAt = &completedAt
		}
		if err := tx.Create(&job).Error; err != nil {
			return fmt.Errorf("create migrated transcription %s: %w", legacyJob.ID, err)
		}
	}
	return nil
}

func migrateExecutions(tx *gorm.DB, userID uint) error {
	if !tx.Migrator().HasTable("transcription_job_executions") {
		return nil
	}
	var executions []legacyTranscriptionExecution
	if err := tx.Table("transcription_job_executions").Order("transcription_job_id ASC, created_at ASC, started_at ASC").Find(&executions).Error; err != nil {
		return fmt.Errorf("load legacy executions: %w", err)
	}
	counters := make(map[string]int)
	for _, legacyExec := range executions {
		counters[legacyExec.TranscriptionJobID]++
		exec := models.TranscriptionJobExecution{
			ID:                 legacyExec.ID,
			TranscriptionJobID: legacyExec.TranscriptionJobID,
			UserID:             userID,
			ExecutionNumber:    counters[legacyExec.TranscriptionJobID],
			TriggerType:        "manual",
			Status:             mapLegacyStatus(legacyExec.Status),
			StartedAt:          legacyExec.StartedAt,
			CompletedAt:        legacyExec.CompletedAt,
			ErrorMessage:       legacyExec.ErrorMessage,
			CreatedAt:          legacyExec.CreatedAt,
			ProcessingDuration: legacyExec.ProcessingDuration,
			ActualParameters:   legacyExec.ActualParameters,
		}
		if exec.Status == models.StatusFailed && legacyExec.CompletedAt != nil {
			exec.FailedAt = legacyExec.CompletedAt
		}
		if err := tx.Create(&exec).Error; err != nil {
			return fmt.Errorf("create migrated execution %s: %w", legacyExec.ID, err)
		}
	}
	return nil
}

func migrateSpeakerMappings(tx *gorm.DB, userID uint) error {
	source := legacyPrefix + "speaker_mappings"
	if !tx.Migrator().HasTable(source) {
		return nil
	}
	var mappings []legacySpeakerMapping
	if err := tx.Table(source).Order("transcription_job_id ASC, updated_at DESC, id DESC").Find(&mappings).Error; err != nil {
		return fmt.Errorf("load legacy speaker mappings: %w", err)
	}
	deduped := make(map[string]legacySpeakerMapping)
	order := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
		key := mapping.TranscriptionJobID + "::" + mapping.OriginalSpeaker
		if _, ok := deduped[key]; ok {
			continue
		}
		deduped[key] = mapping
		order = append(order, key)
	}
	for _, key := range order {
		mapping := deduped[key]
		row := models.SpeakerMapping{
			ID:                 mapping.ID,
			UserID:             userID,
			TranscriptionJobID: mapping.TranscriptionJobID,
			OriginalSpeaker:    mapping.OriginalSpeaker,
			CustomName:         mapping.CustomName,
			CreatedAt:          mapping.CreatedAt,
			UpdatedAt:          mapping.UpdatedAt,
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("create migrated speaker mapping %d: %w", mapping.ID, err)
		}
	}
	return nil
}

func migrateSummaryTemplates(tx *gorm.DB, userID uint) error {
	source := legacyPrefix + "summary_templates"
	if !tx.Migrator().HasTable(source) {
		return nil
	}
	var templates []legacySummaryTemplate
	if err := tx.Table(source).Order("created_at ASC").Find(&templates).Error; err != nil {
		return fmt.Errorf("load legacy summary templates: %w", err)
	}
	for _, legacyTemplate := range templates {
		template := models.SummaryTemplate{
			ID:                 legacyTemplate.ID,
			UserID:             userID,
			Name:               legacyTemplate.Name,
			Description:        legacyTemplate.Description,
			Prompt:             legacyTemplate.Prompt,
			Model:              legacyTemplate.Model,
			IncludeSpeakerInfo: legacyTemplate.IncludeSpeakerInfo,
			CreatedAt:          legacyTemplate.CreatedAt,
			UpdatedAt:          legacyTemplate.UpdatedAt,
		}
		if err := tx.Create(&template).Error; err != nil {
			return fmt.Errorf("create migrated summary template %s: %w", legacyTemplate.ID, err)
		}
	}
	return nil
}

func migrateSummaries(tx *gorm.DB, userID uint) error {
	source := legacyPrefix + "summaries"
	if !tx.Migrator().HasTable(source) {
		return nil
	}
	var summaries []legacySummary
	if err := tx.Table(source).Order("created_at ASC").Find(&summaries).Error; err != nil {
		return fmt.Errorf("load legacy summaries: %w", err)
	}
	for _, legacySummary := range summaries {
		summary := models.Summary{
			ID:              legacySummary.ID,
			TranscriptionID: legacySummary.TranscriptionID,
			UserID:          userID,
			TemplateID:      legacySummary.TemplateID,
			Model:           legacySummary.Model,
			Content:         legacySummary.Content,
			Status:          "completed",
			CreatedAt:       legacySummary.CreatedAt,
			UpdatedAt:       legacySummary.UpdatedAt,
		}
		if err := tx.Create(&summary).Error; err != nil {
			return fmt.Errorf("create migrated summary %s: %w", legacySummary.ID, err)
		}
	}
	return nil
}

func migrateLLMProfiles(tx *gorm.DB, userID uint) error {
	if !tx.Migrator().HasTable("llm_configs") {
		return nil
	}
	var configs []legacyLLMConfig
	if err := tx.Table("llm_configs").Order("created_at ASC, id ASC").Find(&configs).Error; err != nil {
		return fmt.Errorf("load legacy llm configs: %w", err)
	}
	defaultID := uint(0)
	for _, cfg := range configs {
		if cfg.IsActive {
			defaultID = cfg.ID
		}
	}
	if defaultID == 0 && len(configs) > 0 {
		defaultID = configs[0].ID
	}
	for _, legacyConfig := range configs {
		config := models.LLMConfig{
			ID:            legacyConfig.ID,
			UserID:        userID,
			Name:          legacyConfig.Provider,
			Provider:      legacyConfig.Provider,
			BaseURL:       legacyConfig.BaseURL,
			OpenAIBaseURL: legacyConfig.OpenAIBaseURL,
			APIKey:        legacyConfig.APIKey,
			IsDefault:     legacyConfig.ID == defaultID,
			CreatedAt:     legacyConfig.CreatedAt,
			UpdatedAt:     legacyConfig.UpdatedAt,
		}
		if err := tx.Create(&config).Error; err != nil {
			return fmt.Errorf("create migrated llm profile %d: %w", legacyConfig.ID, err)
		}
	}
	return nil
}

func migrateChatSessions(tx *gorm.DB, userID uint) error {
	source := legacyPrefix + "chat_sessions"
	if !tx.Migrator().HasTable(source) {
		return nil
	}
	var sessions []legacyChatSession
	if err := tx.Table(source).Order("created_at ASC").Find(&sessions).Error; err != nil {
		return fmt.Errorf("load legacy chat sessions: %w", err)
	}
	for _, legacySession := range sessions {
		session := models.ChatSession{
			ID:              legacySession.ID,
			UserID:          userID,
			TranscriptionID: legacySession.TranscriptionID,
			Title:           legacySession.Title,
			Model:           legacySession.Model,
			Provider:        legacySession.Provider,
			SystemContext:   legacySession.SystemContext,
			JobID:           legacySession.JobID,
			MessageCount:    legacySession.MessageCount,
			LastActivityAt:  legacySession.LastActivityAt,
			IsActive:        legacySession.IsActive,
			CreatedAt:       legacySession.CreatedAt,
			UpdatedAt:       legacySession.UpdatedAt,
		}
		if err := tx.Create(&session).Error; err != nil {
			return fmt.Errorf("create migrated chat session %s: %w", legacySession.ID, err)
		}
	}
	return nil
}

func migrateChatMessages(tx *gorm.DB, userID uint) error {
	source := legacyPrefix + "chat_messages"
	if !tx.Migrator().HasTable(source) {
		return nil
	}
	var messages []legacyChatMessage
	if err := tx.Table(source).Order("created_at ASC, id ASC").Find(&messages).Error; err != nil {
		return fmt.Errorf("load legacy chat messages: %w", err)
	}
	for _, legacyMessage := range messages {
		chatSessionID := legacyMessage.ChatSessionID
		if chatSessionID == "" {
			chatSessionID = legacyMessage.SessionID
		}
		message := models.ChatMessage{
			ID:            legacyMessage.ID,
			UserID:        userID,
			ChatSessionID: chatSessionID,
			SessionID:     legacyMessage.SessionID,
			Role:          legacyMessage.Role,
			Content:       legacyMessage.Content,
			TokensUsed:    legacyMessage.TokensUsed,
			CreatedAt:     legacyMessage.CreatedAt,
		}
		if err := tx.Create(&message).Error; err != nil {
			return fmt.Errorf("create migrated chat message %d: %w", legacyMessage.ID, err)
		}
	}
	return nil
}

func migrateAPIKeys(tx *gorm.DB, userID uint) error {
	source := legacyPrefix + "api_keys"
	if !tx.Migrator().HasTable(source) {
		return nil
	}
	var keys []legacyAPIKey
	if err := tx.Table(source).Order("created_at ASC").Find(&keys).Error; err != nil {
		return fmt.Errorf("load legacy api keys: %w", err)
	}
	if userID == 0 && len(keys) > 0 {
		return fmt.Errorf("migrate api keys: missing user for %d rows", len(keys))
	}
	for _, legacyKey := range keys {
		migrated := models.APIKey{
			ID:          legacyKey.ID,
			UserID:      userID,
			Name:        legacyKey.Name,
			KeyPrefix:   apiKeyPrefix(legacyKey.Key),
			KeyHash:     hashToken(legacyKey.Key),
			Description: legacyKey.Description,
			LastUsed:    legacyKey.LastUsed,
			CreatedAt:   legacyKey.CreatedAt,
		}
		if !legacyKey.IsActive {
			revokedAt := legacyKey.UpdatedAt
			migrated.RevokedAt = &revokedAt
		}
		if err := tx.Create(&migrated).Error; err != nil {
			return fmt.Errorf("create migrated api key %d: %w", legacyKey.ID, err)
		}
	}
	return nil
}

func migrateRefreshTokens(tx *gorm.DB, userID uint) error {
	source := legacyPrefix + "refresh_tokens"
	if !tx.Migrator().HasTable(source) {
		return nil
	}
	var tokens []legacyRefreshToken
	if err := tx.Table(source).Order("created_at ASC").Find(&tokens).Error; err != nil {
		return fmt.Errorf("load legacy refresh tokens: %w", err)
	}
	for _, legacyToken := range tokens {
		resolvedUserID := chooseUserID(legacyToken.UserID, userID)
		if resolvedUserID == 0 {
			return fmt.Errorf("migrate refresh token %d: missing user", legacyToken.ID)
		}
		token := models.RefreshToken{
			ID:        legacyToken.ID,
			UserID:    resolvedUserID,
			Hashed:    legacyToken.Hashed,
			ExpiresAt: legacyToken.ExpiresAt,
			CreatedAt: legacyToken.CreatedAt,
		}
		if legacyToken.Revoked {
			revokedAt := legacyToken.UpdatedAt
			token.RevokedAt = &revokedAt
		}
		if err := tx.Create(&token).Error; err != nil {
			return fmt.Errorf("create migrated refresh token %d: %w", legacyToken.ID, err)
		}
	}
	return nil
}

func migrateSummarySettings(tx *gorm.DB, userID uint) error {
	source := "summary_settings"
	if !tx.Migrator().HasTable(source) || userID == 0 {
		return nil
	}
	var setting legacySummarySetting
	if err := tx.Table(source).Order("updated_at DESC, id DESC").First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return fmt.Errorf("load legacy summary settings: %w", err)
	}
	user, err := loadUser(tx, userID)
	if err != nil {
		return err
	}
	if setting.DefaultModel != "" {
		applySummaryDefaultModel(user, setting.DefaultModel)
		if err := tx.Save(user).Error; err != nil {
			return fmt.Errorf("persist migrated summary settings: %w", err)
		}
	}
	return nil
}

func updateLatestExecutions(tx *gorm.DB) error {
	var executions []models.TranscriptionJobExecution
	if err := tx.Order("transcription_id ASC, created_at DESC, execution_number DESC").Find(&executions).Error; err != nil {
		return fmt.Errorf("load migrated executions: %w", err)
	}
	latest := make(map[string]string)
	for _, execution := range executions {
		if latest[execution.TranscriptionJobID] == "" {
			latest[execution.TranscriptionJobID] = execution.ID
		}
	}
	ids := make([]string, 0, len(latest))
	for id := range latest {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, transcriptionID := range ids {
		if err := tx.Model(&models.TranscriptionJob{}).Where("id = ?", transcriptionID).Update("latest_execution_id", latest[transcriptionID]).Error; err != nil {
			return fmt.Errorf("update latest execution for %s: %w", transcriptionID, err)
		}
	}
	return nil
}

func mapLegacyStatus(status string) models.JobStatus {
	switch status {
	case "uploaded":
		return models.StatusUploaded
	case "pending", "queued":
		return models.StatusPending
	case "processing":
		return models.StatusProcessing
	case "completed":
		return models.StatusCompleted
	case "failed":
		return models.StatusFailed
	case "stopped", "canceled", "cancelled":
		return models.StatusStopped
	default:
		return models.JobStatus(status)
	}
}

func apiKeyPrefix(key string) string {
	if len(key) <= 8 {
		return key
	}
	return key[:8]
}

func chooseUserID(candidate uint, fallback uint) uint {
	if candidate != 0 {
		return candidate
	}
	return fallback
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func loadUser(tx *gorm.DB, userID uint) (*models.User, error) {
	var user models.User
	if err := tx.First(&user, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("load user %d: %w", userID, err)
	}
	return &user, nil
}

func applySummaryDefaultModel(user *models.User, model string) {
	user.SummaryDefaultModel = model
}
