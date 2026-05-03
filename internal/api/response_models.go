package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	filesdomain "scriberr/internal/files"
	"scriberr/internal/models"
	recordingdomain "scriberr/internal/recording"

	"github.com/gin-gonic/gin"
)

const defaultMaxUploadSizeBytes int64 = 50 << 30

type UserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type RecordingResponse struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	SourceKind      string    `json:"source_kind"`
	MimeType        string    `json:"mime_type"`
	ReceivedChunks  int       `json:"received_chunks"`
	ReceivedBytes   int64     `json:"received_bytes"`
	DurationSeconds any       `json:"duration_seconds"`
	FileID          any       `json:"file_id"`
	TranscriptionID any       `json:"transcription_id"`
	Progress        float64   `json:"progress"`
	ProgressStage   string    `json:"progress_stage"`
	StartedAt       time.Time `json:"started_at"`
	StoppedAt       any       `json:"stopped_at"`
	CompletedAt     any       `json:"completed_at"`
	FailedAt        any       `json:"failed_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type RecordingChunkResponse struct {
	RecordingID    string `json:"recording_id"`
	ChunkIndex     int    `json:"chunk_index"`
	Status         string `json:"status"`
	ReceivedChunks int    `json:"received_chunks"`
	ReceivedBytes  int64  `json:"received_bytes"`
}

type FileResponse struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Kind            string    `json:"kind"`
	Status          string    `json:"status"`
	MimeType        string    `json:"mime_type"`
	SizeBytes       int64     `json:"size_bytes"`
	DurationSeconds any       `json:"duration_seconds"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type TranscriptionResponse struct {
	ID            string    `json:"id"`
	FileID        string    `json:"file_id"`
	Title         string    `json:"title"`
	Status        string    `json:"status"`
	Language      any       `json:"language"`
	Diarization   bool      `json:"diarization"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Progress      float64   `json:"progress"`
	ProgressStage string    `json:"progress_stage"`
	StartedAt     any       `json:"started_at"`
	CompletedAt   any       `json:"completed_at"`
	FailedAt      any       `json:"failed_at"`
	Error         any       `json:"error"`
}

type TranscriptionListResponse struct {
	ID              string    `json:"id"`
	FileID          string    `json:"file_id"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	Progress        float64   `json:"progress"`
	ProgressStage   string    `json:"progress_stage"`
	DurationSeconds any       `json:"duration_seconds"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ProfileResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
	Options     gin.H  `json:"options"`
	Parameters  gin.H  `json:"parameters"`
	CreatedAt   any    `json:"created_at"`
	UpdatedAt   any    `json:"updated_at"`
}

type SettingsResponse struct {
	AutoTranscriptionEnabled bool `json:"auto_transcription_enabled"`
	AutoRenameEnabled        bool `json:"auto_rename_enabled"`
	DefaultProfileID         any  `json:"default_profile_id"`
	LocalOnly                bool `json:"local_only"`
	MaxUploadSizeMB          int  `json:"max_upload_size_mb"`
}

func userResponse(user *models.User) UserResponse {
	return UserResponse{ID: "user_self", Username: user.Username}
}

func recordingResponse(session *models.RecordingSession) RecordingResponse {
	title := ""
	if session.Title != nil {
		title = *session.Title
	}
	durationSeconds := any(nil)
	if session.DurationMs != nil {
		durationSeconds = float64(*session.DurationMs) / 1000
	}
	fileID := any(nil)
	if session.FileID != nil && *session.FileID != "" {
		fileID = "file_" + *session.FileID
	}
	transcriptionID := any(nil)
	if session.TranscriptionID != nil && *session.TranscriptionID != "" {
		transcriptionID = "tr_" + *session.TranscriptionID
	}
	return RecordingResponse{
		ID:              recordingdomain.PublicID(session.ID),
		Title:           title,
		Status:          string(session.Status),
		SourceKind:      string(session.SourceKind),
		MimeType:        session.MimeType,
		ReceivedChunks:  session.ReceivedChunks,
		ReceivedBytes:   session.ReceivedBytes,
		DurationSeconds: durationSeconds,
		FileID:          fileID,
		TranscriptionID: transcriptionID,
		Progress:        session.Progress,
		ProgressStage:   session.ProgressStage,
		StartedAt:       session.StartedAt,
		StoppedAt:       session.StoppedAt,
		CompletedAt:     session.CompletedAt,
		FailedAt:        session.FailedAt,
		CreatedAt:       session.CreatedAt,
		UpdatedAt:       session.UpdatedAt,
	}
}

func recordingChunkResponse(result *recordingdomain.ChunkResult) RecordingChunkResponse {
	status := "stored"
	if result.AlreadyStored {
		status = "already_stored"
	}
	return RecordingChunkResponse{
		RecordingID:    recordingdomain.PublicID(result.Session.ID),
		ChunkIndex:     result.Chunk.ChunkIndex,
		Status:         status,
		ReceivedChunks: result.ReceivedChunks,
		ReceivedBytes:  result.ReceivedBytes,
	}
}
func publicAPIKeyID(id uint) string {
	return fmt.Sprintf("key_%d", id)
}
func parseAPIKeyID(raw string) (uint, bool) {
	trimmed := strings.TrimPrefix(raw, "key_")
	id, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}
func randomHex(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
func keyPreview(prefix string) string {
	if prefix == "" {
		return "sk_..."
	}
	if len(prefix) > 4 {
		return prefix[:4] + "..." + prefix[len(prefix)-4:]
	}
	return prefix + "..."
}
func stringPtr(value string) *string {
	return &value
}
func boolPtr(value bool) *bool {
	return &value
}
func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func fileResponse(job *models.TranscriptionJob, metadata filesdomain.Metadata) FileResponse {
	title := ""
	if job.Title != nil {
		title = *job.Title
	}
	description := ""
	if job.LLMDescription != nil {
		description = *job.LLMDescription
	}
	status := "uploaded"
	if job.Status == models.StatusUploaded {
		status = "ready"
	} else if job.Status != "" {
		status = string(job.Status)
	}
	durationSeconds := any(nil)
	if metadata.DurationMs != nil {
		durationSeconds = float64(*metadata.DurationMs) / 1000
	}
	return FileResponse{
		ID:              "file_" + job.ID,
		Title:           title,
		Description:     description,
		Kind:            metadata.Kind,
		Status:          status,
		MimeType:        metadata.MimeType,
		SizeBytes:       metadata.SizeBytes,
		DurationSeconds: durationSeconds,
		CreatedAt:       job.CreatedAt,
		UpdatedAt:       job.UpdatedAt,
	}
}

func transcriptionResponse(job *models.TranscriptionJob) TranscriptionResponse {
	title := ""
	if job.Title != nil {
		title = *job.Title
	}
	language := any(nil)
	if job.Language != nil {
		language = *job.Language
	}
	errorValue := any(nil)
	if job.ErrorMessage != nil && *job.ErrorMessage != "" {
		errorValue = sanitizePublicText(*job.ErrorMessage)
	}
	return TranscriptionResponse{
		ID:            "tr_" + job.ID,
		FileID:        fileIDForTranscription(job),
		Title:         title,
		Status:        string(job.Status),
		Language:      language,
		Diarization:   job.Diarization,
		CreatedAt:     job.CreatedAt,
		UpdatedAt:     job.UpdatedAt,
		Progress:      job.Progress,
		ProgressStage: job.ProgressStage,
		StartedAt:     job.StartedAt,
		CompletedAt:   job.CompletedAt,
		FailedAt:      job.FailedAt,
		Error:         errorValue,
	}
}

func transcriptionListResponse(job *models.TranscriptionJob) TranscriptionListResponse {
	title := ""
	if job.Title != nil {
		title = *job.Title
	}
	durationSeconds := any(nil)
	if job.SourceDurationMs != nil {
		durationSeconds = float64(*job.SourceDurationMs) / 1000
	}
	return TranscriptionListResponse{
		ID:              "tr_" + job.ID,
		FileID:          fileIDForTranscription(job),
		Title:           title,
		Status:          string(job.Status),
		Progress:        job.Progress,
		ProgressStage:   job.ProgressStage,
		DurationSeconds: durationSeconds,
		CreatedAt:       job.CreatedAt,
		UpdatedAt:       job.UpdatedAt,
	}
}
func fileIDForTranscription(job *models.TranscriptionJob) string {
	if job.SourceFileHash != nil && *job.SourceFileHash != "" {
		return "file_" + *job.SourceFileHash
	}
	return "file_" + job.ID
}
func validLanguage(language string) bool {
	if len(language) != 2 {
		return false
	}
	for _, r := range language {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}
func validTranscriptionStatus(status string) bool {
	switch models.JobStatus(status) {
	case models.StatusPending, models.StatusProcessing, models.StatusCompleted, models.StatusFailed, models.StatusStopped, models.StatusCanceled:
		return true
	default:
		return false
	}
}
func validFileStatus(status string) bool {
	switch status {
	case "uploaded", "ready", string(models.StatusProcessing), string(models.StatusFailed):
		return true
	default:
		return false
	}
}
func profileResponse(profile *models.TranscriptionProfile) ProfileResponse {
	description := ""
	if profile.Description != nil {
		description = *profile.Description
	}
	options := profileOptionsMap(profile.Parameters)
	return ProfileResponse{
		ID:          publicIDForProfile(profile.ID),
		Name:        profile.Name,
		Description: description,
		IsDefault:   profile.IsDefault,
		Options:     options,
		Parameters:  options,
		CreatedAt:   profile.CreatedAt,
		UpdatedAt:   profile.UpdatedAt,
	}
}
func profileOptionsMap(params models.WhisperXParams) gin.H {
	params = supportedProfileParams(params)
	params.EnableTokenTimestamps = nil
	params.EnableSegmentTimestamps = nil
	var options gin.H
	bytes, err := json.Marshal(params)
	if err != nil || json.Unmarshal(bytes, &options) != nil {
		options = gin.H{}
	}
	options["diarization"] = params.Diarize
	return options
}
func settingsResponse(h *Handler, user *models.User) SettingsResponse {
	defaultProfileID := any(nil)
	if user.DefaultProfileID != nil && *user.DefaultProfileID != "" {
		defaultProfileID = publicIDForProfile(*user.DefaultProfileID)
	}
	return SettingsResponse{
		AutoTranscriptionEnabled: user.AutoTranscriptionEnabled,
		AutoRenameEnabled:        user.AutoRenameEnabled,
		DefaultProfileID:         defaultProfileID,
		LocalOnly:                true,
		MaxUploadSizeMB:          maxUploadSizeMB(h),
	}
}
func maxUploadSizeMB(h *Handler) int {
	return int(uploadSizeLimit(h) / (1 << 20))
}
func uploadSizeLimit(h *Handler) int64 {
	if h != nil && h.maxUploadBytes > 0 {
		return h.maxUploadBytes
	}
	return defaultMaxUploadSizeBytes
}
