package models

import (
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const primaryUserID uint = 1

// JobStatus represents the status of a transcription job.
type JobStatus string

const (
	StatusUploaded   JobStatus = "uploaded"
	StatusPending    JobStatus = "queued"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusStopped    JobStatus = "stopped"
	StatusCanceled   JobStatus = "canceled" // legacy persisted value
)

const (
	ASRStepTranscription         = "transcription"
	ASRStepDiarization           = "diarization"
	ASRStepSpeakerIdentification = "speaker_identification"
)

// ASRStep describes one provider operation in an ASR pipeline.
type ASRStep struct {
	Kind        string         `json:"kind"`
	Provider    string         `json:"provider,omitempty"`
	Model       string         `json:"model,omitempty"`
	ModelFamily string         `json:"model_family,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
}

// ASRParams contains provider-neutral parameters for ASR execution.
type ASRParams struct {
	Pipeline                []ASRStep `json:"pipeline,omitempty" gorm:"-"`
	Provider                string    `json:"provider,omitempty"`
	ModelFamily             string    `json:"model_family,omitempty"`
	Model                   string    `json:"model,omitempty"`
	Task                    string    `json:"task,omitempty"`
	Language                *string   `json:"language,omitempty"`
	Threads                 int       `json:"threads,omitempty"`
	TailPaddings            *int      `json:"tail_paddings,omitempty"`
	EnableTokenTimestamps   *bool     `json:"enable_token_timestamps,omitempty"`
	EnableSegmentTimestamps *bool     `json:"enable_segment_timestamps,omitempty"`
	DecodingMethod          string    `json:"decoding_method,omitempty"`
	ChunkingStrategy        string    `json:"chunking_strategy,omitempty"`
	ChunkSize               int       `json:"chunk_size,omitempty"`
	NumSpeakers             int       `json:"num_speakers,omitempty"`
	DiarizationThreshold    float64   `json:"diarization_threshold,omitempty"`
	MinDurationOn           float64   `json:"min_duration_on,omitempty"`
	MinDurationOff          float64   `json:"min_duration_off,omitempty"`
}

type transcriptionMetadata struct {
	Diarization bool      `json:"diarization,omitempty"`
	Summary     *string   `json:"summary,omitempty"`
	Parameters  ASRParams `json:"parameters,omitempty"`
}

// TranscriptionJob represents the durable transcription record.
type TranscriptionJob struct {
	ID                            string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID                        uint           `json:"user_id" gorm:"not null;index"`
	Title                         *string        `json:"title,omitempty" gorm:"type:text"`
	Status                        JobStatus      `json:"status" gorm:"column:status;type:varchar(20);not null;default:'uploaded';index"`
	AudioPath                     string         `json:"audio_path" gorm:"column:source_file_path;type:text;not null"`
	SourceFileName                string         `json:"source_file_name,omitempty" gorm:"type:text"`
	SourceFileHash                *string        `json:"source_file_hash,omitempty" gorm:"type:varchar(128);index"`
	SourceDurationMs              *int64         `json:"source_duration_ms,omitempty" gorm:"type:integer"`
	Language                      *string        `json:"language,omitempty" gorm:"type:varchar(32)"`
	Transcript                    *string        `json:"transcript,omitempty" gorm:"column:transcript_text;type:text"`
	OutputJSONPath                *string        `json:"output_json_path,omitempty" gorm:"column:output_json_path;type:text"`
	OutputSRTPath                 *string        `json:"output_srt_path,omitempty" gorm:"column:output_srt_path;type:text"`
	OutputVTTPath                 *string        `json:"output_vtt_path,omitempty" gorm:"column:output_vtt_path;type:text"`
	LatestExecutionID             *string        `json:"latest_execution_id,omitempty" gorm:"type:varchar(36);index"`
	ErrorMessage                  *string        `json:"error_message,omitempty" gorm:"column:last_error;type:text"`
	MetadataJSON                  string         `json:"-" gorm:"column:metadata_json;type:json"`
	QueuedAt                      *time.Time     `json:"queued_at,omitempty"`
	Priority                      int            `json:"priority" gorm:"not null;default:0"`
	StartedAt                     *time.Time     `json:"started_at,omitempty"`
	FailedAt                      *time.Time     `json:"failed_at,omitempty"`
	Progress                      float64        `json:"progress" gorm:"not null;default:0"`
	ProgressStage                 string         `json:"progress_stage,omitempty" gorm:"type:varchar(50)"`
	ClaimedBy                     *string        `json:"claimed_by,omitempty" gorm:"type:varchar(128)"`
	ClaimExpiresAt                *time.Time     `json:"claim_expires_at,omitempty"`
	EngineID                      *string        `json:"engine_id,omitempty" gorm:"type:varchar(50)"`
	CompletedAt                   *time.Time     `json:"completed_at,omitempty"`
	LLMTitleGenerated             bool           `json:"llm_title_generated" gorm:"not null;default:false"`
	LLMTitleGeneratedAt           *time.Time     `json:"llm_title_generated_at,omitempty"`
	LLMDescription                *string        `json:"llm_description,omitempty" gorm:"type:text"`
	LLMDescriptionAt              *time.Time     `json:"llm_description_generated_at,omitempty" gorm:"column:llm_description_generated_at"`
	LLMDescriptionSourceSummaryID *string        `json:"llm_description_source_summary_id,omitempty" gorm:"type:varchar(36);index"`
	CreatedAt                     time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt                     time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt                     gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`

	Diarization bool      `json:"diarization" gorm:"-"`
	Summary     *string   `json:"summary,omitempty" gorm:"-"`
	Parameters  ASRParams `json:"parameters" gorm:"-"`
}

func (TranscriptionJob) TableName() string { return "transcriptions" }

func (tj *TranscriptionJob) BeforeCreate(tx *gorm.DB) error {
	if tj.ID == "" {
		tj.ID = uuid.New().String()
	}
	if err := tj.applyDefaults(); err != nil {
		return err
	}
	return tj.syncColumnsFromCompat()
}

func (tj *TranscriptionJob) BeforeSave(tx *gorm.DB) error {
	if err := tj.applyDefaults(); err != nil {
		return err
	}
	return tj.syncColumnsFromCompat()
}

func (tj *TranscriptionJob) AfterFind(tx *gorm.DB) error {
	return tj.syncCompatFromColumns()
}

func (tj *TranscriptionJob) applyDefaults() error {
	if err := requireUserIDForIdentifiedSave("transcription", tj.UserID, tj.ID != ""); err != nil {
		return err
	}
	if tj.SourceFileName == "" && tj.AudioPath != "" {
		tj.SourceFileName = filepath.Base(tj.AudioPath)
	}
	return nil
}

func (tj *TranscriptionJob) syncColumnsFromCompat() error {
	if tj.Parameters.Language != nil {
		tj.Language = tj.Parameters.Language
	}
	if tj.Status == StatusCompleted && tj.CompletedAt == nil {
		now := time.Now()
		tj.CompletedAt = &now
	}
	if tj.Status != StatusCompleted {
		tj.CompletedAt = nil
	}

	metadata := transcriptionMetadata{
		Diarization: tj.Diarization || hasASRStep(tj.Parameters.Pipeline, ASRStepDiarization),
		Summary:     tj.Summary,
		Parameters:  tj.Parameters,
	}
	metadataJSON, err := marshalJSONColumn("transcriptions.metadata_json", metadata)
	if err != nil {
		return err
	}
	tj.MetadataJSON = metadataJSON
	return nil
}

func (tj *TranscriptionJob) SyncColumnsForMigration() error {
	if tj.UserID == 0 {
		tj.UserID = primaryUserID
	}
	if tj.SourceFileName == "" && tj.AudioPath != "" {
		tj.SourceFileName = filepath.Base(tj.AudioPath)
	}
	if tj.Parameters.Language != nil {
		tj.Language = tj.Parameters.Language
	}
	metadata := transcriptionMetadata{
		Diarization: tj.Diarization || hasASRStep(tj.Parameters.Pipeline, ASRStepDiarization),
		Summary:     tj.Summary,
		Parameters:  tj.Parameters,
	}
	metadataJSON, err := marshalJSONColumn("transcriptions.metadata_json", metadata)
	if err != nil {
		return err
	}
	tj.MetadataJSON = metadataJSON
	return nil
}

func (tj *TranscriptionJob) syncCompatFromColumns() error {
	tj.SourceFileName = coalesceString(tj.SourceFileName, filepath.Base(tj.AudioPath))
	if tj.MetadataJSON == "" {
		if tj.Language != nil {
			tj.Parameters.Language = tj.Language
		}
		return nil
	}
	var metadata transcriptionMetadata
	if err := unmarshalJSONColumn("transcriptions.metadata_json", tj.MetadataJSON, &metadata); err != nil {
		return err
	}
	tj.Diarization = metadata.Diarization
	tj.Summary = metadata.Summary
	tj.Parameters = metadata.Parameters
	if tj.Language != nil && tj.Parameters.Language == nil {
		tj.Parameters.Language = tj.Language
	}
	return nil
}

func coalesceString(current, fallback string) string {
	if current != "" {
		return current
	}
	return fallback
}

type executionPayload struct {
	Parameters         ASRParams `json:"parameters,omitempty"`
	ProcessingDuration *int64    `json:"processing_duration,omitempty"`
}

// TranscriptionJobExecution represents execution metadata for a transcription.
type TranscriptionJobExecution struct {
	ID                 string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TranscriptionJobID string     `json:"transcription_job_id" gorm:"column:transcription_id;type:varchar(36);not null;index"`
	UserID             uint       `json:"user_id" gorm:"not null;index"`
	ExecutionNumber    int        `json:"execution_number" gorm:"not null;default:1"`
	TriggerType        string     `json:"trigger_type" gorm:"type:varchar(20);not null;default:'manual'"`
	Status             JobStatus  `json:"status" gorm:"type:varchar(20);not null;index"`
	ProfileID          *string    `json:"profile_id,omitempty" gorm:"type:varchar(36);index"`
	ModelName          string     `json:"model_name,omitempty" gorm:"type:varchar(100)"`
	ModelFamily        string     `json:"model_family,omitempty" gorm:"type:varchar(50)"`
	Provider           string     `json:"provider,omitempty" gorm:"type:varchar(50)"`
	Device             string     `json:"device,omitempty" gorm:"type:varchar(50)"`
	ComputeType        string     `json:"compute_type,omitempty" gorm:"type:varchar(50)"`
	RequestJSON        string     `json:"-" gorm:"column:request_json;type:json"`
	ConfigJSON         string     `json:"-" gorm:"column:config_json;type:json"`
	StartedAt          time.Time  `json:"started_at" gorm:"not null"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	FailedAt           *time.Time `json:"failed_at,omitempty"`
	ErrorMessage       *string    `json:"error_message,omitempty" gorm:"type:text"`
	LogsPath           *string    `json:"logs_path,omitempty" gorm:"type:text"`
	OutputJSONPath     *string    `json:"output_json_path,omitempty" gorm:"type:text"`
	CreatedAt          time.Time  `json:"created_at" gorm:"autoCreateTime"`

	ProcessingDuration *int64    `json:"processing_duration,omitempty" gorm:"-"`
	ActualParameters   ASRParams `json:"actual_parameters" gorm:"-"`

	TranscriptionJob TranscriptionJob `json:"transcription_job,omitempty" gorm:"foreignKey:TranscriptionJobID;references:ID;constraint:OnDelete:CASCADE"`
}

func (TranscriptionJobExecution) TableName() string { return "transcription_executions" }

func (tje *TranscriptionJobExecution) BeforeCreate(tx *gorm.DB) error {
	if tje.ID == "" {
		tje.ID = uuid.New().String()
	}
	return tje.syncColumnsFromCompat()
}

func (tje *TranscriptionJobExecution) BeforeSave(tx *gorm.DB) error {
	return tje.syncColumnsFromCompat()
}

func (tje *TranscriptionJobExecution) AfterFind(tx *gorm.DB) error {
	return tje.syncCompatFromColumns()
}

func (tje *TranscriptionJobExecution) syncColumnsFromCompat() error {
	if err := requireUserIDForIdentifiedSave("transcription execution", tje.UserID, tje.ID != ""); err != nil {
		return err
	}
	if tje.ModelName == "" {
		tje.ModelName = tje.ActualParameters.Model
	}
	if tje.ModelFamily == "" {
		tje.ModelFamily = tje.ActualParameters.ModelFamily
	}
	payload := executionPayload{Parameters: tje.ActualParameters, ProcessingDuration: tje.ProcessingDuration}
	requestJSON, err := marshalJSONColumn("transcription_executions.request_json", payload)
	if err != nil {
		return err
	}
	tje.RequestJSON = requestJSON
	if tje.ConfigJSON == "" {
		tje.ConfigJSON = tje.RequestJSON
	}
	if tje.Status == StatusCompleted {
		tje.FailedAt = nil
	}
	if tje.Status == StatusFailed && tje.FailedAt == nil {
		now := time.Now()
		tje.FailedAt = &now
	}
	return nil
}

func (tje *TranscriptionJobExecution) SyncColumnsForMigration() error {
	if tje.UserID == 0 {
		tje.UserID = primaryUserID
	}
	if tje.ModelName == "" {
		tje.ModelName = tje.ActualParameters.Model
	}
	if tje.ModelFamily == "" {
		tje.ModelFamily = tje.ActualParameters.ModelFamily
	}
	payload := executionPayload{
		Parameters:         tje.ActualParameters,
		ProcessingDuration: tje.ProcessingDuration,
	}
	requestJSON, err := marshalJSONColumn("transcription_executions.request_json", payload)
	if err != nil {
		return err
	}
	tje.RequestJSON = requestJSON
	if tje.ConfigJSON == "" {
		tje.ConfigJSON = tje.RequestJSON
	}
	return nil
}

func (tje *TranscriptionJobExecution) syncCompatFromColumns() error {
	if tje.RequestJSON == "" {
		return nil
	}
	var payload executionPayload
	if err := unmarshalJSONColumn("transcription_executions.request_json", tje.RequestJSON, &payload); err != nil {
		return err
	}
	tje.ActualParameters = payload.Parameters
	tje.ProcessingDuration = payload.ProcessingDuration
	return nil
}

// CalculateProcessingDuration calculates and sets the processing duration.
func (tje *TranscriptionJobExecution) CalculateProcessingDuration() {
	if tje.CompletedAt != nil {
		duration := tje.CompletedAt.Sub(tje.StartedAt).Milliseconds()
		tje.ProcessingDuration = &duration
	}
}

// SpeakerMapping represents transcript-local speaker naming.
type SpeakerMapping struct {
	ID                 uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID             uint      `json:"user_id" gorm:"not null;index"`
	TranscriptionJobID string    `json:"transcription_job_id" gorm:"column:transcription_id;type:varchar(36);not null;index"`
	OriginalSpeaker    string    `json:"original_speaker" gorm:"type:varchar(100);not null"`
	CustomName         string    `json:"custom_name" gorm:"column:display_name;type:varchar(255);not null"`
	CreatedAt          time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt          time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	TranscriptionJob TranscriptionJob `json:"transcription_job,omitempty" gorm:"foreignKey:TranscriptionJobID;references:ID;constraint:OnDelete:CASCADE"`
}

func (SpeakerMapping) TableName() string { return "speaker_mappings" }

func (sm *SpeakerMapping) BeforeCreate(tx *gorm.DB) error {
	return requireUserID("speaker mapping", sm.UserID)
}

// TranscriptionProfile represents a saved transcription profile.
type TranscriptionProfile struct {
	ID                 string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID             uint      `json:"user_id" gorm:"not null;index"`
	Name               string    `json:"name" gorm:"type:varchar(255);not null"`
	Description        *string   `json:"description,omitempty" gorm:"type:text"`
	ModelName          string    `json:"model_name,omitempty" gorm:"type:varchar(100)"`
	ModelFamily        string    `json:"model_family,omitempty" gorm:"type:varchar(50)"`
	Provider           string    `json:"provider,omitempty" gorm:"type:varchar(50)"`
	Language           *string   `json:"language,omitempty" gorm:"type:varchar(32)"`
	DiarizationEnabled bool      `json:"diarization_enabled" gorm:"not null;default:false"`
	Device             string    `json:"device,omitempty" gorm:"type:varchar(50)"`
	ComputeType        string    `json:"compute_type,omitempty" gorm:"type:varchar(50)"`
	ConfigJSON         string    `json:"-" gorm:"column:config_json;type:json"`
	IsDefault          bool      `json:"is_default" gorm:"not null;default:false;index"`
	CreatedAt          time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt          time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Parameters ASRParams `json:"parameters" gorm:"-"`
}

func (TranscriptionProfile) TableName() string { return "transcription_profiles" }

func (tp *TranscriptionProfile) BeforeCreate(tx *gorm.DB) error {
	if tp.ID == "" {
		tp.ID = uuid.New().String()
	}
	return tp.BeforeSave(tx)
}

func (tp *TranscriptionProfile) BeforeSave(tx *gorm.DB) error {
	if err := requireUserID("transcription profile", tp.UserID); err != nil {
		return err
	}
	if step, ok := firstASRStep(tp.Parameters.Pipeline, ASRStepTranscription); ok {
		tp.Provider = step.Provider
		tp.ModelName = step.Model
		tp.ModelFamily = step.ModelFamily
	} else {
		tp.Provider = ""
		tp.ModelName = ""
		tp.ModelFamily = ""
	}
	tp.Language = nil
	tp.DiarizationEnabled = hasASRStep(tp.Parameters.Pipeline, ASRStepDiarization)
	configJSON, err := marshalJSONColumn("transcription_profiles.config_json", tp.Parameters)
	if err != nil {
		return err
	}
	tp.ConfigJSON = configJSON
	if tp.IsDefault {
		if err := clearOtherDefaultsForUser(tx, &TranscriptionProfile{}, tp.UserID, tp.ID); err != nil {
			return err
		}
	}
	return nil
}

func (tp *TranscriptionProfile) AfterFind(tx *gorm.DB) error {
	if tp.ConfigJSON != "" {
		if err := unmarshalJSONColumn("transcription_profiles.config_json", tp.ConfigJSON, &tp.Parameters); err != nil {
			return err
		}
	}
	return nil
}

func hasASRStep(steps []ASRStep, kind string) bool {
	for _, step := range steps {
		if step.Kind == kind {
			return true
		}
	}
	return false
}

func firstASRStep(steps []ASRStep, kind string) (ASRStep, bool) {
	for _, step := range steps {
		if step.Kind == kind {
			return step, true
		}
	}
	return ASRStep{}, false
}

// LLMConfig represents a saved LLM profile.
type LLMConfig struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"not null;index"`
	Name       string    `json:"name" gorm:"type:varchar(255);not null;default:'default'"`
	Provider   string    `json:"provider" gorm:"not null;type:varchar(50)"`
	ModelName  string    `json:"model_name,omitempty" gorm:"type:varchar(100)"`
	BaseURL    *string   `json:"base_url,omitempty" gorm:"type:text"`
	ConfigJSON string    `json:"-" gorm:"column:config_json;type:json"`
	IsDefault  bool      `json:"is_default" gorm:"not null;default:false;index"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	OpenAIBaseURL *string `json:"openai_base_url,omitempty" gorm:"-"`
	APIKey        *string `json:"api_key,omitempty" gorm:"-"`
	LargeModel    *string `json:"large_model,omitempty" gorm:"-"`
	SmallModel    *string `json:"small_model,omitempty" gorm:"-"`
	IsActive      bool    `json:"is_active" gorm:"-"`
}

func (LLMConfig) TableName() string { return "llm_profiles" }

func (lc *LLMConfig) BeforeCreate(tx *gorm.DB) error {
	return requireUserID("llm profile", lc.UserID)
}

func (lc *LLMConfig) BeforeSave(tx *gorm.DB) error {
	if err := requireUserIDForIdentifiedSave("llm profile", lc.UserID, lc.ID != 0); err != nil {
		return err
	}
	configJSON, err := marshalJSONColumn("llm_profiles.config_json", map[string]any{
		"openai_base_url": lc.OpenAIBaseURL,
		"api_key":         lc.APIKey,
		"large_model":     lc.LargeModel,
		"small_model":     lc.SmallModel,
	})
	if err != nil {
		return err
	}
	lc.ConfigJSON = configJSON
	lc.IsActive = lc.IsDefault
	if lc.IsDefault {
		if err := clearOtherDefaultsForUser(tx, &LLMConfig{}, lc.UserID, lc.ID); err != nil {
			return err
		}
	}
	return nil
}

func (lc *LLMConfig) AfterFind(tx *gorm.DB) error {
	lc.IsActive = lc.IsDefault
	if lc.ConfigJSON == "" {
		return nil
	}
	var cfg struct {
		OpenAIBaseURL *string `json:"openai_base_url,omitempty"`
		APIKey        *string `json:"api_key,omitempty"`
		LargeModel    *string `json:"large_model,omitempty"`
		SmallModel    *string `json:"small_model,omitempty"`
	}
	if err := unmarshalJSONColumn("llm_profiles.config_json", lc.ConfigJSON, &cfg); err != nil {
		return err
	}
	lc.OpenAIBaseURL = cfg.OpenAIBaseURL
	lc.APIKey = cfg.APIKey
	lc.LargeModel = cfg.LargeModel
	lc.SmallModel = cfg.SmallModel
	return nil
}
