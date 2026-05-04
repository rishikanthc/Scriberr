package api

import "scriberr/internal/models"

type ErrorBody struct {
	Error APIError `json:"error"`
}
type APIError struct {
	Code      string  `json:"code"`
	Message   string  `json:"message"`
	Field     *string `json:"field,omitempty"`
	RequestID string  `json:"request_id"`
}
type registerRequest struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}
type changeUsernameRequest struct {
	NewUsername string `json:"new_username"`
	Password    string `json:"password"`
}
type createAPIKeyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type updateFileRequest struct {
	Title string `json:"title"`
}
type importYouTubeRequest struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}
type createRecordingRequest struct {
	Title           string  `json:"title"`
	SourceKind      string  `json:"source_kind"`
	MimeType        string  `json:"mime_type"`
	Codec           *string `json:"codec"`
	ChunkDurationMs *int64  `json:"chunk_duration_ms"`
	AutoTranscribe  bool    `json:"auto_transcribe"`
	ProfileID       *string `json:"profile_id"`
	Options         struct {
		Language    string `json:"language"`
		Diarization *bool  `json:"diarization"`
	} `json:"options"`
}
type stopRecordingRequest struct {
	FinalChunkIndex int    `json:"final_chunk_index"`
	DurationMs      *int64 `json:"duration_ms"`
	AutoTranscribe  *bool  `json:"auto_transcribe"`
}
type createTranscriptionRequest struct {
	FileID    string `json:"file_id"`
	Title     string `json:"title"`
	ProfileID string `json:"profile_id"`
	Options   struct {
		Language    string `json:"language"`
		Diarization *bool  `json:"diarization"`
	} `json:"options"`
}
type updateTranscriptionRequest struct {
	Title string `json:"title"`
}
type profileOptionsRequest struct {
	Pipeline             []models.ASRStep `json:"pipeline,omitempty"`
	Language             *string          `json:"language,omitempty"`
	Task                 string           `json:"task"`
	Threads              int              `json:"threads"`
	TailPaddings         *int             `json:"tail_paddings,omitempty"`
	DecodingMethod       string           `json:"decoding_method"`
	ChunkingStrategy     string           `json:"chunking_strategy"`
	NumSpeakers          int              `json:"num_speakers"`
	DiarizationThreshold float64          `json:"diarization_threshold"`
	MinDurationOn        float64          `json:"min_duration_on"`
	MinDurationOff       float64          `json:"min_duration_off"`
}
type createProfileRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	IsDefault   bool                  `json:"is_default"`
	Options     profileOptionsRequest `json:"options"`
}
type updateProfileRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	IsDefault   *bool                 `json:"is_default"`
	Options     profileOptionsRequest `json:"options"`
}
type updateSettingsRequest struct {
	AutoTranscriptionEnabled *bool   `json:"auto_transcription_enabled"`
	AutoRenameEnabled        *bool   `json:"auto_rename_enabled"`
	DefaultProfileID         *string `json:"default_profile_id"`
}
type loadASRProviderModelRequest struct {
	Model      string         `json:"model"`
	Operation  string         `json:"operation"`
	LoadPolicy string         `json:"load_policy"`
	Options    map[string]any `json:"options"`
}
type unloadASRProviderModelRequest struct {
	Model   string         `json:"model"`
	Force   bool           `json:"force"`
	Options map[string]any `json:"options"`
}
type adminCreateUserRequest struct {
	Username    string  `json:"username"`
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
	Role        string  `json:"role"`
	Password    string  `json:"password"`
}
type adminUpdateUserRequest struct {
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
	Role        *string `json:"role"`
	Status      *string `json:"status"`
}
type adminResetPasswordRequest struct {
	Password string `json:"password"`
}
type adminSchedulerRequest struct {
	Policy               string `json:"policy"`
	MaxConcurrentPerUser int    `json:"max_concurrent_per_user"`
}
type updateLLMProviderRequest struct {
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key"`
	LargeModel string `json:"large_model"`
	SmallModel string `json:"small_model"`
}
type summaryWidgetRequest struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	AlwaysEnabled  bool    `json:"always_enabled"`
	WhenToUse      *string `json:"when_to_use"`
	ContextSource  string  `json:"context_source"`
	Prompt         string  `json:"prompt"`
	RenderMarkdown bool    `json:"render_markdown"`
	DisplayTitle   string  `json:"display_title"`
	Enabled        *bool   `json:"enabled"`
}
