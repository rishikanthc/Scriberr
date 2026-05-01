package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatSessionStatus string

const (
	ChatSessionStatusActive   ChatSessionStatus = "active"
	ChatSessionStatusArchived ChatSessionStatus = "archived"
)

type ChatContextSourceKind string

const (
	ChatContextSourceKindParentTranscript ChatContextSourceKind = "parent_transcript"
	ChatContextSourceKindTranscript       ChatContextSourceKind = "transcript"
)

type ChatContextCompactionStatus string

const (
	ChatContextCompactionStatusNone       ChatContextCompactionStatus = "none"
	ChatContextCompactionStatusCompacting ChatContextCompactionStatus = "compacting"
	ChatContextCompactionStatusCompacted  ChatContextCompactionStatus = "compacted"
	ChatContextCompactionStatusFailed     ChatContextCompactionStatus = "failed"
)

type ChatMessageRole string

const (
	ChatMessageRoleUser      ChatMessageRole = "user"
	ChatMessageRoleAssistant ChatMessageRole = "assistant"
	ChatMessageRoleSystem    ChatMessageRole = "system"
	ChatMessageRoleTool      ChatMessageRole = "tool"
)

type ChatMessageStatus string

const (
	ChatMessageStatusPending   ChatMessageStatus = "pending"
	ChatMessageStatusStreaming ChatMessageStatus = "streaming"
	ChatMessageStatusCompleted ChatMessageStatus = "completed"
	ChatMessageStatusFailed    ChatMessageStatus = "failed"
	ChatMessageStatusCanceled  ChatMessageStatus = "canceled"
)

type ChatGenerationRunStatus string

const (
	ChatGenerationRunStatusPending   ChatGenerationRunStatus = "pending"
	ChatGenerationRunStatusStreaming ChatGenerationRunStatus = "streaming"
	ChatGenerationRunStatusCompleted ChatGenerationRunStatus = "completed"
	ChatGenerationRunStatusFailed    ChatGenerationRunStatus = "failed"
	ChatGenerationRunStatusCanceled  ChatGenerationRunStatus = "canceled"
)

type ChatContextSummaryType string

const (
	ChatContextSummaryTypeTranscript ChatContextSummaryType = "transcript"
	ChatContextSummaryTypeSession    ChatContextSummaryType = "session"
)

