package database

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"scriberr/internal/models"

	"gorm.io/gorm"
)

const latestSchemaVersion = 3

var schemaModels = []any{
	&models.User{},
	&models.APIKey{},
	&models.RefreshToken{},
	&models.TranscriptionProfile{},
	&models.TranscriptionJob{},
	&models.TranscriptionJobExecution{},
	&models.SpeakerMapping{},
	&models.SummaryTemplate{},
	&models.Summary{},
	&models.SummaryWidget{},
	&models.SummaryWidgetRun{},
	&models.TranscriptAnnotation{},
	&models.TranscriptAnnotationEntry{},
	&models.ChatSession{},
	&models.ChatContextSource{},
	&models.ChatMessage{},
	&models.ChatGenerationRun{},
	&models.ChatContextSummary{},
	&models.LLMConfig{},
}

type schemaMigration struct {
	Version   int   `gorm:"primaryKey"`
	AppliedAt int64 `gorm:"not null"`
}

func (schemaMigration) TableName() string { return "schema_migrations" }

func createTargetSchema(tx *gorm.DB) error {
	if err := removeObsoleteChatSchema(tx); err != nil {
		return fmt.Errorf("remove obsolete chat schema: %w", err)
	}
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
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_transcription_executions_unique ON transcription_executions(transcription_id, execution_number)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_transcription_profiles_user_default_unique ON transcription_profiles(user_id) WHERE is_default = 1`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_summary_templates_user_default_unique ON summary_templates(user_id) WHERE is_default = 1`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_llm_profiles_user_default_unique ON llm_profiles(user_id) WHERE is_default = 1`,
		`CREATE INDEX IF NOT EXISTS idx_transcriptions_status_created_at ON transcriptions(status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_transcriptions_queue_claim ON transcriptions(status, queued_at)`,
		`CREATE INDEX IF NOT EXISTS idx_transcriptions_claim_expires_at ON transcriptions(claim_expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_parent_updated_at ON chat_sessions(user_id, parent_transcription_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_context_sources_session_enabled_position ON chat_context_sources(chat_session_id, enabled, position)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_context_sources_user_transcription ON chat_context_sources(user_id, transcription_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_session_created_at ON chat_messages(chat_session_id, created_at ASC)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_generation_runs_session_created_at ON chat_generation_runs(chat_session_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_generation_runs_status_created_at ON chat_generation_runs(status, created_at ASC)`,
		`CREATE INDEX IF NOT EXISTS idx_summaries_transcription_created_at ON summaries(transcription_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_summaries_status_created_at ON summaries(status, created_at ASC)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_summary_widgets_user_name_active_unique ON summary_widgets(user_id, name) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_summary_widgets_user_enabled ON summary_widgets(user_id, enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_summary_widget_runs_status_created_at ON summary_widget_runs(status, created_at ASC)`,
		`CREATE INDEX IF NOT EXISTS idx_summary_widget_runs_summary_created_at ON summary_widget_runs(summary_id, created_at ASC)`,
		`CREATE INDEX IF NOT EXISTS idx_transcript_annotations_user_transcription_created_at ON transcript_annotations(user_id, transcription_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_transcript_annotations_user_kind_updated_at ON transcript_annotations(user_id, kind, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_transcript_annotations_transcription_time ON transcript_annotations(transcription_id, anchor_start_ms, anchor_end_ms)`,
		`CREATE INDEX IF NOT EXISTS idx_transcript_annotation_entries_annotation_created_at ON transcript_annotation_entries(annotation_id, created_at ASC)`,
	}
	for _, stmt := range statements {
		if err := tx.Exec(stmt).Error; err != nil {
			return fmt.Errorf("apply schema index: %w", err)
		}
	}
	return nil
}

func removeObsoleteChatSchema(tx *gorm.DB) error {
	if !tx.Migrator().HasTable("chat_sessions") {
		return nil
	}
	if !tx.Migrator().HasColumn("chat_sessions", "transcription_id") {
		return nil
	}
	tables := []string{
		"chat_context_summaries",
		"chat_generation_runs",
		"chat_messages",
		"chat_context_sources",
		"chat_sessions",
	}
	for _, table := range tables {
		if tx.Migrator().HasTable(table) {
			if err := tx.Exec("DROP TABLE IF EXISTS " + table).Error; err != nil {
				return err
			}
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
	Table          string
	Columns        []string
	Unique         bool
	Partial        bool
	WherePredicate string
}

var expectedSQLiteIndexes = map[string]expectedSQLiteIndex{
	"idx_users_deleted_at":                                     {Table: "users", Columns: []string{"deleted_at"}, Unique: false},
	"idx_users_username":                                       {Table: "users", Columns: []string{"username"}, Unique: true},
	"idx_users_email":                                          {Table: "users", Columns: []string{"email"}, Unique: true},
	"idx_refresh_tokens_user_id":                               {Table: "refresh_tokens", Columns: []string{"user_id"}, Unique: false},
	"idx_refresh_tokens_token_hash":                            {Table: "refresh_tokens", Columns: []string{"token_hash"}, Unique: true},
	"idx_refresh_tokens_expires_at":                            {Table: "refresh_tokens", Columns: []string{"expires_at"}, Unique: false},
	"idx_refresh_tokens_revoked_at":                            {Table: "refresh_tokens", Columns: []string{"revoked_at"}, Unique: false},
	"idx_api_keys_user_id":                                     {Table: "api_keys", Columns: []string{"user_id"}, Unique: false},
	"idx_api_keys_key_prefix":                                  {Table: "api_keys", Columns: []string{"key_prefix"}, Unique: false},
	"idx_api_keys_key_hash":                                    {Table: "api_keys", Columns: []string{"key_hash"}, Unique: true},
	"idx_api_keys_expires_at":                                  {Table: "api_keys", Columns: []string{"expires_at"}, Unique: false},
	"idx_api_keys_revoked_at":                                  {Table: "api_keys", Columns: []string{"revoked_at"}, Unique: false},
	"idx_transcription_profiles_user_id":                       {Table: "transcription_profiles", Columns: []string{"user_id"}, Unique: false},
	"idx_transcription_profiles_is_default":                    {Table: "transcription_profiles", Columns: []string{"is_default"}, Unique: false},
	"idx_transcription_profiles_user_default_unique":           {Table: "transcription_profiles", Columns: []string{"user_id"}, Unique: true, Partial: true, WherePredicate: "is_default=1"},
	"idx_transcriptions_user_id":                               {Table: "transcriptions", Columns: []string{"user_id"}, Unique: false},
	"idx_transcriptions_status":                                {Table: "transcriptions", Columns: []string{"status"}, Unique: false},
	"idx_transcriptions_source_file_hash":                      {Table: "transcriptions", Columns: []string{"source_file_hash"}, Unique: false},
	"idx_transcriptions_latest_execution_id":                   {Table: "transcriptions", Columns: []string{"latest_execution_id"}, Unique: false},
	"idx_transcriptions_deleted_at":                            {Table: "transcriptions", Columns: []string{"deleted_at"}, Unique: false},
	"idx_transcriptions_queue_claim":                           {Table: "transcriptions", Columns: []string{"status", "queued_at"}, Unique: false},
	"idx_transcriptions_claim_expires_at":                      {Table: "transcriptions", Columns: []string{"claim_expires_at"}, Unique: false},
	"idx_transcription_executions_transcription_job_id":        {Table: "transcription_executions", Columns: []string{"transcription_id"}, Unique: false},
	"idx_transcription_executions_user_id":                     {Table: "transcription_executions", Columns: []string{"user_id"}, Unique: false},
	"idx_transcription_executions_status":                      {Table: "transcription_executions", Columns: []string{"status"}, Unique: false},
	"idx_transcription_executions_profile_id":                  {Table: "transcription_executions", Columns: []string{"profile_id"}, Unique: false},
	"idx_speaker_mappings_user_id":                             {Table: "speaker_mappings", Columns: []string{"user_id"}, Unique: false},
	"idx_speaker_mappings_transcription_job_id":                {Table: "speaker_mappings", Columns: []string{"transcription_id"}, Unique: false},
	"idx_summary_templates_user_id":                            {Table: "summary_templates", Columns: []string{"user_id"}, Unique: false},
	"idx_summary_templates_is_default":                         {Table: "summary_templates", Columns: []string{"is_default"}, Unique: false},
	"idx_summary_templates_deleted_at":                         {Table: "summary_templates", Columns: []string{"deleted_at"}, Unique: false},
	"idx_summary_templates_user_default_unique":                {Table: "summary_templates", Columns: []string{"user_id"}, Unique: true, Partial: true, WherePredicate: "is_default=1"},
	"idx_summaries_transcription_id":                           {Table: "summaries", Columns: []string{"transcription_id"}, Unique: false},
	"idx_summaries_user_id":                                    {Table: "summaries", Columns: []string{"user_id"}, Unique: false},
	"idx_summaries_template_id":                                {Table: "summaries", Columns: []string{"template_id"}, Unique: false},
	"idx_summaries_status_created_at":                          {Table: "summaries", Columns: []string{"status", "created_at"}, Unique: false},
	"idx_summary_widgets_user_id":                              {Table: "summary_widgets", Columns: []string{"user_id"}, Unique: false},
	"idx_summary_widgets_enabled":                              {Table: "summary_widgets", Columns: []string{"enabled"}, Unique: false},
	"idx_summary_widgets_deleted_at":                           {Table: "summary_widgets", Columns: []string{"deleted_at"}, Unique: false},
	"idx_summary_widgets_user_name_active_unique":              {Table: "summary_widgets", Columns: []string{"user_id", "name"}, Unique: true, Partial: true, WherePredicate: "deleted_at IS NULL"},
	"idx_summary_widgets_user_enabled":                         {Table: "summary_widgets", Columns: []string{"user_id", "enabled"}, Unique: false},
	"idx_summary_widget_runs_summary_id":                       {Table: "summary_widget_runs", Columns: []string{"summary_id"}, Unique: false},
	"idx_summary_widget_runs_transcription_id":                 {Table: "summary_widget_runs", Columns: []string{"transcription_id"}, Unique: false},
	"idx_summary_widget_runs_widget_id":                        {Table: "summary_widget_runs", Columns: []string{"widget_id"}, Unique: false},
	"idx_summary_widget_runs_user_id":                          {Table: "summary_widget_runs", Columns: []string{"user_id"}, Unique: false},
	"idx_summary_widget_runs_status_created_at":                {Table: "summary_widget_runs", Columns: []string{"status", "created_at"}, Unique: false},
	"idx_summary_widget_runs_summary_created_at":               {Table: "summary_widget_runs", Columns: []string{"summary_id", "created_at"}, Unique: false},
	"idx_transcript_annotations_user_id":                       {Table: "transcript_annotations", Columns: []string{"user_id"}, Unique: false},
	"idx_transcript_annotations_transcription_id":              {Table: "transcript_annotations", Columns: []string{"transcription_id"}, Unique: false},
	"idx_transcript_annotations_kind":                          {Table: "transcript_annotations", Columns: []string{"kind"}, Unique: false},
	"idx_transcript_annotations_deleted_at":                    {Table: "transcript_annotations", Columns: []string{"deleted_at"}, Unique: false},
	"idx_transcript_annotations_user_transcription_created_at": {Table: "transcript_annotations", Columns: []string{"user_id", "transcription_id", "created_at"}, Unique: false},
	"idx_transcript_annotations_user_kind_updated_at":          {Table: "transcript_annotations", Columns: []string{"user_id", "kind", "updated_at"}, Unique: false},
	"idx_transcript_annotations_transcription_time":            {Table: "transcript_annotations", Columns: []string{"transcription_id", "anchor_start_ms", "anchor_end_ms"}, Unique: false},
	"idx_transcript_annotation_entries_annotation_id":          {Table: "transcript_annotation_entries", Columns: []string{"annotation_id"}, Unique: false},
	"idx_transcript_annotation_entries_user_id":                {Table: "transcript_annotation_entries", Columns: []string{"user_id"}, Unique: false},
	"idx_transcript_annotation_entries_deleted_at":             {Table: "transcript_annotation_entries", Columns: []string{"deleted_at"}, Unique: false},
	"idx_transcript_annotation_entries_annotation_created_at":  {Table: "transcript_annotation_entries", Columns: []string{"annotation_id", "created_at"}, Unique: false},
	"idx_chat_sessions_user_id":                                {Table: "chat_sessions", Columns: []string{"user_id"}, Unique: false},
	"idx_chat_sessions_parent_transcription_id":                {Table: "chat_sessions", Columns: []string{"parent_transcription_id"}, Unique: false},
	"idx_chat_sessions_deleted_at":                             {Table: "chat_sessions", Columns: []string{"deleted_at"}, Unique: false},
	"idx_chat_sessions_user_parent_updated_at":                 {Table: "chat_sessions", Columns: []string{"user_id", "parent_transcription_id", "updated_at"}, Unique: false},
	"idx_chat_context_sources_user_id":                         {Table: "chat_context_sources", Columns: []string{"user_id"}, Unique: false},
	"idx_chat_context_sources_chat_session_id":                 {Table: "chat_context_sources", Columns: []string{"chat_session_id"}, Unique: false},
	"idx_chat_context_sources_transcription_id":                {Table: "chat_context_sources", Columns: []string{"transcription_id"}, Unique: false},
	"idx_chat_context_sources_session_enabled_position":        {Table: "chat_context_sources", Columns: []string{"chat_session_id", "enabled", "position"}, Unique: false},
	"idx_chat_context_sources_user_transcription":              {Table: "chat_context_sources", Columns: []string{"user_id", "transcription_id"}, Unique: false},
	"idx_chat_messages_user_id":                                {Table: "chat_messages", Columns: []string{"user_id"}, Unique: false},
	"idx_chat_messages_chat_session_id":                        {Table: "chat_messages", Columns: []string{"chat_session_id"}, Unique: false},
	"idx_chat_messages_run_id":                                 {Table: "chat_messages", Columns: []string{"run_id"}, Unique: false},
	"idx_chat_generation_runs_user_id":                         {Table: "chat_generation_runs", Columns: []string{"user_id"}, Unique: false},
	"idx_chat_generation_runs_chat_session_id":                 {Table: "chat_generation_runs", Columns: []string{"chat_session_id"}, Unique: false},
	"idx_chat_generation_runs_assistant_message_id":            {Table: "chat_generation_runs", Columns: []string{"assistant_message_id"}, Unique: false},
	"idx_chat_generation_runs_status":                          {Table: "chat_generation_runs", Columns: []string{"status"}, Unique: false},
	"idx_chat_generation_runs_session_created_at":              {Table: "chat_generation_runs", Columns: []string{"chat_session_id", "created_at"}, Unique: false},
	"idx_chat_generation_runs_status_created_at":               {Table: "chat_generation_runs", Columns: []string{"status", "created_at"}, Unique: false},
	"idx_chat_context_summaries_user_id":                       {Table: "chat_context_summaries", Columns: []string{"user_id"}, Unique: false},
	"idx_chat_context_summaries_chat_session_id":               {Table: "chat_context_summaries", Columns: []string{"chat_session_id"}, Unique: false},
	"idx_chat_context_summaries_source_transcription_id":       {Table: "chat_context_summaries", Columns: []string{"source_transcription_id"}, Unique: false},
	"idx_chat_context_summaries_source_message_through_id":     {Table: "chat_context_summaries", Columns: []string{"source_message_through_id"}, Unique: false},
	"idx_llm_profiles_user_id":                                 {Table: "llm_profiles", Columns: []string{"user_id"}, Unique: false},
	"idx_llm_profiles_is_default":                              {Table: "llm_profiles", Columns: []string{"is_default"}, Unique: false},
	"idx_llm_profiles_user_default_unique":                     {Table: "llm_profiles", Columns: []string{"user_id"}, Unique: true, Partial: true, WherePredicate: "is_default=1"},
}

func duplicateIndexName(errMsg string) string {
	matches := duplicateIndexRegexp.FindStringSubmatch(errMsg)
	if len(matches) != 2 {
		return ""
	}
	return strings.Trim(matches[1], "`\"")
}

func normalizeIndexPredicate(predicate string) string {
	normalized := strings.ToLower(strings.TrimSpace(predicate))
	replacer := strings.NewReplacer(
		"`", "",
		`"`, "",
		"[", "",
		"]", "",
		"(", "",
		")", "",
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
	)
	normalized = replacer.Replace(normalized)
	normalized = strings.ReplaceAll(normalized, "true", "1")
	normalized = strings.ReplaceAll(normalized, "==", "=")
	return normalized
}

var whereTokenRegexp = regexp.MustCompile(`(?i)\bwhere\b`)

func extractNormalizedWherePredicate(createIndexSQL string) string {
	sql := strings.TrimSpace(createIndexSQL)
	whereLoc := whereTokenRegexp.FindStringIndex(sql)
	if whereLoc == nil {
		return ""
	}
	return normalizeIndexPredicate(sql[whereLoc[1]:])
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
		Name    string `gorm:"column:name"`
		Unique  int    `gorm:"column:unique"`
		Partial int    `gorm:"column:partial"`
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
			if (row.Partial == 1) != expected.Partial {
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
	if expected.Partial {
		type sqliteMasterSQLRow struct {
			SQL string `gorm:"column:sql"`
		}
		var sqlRow sqliteMasterSQLRow
		if err := tx.Raw("SELECT sql FROM sqlite_master WHERE type = 'index' AND name = ?", indexName).Scan(&sqlRow).Error; err != nil {
			return false
		}
		if extractNormalizedWherePredicate(sqlRow.SQL) != normalizeIndexPredicate(expected.WherePredicate) {
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
