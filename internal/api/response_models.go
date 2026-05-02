package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"scriberr/internal/models"

	"github.com/gin-gonic/gin"
)

const defaultMaxUploadSizeBytes int64 = 50 << 30

func userResponse(user *models.User) gin.H {
	return gin.H{
		"id":       "user_self",
		"username": user.Username,
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
func fileResponse(job *models.TranscriptionJob, mimeType, kind string) gin.H {
	title := ""
	if job.Title != nil {
		title = *job.Title
	}
	description := ""
	if job.LLMDescription != nil {
		description = *job.LLMDescription
	}
	size := int64(0)
	if stat, err := os.Stat(job.AudioPath); err == nil {
		size = stat.Size()
	}
	status := "uploaded"
	if job.Status == models.StatusUploaded {
		status = "ready"
	} else if job.Status != "" {
		status = string(job.Status)
	}
	if strings.HasPrefix(job.SourceFileName, "youtube:") {
		kind = "youtube"
		if mimeType == "" {
			mimeType = mediaType("", strings.TrimPrefix(job.SourceFileName, "youtube:"))
		}
	}
	durationSeconds := any(nil)
	if job.SourceDurationMs != nil {
		durationSeconds = float64(*job.SourceDurationMs) / 1000
	}
	return gin.H{
		"id":               "file_" + job.ID,
		"title":            title,
		"description":      description,
		"kind":             kind,
		"status":           status,
		"mime_type":        mimeType,
		"size_bytes":       size,
		"duration_seconds": durationSeconds,
		"created_at":       job.CreatedAt,
		"updated_at":       job.UpdatedAt,
	}
}
func mediaType(headerValue, filename string) string {
	cleanHeader := strings.ToLower(strings.TrimSpace(strings.Split(headerValue, ";")[0]))
	if strings.HasPrefix(cleanHeader, "audio/") || strings.HasPrefix(cleanHeader, "video/") {
		return cleanHeader
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".wav":
		return "audio/wav"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a":
		return "audio/mp4"
	case ".aac":
		return "audio/aac"
	case ".flac":
		return "audio/flac"
	case ".ogg":
		return "audio/ogg"
	case ".opus":
		return "audio/opus"
	case ".webm":
		return "video/webm"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".mkv":
		return "video/x-matroska"
	case ".avi":
		return "video/x-msvideo"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".flv":
		return "video/x-flv"
	default:
		return cleanHeader
	}
}
func fileKind(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	default:
		return ""
	}
}
func safeFilename(filename string) string {
	base := filepath.Base(strings.TrimSpace(filename))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return strings.NewReplacer("/", "_", "\\", "_", "\x00", "").Replace(base)
}
func transcriptionResponse(job *models.TranscriptionJob) gin.H {
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
	return gin.H{
		"id":             "tr_" + job.ID,
		"file_id":        fileIDForTranscription(job),
		"title":          title,
		"status":         string(job.Status),
		"language":       language,
		"diarization":    job.Diarization,
		"created_at":     job.CreatedAt,
		"updated_at":     job.UpdatedAt,
		"progress":       job.Progress,
		"progress_stage": job.ProgressStage,
		"started_at":     job.StartedAt,
		"completed_at":   job.CompletedAt,
		"failed_at":      job.FailedAt,
		"error":          errorValue,
	}
}
func transcriptionListResponse(job *models.TranscriptionJob) gin.H {
	title := ""
	if job.Title != nil {
		title = *job.Title
	}
	durationSeconds := any(nil)
	if job.SourceDurationMs != nil {
		durationSeconds = float64(*job.SourceDurationMs) / 1000
	}
	return gin.H{
		"id":               "tr_" + job.ID,
		"file_id":          fileIDForTranscription(job),
		"title":            title,
		"status":           string(job.Status),
		"progress":         job.Progress,
		"progress_stage":   job.ProgressStage,
		"duration_seconds": durationSeconds,
		"created_at":       job.CreatedAt,
		"updated_at":       job.UpdatedAt,
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
func profileResponse(profile *models.TranscriptionProfile) gin.H {
	description := ""
	if profile.Description != nil {
		description = *profile.Description
	}
	options := profileOptionsMap(profile.Parameters)
	return gin.H{
		"id":          publicIDForProfile(profile.ID),
		"name":        profile.Name,
		"description": description,
		"is_default":  profile.IsDefault,
		"options":     options,
		"parameters":  options,
		"created_at":  profile.CreatedAt,
		"updated_at":  profile.UpdatedAt,
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
func settingsResponse(h *Handler, user *models.User) gin.H {
	defaultProfileID := any(nil)
	if user.DefaultProfileID != nil && *user.DefaultProfileID != "" {
		defaultProfileID = publicIDForProfile(*user.DefaultProfileID)
	}
	return gin.H{
		"auto_transcription_enabled": user.AutoTranscriptionEnabled,
		"default_profile_id":         defaultProfileID,
		"local_only":                 true,
		"max_upload_size_mb":         maxUploadSizeMB(h),
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