// ChatSession is the durable root for one transcript chat workflow.
type ChatSession struct {
	ID                    string            `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID                uint              `json:"user_id" gorm:"not null;index;default:1"`
	ParentTranscriptionID string            `json:"parent_transcription_id" gorm:"type:varchar(36);not null;index"`
	Title                 string            `json:"title" gorm:"type:varchar(255);not null"`
	Provider              string            `json:"provider" gorm:"type:varchar(50);not null"`
	Model                 string            `json:"model" gorm:"column:model_name;type:varchar(255);not null"`
	SystemPrompt          *string           `json:"system_prompt,omitempty" gorm:"type:text"`
	Status                ChatSessionStatus `json:"status" gorm:"type:varchar(20);not null;default:'active'"`
	ContextPolicyJSON     string            `json:"-" gorm:"column:context_policy_json;type:json;not null;default:'{}'"`
	LastMessageAt         *time.Time        `json:"last_message_at,omitempty"`
	CreatedAt             time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt             time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt             gorm.DeletedAt    `json:"deleted_at,omitempty" gorm:"index" swaggertype:"string"`

	ParentTranscription TranscriptionJob     `json:"parent_transcription,omitempty" gorm:"foreignKey:ParentTranscriptionID;references:ID;constraint:OnDelete:CASCADE"`
	ContextSources      []ChatContextSource  `json:"context_sources,omitempty" gorm:"foreignKey:ChatSessionID;references:ID;constraint:OnDelete:CASCADE"`
	Messages            []ChatMessage        `json:"messages,omitempty" gorm:"foreignKey:ChatSessionID;references:ID;constraint:OnDelete:CASCADE"`
	GenerationRuns      []ChatGenerationRun  `json:"generation_runs,omitempty" gorm:"foreignKey:ChatSessionID;references:ID;constraint:OnDelete:CASCADE"`
	ContextSummaries    []ChatContextSummary `json:"context_summaries,omitempty" gorm:"foreignKey:ChatSessionID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ChatSession) TableName() string { return "chat_sessions" }

func (s *ChatSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return s.BeforeSave(tx)
}

func (s *ChatSession) BeforeSave(tx *gorm.DB) error {
	if s.UserID == 0 {
		s.UserID = primaryUserID
	}
	if s.Title == "" {
		s.Title = "New Chat Session"
	}
	if s.Status == "" {
		s.Status = ChatSessionStatusActive
	}
	if s.ContextPolicyJSON == "" {
		s.ContextPolicyJSON = "{}"
	}
	if err := validateChatSessionStatus(s.Status); err != nil {
		return err
	}
	return nil
}

// ChatContextSource is a backend-managed transcript included in a chat context.
type ChatContextSource struct {
	ID                string                      `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID            uint                        `json:"user_id" gorm:"not null;index;default:1"`
	ChatSessionID     string                      `json:"chat_session_id" gorm:"type:varchar(36);not null;index"`
	TranscriptionID   string                      `json:"transcription_id" gorm:"type:varchar(36);not null;index"`
	Kind              ChatContextSourceKind       `json:"kind" gorm:"type:varchar(30);not null"`
	Enabled           bool                        `json:"enabled" gorm:"not null;default:true"`
	Position          int                         `json:"position" gorm:"not null;default:0"`
	PlainTextSnapshot *string                     `json:"plain_text_snapshot,omitempty" gorm:"type:text"`
	SnapshotHash      *string                     `json:"snapshot_hash,omitempty" gorm:"type:varchar(128)"`
	SourceVersion     *string                     `json:"source_version,omitempty" gorm:"type:varchar(128)"`
	CompactedSnapshot *string                     `json:"compacted_snapshot,omitempty" gorm:"type:text"`
	CompactionStatus  ChatContextCompactionStatus `json:"compaction_status" gorm:"type:varchar(20);not null;default:'none'"`
	MetadataJSON      string                      `json:"-" gorm:"column:metadata_json;type:json;not null;default:'{}'"`
	CreatedAt         time.Time                   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time                   `json:"updated_at" gorm:"autoUpdateTime"`

	ChatSession   ChatSession      `json:"chat_session,omitempty" gorm:"foreignKey:ChatSessionID;references:ID;constraint:OnDelete:CASCADE"`
	Transcription TranscriptionJob `json:"transcription,omitempty" gorm:"foreignKey:TranscriptionID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ChatContextSource) TableName() string { return "chat_context_sources" }

func (s *ChatContextSource) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return s.BeforeSave(tx)
}

func (s *ChatContextSource) BeforeSave(tx *gorm.DB) error {
	if s.UserID == 0 {
		s.UserID = primaryUserID
	}
	if s.Kind == "" {
		s.Kind = ChatContextSourceKindTranscript
	}
	if s.CompactionStatus == "" {
		s.CompactionStatus = ChatContextCompactionStatusNone
	}
	if s.MetadataJSON == "" {
		s.MetadataJSON = "{}"
	}
	if err := validateChatContextSourceKind(s.Kind); err != nil {
		return err
	}
	if err := validateChatContextCompactionStatus(s.CompactionStatus); err != nil {
		return err
	}
	return nil
}

// ChatMessage stores user, assistant, system, and tool messages separately from provider runs.
type ChatMessage struct {
	ID               string            `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID           uint              `json:"user_id" gorm:"not null;index;default:1"`
	ChatSessionID    string            `json:"chat_session_id" gorm:"type:varchar(36);not null;index"`
	Role             ChatMessageRole   `json:"role" gorm:"type:varchar(20);not null"`
	Content          string            `json:"content" gorm:"type:text;not null;default:''"`
	ReasoningContent string            `json:"reasoning_content" gorm:"type:text;not null;default:''"`
	Status           ChatMessageStatus `json:"status" gorm:"type:varchar(20);not null;default:'completed'"`
	Provider         *string           `json:"provider,omitempty" gorm:"type:varchar(50)"`
	Model            *string           `json:"model,omitempty" gorm:"column:model_name;type:varchar(255)"`
	RunID            *string           `json:"run_id,omitempty" gorm:"type:varchar(36);index"`
	PromptTokens     *int              `json:"prompt_tokens,omitempty"`
	CompletionTokens *int              `json:"completion_tokens,omitempty"`
	ReasoningTokens  *int              `json:"reasoning_tokens,omitempty"`
	TotalTokens      *int              `json:"total_tokens,omitempty"`
	MetadataJSON     string            `json:"-" gorm:"column:metadata_json;type:json;not null;default:'{}'"`
	CreatedAt        time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time         `json:"updated_at" gorm:"autoUpdateTime"`

	ChatSession ChatSession `json:"chat_session,omitempty" gorm:"foreignKey:ChatSessionID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ChatMessage) TableName() string { return "chat_messages" }

func (m *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return m.BeforeSave(tx)
}

func (m *ChatMessage) BeforeSave(tx *gorm.DB) error {
	if m.UserID == 0 {
		m.UserID = primaryUserID
	}
	if m.Status == "" {
		m.Status = ChatMessageStatusCompleted
	}
	if m.MetadataJSON == "" {
		m.MetadataJSON = "{}"
	}
	if err := validateChatMessageRole(m.Role); err != nil {
		return err
	}
	if err := validateChatMessageStatus(m.Status); err != nil {
		return err
	}
	return nil
}

// ChatGenerationRun tracks a durable model generation lifecycle.
type ChatGenerationRun struct {
	ID                     string                  `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID                 uint                    `json:"user_id" gorm:"not null;index;default:1"`
	ChatSessionID          string                  `json:"chat_session_id" gorm:"type:varchar(36);not null;index"`
	AssistantMessageID     *string                 `json:"assistant_message_id,omitempty" gorm:"type:varchar(36);index"`
	Status                 ChatGenerationRunStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending';index"`
	Provider               string                  `json:"provider" gorm:"type:varchar(50);not null"`
	Model                  string                  `json:"model" gorm:"column:model_name;type:varchar(255);not null"`
	ContextWindow          int                     `json:"context_window" gorm:"not null"`
	ContextTokensEstimated int                     `json:"context_tokens_estimated" gorm:"not null;default:0"`
	CompactionApplied      bool                    `json:"compaction_applied" gorm:"not null;default:false"`
	ErrorMessage           *string                 `json:"error_message,omitempty" gorm:"type:text"`
	StartedAt              *time.Time              `json:"started_at,omitempty"`
	CompletedAt            *time.Time              `json:"completed_at,omitempty"`
	FailedAt               *time.Time              `json:"failed_at,omitempty"`
	CreatedAt              time.Time               `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt              time.Time               `json:"updated_at" gorm:"autoUpdateTime"`

	ChatSession      ChatSession  `json:"chat_session,omitempty" gorm:"foreignKey:ChatSessionID;references:ID;constraint:OnDelete:CASCADE"`
	AssistantMessage *ChatMessage `json:"assistant_message,omitempty" gorm:"foreignKey:AssistantMessageID;references:ID;constraint:OnDelete:SET NULL"`
}

func (ChatGenerationRun) TableName() string { return "chat_generation_runs" }

func (r *ChatGenerationRun) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return r.BeforeSave(tx)
}

func (r *ChatGenerationRun) BeforeSave(tx *gorm.DB) error {
	if r.UserID == 0 {
		r.UserID = primaryUserID
	}
	if r.Status == "" {
		r.Status = ChatGenerationRunStatusPending
	}
	return validateChatGenerationRunStatus(r.Status)
}

// ChatContextSummary stores compacted transcript or session context.
type ChatContextSummary struct {
	ID                     string                 `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID                 uint                   `json:"user_id" gorm:"not null;index;default:1"`
	ChatSessionID          string                 `json:"chat_session_id" gorm:"type:varchar(36);not null;index"`
	SummaryType            ChatContextSummaryType `json:"summary_type" gorm:"type:varchar(20);not null"`
	SourceTranscriptionID  *string                `json:"source_transcription_id,omitempty" gorm:"type:varchar(36);index"`
	SourceMessageThroughID *string                `json:"source_message_through_id,omitempty" gorm:"type:varchar(36);index"`
	Content                string                 `json:"content" gorm:"type:text;not null"`
	Model                  string                 `json:"model" gorm:"column:model_name;type:varchar(255);not null"`
	Provider               string                 `json:"provider" gorm:"type:varchar(50);not null"`
	InputTokensEstimated   int                    `json:"input_tokens_estimated" gorm:"not null;default:0"`
	OutputTokensEstimated  int                    `json:"output_tokens_estimated" gorm:"not null;default:0"`
	CreatedAt              time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt              time.Time              `json:"updated_at" gorm:"autoUpdateTime"`

	ChatSession          ChatSession       `json:"chat_session,omitempty" gorm:"foreignKey:ChatSessionID;references:ID;constraint:OnDelete:CASCADE"`
	SourceTranscription  *TranscriptionJob `json:"source_transcription,omitempty" gorm:"foreignKey:SourceTranscriptionID;references:ID;constraint:OnDelete:CASCADE"`
	SourceMessageThrough *ChatMessage      `json:"source_message_through,omitempty" gorm:"foreignKey:SourceMessageThroughID;references:ID;constraint:OnDelete:SET NULL"`
}

func (ChatContextSummary) TableName() string { return "chat_context_summaries" }

func (s *ChatContextSummary) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return s.BeforeSave(tx)
}

func (s *ChatContextSummary) BeforeSave(tx *gorm.DB) error {
	if s.UserID == 0 {
		s.UserID = primaryUserID
	}
	return validateChatContextSummaryType(s.SummaryType)
}

func validateChatSessionStatus(value ChatSessionStatus) error {
	switch value {
	case ChatSessionStatusActive, ChatSessionStatusArchived:
		return nil
	default:
		return fmt.Errorf("invalid chat session status %q", value)
	}
}

func validateChatContextSourceKind(value ChatContextSourceKind) error {
	switch value {
	case ChatContextSourceKindParentTranscript, ChatContextSourceKindTranscript:
		return nil
	default:
		return fmt.Errorf("invalid chat context source kind %q", value)
	}
}

func validateChatContextCompactionStatus(value ChatContextCompactionStatus) error {
	switch value {
	case ChatContextCompactionStatusNone, ChatContextCompactionStatusCompacting, ChatContextCompactionStatusCompacted, ChatContextCompactionStatusFailed:
		return nil
	default:
		return fmt.Errorf("invalid chat context compaction status %q", value)
	}
}

func validateChatMessageRole(value ChatMessageRole) error {
	switch value {
	case ChatMessageRoleUser, ChatMessageRoleAssistant, ChatMessageRoleSystem, ChatMessageRoleTool:
		return nil
	default:
		return fmt.Errorf("invalid chat message role %q", value)
	}
}

func validateChatMessageStatus(value ChatMessageStatus) error {
	switch value {
	case ChatMessageStatusPending, ChatMessageStatusStreaming, ChatMessageStatusCompleted, ChatMessageStatusFailed, ChatMessageStatusCanceled:
		return nil
	default:
		return fmt.Errorf("invalid chat message status %q", value)
	}
}

func validateChatGenerationRunStatus(value ChatGenerationRunStatus) error {
	switch value {
	case ChatGenerationRunStatusPending, ChatGenerationRunStatusStreaming, ChatGenerationRunStatusCompleted, ChatGenerationRunStatusFailed, ChatGenerationRunStatusCanceled:
		return nil
	default:
		return fmt.Errorf("invalid chat generation run status %q", value)
	}
}

func validateChatContextSummaryType(value ChatContextSummaryType) error {
	switch value {
	case ChatContextSummaryTypeTranscript, ChatContextSummaryTypeSession:
		return nil
	default:
		return fmt.Errorf("invalid chat context summary type %q", value)
	}
}
