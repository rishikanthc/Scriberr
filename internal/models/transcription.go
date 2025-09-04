package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TranscriptionJob represents a transcription job record
type TranscriptionJob struct {
	ID           string          `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Title        *string         `json:"title,omitempty" gorm:"type:text"`
	Status       JobStatus       `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	AudioPath    string          `json:"audio_path" gorm:"type:text;not null"`
	Transcript   *string         `json:"transcript,omitempty" gorm:"type:text"`
	Diarization  bool            `json:"diarization" gorm:"type:boolean;default:false"`
	Summary      *string         `json:"summary,omitempty" gorm:"type:text"`
	ErrorMessage *string         `json:"error_message,omitempty" gorm:"type:text"`
	CreatedAt    time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	
	// WhisperX parameters
	Parameters WhisperXParams `json:"parameters" gorm:"embedded"`
}

// JobStatus represents the status of a transcription job
type JobStatus string

const (
	StatusUploaded    JobStatus = "uploaded"
	StatusPending     JobStatus = "pending"
	StatusProcessing  JobStatus = "processing"
	StatusCompleted   JobStatus = "completed"
	StatusFailed      JobStatus = "failed"
)

// WhisperXParams contains parameters for WhisperX transcription
type WhisperXParams struct {
	// Model parameters
	Model                string  `json:"model" gorm:"type:varchar(50);default:'small'"`
	ModelCacheOnly       bool    `json:"model_cache_only" gorm:"type:boolean;default:false"`
	ModelDir             *string `json:"model_dir,omitempty" gorm:"type:text"`
	
	// Device and computation
	Device               string  `json:"device" gorm:"type:varchar(20);default:'cpu'"`
	DeviceIndex          int     `json:"device_index" gorm:"type:int;default:0"`
	BatchSize            int     `json:"batch_size" gorm:"type:int;default:8"`
	ComputeType          string  `json:"compute_type" gorm:"type:varchar(20);default:'float32'"`
	Threads              int     `json:"threads" gorm:"type:int;default:0"`
	
	// Output settings
	OutputFormat         string  `json:"output_format" gorm:"type:varchar(20);default:'all'"`
	Verbose              bool    `json:"verbose" gorm:"type:boolean;default:true"`
	
	// Task and language
	Task                 string  `json:"task" gorm:"type:varchar(20);default:'transcribe'"`
	Language             *string `json:"language,omitempty" gorm:"type:varchar(10)"`
	
	// Alignment settings
	AlignModel           *string `json:"align_model,omitempty" gorm:"type:varchar(100)"`
	InterpolateMethod    string  `json:"interpolate_method" gorm:"type:varchar(20);default:'nearest'"`
	NoAlign              bool    `json:"no_align" gorm:"type:boolean;default:false"`
	ReturnCharAlignments bool    `json:"return_char_alignments" gorm:"type:boolean;default:false"`
	
	// VAD (Voice Activity Detection) settings
	VadMethod            string  `json:"vad_method" gorm:"type:varchar(20);default:'pyannote'"`
	VadOnset             float64 `json:"vad_onset" gorm:"type:real;default:0.5"`
	VadOffset            float64 `json:"vad_offset" gorm:"type:real;default:0.363"`
	ChunkSize            int     `json:"chunk_size" gorm:"type:int;default:30"`
	
	// Diarization settings
	Diarize              bool    `json:"diarize" gorm:"type:boolean;default:false"`
	MinSpeakers          *int    `json:"min_speakers,omitempty" gorm:"type:int"`
	MaxSpeakers          *int    `json:"max_speakers,omitempty" gorm:"type:int"`
	DiarizeModel         string  `json:"diarize_model" gorm:"type:varchar(100);default:'pyannote/speaker-diarization-3.1'"`
	SpeakerEmbeddings    bool    `json:"speaker_embeddings" gorm:"type:boolean;default:false"`
	
	// Transcription quality settings
	Temperature                           float64  `json:"temperature" gorm:"type:real;default:0"`
	BestOf                               int      `json:"best_of" gorm:"type:int;default:5"`
	BeamSize                             int      `json:"beam_size" gorm:"type:int;default:5"`
	Patience                             float64  `json:"patience" gorm:"type:real;default:1.0"`
	LengthPenalty                        float64  `json:"length_penalty" gorm:"type:real;default:1.0"`
	SuppressTokens                       *string  `json:"suppress_tokens,omitempty" gorm:"type:text"`
	SuppressNumerals                     bool     `json:"suppress_numerals" gorm:"type:boolean;default:false"`
	InitialPrompt                        *string  `json:"initial_prompt,omitempty" gorm:"type:text"`
	ConditionOnPreviousText              bool     `json:"condition_on_previous_text" gorm:"type:boolean;default:false"`
	Fp16                                 bool     `json:"fp16" gorm:"type:boolean;default:true"`
	TemperatureIncrementOnFallback       float64  `json:"temperature_increment_on_fallback" gorm:"type:real;default:0.2"`
	CompressionRatioThreshold            float64  `json:"compression_ratio_threshold" gorm:"type:real;default:2.4"`
	LogprobThreshold                     float64  `json:"logprob_threshold" gorm:"type:real;default:-1.0"`
	NoSpeechThreshold                    float64  `json:"no_speech_threshold" gorm:"type:real;default:0.6"`
	
	// Output formatting
	MaxLineWidth                         *int     `json:"max_line_width,omitempty" gorm:"type:int"`
	MaxLineCount                         *int     `json:"max_line_count,omitempty" gorm:"type:int"`
	HighlightWords                       bool     `json:"highlight_words" gorm:"type:boolean;default:false"`
	SegmentResolution                    string   `json:"segment_resolution" gorm:"type:varchar(20);default:'sentence'"`
	
	// Token and progress
	HfToken                              *string  `json:"hf_token,omitempty" gorm:"type:text"`
	PrintProgress                        bool     `json:"print_progress" gorm:"type:boolean;default:false"`
}

