package database

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"scriberr/internal/models"

	"gorm.io/gorm"
)

const latestSchemaVersion = 2

var schemaModels = []any{
	&models.User{},
	&models.APIKey{},
	&models.RefreshToken{},
	&models.TranscriptionProfile{},
	&models.TranscriptionJob{},
	&models.TranscriptionJobExecution{},
	&models.MultiTrackFile{},
	&models.SpeakerMapping{},
	&models.SummaryTemplate{},
	&models.Summary{},
	&models.Note{},
	&models.ChatSession{},
	&models.ChatMessage{},
	&models.LLMConfig{},
}

type schemaMigration struct {
	Version   int   `gorm:"primaryKey"`
	AppliedAt int64 `gorm:"not null"`
}

func (schemaMigration) TableName() string { return "schema_migrations" }

func createTargetSchema(tx *gorm.DB) error {
	for _, model := range schemaModels {
		if err := tx.AutoMigrate(model); err != nil {
			if !isIgnorableSQLiteDuplicateIndexError(tx, err) {
				return fmt.Errorf("auto migrate target schema: %w", err)
			}
		}
	}
	if err := tx.AutoMigrate(&schemaMigration{}); err != nil {
		if !isIgnorableSQLiteDuplicateIndexError(tx, err) {
			return fmt.Errorf("auto migrate schema state: %w", err)
		}
	}
	if err := ensureSingleDefaultPerUser(tx); err != nil {
		return fmt.Errorf("enforce default-selection invariants: %w", err)
	}

	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_speaker_mappings_unique ON speaker_mappings(transcription_id, original_speaker)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_transcription_tracks_unique ON transcription_tracks(transcription_id, track_index)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_transcription_executions_unique ON transcription_executions(transcription_id, execution_number)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_transcription_profiles_user_default_unique ON transcription_profiles(user_id) WHERE is_default = 1`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_summary_templates_user_default_unique ON summary_templates(user_id) WHERE is_default = 1`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_llm_profiles_user_default_unique ON llm_profiles(user_id) WHERE is_default = 1`,
		`CREATE INDEX IF NOT EXISTS idx_transcriptions_status_created_at ON transcriptions(status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_session_created_at ON chat_messages(chat_session_id, created_at ASC)`,
		`CREATE INDEX IF NOT EXISTS idx_summaries_transcription_created_at ON summaries(transcription_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_transcription_created_at ON notes(transcription_id, created_at DESC)`,
	}
	for _, stmt := range statements {
		if err := tx.Exec(stmt).Error; err != nil {
			return fmt.Errorf("apply schema index: %w", err)
		}
	}
	return nil
}

func isIgnorableSQLiteDuplicateIndexError(tx *gorm.DB, err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	if !strings.Contains(errMsg, "sql logic error") {
		return false
	}
	if !strings.Contains(errMsg, "already exists") {
		return false
	}
	indexName := duplicateIndexName(err.Error())
	if indexName == "" {
		return false
	}
	expected, ok := expectedSQLiteIndexes[indexName]
	if !ok {
		return false
	}
	return existingIndexMatches(tx, indexName, expected)
}

var duplicateIndexRegexp = regexp.MustCompile(`index ([^ ]+) already exists`)

type expectedSQLiteIndex struct {
	Table   string
	Columns []string
	Unique  bool
}

var expectedSQLiteIndexes = map[string]expectedSQLiteIndex{
	"idx_users_deleted_at":                  {Table: "users", Columns: []string{"deleted_at"}, Unique: false},
	"idx_users_username":                    {Table: "users", Columns: []string{"username"}, Unique: true},
	"idx_users_email":                       {Table: "users", Columns: []string{"email"}, Unique: true},
	"idx_refresh_tokens_user_id":            {Table: "refresh_tokens", Columns: []string{"user_id"}, Unique: false},
	"idx_refresh_tokens_token_hash":         {Table: "refresh_tokens", Columns: []string{"token_hash"}, Unique: true},
	"idx_refresh_tokens_expires_at":         {Table: "refresh_tokens", Columns: []string{"expires_at"}, Unique: false},
	"idx_refresh_tokens_revoked_at":         {Table: "refresh_tokens", Columns: []string{"revoked_at"}, Unique: false},
	"idx_api_keys_user_id":                  {Table: "api_keys", Columns: []string{"user_id"}, Unique: false},
	"idx_api_keys_key_prefix":               {Table: "api_keys", Columns: []string{"key_prefix"}, Unique: false},
	"idx_api_keys_key_hash":                 {Table: "api_keys", Columns: []string{"key_hash"}, Unique: true},
	"idx_api_keys_expires_at":               {Table: "api_keys", Columns: []string{"expires_at"}, Unique: false},
	"idx_api_keys_revoked_at":               {Table: "api_keys", Columns: []string{"revoked_at"}, Unique: false},
	"idx_transcription_profiles_user_id":    {Table: "transcription_profiles", Columns: []string{"user_id"}, Unique: false},
	"idx_transcription_profiles_is_default": {Table: "transcription_profiles", Columns: []string{"is_default"}, Unique: false},
	"idx_transcriptions_user_id":            {Table: "transcriptions", Columns: []string{"user_id"}, Unique: false},
	"idx_transcriptions_status":             {Table: "transcriptions", Columns: []string{"status"}, Unique: false},
	"idx_transcriptions_source_file_hash":   {Table: "transcriptions", Columns: []string{"source_file_hash"}, Unique: false},
	"idx_transcriptions_latest_execution_id": {Table: "transcriptions", Columns: []string{"latest_execution_id"}, Unique: false},
	"idx_transcriptions_deleted_at":         {Table: "transcriptions", Columns: []string{"deleted_at"}, Unique: false},
	"idx_transcription_executions_transcription_job_id": {Table: "transcription_executions", Columns: []string{"transcription_id"}, Unique: false},
	"idx_transcription_executions_user_id":  {Table: "transcription_executions", Columns: []string{"user_id"}, Unique: false},
	"idx_transcription_executions_status":   {Table: "transcription_executions", Columns: []string{"status"}, Unique: false},
	"idx_transcription_executions_profile_id": {Table: "transcription_executions", Columns: []string{"profile_id"}, Unique: false},
	"idx_speaker_mappings_user_id":          {Table: "speaker_mappings", Columns: []string{"user_id"}, Unique: false},
	"idx_speaker_mappings_transcription_job_id": {Table: "speaker_mappings", Columns: []string{"transcription_id"}, Unique: false},
	"idx_transcription_tracks_user_id":      {Table: "transcription_tracks", Columns: []string{"user_id"}, Unique: false},
	"idx_transcription_tracks_transcription_job_id": {Table: "transcription_tracks", Columns: []string{"transcription_id"}, Unique: false},
	"idx_summary_templates_user_id":         {Table: "summary_templates", Columns: []string{"user_id"}, Unique: false},
	"idx_summary_templates_is_default":      {Table: "summary_templates", Columns: []string{"is_default"}, Unique: false},
	"idx_summary_templates_deleted_at":      {Table: "summary_templates", Columns: []string{"deleted_at"}, Unique: false},
	"idx_summaries_transcription_id":        {Table: "summaries", Columns: []string{"transcription_id"}, Unique: false},
	"idx_summaries_user_id":                 {Table: "summaries", Columns: []string{"user_id"}, Unique: false},
	"idx_summaries_template_id":             {Table: "summaries", Columns: []string{"template_id"}, Unique: false},
	"idx_notes_user_id":                     {Table: "notes", Columns: []string{"user_id"}, Unique: false},
	"idx_notes_transcription_id":            {Table: "notes", Columns: []string{"transcription_id"}, Unique: false},
	"idx_notes_deleted_at":                  {Table: "notes", Columns: []string{"deleted_at"}, Unique: false},
	"idx_chat_sessions_user_id":             {Table: "chat_sessions", Columns: []string{"user_id"}, Unique: false},
	"idx_chat_sessions_transcription_id":    {Table: "chat_sessions", Columns: []string{"transcription_id"}, Unique: false},
	"idx_chat_messages_user_id":             {Table: "chat_messages", Columns: []string{"user_id"}, Unique: false},
	"idx_chat_messages_chat_session_id":     {Table: "chat_messages", Columns: []string{"chat_session_id"}, Unique: false},
	"idx_llm_profiles_user_id":              {Table: "llm_profiles", Columns: []string{"user_id"}, Unique: false},
	"idx_llm_profiles_is_default":           {Table: "llm_profiles", Columns: []string{"is_default"}, Unique: false},
}

func duplicateIndexName(errMsg string) string {
	matches := duplicateIndexRegexp.FindStringSubmatch(errMsg)
	if len(matches) != 2 {
		return ""
	}
	return strings.Trim(matches[1], "`\"")
}

func existingIndexMatches(tx *gorm.DB, indexName string, expected expectedSQLiteIndex) bool {
	type sqliteMasterRow struct {
		Table string `gorm:"column:tbl_name"`
	}
	var master sqliteMasterRow
	if err := tx.Raw("SELECT tbl_name FROM sqlite_master WHERE type = 'index' AND name = ?", indexName).Scan(&master).Error; err != nil {
		return false
	}
	if master.Table != expected.Table {
		return false
	}
	type indexListRow struct {
		Name   string `gorm:"column:name"`
		Unique int    `gorm:"column:unique"`
	}
	var listRows []indexListRow
	if err := tx.Raw("PRAGMA index_list('" + expected.Table + "')").Scan(&listRows).Error; err != nil {
		return false
	}
	found := false
	for _, row := range listRows {
		if row.Name == indexName {
			if (row.Unique == 1) != expected.Unique {
				return false
			}
			found = true
			break
		}
	}
	if !found {
		return false
	}
	type indexInfoRow struct {
		Name string `gorm:"column:name"`
	}
	var infoRows []indexInfoRow
	if err := tx.Raw("PRAGMA index_info('" + indexName + "')").Scan(&infoRows).Error; err != nil {
		return false
	}
	if len(infoRows) != len(expected.Columns) {
		return false
	}
	for i, row := range infoRows {
		if row.Name != expected.Columns[i] {
			return false
		}
	}
	return true
}

func ensureSingleDefaultPerUser(tx *gorm.DB) error {
	if err := enforceLatestDefaultPerUserForProfiles(tx); err != nil {
		return err
	}
	if err := enforceLatestDefaultPerUserForSummaryTemplates(tx); err != nil {
		return err
	}
	if err := enforceLatestDefaultPerUserForLLMProfiles(tx); err != nil {
		return err
	}
	return nil
}

type defaultProfileRow struct {
	ID        string    `gorm:"column:id"`
	UserID    uint      `gorm:"column:user_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

type defaultStringRow struct {
	ID        string
	UserID    uint
	CreatedAt time.Time
}

type defaultUintRow struct {
	ID        uint
	UserID    uint
	CreatedAt time.Time
}

func enforceLatestDefaultPerUserForProfiles(tx *gorm.DB) error {
	var rows []defaultProfileRow
	if err := tx.Model(&models.TranscriptionProfile{}).
		Where("is_default = ?", true).
		Order("user_id ASC, created_at DESC, id DESC").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("load default transcription profiles: %w", err)
	}

	latestIDByUser := make(map[uint]string)
	for _, row := range rows {
		if _, ok := latestIDByUser[row.UserID]; ok {
			continue
		}
		latestIDByUser[row.UserID] = row.ID
	}

	idsToClear := make([]string, 0, len(rows))
	for _, row := range rows {
		if latestIDByUser[row.UserID] != row.ID {
			idsToClear = append(idsToClear, row.ID)
		}
	}
	if len(idsToClear) == 0 {
		return nil
	}
	if err := tx.Model(&models.TranscriptionProfile{}).
		Where("id IN ?", idsToClear).
		Update("is_default", false).Error; err != nil {
		return fmt.Errorf("clear duplicate profile defaults: %w", err)
	}
	return nil
}

func enforceLatestDefaultPerUserForSummaryTemplates(tx *gorm.DB) error {
	var rows []defaultStringRow
	if err := tx.Model(&models.SummaryTemplate{}).
		Where("is_default = ?", true).
		Order("user_id ASC, created_at DESC, id DESC").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("load default summary templates: %w", err)
	}

	latestIDByUser := make(map[uint]string)
	for _, row := range rows {
		if _, ok := latestIDByUser[row.UserID]; ok {
			continue
		}
		latestIDByUser[row.UserID] = row.ID
	}

	idsToClear := make([]string, 0, len(rows))
	for _, row := range rows {
		if latestIDByUser[row.UserID] != row.ID {
			idsToClear = append(idsToClear, row.ID)
		}
	}
	if len(idsToClear) == 0 {
		return nil
	}
	if err := tx.Model(&models.SummaryTemplate{}).
		Where("id IN ?", idsToClear).
		Update("is_default", false).Error; err != nil {
		return fmt.Errorf("clear duplicate summary template defaults: %w", err)
	}
	return nil
}

func enforceLatestDefaultPerUserForLLMProfiles(tx *gorm.DB) error {
	var rows []defaultUintRow
	if err := tx.Model(&models.LLMConfig{}).
		Where("is_default = ?", true).
		Order("user_id ASC, created_at DESC, id DESC").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("load default llm profiles: %w", err)
	}

	latestIDByUser := make(map[uint]uint)
	for _, row := range rows {
		if _, ok := latestIDByUser[row.UserID]; ok {
			continue
		}
		latestIDByUser[row.UserID] = row.ID
	}

	idsToClear := make([]uint, 0, len(rows))
	for _, row := range rows {
		if latestIDByUser[row.UserID] != row.ID {
			idsToClear = append(idsToClear, row.ID)
		}
	}
	if len(idsToClear) == 0 {
		return nil
	}
	if err := tx.Model(&models.LLMConfig{}).
		Where("id IN ?", idsToClear).
		Update("is_default", false).Error; err != nil {
		return fmt.Errorf("clear duplicate llm defaults: %w", err)
	}
	return nil
}
