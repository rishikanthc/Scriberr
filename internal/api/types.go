package api

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
	Model       string `json:"model"`
	Language    string `json:"language"`
	Diarization bool   `json:"diarization"`
	Device      string `json:"device"`
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
	DefaultProfileID         *string `json:"default_profile_id"`
}