// BeforeCreate sets the ID if not already set
func (tj *TranscriptionJob) BeforeCreate(tx *gorm.DB) error {
	if tj.ID == "" {
		tj.ID = uuid.New().String()
	}
	return nil
}

// User represents a user for authentication
type User struct {
	ID                     uint      `json:"id" gorm:"primaryKey"`
	Username               string    `json:"username" gorm:"uniqueIndex;not null;type:varchar(50)"`
	Password               string    `json:"-" gorm:"not null;type:varchar(255)"`
	DefaultProfileID       *string   `json:"default_profile_id,omitempty" gorm:"type:varchar(36)"`
	AutoTranscriptionEnabled bool      `json:"auto_transcription_enabled" gorm:"not null;default:false"`
	CreatedAt              time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt              time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// APIKey represents an API key for external authentication
type APIKey struct {
    ID          uint      `json:"id" gorm:"primaryKey"`
    Key         string    `json:"key" gorm:"uniqueIndex;not null;type:varchar(255)"`
    Name        string    `json:"name" gorm:"not null;type:varchar(100)"`
    Description *string   `json:"description,omitempty" gorm:"type:text"`
    // IsActive should persist explicit false values; avoid default tag to prevent
    // GORM from overriding false with DB defaults during inserts.
    IsActive    bool      `json:"is_active" gorm:"type:boolean;not null"`
    CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// BeforeCreate sets the API key if not already set
func (ak *APIKey) BeforeCreate(tx *gorm.DB) error {
	if ak.Key == "" {
		ak.Key = uuid.New().String()
	}
	return nil
}

// TranscriptionProfile represents a saved transcription configuration profile
type TranscriptionProfile struct {
	ID          string          `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string          `json:"name" gorm:"type:varchar(255);not null"`
	Description *string         `json:"description,omitempty" gorm:"type:text"`
	IsDefault   bool            `json:"is_default" gorm:"type:boolean;default:false"`
	Parameters  WhisperXParams  `json:"parameters" gorm:"embedded"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// BeforeCreate sets the ID if not already set
func (tp *TranscriptionProfile) BeforeCreate(tx *gorm.DB) error {
	if tp.ID == "" {
		tp.ID = uuid.New().String()
	}
	return nil
}

// BeforeSave ensures only one profile can be default
func (tp *TranscriptionProfile) BeforeSave(tx *gorm.DB) error {
	if tp.IsDefault {
		// Set all other profiles to not default
		if err := tx.Model(&TranscriptionProfile{}).Where("id != ?", tp.ID).Update("is_default", false).Error; err != nil {
			return err
		}
	}
	return nil
}

// LLMConfig represents LLM configuration settings
type LLMConfig struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Provider  string    `json:"provider" gorm:"not null;type:varchar(50)"` // "ollama" or "openai"
	BaseURL   *string   `json:"base_url,omitempty" gorm:"type:text"`       // For Ollama
	APIKey    *string   `json:"api_key,omitempty" gorm:"type:text"`        // For OpenAI (encrypted)
	IsActive  bool      `json:"is_active" gorm:"type:boolean;default:false"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// BeforeSave ensures only one LLM config can be active
func (lc *LLMConfig) BeforeSave(tx *gorm.DB) error {
	if lc.IsActive {
		// Set all other configs to not active
		if err := tx.Model(&LLMConfig{}).Where("id != ?", lc.ID).Update("is_active", false).Error; err != nil {
			return err
		}
	}
	return nil
}

// ChatSession represents a chat session with a transcript
type ChatSession struct {
	ID                string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	JobID             string         `json:"job_id" gorm:"type:varchar(36);not null"`
	TranscriptionID   string         `json:"transcription_id" gorm:"type:varchar(36);not null;index"`
	Title             string         `json:"title" gorm:"type:varchar(255);not null"`
	Model             string         `json:"model" gorm:"type:varchar(100);not null"`
	Provider          string         `json:"provider" gorm:"type:varchar(50);not null;default:'openai'"`
	SystemContext     *string        `json:"system_context,omitempty" gorm:"type:text"`
	MessageCount      int            `json:"message_count" gorm:"type:integer;default:0"`
	LastActivityAt    *time.Time     `json:"last_activity_at,omitempty" gorm:"type:datetime"`
	IsActive          bool           `json:"is_active" gorm:"type:boolean;default:true"`
	CreatedAt         time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	
	// Relationships
	Transcription     TranscriptionJob `json:"transcription,omitempty" gorm:"foreignKey:TranscriptionID"`
	Job               TranscriptionJob `json:"job,omitempty" gorm:"foreignKey:JobID"`
	Messages          []ChatMessage    `json:"messages,omitempty" gorm:"foreignKey:ChatSessionID"`
}

// BeforeCreate sets the ID if not already set
func (cs *ChatSession) BeforeCreate(tx *gorm.DB) error {
	if cs.ID == "" {
		cs.ID = uuid.New().String()
	}
	if cs.Title == "" {
		cs.Title = "New Chat Session"
	}
	return nil
}

// ChatMessage represents a message in a chat session
type ChatMessage struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionID     string    `json:"session_id" gorm:"type:varchar(36);not null;index"`
	ChatSessionID string    `json:"chat_session_id" gorm:"type:varchar(36);not null;index"`
	Role          string    `json:"role" gorm:"type:varchar(20);not null"` // "user" or "assistant"
	Content       string    `json:"content" gorm:"type:text;not null"`
	TokensUsed    *int      `json:"tokens_used,omitempty" gorm:"type:integer"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	
	// Relationships
	ChatSession   ChatSession `json:"chat_session,omitempty" gorm:"foreignKey:ChatSessionID"`
}

// BeforeCreate sets both session IDs to the same value for compatibility
func (cm *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if cm.SessionID == "" {
		cm.SessionID = cm.ChatSessionID
	}
	if cm.ChatSessionID == "" {
		cm.ChatSessionID = cm.SessionID
	}
	return nil
}

// TranscriptionJobExecution represents execution metadata for completed transcription jobs
type TranscriptionJobExecution struct {
	ID                 string          `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TranscriptionJobID string          `json:"transcription_job_id" gorm:"type:varchar(36);not null;index"`
	
	// Execution timing
	StartedAt          time.Time       `json:"started_at" gorm:"not null"`
	CompletedAt        *time.Time      `json:"completed_at,omitempty"`
	ProcessingDuration *int64          `json:"processing_duration,omitempty"` // Duration in milliseconds
	
	// Parameters used for this execution (may differ from job parameters due to profiles)
	ActualParameters   WhisperXParams  `json:"actual_parameters" gorm:"embedded;embeddedPrefix:actual_"`
	
	// Execution results
	Status             JobStatus       `json:"status" gorm:"type:varchar(20);not null"`
	ErrorMessage       *string         `json:"error_message,omitempty" gorm:"type:text"`
	
	// Metadata
	CreatedAt          time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt          time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	
	// Relationship
	TranscriptionJob   TranscriptionJob `json:"transcription_job,omitempty" gorm:"foreignKey:TranscriptionJobID"`
}

// BeforeCreate sets the ID if not already set
func (tje *TranscriptionJobExecution) BeforeCreate(tx *gorm.DB) error {
	if tje.ID == "" {
		tje.ID = uuid.New().String()
	}
	return nil
}

// CalculateProcessingDuration calculates and sets the processing duration
func (tje *TranscriptionJobExecution) CalculateProcessingDuration() {
	if tje.CompletedAt != nil {
		duration := tje.CompletedAt.Sub(tje.StartedAt)
		durationMs := duration.Milliseconds()
		tje.ProcessingDuration = &durationMs
	}
}

// SpeakerMapping represents custom speaker names for a transcription job
type SpeakerMapping struct {
	ID                 uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	TranscriptionJobID string    `json:"transcription_job_id" gorm:"type:varchar(36);not null;index"`
	OriginalSpeaker    string    `json:"original_speaker" gorm:"type:varchar(50);not null"` // e.g., "speaker_00"
	CustomName         string    `json:"custom_name" gorm:"type:varchar(100);not null"`     // e.g., "John Doe"
	CreatedAt          time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt          time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	
	// Relationships
	TranscriptionJob   TranscriptionJob `json:"transcription_job,omitempty" gorm:"foreignKey:TranscriptionJobID"`
}

// Ensure unique constraint on job_id + original_speaker combination
func (SpeakerMapping) TableName() string {
	return "speaker_mappings"
}
