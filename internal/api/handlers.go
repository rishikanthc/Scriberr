package api

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"scriberr/internal/auth"
	"scriberr/internal/config"
	"scriberr/internal/csvbatch"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/processing"
	"scriberr/internal/queue"
	"scriberr/internal/repository"
	"scriberr/internal/service"
	"scriberr/internal/sse"
	"scriberr/internal/transcription"
	"scriberr/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Handler contains all the API handlers
type Handler struct {
	config              *config.Config
	authService         *auth.AuthService
	userService         service.UserService
	fileService         service.FileService
	jobRepo             repository.JobRepository
	apiKeyRepo          repository.APIKeyRepository
	profileRepo         repository.ProfileRepository
	userRepo            repository.UserRepository
	llmConfigRepo       repository.LLMConfigRepository
	summaryRepo         repository.SummaryRepository
	chatRepo            repository.ChatRepository
	noteRepo            repository.NoteRepository
	speakerMappingRepo  repository.SpeakerMappingRepository
	refreshTokenRepo    repository.RefreshTokenRepository
	taskQueue           *queue.TaskQueue
	unifiedProcessor    *transcription.UnifiedJobProcessor
	quickTranscription  *transcription.QuickTranscriptionService
	multiTrackProcessor *processing.MultiTrackProcessor
	csvBatchProcessor   *csvbatch.Processor
}

// NewHandler creates a new handler
func NewHandler(cfg *config.Config, authService *auth.AuthService, taskQueue *queue.TaskQueue, unifiedProcessor *transcription.UnifiedJobProcessor, quickTranscription *transcription.QuickTranscriptionService) *Handler {
	// Create CSV batch processor
	csvProcessor := csvbatch.New(cfg)

	broadcaster         *sse.Broadcaster
}

// NewHandler creates a new handler
func NewHandler(
	cfg *config.Config,
	authService *auth.AuthService,
	userService service.UserService,
	fileService service.FileService,
	jobRepo repository.JobRepository,
	apiKeyRepo repository.APIKeyRepository,
	profileRepo repository.ProfileRepository,
	userRepo repository.UserRepository,
	llmConfigRepo repository.LLMConfigRepository,
	summaryRepo repository.SummaryRepository,
	chatRepo repository.ChatRepository,
	noteRepo repository.NoteRepository,
	speakerMappingRepo repository.SpeakerMappingRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	taskQueue *queue.TaskQueue,
	unifiedProcessor *transcription.UnifiedJobProcessor,
	quickTranscription *transcription.QuickTranscriptionService,
	multiTrackProcessor *processing.MultiTrackProcessor,
	broadcaster *sse.Broadcaster,
) *Handler {
	return &Handler{
		config:              cfg,
		authService:         authService,
		userService:         userService,
		fileService:         fileService,
		jobRepo:             jobRepo,
		apiKeyRepo:          apiKeyRepo,
		profileRepo:         profileRepo,
		userRepo:            userRepo,
		llmConfigRepo:       llmConfigRepo,
		summaryRepo:         summaryRepo,
		chatRepo:            chatRepo,
		noteRepo:            noteRepo,
		speakerMappingRepo:  speakerMappingRepo,
		refreshTokenRepo:    refreshTokenRepo,
		taskQueue:           taskQueue,
		unifiedProcessor:    unifiedProcessor,
		quickTranscription:  quickTranscription,
		multiTrackProcessor: processing.NewMultiTrackProcessor(),
		csvBatchProcessor:   csvProcessor,
		multiTrackProcessor: multiTrackProcessor,
		broadcaster:         broadcaster,
	}
}

// GetCSVBatchProcessor returns the CSV batch processor for route registration
func (h *Handler) GetCSVBatchProcessor() *csvbatch.Processor {
	return h.csvBatchProcessor
}

// SubmitJobRequest represents the submit job request
type SubmitJobRequest struct {
	Title       *string               `json:"title,omitempty"`
	Diarization bool                  `json:"diarization"`
	Parameters  models.WhisperXParams `json:"parameters"`
}

// LoginRequest represents the login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
}

// RegisterRequest represents the registration request
type RegisterRequest struct {
	Username        string `json:"username" binding:"required,min=3,max=50"`
	Password        string `json:"password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
}

// RegistrationStatusResponse represents the registration status
type RegistrationStatusResponse struct {
	// Match tests expecting snake_case key
	RegistrationEnabled bool `json:"registration_enabled"`
}

// ChangePasswordRequest represents the change password request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
}

// ChangeUsernameRequest represents the change username request
type ChangeUsernameRequest struct {
	NewUsername string `json:"newUsername" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required"`
}

// CreateAPIKeyRequest represents the create API key request
type CreateAPIKeyRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description,omitempty"`
}

// CreateAPIKeyResponse represents the create API key response
type CreateAPIKeyResponse struct {
	ID          uint   `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// YouTubeDownloadRequest represents the YouTube download request
type YouTubeDownloadRequest struct {
	URL   string  `json:"url" binding:"required"`
	Title *string `json:"title,omitempty"`
}

// YouTubeDownloadResponse represents the YouTube download response
type YouTubeDownloadResponse struct {
	JobID    string `json:"job_id"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	Title    string `json:"title,omitempty"`
	Progress int    `json:"progress,omitempty"`
}

// LLMConfigRequest represents the LLM configuration request
type LLMConfigRequest struct {
	Provider      string  `json:"provider" binding:"required,oneof=ollama openai"`
	BaseURL       *string `json:"base_url,omitempty"`
	OpenAIBaseURL *string `json:"openai_base_url,omitempty"`
	APIKey        *string `json:"api_key,omitempty"`
	IsActive      bool    `json:"is_active"`
}

// LLMConfigResponse represents the LLM configuration response
type LLMConfigResponse struct {
	ID            uint    `json:"id"`
	Provider      string  `json:"provider"`
	BaseURL       *string `json:"base_url,omitempty"`
	OpenAIBaseURL *string `json:"openai_base_url,omitempty"`
	HasAPIKey     bool    `json:"has_api_key"` // Don't return actual API key
	IsActive      bool    `json:"is_active"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// APIKeyListResponse represents an API key in the list (without the actual key)
type APIKeyListResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	KeyPreview  string `json:"key_preview"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	LastUsed    string `json:"last_used,omitempty"`
}

// APIKeysWrapper wraps the API keys list response
type APIKeysWrapper struct {
	APIKeys []APIKeyListResponse `json:"api_keys"`
}

// transformAPIKeyForList converts a models.APIKey to APIKeyListResponse
func transformAPIKeyForList(apiKey models.APIKey) APIKeyListResponse {
	keyPreview := ""
	if len(apiKey.Key) > 8 {
		keyPreview = apiKey.Key[:8] + "..."
	} else if apiKey.Key != "" {
		keyPreview = apiKey.Key + "..."
	}

	lastUsed := ""
	if apiKey.LastUsed != nil {
		lastUsed = apiKey.LastUsed.Format(time.RFC3339)
	}

	description := ""
	if apiKey.Description != nil {
		description = *apiKey.Description
	}

	return APIKeyListResponse{
		ID:          apiKey.ID,
		Name:        apiKey.Name,
		Description: description,
		KeyPreview:  keyPreview,
		IsActive:    apiKey.IsActive,
		CreatedAt:   apiKey.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   apiKey.UpdatedAt.Format(time.RFC3339),
		LastUsed:    lastUsed,
	}
}

// @Summary Upload audio file
// @Description Upload an audio file without starting transcription
// @Tags transcription
// @Accept multipart/form-data
// @Produce json
// @Param audio formData file true "Audio file"
// @Param title formData string false "Job title"
// @Success 200 {object} models.TranscriptionJob
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/transcription/upload [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) UploadAudio(c *gin.Context) {
	// Note: This endpoint is also used by the CLI watcher to upload files.
	// The CLI authenticates using a long-lived JWT token.

	// Parse multipart form
	header, err := c.FormFile(paramAudio)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Audio file is required"})
		return
	}

	// Save file using FileService
	uploadDir := h.config.UploadDir
	filePath, err := h.fileService.SaveUpload(header, uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Create job record
	jobID := filepath.Base(filePath)
	jobID = jobID[:len(jobID)-len(filepath.Ext(jobID))] // Extract ID from filename

	job := models.TranscriptionJob{
		ID:        jobID,
		AudioPath: filePath,
		Status:    models.StatusUploaded,
	}

	if title := c.PostForm(paramTitle); title != "" {
		job.Title = &title
	}

	// Save to database using Repository
	if err := h.jobRepo.Create(c.Request.Context(), &job); err != nil {
		_ = h.fileService.RemoveFile(filePath) // Clean up file
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}

	// Check for auto-transcription if user is authenticated via JWT
	if userID, exists := c.Get("user_id"); exists {
		// Use UserService to get user
		user, err := h.userService.GetUser(c.Request.Context(), userID.(uint))
		if err == nil && user.AutoTranscriptionEnabled {
			// Get user's default profile or use system default
			var profile *models.TranscriptionProfile

			if user.DefaultProfileID != nil {
				profile, _ = h.profileRepo.FindByID(c.Request.Context(), *user.DefaultProfileID)
			}

			// If no user default or user default not found, try to find a system default
			if profile == nil {
				profile, _ = h.profileRepo.FindDefault(c.Request.Context())
			}

			// If still no profile found, use the first available profile
			if profile == nil {
				profiles, _, _ := h.profileRepo.List(c.Request.Context(), 0, 1)
				if len(profiles) > 0 {
					profile = &profiles[0]
				}
			}

			// If we found a profile, update the job and queue it
			if profile != nil {
				job.Parameters = profile.Parameters
				job.Diarization = profile.Parameters.Diarize
				job.Status = models.StatusPending

				// Update the job in database
				if err := h.jobRepo.Update(c.Request.Context(), &job); err == nil {
					// Enqueue the job for transcription
					if err := h.taskQueue.EnqueueJob(jobID); err != nil {
						// If enqueueing fails, revert status but don't fail the upload
						job.Status = models.StatusUploaded
						_ = h.jobRepo.Update(c.Request.Context(), &job)
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, job)
}

// @Summary Upload video file for transcription
// @Description Upload a video file, extract audio from it using ffmpeg, and create a transcription job
// @Tags transcription
// @Accept multipart/form-data
// @Produce json
// @Param video formData file true "Video file"
// @Param title formData string false "Job title"
// @Success 200 {object} models.TranscriptionJob
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/transcription/upload-video [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) UploadVideo(c *gin.Context) {
	// Parse multipart form
	header, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Video file is required"})
		return
	}

	// Save file using FileService
	uploadDir := h.config.UploadDir
	videoPath, err := h.fileService.SaveUpload(header, uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Generate job ID from filename
	jobID := filepath.Base(videoPath)
	jobID = jobID[:len(jobID)-len(filepath.Ext(jobID))]

	// Extract audio using ffmpeg (keep this logic here for now, or move to a MediaService)
	audioPath := strings.TrimSuffix(videoPath, filepath.Ext(videoPath)) + ".mp3"
	cmd := exec.Command("ffmpeg", "-i", videoPath, "-vn", "-acodec", "libmp3lame", "-q:a", "2", audioPath)
	if err := cmd.Run(); err != nil {
		_ = h.fileService.RemoveFile(videoPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract audio from video"})
		return
	}

	// Create job record
	job := models.TranscriptionJob{
		ID:        jobID,
		AudioPath: audioPath, // Use the extracted audio path
		Status:    models.StatusUploaded,
	}

	if title := c.PostForm(paramTitle); title != "" {
		job.Title = &title
	}

	// Save to database
	if err := h.jobRepo.Create(c.Request.Context(), &job); err != nil {
		_ = h.fileService.RemoveFile(videoPath)
		_ = h.fileService.RemoveFile(audioPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}

	// Clean up video file as we only need audio
	// TODO: Make this configurable? Some users might want to keep the video.
	_ = h.fileService.RemoveFile(videoPath)

	// Check for auto-transcription (same logic as UploadAudio)
	if userID, exists := c.Get("user_id"); exists {
		user, err := h.userService.GetUser(c.Request.Context(), userID.(uint))
		if err == nil && user.AutoTranscriptionEnabled {
			var profile *models.TranscriptionProfile
			if user.DefaultProfileID != nil {
				profile, _ = h.profileRepo.FindByID(c.Request.Context(), *user.DefaultProfileID)
			}
			if profile == nil {
				profile, _ = h.profileRepo.FindDefault(c.Request.Context())
			}
			if profile == nil {
				profiles, _, _ := h.profileRepo.List(c.Request.Context(), 0, 1)
				if len(profiles) > 0 {
					profile = &profiles[0]
				}
			}

			if profile != nil {
				job.Parameters = profile.Parameters
				job.Diarization = profile.Parameters.Diarize
				job.Status = models.StatusPending
				if err := h.jobRepo.Update(c.Request.Context(), &job); err == nil {
					if err := h.taskQueue.EnqueueJob(jobID); err != nil {
						job.Status = models.StatusUploaded
						_ = h.jobRepo.Update(c.Request.Context(), &job)
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, job)
}

// @Summary Upload multi-track audio files
// @Description Upload multiple audio files for multi-track transcription
// @Tags transcription
// @Accept multipart/form-data
// @Produce json
// @Param title formData string false "Job title"
// @Param files formData file true "Audio track files" multiple
// @Success 200 {object} models.TranscriptionJob
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/transcription/upload-multitrack [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) UploadMultiTrack(c *gin.Context) {
	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files uploaded"})
		return
	}

	// Create a unique job ID
	jobID := uuid.New().String()
	uploadDir := h.config.UploadDir

	// Create job directory
	jobDir := filepath.Join(uploadDir, jobID)
	if err := h.fileService.CreateDirectory(jobDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job directory"})
		return
	}

	var trackFiles []models.MultiTrackFile

	// Process each file
	for i, fileHeader := range files {
		// Save file using FileService
		filePath, err := h.fileService.SaveUpload(fileHeader, jobDir)
		if err != nil {
			// Cleanup
			_ = h.fileService.RemoveDirectory(jobDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save file %s", fileHeader.Filename)})
			return
		}

		// Create track record
		trackFiles = append(trackFiles, models.MultiTrackFile{
			TranscriptionJobID: jobID,
			FilePath:           filePath,
			FileName:           fileHeader.Filename,
			TrackIndex:         i,
		})
	}

	// Create job record
	job := models.TranscriptionJob{
		ID:              jobID,
		Status:          models.StatusUploaded,
		IsMultiTrack:    true,
		MultiTrackFiles: trackFiles,
	}

	if title := c.PostForm(paramTitle); title != "" {
		job.Title = &title
	} else {
		defaultTitle := fmt.Sprintf("Multi-track Job %s", jobID)
		job.Title = &defaultTitle
	}

	// Save to database
	if err := h.jobRepo.Create(c.Request.Context(), &job); err != nil {
		_ = h.fileService.RemoveDirectory(jobDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}
}

// @Summary Get multi-track merge status
// @Description Get the current merge status for a multi-track job
// @Tags transcription
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /api/v1/transcription/{id}/merge-status [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetMergeStatus(c *gin.Context) {
	jobID := c.Param("id")

	status, errorMsg, err := h.multiTrackProcessor.GetMergeStatus(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	response := gin.H{
		"merge_status": status,
	}

	if errorMsg != nil {
		response["merge_error"] = *errorMsg
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Get multi-track job progress
// @Description Get real-time progress information for individual tracks in a multi-track job
// @Tags transcription
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /api/v1/transcription/{id}/track-progress [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetTrackProgress(c *gin.Context) {
	jobID := c.Param("id")

	// Get the main job details using repository
	job, err := h.jobRepo.FindWithAssociations(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Only provide track progress for multi-track jobs
	if !job.IsMultiTrack {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not a multi-track job"})
		return
	}

	// Get individual transcripts to see which tracks are completed
	var individualTranscripts map[string]string
	if job.IndividualTranscripts != nil {
		_ = json.Unmarshal([]byte(*job.IndividualTranscripts), &individualTranscripts)
	}

	// Find active track jobs (temp jobs still in progress)
	activeTrackJobs, _ := h.jobRepo.FindActiveTrackJobs(c.Request.Context(), jobID)

	// Build track progress information
	trackProgress := make([]map[string]interface{}, 0)
	totalTracks := len(job.MultiTrackFiles)
	completedTracks := 0

	for _, trackFile := range job.MultiTrackFiles {
		trackInfo := map[string]interface{}{
			"track_name":  trackFile.FileName,
			"track_index": trackFile.TrackIndex,
		}

		// Check if this track is completed (only count if actually in individualTranscripts)
		if _, exists := individualTranscripts[trackFile.FileName]; exists {
			trackInfo["status"] = "completed"
			completedTracks++
		} else {
			// Check if there's an active job for this track
			isActive := false
			for _, activeJob := range activeTrackJobs {
				if strings.Contains(activeJob.ID, trackFile.FileName) {
					trackInfo["status"] = "processing"
					isActive = true
					break
				}
			}
			if !isActive {
				if job.Status == "processing" {
					trackInfo["status"] = "pending"
				} else {
					trackInfo["status"] = "failed"
				}
			}
		}

		trackProgress = append(trackProgress, trackInfo)
	}

	// Calculate overall progress based on actual track status
	progressPercentage := 0.0
	if totalTracks > 0 {
		progressPercentage = float64(completedTracks) / float64(totalTracks) * 100
	}

	response := gin.H{
		paramJobID:       job.ID,
		"is_multi_track": true,
		"overall_status": job.Status,
		"merge_status":   job.MergeStatus,
		"tracks":         trackProgress,
		"progress": map[string]interface{}{
			"completed_tracks": completedTracks,
			"total_tracks":     totalTracks,
			"percentage":       progressPercentage,
		},
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Submit a transcription job
// @Description Submit an audio file for transcription with WhisperX
// @Tags transcription
// @Accept multipart/form-data
// @Produce json
// @Param audio formData file true "Audio file"
// @Param title formData string false "Job title"
// @Param diarization formData boolean false "Enable speaker diarization"
// @Param model formData string false "Whisper model" default(base)
// @Param language formData string false "Language code"
// @Param batch_size formData int false "Batch size" default(16)
// @Param compute_type formData string false "Compute type" default(float16)
// @Param device formData string false "Device" default(auto)
// @Param vad_filter formData boolean false "Enable VAD filter"
// @Param vad_onset formData number false "VAD onset" default(0.500)
// @Param vad_offset formData number false "VAD offset" default(0.363)
// @Param min_speakers formData int false "Minimum speakers for diarization"
// @Param max_speakers formData int false "Maximum speakers for diarization"
// @Success 200 {object} models.TranscriptionJob
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/transcription/submit [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) SubmitJob(c *gin.Context) {
	// Parse multipart form
	header, err := c.FormFile(paramAudio)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Audio file is required"})
		return
	}

	// Save file using FileService
	uploadDir := h.config.UploadDir
	filePath, err := h.fileService.SaveUpload(header, uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Generate job ID from filename
	jobID := filepath.Base(filePath)
	jobID = jobID[:len(jobID)-len(filepath.Ext(jobID))]

	// Parse parameters (accept both 'diarization' and 'diarize')
	diarize := false
	if v := c.PostForm("diarization"); v != "" {
		diarize = strings.EqualFold(v, "true") || v == "1"
	} else {
		diarize = getFormBoolWithDefault(c, "diarize", false)
	}
	params := models.WhisperXParams{
		Model:       getFormValueWithDefault(c, "model", "base"),
		BatchSize:   getFormIntWithDefault(c, "batch_size", 16),
		ComputeType: getFormValueWithDefault(c, "compute_type", "int8"),
		Device:      getFormValueWithDefault(c, "device", "cpu"),
		VadOnset:    getFormFloatWithDefault(c, "vad_onset", 0.500),
		VadOffset:   getFormFloatWithDefault(c, "vad_offset", 0.363),
		Diarize:     diarize,
	}

	if lang := c.PostForm("language"); lang != "" {
		params.Language = &lang
	}

	if minSpeakers := c.PostForm("min_speakers"); minSpeakers != "" {
		if min, err := strconv.Atoi(minSpeakers); err == nil {
			params.MinSpeakers = &min
		}
	}

	if maxSpeakers := c.PostForm("max_speakers"); maxSpeakers != "" {
		if max, err := strconv.Atoi(maxSpeakers); err == nil {
			params.MaxSpeakers = &max
		}
	}

	if hfToken := c.PostForm("hf_token"); hfToken != "" {
		params.HfToken = &hfToken
	}

	// Parse and validate diarization model
	diarizeModel := getFormValueWithDefault(c, "diarize_model", "pyannote")
	if diarizeModel != "pyannote" && diarizeModel != "nvidia_sortformer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diarize_model. Must be 'pyannote' or 'nvidia_sortformer'"})
		_ = h.fileService.RemoveFile(filePath)
		return
	}
	params.DiarizeModel = diarizeModel

	// Create job
	job := models.TranscriptionJob{
		ID:          jobID,
		AudioPath:   filePath,
		Status:      models.StatusPending,
		Diarization: diarize,
		Parameters:  params,
	}

	if title := c.PostForm(paramTitle); title != "" {
		job.Title = &title
	}

	// Save to database
	if err := h.jobRepo.Create(c.Request.Context(), &job); err != nil {
		_ = h.fileService.RemoveFile(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}

	// Enqueue job
	if err := h.taskQueue.EnqueueJob(jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue job"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// @Summary Get job status
// @Description Get the current status of a transcription job
// @Tags transcription
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} models.TranscriptionJob
// @Failure 404 {object} map[string]string
// @Router /api/v1/transcription/{id}/status [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetJobStatus(c *gin.Context) {
	jobID := c.Param("id")

	job, err := h.taskQueue.GetJobStatus(jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job status"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// @Summary Get transcript
// @Description Get the transcript for a completed transcription job
// @Tags transcription
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/v1/transcription/{id}/transcript [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetTranscript(c *gin.Context) {
	jobID := c.Param("id")

	job, err := h.jobRepo.FindByID(c.Request.Context(), jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}

	// Return empty transcript gracefully for non-completed jobs
	if job.Status != models.StatusCompleted {
		c.JSON(http.StatusOK, gin.H{
			"job_id":     job.ID,
			"title":      job.Title,
			"transcript": nil,
			"status":     job.Status,
			"available":  false,
			"message":    fmt.Sprintf("Transcript not ready, current status: %s", job.Status),
			"created_at": job.CreatedAt,
			"updated_at": job.UpdatedAt,
		})
		return
	}

	// Return empty transcript gracefully if nil
	if job.Transcript == nil {
		c.JSON(http.StatusOK, gin.H{
			"job_id":     job.ID,
			"title":      job.Title,
			"transcript": nil,
			"status":     job.Status,
			"available":  false,
			"message":    "Transcript not available",
			"created_at": job.CreatedAt,
			"updated_at": job.UpdatedAt,
		})
		return
	}

	var transcript interface{}
	if err := json.Unmarshal([]byte(*job.Transcript), &transcript); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse transcript"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":     job.ID,
		"title":      job.Title,
		"transcript": transcript,
		"status":     job.Status,
		"available":  true,
		"created_at": job.CreatedAt,
		"updated_at": job.UpdatedAt,
	})
}

// @Summary List all transcription records
// @Description Get a list of all transcription jobs with optional search and filtering
// @Tags transcription
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Summary List all transcription records
// @Description Get a list of all transcription jobs with optional search and filtering
// @Tags transcription
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param sort_by query string false "Sort By"
// @Param sort_order query string false "Sort Order (asc/desc)"
// @Param status query string false "Filter by status"
// @Param q query string false "Search in title and audio filename"
// @Param updated_after query string false "Filter by updated_at > timestamp (RFC3339)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/v1/transcription/list [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) ListTranscriptionJobs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	sortBy := c.Query("sort_by")
	sortOrder := c.Query("sort_order")
	searchQuery := c.Query("q")
	updatedAfterStr := c.Query("updated_after")

	var updatedAfter *time.Time
	if updatedAfterStr != "" {
		if t, err := time.Parse(time.RFC3339, updatedAfterStr); err == nil {
			updatedAfter = &t
		}
	}

	jobs, total, err := h.jobRepo.ListWithParams(c.Request.Context(), offset, limit, sortBy, sortOrder, searchQuery, updatedAfter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs": jobs,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// @Summary Get transcription job details
// @Description Get details of a specific transcription job
// @Tags transcription
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} models.TranscriptionJob
// @Failure 404 {object} map[string]string
// @Router /api/v1/transcription/{id} [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetTranscriptionJob(c *gin.Context) {
	id := c.Param("id")

	job, err := h.jobRepo.FindWithAssociations(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// @Summary Start transcription for uploaded file
// @Description Start transcription for an already uploaded audio file
// @Tags transcription
// @Accept json
// @Produce json
// @Param id path string true "Job ID"
// @Param parameters body models.WhisperXParams true "Transcription parameters"
// @Success 200 {object} models.TranscriptionJob
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/transcription/{id}/start [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) StartTranscription(c *gin.Context) {
	jobID := c.Param("id")

	job, err := h.getJobForTranscription(c, jobID)
	if err != nil {
		return
	}

	requestParams, err := h.getValidatedTranscriptionParams(c, job, jobID)
	if err != nil {
		return
	}

	// Update job with parameters
	job.Parameters = *requestParams
	job.Diarization = requestParams.Diarize
	job.Status = models.StatusPending

	// Clear previous results for re-transcription
	job.Transcript = nil
	job.Summary = nil
	job.ErrorMessage = nil

	// Save updated job
	if err := h.jobRepo.Update(c.Request.Context(), job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job"})
		return
	}

	// Enqueue job for transcription
	if err := h.taskQueue.EnqueueJob(jobID); err != nil {
		logger.Error("Failed to enqueue job", "job_id", jobID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue job"})
		return
	}

	// Log job started
	params := make(map[string]any)
	params["model"] = requestParams.Model
	params["model_family"] = requestParams.ModelFamily
	params["diarization"] = requestParams.Diarize
	if requestParams.Diarize && requestParams.DiarizeModel != "" {
		params["diarize_model"] = requestParams.DiarizeModel
	}
	params["language"] = requestParams.Language
	params["device"] = requestParams.Device

	filename := filepath.Base(job.AudioPath)
	logger.JobStarted(jobID, filename, requestParams.ModelFamily, params)

	c.JSON(http.StatusOK, job)
}

func (h *Handler) getJobForTranscription(c *gin.Context, jobID string) (*models.TranscriptionJob, error) {
	job, err := h.jobRepo.FindByID(c.Request.Context(), jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return nil, err
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return nil, err
	}

	// Allow transcription for uploaded, completed, and failed jobs (re-transcription)
	if job.Status != models.StatusUploaded && job.Status != models.StatusCompleted && job.Status != models.StatusFailed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot start transcription: job is currently processing or pending"})
		return nil, fmt.Errorf("invalid job status")
	}
	return job, nil
}

func (h *Handler) getValidatedTranscriptionParams(c *gin.Context, job *models.TranscriptionJob, jobID string) (*models.WhisperXParams, error) {
	// Set defaults
	requestParams := models.WhisperXParams{
		ModelFamily:                    "whisper", // Default to whisper for backward compatibility
		Model:                          "small",
		ModelCacheOnly:                 false,
		Device:                         "cpu",
		DeviceIndex:                    0,
		BatchSize:                      8,
		ComputeType:                    "float32",
		Threads:                        0,
		OutputFormat:                   "all",
		Verbose:                        true,
		Task:                           "transcribe",
		InterpolateMethod:              "nearest",
		NoAlign:                        false,
		ReturnCharAlignments:           false,
		VadMethod:                      "pyannote",
		VadOnset:                       0.5,
		VadOffset:                      0.363,
		ChunkSize:                      30,
		Diarize:                        false,
		DiarizeModel:                   "pyannote/speaker-diarization-3.1",
		SpeakerEmbeddings:              false,
		Temperature:                    0,
		BestOf:                         5,
		BeamSize:                       5,
		Patience:                       1.0,
		LengthPenalty:                  1.0,
		SuppressNumerals:               false,
		ConditionOnPreviousText:        false,
		Fp16:                           true,
		TemperatureIncrementOnFallback: 0.2,
		CompressionRatioThreshold:      2.4,
		LogprobThreshold:               -1.0,
		NoSpeechThreshold:              0.6,
		HighlightWords:                 false,
		SegmentResolution:              "sentence",
		PrintProgress:                  false,
		AttentionContextLeft:           256,
		AttentionContextRight:          256,
		IsMultiTrackEnabled:            false,
	}

	// Parse request body parameters, overriding defaults
	if err := c.ShouldBindJSON(&requestParams); err != nil {
		// Use defaults if JSON parsing fails
		logger.Debug("Failed to parse JSON parameters, using defaults", "error", err)
	}

	// Debug: log what we received
	logger.Debug("Parsed transcription parameters",
		"job_id", jobID,
		"model_family", requestParams.ModelFamily,
		"model", requestParams.Model,
		"diarization", requestParams.Diarize,
		"diarize_model", requestParams.DiarizeModel,
		"language", requestParams.Language)

	// Validate NVIDIA-specific constraints
	if requestParams.ModelFamily == "nvidia_parakeet" || requestParams.ModelFamily == "nvidia_canary" {
		// Both NVIDIA models support multiple European languages
		// No language restriction needed - models support auto-detection

		// NVIDIA models support diarization via Pyannote integration or NVIDIA Sortformer
		if requestParams.Diarize && requestParams.DiarizeModel == "pyannote" && (requestParams.HfToken == nil || *requestParams.HfToken == "") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Hugging Face token (hf_token) is required for Pyannote diarization"})
			return nil, fmt.Errorf("hf_token required")
		}
	}

	// Validate multi-track compatibility
	if job.IsMultiTrack && !requestParams.IsMultiTrackEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Multi-track audio requires multi-track transcription to be enabled in the parameters"})
		return nil, fmt.Errorf("multi-track mismatch")
	}

	if !job.IsMultiTrack && requestParams.IsMultiTrackEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Multi-track transcription cannot be used with single-track audio files"})
		return nil, fmt.Errorf("single-track mismatch")
	}

	// Multi-track transcription should automatically disable diarization
	if requestParams.IsMultiTrackEnabled && requestParams.Diarize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Diarization must be disabled when using multi-track transcription"})
		return nil, fmt.Errorf("diarization conflict")
	}

	return &requestParams, nil
}

// @Summary Kill running transcription job
// @Description Cancel a currently running transcription job
// @Tags transcription
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/v1/transcription/{id}/kill [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) KillJob(c *gin.Context) {
	jobID := c.Param("id")

	job, err := h.jobRepo.FindByID(c.Request.Context(), jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}

	// Check if job is currently processing
	if job.Status != models.StatusProcessing {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job is not currently running"})
		return
	}

	// Attempt to kill the job
	if err := h.taskQueue.KillJob(jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job cancellation requested"})
}

// UpdateTranscriptionTitle updates the title of a transcription job
// @Summary Update transcription title
// @Description Update the title of an audio file / transcription
// @Tags transcription
// @Accept json
// @Produce json
// @Param id path string true "Job ID"
// @Param request body map[string]string true "Title update request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/transcription/{id}/title [put]
// @Security ApiKeyAuth
// @Security BearerAuth
// @Security BearerAuth
func (h *Handler) UpdateTranscriptionTitle(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID required"})
		return
	}

	var body struct {
		Title string `json:"title" binding:"required,min=1,max=255"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := h.jobRepo.FindByID(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	job.Title = &body.Title
	if err := h.jobRepo.Update(c.Request.Context(), job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update title"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         job.ID,
		"title":      job.Title,
		"status":     job.Status,
		"created_at": job.CreatedAt,
		"audio_path": job.AudioPath,
	})
}

// @Summary Delete transcription job
// @Description Delete a transcription job and its associated files
// @Tags transcription
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/v1/transcription/{id} [delete]
// @Security ApiKeyAuth
// @Security BearerAuth
// @Summary Delete transcription job
// @Description Delete a transcription job and its associated files
// @Tags transcription
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/v1/transcription/{id} [delete]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) DeleteTranscriptionJob(c *gin.Context) {
	jobID := c.Param("id")

	job, err := h.jobRepo.FindByID(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Prevent deletion of jobs that are currently processing
	if job.Status == models.StatusProcessing {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete job that is currently processing"})
		return
	}

	// Delete files
	if job.IsMultiTrack && job.MultiTrackFolder != nil {
		_ = h.fileService.RemoveDirectory(*job.MultiTrackFolder)
	} else {
		_ = h.fileService.RemoveFile(job.AudioPath)
	}

	// Also remove .aup file if exists
	if job.AupFilePath != nil {
		_ = h.fileService.RemoveFile(*job.AupFilePath)
	}

	// Manually delete related records to handle legacy DBs without CASCADE constraints
	// 1. Delete Chat Sessions (and their messages via GORM hooks or manual if needed, but let's assume messages are cascaded by session deletion or we delete them too)
	// Actually, we should use the repositories if available, or direct DB calls if not exposed.
	// Since we have repositories, let's try to use them or add methods.
	// However, for speed and robustness here, we can use the jobRepo's DB instance if we had access, but we don't directly.
	// We should add DeleteByJobID methods to repositories or use a transaction.
	// Given the constraints, let's add a helper in jobRepo or just rely on the fact that we can't easily access other repos here without adding them to Handler if they aren't already.
	// Wait, Handler HAS all repos.

	ctx := c.Request.Context()

	// Delete Chat Sessions
	// We need a method in ChatRepository to delete by JobID or TranscriptionID
	if err := h.chatRepo.DeleteByJobID(ctx, jobID); err != nil {
		// Log error but continue? Or fail? Best to try to clean up as much as possible.
		fmt.Printf("Failed to delete chat sessions for job %s: %v\n", jobID, err)
	}

	// Delete Notes
	if err := h.noteRepo.DeleteByTranscriptionID(ctx, jobID); err != nil {
		fmt.Printf("Failed to delete notes for job %s: %v\n", jobID, err)
	}

	// Delete Summaries
	if err := h.summaryRepo.DeleteByTranscriptionID(ctx, jobID); err != nil {
		fmt.Printf("Failed to delete summaries for job %s: %v\n", jobID, err)
	}

	// Delete Speaker Mappings
	if err := h.speakerMappingRepo.DeleteByJobID(ctx, jobID); err != nil {
		fmt.Printf("Failed to delete speaker mappings for job %s: %v\n", jobID, err)
	}

	// Delete Job Executions
	if err := h.jobRepo.DeleteExecutionsByJobID(ctx, jobID); err != nil {
		fmt.Printf("Failed to delete job executions for job %s: %v\n", jobID, err)
	}

	// Delete MultiTrack Files (DB records)
	if err := h.jobRepo.DeleteMultiTrackFilesByJobID(ctx, jobID); err != nil {
		fmt.Printf("Failed to delete multi-track file records for job %s: %v\n", jobID, err)
	}

	// Delete from database
	if err := h.jobRepo.Delete(c.Request.Context(), jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete job: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
}

// @Summary Get transcription job execution data
// @Description Get execution parameters and timing for a transcription job
// @Tags transcription
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} models.TranscriptionJobExecution
// @Failure 404 {object} map[string]string
// @Router /api/v1/transcription/{id}/execution [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetJobExecutionData(c *gin.Context) {
	jobID := c.Param("id")

	// Get the transcription job to check if it's multi-track
	job, err := h.jobRepo.FindWithAssociations(c.Request.Context(), jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transcription job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transcription job"})
		return
	}

	execution, err := h.jobRepo.FindLatestCompletedExecution(c.Request.Context(), jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return graceful empty response instead of 404
			c.JSON(http.StatusOK, gin.H{
				"transcription_job_id": jobID,
				"available":            false,
				"message":              "No execution data available for this job",
				"is_multi_track":       job.IsMultiTrack,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get execution data"})
		return
	}

	// Create enhanced response with multi-track data
	response := gin.H{
		"id":                   execution.ID,
		"transcription_job_id": execution.TranscriptionJobID,
		"started_at":           execution.StartedAt,
		"completed_at":         execution.CompletedAt,
		"processing_duration":  execution.ProcessingDuration,
		"actual_parameters":    execution.ActualParameters,
		"status":               execution.Status,
		"error_message":        execution.ErrorMessage,
		"created_at":           execution.CreatedAt,
		"updated_at":           execution.UpdatedAt,
		"is_multi_track":       job.IsMultiTrack,
	}

	// Add multi-track specific data if available
	if job.IsMultiTrack && execution.MultiTrackTimings != nil {
		// Deserialize track timings
		var trackTimings []models.MultiTrackTiming
		if err := json.Unmarshal([]byte(*execution.MultiTrackTimings), &trackTimings); err == nil {
			response["multi_track_timings"] = trackTimings
		}

		// Add merge timing data
		response["merge_start_time"] = execution.MergeStartTime
		response["merge_end_time"] = execution.MergeEndTime
		response["merge_duration"] = execution.MergeDuration

		// Add multi-track files information
		response["multi_track_files"] = job.MultiTrackFiles
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Get audio file
// @Description Serve the audio file for a transcription job
// @Tags transcription
// @Produce audio/mpeg,audio/wav,audio/mp4
// @Param id path string true "Job ID"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /api/v1/transcription/{id}/audio [get]
// @Security ApiKeyAuth
func (h *Handler) GetAudioFile(c *gin.Context) {
	jobID := c.Param("id")

	job, err := h.jobRepo.FindByID(c.Request.Context(), jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}

	// Debug logging
	fmt.Printf("DEBUG: GetAudioFile for job %s\n", jobID)
	fmt.Printf("DEBUG: Job status: %s\n", job.Status)
	fmt.Printf("DEBUG: Audio path: '%s'\n", job.AudioPath)

	// For multi-track jobs, prefer merged audio if available
	audioPath := job.AudioPath
	if job.IsMultiTrack && job.MergedAudioPath != nil && *job.MergedAudioPath != "" {
		// Check if merged audio file exists
		if _, err := os.Stat(*job.MergedAudioPath); err == nil {
			audioPath = *job.MergedAudioPath
			fmt.Printf("DEBUG: Using merged audio: %s\n", audioPath)
		} else {
			fmt.Printf("DEBUG: Merged audio not found, falling back to original: %s\n", job.AudioPath)
		}
	}

	// Check if audio file exists
	if audioPath == "" {
		fmt.Printf("DEBUG: Audio path is empty\n")
		c.JSON(http.StatusNotFound, gin.H{"error": "Audio file path not found"})
		return
	}

	// Check if file exists on filesystem
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		fmt.Printf("DEBUG: Audio file does not exist on disk: %s\n", audioPath)
		c.JSON(http.StatusNotFound, gin.H{"error": "Audio file not found on disk"})
		return
	}

	fmt.Printf("DEBUG: Audio file exists, serving: %s\n", audioPath)

	// Set appropriate content type based on file extension
	ext := filepath.Ext(job.AudioPath)
	switch ext {
	case ".mp3":
		c.Header("Content-Type", "audio/mpeg")
	case ".wav":
		c.Header("Content-Type", "audio/wav")
	case ".m4a":
		c.Header("Content-Type", "audio/mp4")
	case ".ogg":
		c.Header("Content-Type", "audio/ogg")
	default:
		c.Header("Content-Type", "audio/mpeg")
	}

	// Add CORS headers for audio visualization and streaming
	origin := c.Request.Header.Get("Origin")
	allowOrigin := "*"
	if h.config.IsProduction() && len(h.config.AllowedOrigins) > 0 {
		// In production, validate against configured origins
		allowOrigin = ""
		for _, allowed := range h.config.AllowedOrigins {
			if origin == allowed {
				allowOrigin = origin
				break
			}
		}
	} else if origin != "" {
		// In development, echo back the origin for credentials support
		allowOrigin = origin
	}

	if allowOrigin != "" {
		c.Header("Access-Control-Allow-Origin", allowOrigin)
		c.Header("Access-Control-Allow-Credentials", "true")
	}
	c.Header("Access-Control-Expose-Headers", "Content-Range, Accept-Ranges, Content-Length")
	c.Header("Accept-Ranges", "bytes")

	// Open the file
	file, err := os.Open(audioPath)
	if err != nil {
		fmt.Printf("ERROR: Failed to open audio file: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open audio file"})
		return
	}
	defer file.Close()

	// Get file stats
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Printf("ERROR: Failed to stat audio file: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stat audio file"})
		return
	}

	// Use http.ServeContent for efficient streaming and range request support
	http.ServeContent(c.Writer, c.Request, filepath.Base(audioPath), fileInfo.ModTime(), file)
}

// @Summary Login
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "User credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user, err := h.userRepo.FindByUsername(c.Request.Context(), req.Username)
	if err != nil {
		logger.AuthEvent("login", req.Username, c.ClientIP(), false, "user_not_found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !auth.CheckPassword(req.Password, user.Password) {
		logger.AuthEvent("login", req.Username, c.ClientIP(), false, "invalid_password")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := h.authService.GenerateToken(user)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Set refresh token cookie
	if err := h.issueRefreshToken(c, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Set access token cookie for streaming/media access
	// Use Lax mode because Strict mode blocks <audio>/<video> subresource requests on mobile browsers.
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "scriberr_access_token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour), // Match your token duration constant
		HttpOnly: true,
		Secure:   h.config.SecureCookies, // Use explicit secure flag
		SameSite: http.SameSiteLaxMode,
	})

	response := LoginResponse{Token: token}
	response.User.ID = user.ID
	response.User.Username = user.Username

	logger.AuthEvent("login", req.Username, c.ClientIP(), true)
	c.JSON(http.StatusOK, response)
}

// @Summary Logout user
// @Description Logout user and invalidate token (client-side action)
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	// Best-effort refresh token revocation and cookie clear
	if cookie, err := c.Cookie("scriberr_refresh_token"); err == nil {
		h.revokeRefreshToken(c, cookie)

	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "scriberr_refresh_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.config.SecureCookies,
	})
	// Also clear access token
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "scriberr_access_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.config.SecureCookies,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// @Summary Check registration status
// @Description Check if the application requires initial user registration
// @Tags auth
// @Produce json
// @Success 200 {object} RegistrationStatusResponse
// @Router /api/v1/auth/registration-status [get]
func (h *Handler) GetRegistrationStatus(c *gin.Context) {
	userCount, err := h.userRepo.Count(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check registration status"})
		return
	}

	response := RegistrationStatusResponse{
		RegistrationEnabled: userCount == 0,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Register initial admin user
// @Description Register the initial admin user (only allowed when no users exist)
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	// Check if any users already exist
	userCount, err := h.userRepo.Count(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing users"})
		return
	}

	if userCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Registration is not allowed. Admin user already exists"})
		return
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate password confirmation
	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Passwords do not match"})
		return
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to secure password"})
		return
	}

	// Create user
	user := models.User{
		Username: req.Username,
		Password: hashedPassword,
	}

	if err := h.userRepo.Create(c.Request.Context(), &user); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate token for immediate login
	token, err := h.authService.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate login token"})
		return
	}
	// Set refresh token cookie
	if err := h.issueRefreshToken(c, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}
	response := LoginResponse{Token: token}
	response.User.ID = user.ID
	response.User.Username = user.Username

	c.JSON(http.StatusCreated, response)
}

// RefreshTokenResponse represents the refresh response
type RefreshTokenResponse struct {
	Token string `json:"token"`
}

// @Summary Refresh access token
// @Description Rotate refresh token and return new access token
// @Tags auth
// @Produce json
// @Success 200 {object} RefreshTokenResponse
// @Failure 401 {object} map[string]string
// @Router /api/v1/auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	cookie, err := c.Cookie("scriberr_refresh_token")
	if err != nil || cookie == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing refresh token"})
		return
	}
	userID, err := h.validateAndRotateRefreshToken(c, cookie)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}
	user, err := h.userRepo.FindByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	token, err := h.authService.GenerateToken(user)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Set access token cookie for streaming/media access
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "scriberr_access_token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   h.config.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	c.JSON(http.StatusOK, RefreshTokenResponse{Token: token})
}

// issueRefreshToken creates a refresh token and sets cookie
func (h *Handler) issueRefreshToken(c *gin.Context, userID uint) error {
	tokenValue := generateSecureAPIKey(64)
	hashed := sha256Hex(tokenValue)
	rt := models.RefreshToken{
		UserID:    userID,
		Hashed:    hashed,
		ExpiresAt: time.Now().Add(14 * 24 * time.Hour),
		Revoked:   false,
	}
	if err := h.refreshTokenRepo.Create(c.Request.Context(), &rt); err != nil {
		return err
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "scriberr_refresh_token",
		Value:    tokenValue,
		Path:     "/",
		Expires:  rt.ExpiresAt,
		MaxAge:   int((14 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.config.SecureCookies,
	})
	return nil
}

// validateAndRotateRefreshToken validates refresh token, revokes old, and issues new
func (h *Handler) validateAndRotateRefreshToken(c *gin.Context, tokenValue string) (uint, error) {
	hashed := sha256Hex(tokenValue)
	rt, err := h.refreshTokenRepo.FindByHash(c.Request.Context(), hashed)
	if err != nil {
		return 0, err
	}
	if rt.Revoked || time.Now().After(rt.ExpiresAt) {
		return 0, fmt.Errorf("expired or revoked")
	}
	// Revoke current
	_ = h.refreshTokenRepo.Revoke(c.Request.Context(), rt.ID)
	// Issue new
	if err := h.issueRefreshToken(c, rt.UserID); err != nil {
		return 0, err
	}
	return rt.UserID, nil
}

func (h *Handler) revokeRefreshToken(c *gin.Context, tokenValue string) {
	hashed := sha256Hex(tokenValue)
	_ = h.refreshTokenRepo.RevokeByHash(c.Request.Context(), hashed)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// @Summary Change user password
// @Description Change the current user's password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ChangePasswordRequest true "Password change details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/auth/change-password [post]
func (h *Handler) ChangePassword(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate password confirmation
	if req.NewPassword != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New passwords do not match"})
		return
	}

	// Use UserService to change password
	if err := h.userService.ChangePassword(c.Request.Context(), userID.(uint), req.CurrentPassword, req.NewPassword); err != nil {
		if err.Error() == "incorrect password" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// @Summary Change username
// @Description Change the current user's username
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ChangeUsernameRequest true "Username change details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/auth/change-username [post]
func (h *Handler) ChangeUsername(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req ChangeUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	// Use UserService to change username
	if err := h.userService.ChangeUsername(c.Request.Context(), userID.(uint), req.NewUsername, req.Password); err != nil {
		if err.Error() == "incorrect password" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password is incorrect"})
			return
		}
		if err.Error() == "username already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update username"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Username changed successfully"})
}

// @Summary List API keys
// @Description Get all API keys for the current user (without exposing the actual keys)
// @Tags api-keys
// @Produce json
// @Success 200 {object} APIKeysWrapper
// @Security BearerAuth
// @Router /api/v1/api-keys [get]
func (h *Handler) ListAPIKeys(c *gin.Context) {
	apiKeys, err := h.apiKeyRepo.ListActive(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch API keys"})
		return
	}

	// Transform API keys to list response format
	var responseKeys []APIKeyListResponse
	for _, apiKey := range apiKeys {
		responseKeys = append(responseKeys, transformAPIKeyForList(apiKey))
	}

	c.JSON(http.StatusOK, APIKeysWrapper{APIKeys: responseKeys})
}

// @Summary Create API key
// @Description Create a new API key for external API access
// @Tags api-keys
// @Accept json
// @Produce json
// @Param request body CreateAPIKeyRequest true "API key creation details"
// @Success 201 {object} CreateAPIKeyResponse
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/api-keys [post]
func (h *Handler) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Generate a secure API key
	apiKey := generateSecureAPIKey(32)

	// Create the API key record
	newKey := models.APIKey{
		Key:         apiKey,
		Name:        req.Name,
		Description: &req.Description,
		IsActive:    true,
	}

	if err := h.apiKeyRepo.Create(c.Request.Context(), &newKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	// Return full model with 200 to match tests
	c.JSON(http.StatusOK, newKey)
}

// @Summary Delete API key
// @Description Delete an API key
// @Tags api-keys
// @Param id path int true "API Key ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/api-keys/{id} [delete]
func (h *Handler) DeleteAPIKey(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	// Check if the API key exists
	_, err = h.apiKeyRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	// Delete the API key (soft delete by setting is_active to false)
	if err := h.apiKeyRepo.Revoke(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted successfully"})
}

// @Summary Get LLM configuration
// @Description Get the current active LLM configuration
// @Tags llm
// @Produce json
// @Success 200 {object} LLMConfigResponse
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/llm/config [get]
func (h *Handler) GetLLMConfig(c *gin.Context) {
	config, err := h.llmConfigRepo.GetActive(c.Request.Context())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "No active LLM configuration found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch LLM configuration"})
		return
	}

	response := LLMConfigResponse{
		ID:            config.ID,
		Provider:      config.Provider,
		BaseURL:       config.BaseURL,
		OpenAIBaseURL: config.OpenAIBaseURL,
		HasAPIKey:     config.APIKey != nil && *config.APIKey != "",
		IsActive:      config.IsActive,
		CreatedAt:     config.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     config.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Create or update LLM configuration
// @Description Create or update LLM configuration settings
// @Tags llm
// @Accept json
// @Produce json
// @Param request body LLMConfigRequest true "LLM configuration details"
// @Success 200 {object} LLMConfigResponse
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/llm/config [post]
func (h *Handler) SaveLLMConfig(c *gin.Context) {
	var req LLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate provider-specific requirements
	if req.Provider == "ollama" && (req.BaseURL == nil || *req.BaseURL == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Base URL is required for Ollama provider"})
		return
	}

	// Check if there's an existing active configuration
	existingConfig, err := h.llmConfigRepo.GetActive(c.Request.Context())
	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing configuration"})
		return
	}

	// Handle API Key logic for OpenAI
	var apiKeyToSave *string
	if req.Provider == "openai" {
		if req.APIKey != nil && *req.APIKey != "" {
			// New key provided
			apiKeyToSave = req.APIKey
		} else if existingConfig != nil && existingConfig.APIKey != nil && *existingConfig.APIKey != "" {
			// Reuse existing key
			apiKeyToSave = existingConfig.APIKey
		} else {
			// No key provided and no existing key
			c.JSON(http.StatusBadRequest, gin.H{"error": "API key is required for OpenAI provider"})
			return
		}
	}

	var config *models.LLMConfig

	if err == gorm.ErrRecordNotFound {
		// No existing active config, create new one
		config = &models.LLMConfig{
			Provider:      req.Provider,
			BaseURL:       req.BaseURL,
			OpenAIBaseURL: req.OpenAIBaseURL,
			APIKey:        apiKeyToSave,
			IsActive:      req.IsActive,
		}

		if err := h.llmConfigRepo.Create(c.Request.Context(), config); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create LLM configuration"})
			return
		}
	} else {
		// Update existing config
		existingConfig.Provider = req.Provider
		existingConfig.BaseURL = req.BaseURL
		existingConfig.OpenAIBaseURL = req.OpenAIBaseURL
		existingConfig.APIKey = apiKeyToSave
		existingConfig.IsActive = req.IsActive

		if err := h.llmConfigRepo.Update(c.Request.Context(), existingConfig); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update LLM configuration"})
			return
		}
		config = existingConfig
	}

	response := LLMConfigResponse{
		ID:            config.ID,
		Provider:      config.Provider,
		BaseURL:       config.BaseURL,
		OpenAIBaseURL: config.OpenAIBaseURL,
		HasAPIKey:     config.APIKey != nil && *config.APIKey != "",
		IsActive:      config.IsActive,
		CreatedAt:     config.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     config.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	c.JSON(http.StatusOK, response)
}

// generateSecureAPIKey generates a cryptographically secure API key
func generateSecureAPIKey(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to a UUID if crypto/rand fails
		return uuid.New().String()
	}

	for i := range b {
		b[i] = charset[randomBytes[i]%byte(len(charset))]
	}
	return string(b)
}

// @Summary Get queue statistics
// @Description Get current queue statistics
// @Tags admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/queue/stats [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetQueueStats(c *gin.Context) {
	stats := h.taskQueue.GetQueueStats()
	c.JSON(http.StatusOK, stats)
}

// @Summary Get supported models
// @Description Get list of supported WhisperX models
// @Tags transcription
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/transcription/models [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetSupportedModels(c *gin.Context) {
	models := h.unifiedProcessor.GetSupportedModels()
	languages := h.unifiedProcessor.GetSupportedLanguages()

	c.JSON(http.StatusOK, gin.H{
		"models":    models,
		"languages": languages,
	})
}

// Health check endpoint
// @Summary Health check
// @Description Check if the API is healthy
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"version": "1.0.0",
	})
}

// Helper functions
func getFormValueWithDefault(c *gin.Context, key, defaultValue string) string {
	if value := c.PostForm(key); value != "" {
		return value
	}
	return defaultValue
}

func getFormIntWithDefault(c *gin.Context, key string, defaultValue int) int {
	if value := c.PostForm(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getFormFloatWithDefault(c *gin.Context, key string, defaultValue float64) float64 {
	if value := c.PostForm(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getFormBoolWithDefault(c *gin.Context, key string, defaultValue bool) bool {
	if value := c.PostForm(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// Profile API Handlers

// @Summary List transcription profiles
// @Description Get list of all transcription profiles
// @Tags profiles
// @Produce json
// @Success 200 {array} models.TranscriptionProfile
// @Router /api/v1/profiles [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) ListProfiles(c *gin.Context) {
	// TODO: Add pagination support to API if needed. For now, list all (limit 1000)
	profiles, _, err := h.profileRepo.List(c.Request.Context(), 0, 1000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch profiles"})
		return
	}
	c.JSON(http.StatusOK, profiles)
}

// @Summary Create transcription profile
// @Description Create a new transcription profile
// @Tags profiles
// @Accept json
// @Produce json
// @Param profile body models.TranscriptionProfile true "Profile data"
// @Success 201 {object} models.TranscriptionProfile
// @Failure 400 {object} map[string]string
// @Router /api/v1/profiles [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) CreateProfile(c *gin.Context) {
	var profile models.TranscriptionProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate required fields
	if profile.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Profile name is required"})
		return
	}

	// Check if profile name already exists
	// TODO: Add FindByName to ProfileRepository if needed, or rely on unique constraint error
	// For now, we'll skip explicit check or implement it in repository.
	// Assuming unique constraint on Name in DB or we can check via List.

	if err := h.profileRepo.Create(c.Request.Context(), &profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create profile"})
		return
	}

	// Tests expect 200 on create
	c.JSON(http.StatusOK, profile)
}

// @Summary Get transcription profile
// @Description Get a transcription profile by ID
// @Tags profiles
// @Produce json
// @Param id path string true "Profile ID"
// @Success 200 {object} models.TranscriptionProfile
// @Failure 404 {object} map[string]string
// @Router /api/v1/profiles/{id} [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetProfile(c *gin.Context) {
	profileID := c.Param("id")

	profile, err := h.profileRepo.FindByID(c.Request.Context(), profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// @Summary Update transcription profile
// @Description Update a transcription profile
// @Tags profiles
// @Accept json
// @Produce json
// @Param id path string true "Profile ID"
// @Param profile body models.TranscriptionProfile true "Updated profile data"
// @Success 200 {object} models.TranscriptionProfile
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/profiles/{id} [put]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) UpdateProfile(c *gin.Context) {
	profileID := c.Param("id")

	existingProfile, err := h.profileRepo.FindByID(c.Request.Context(), profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}

	var updatedProfile models.TranscriptionProfile
	if err := c.ShouldBindJSON(&updatedProfile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate required fields
	if updatedProfile.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Profile name is required"})
		return
	}

	// Check if profile name already exists (excluding current profile)
	// TODO: Add check to repository

	// Update the profile
	// We need to preserve ID and CreatedAt, and update other fields
	// GORM Save updates all fields.
	updatedProfile.ID = existingProfile.ID
	updatedProfile.CreatedAt = existingProfile.CreatedAt

	if err := h.profileRepo.Update(c.Request.Context(), &updatedProfile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, updatedProfile)
}

// @Summary Delete transcription profile
// @Description Delete a transcription profile
// @Tags profiles
// @Produce json
// @Param id path string true "Profile ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/profiles/{id} [delete]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) DeleteProfile(c *gin.Context) {
	profileID := c.Param("id")

	_, err := h.profileRepo.FindByID(c.Request.Context(), profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}

	if err := h.profileRepo.Delete(c.Request.Context(), profileID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile deleted successfully"})
}

// SetDefaultProfile sets a profile as the default profile
// @Summary Set default transcription profile
// @Description Mark the specified profile as the default profile
// @Tags profiles
// @Produce json
// @Param id path string true "Profile ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Security BearerAuth
// @Router /api/v1/profiles/{id}/set-default [post]
func (h *Handler) SetDefaultProfile(c *gin.Context) {
	profileID := c.Param("id")
	if profileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Profile ID is required"})
		return
	}

	// Find the profile
	profile, err := h.profileRepo.FindByID(c.Request.Context(), profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}

	// Set this profile as default (the BeforeSave hook will handle unsetting other defaults)
	profile.IsDefault = true
	if err := h.profileRepo.Update(c.Request.Context(), profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default profile set successfully", "profile": profile})
}

// QuickTranscriptionRequest represents the quick transcription request
type QuickTranscriptionRequest struct {
	Parameters  *models.WhisperXParams `json:"parameters,omitempty"`
	ProfileName *string                `json:"profile_name,omitempty"`
}

// @Summary Submit quick transcription job
// @Description Submit an audio file for temporary transcription (data discarded after 6 hours)
// @Tags transcription
// @Accept multipart/form-data
// @Produce json
// @Param audio formData file true "Audio file"
// @Param parameters formData string false "JSON string of transcription parameters"
// @Param profile_name formData string false "Profile name to use for transcription"
// @Success 200 {object} transcription.QuickTranscriptionJob
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/transcription/quick [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) SubmitQuickTranscription(c *gin.Context) {
	// Parse multipart form
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Audio file is required"})
		return
	}
	defer file.Close()

	var params models.WhisperXParams

	// Check if profile_name was provided
	if profileName := c.PostForm("profile_name"); profileName != "" {
		// Load parameters from profile
		profile, err := h.profileRepo.FindByName(c.Request.Context(), profileName)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Profile '%s' not found", profileName)})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load profile"})
			return
		}
		params = profile.Parameters

	} else if parametersJSON := c.PostForm("parameters"); parametersJSON != "" {
		// Parse parameters from JSON string
		if err := json.Unmarshal([]byte(parametersJSON), &params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameters JSON"})
			return
		}
	} else {
		// Use default parameters with all required fields
		params = models.WhisperXParams{
			// Model parameters
			Model:          "small",
			ModelCacheOnly: false,

			// Device and computation
			Device:      "cpu",
			DeviceIndex: 0,
			BatchSize:   8,
			ComputeType: "float32",
			Threads:     0,

			// Output settings
			OutputFormat: "all",
			Verbose:      true,

			// Task and language
			Task: "transcribe",

			// Alignment settings
			InterpolateMethod:    "nearest",
			NoAlign:              false,
			ReturnCharAlignments: false,

			// VAD (Voice Activity Detection) settings
			VadMethod: "pyannote",
			VadOnset:  0.5,
			VadOffset: 0.363,
			ChunkSize: 30,

			// Diarization settings
			Diarize:           false,
			DiarizeModel:      "pyannote/speaker-diarization-3.1",
			SpeakerEmbeddings: false,

			// Transcription quality settings
			Temperature:                    0,
			BestOf:                         5,
			BeamSize:                       5,
			Patience:                       1.0,
			LengthPenalty:                  1.0,
			SuppressNumerals:               false,
			ConditionOnPreviousText:        false,
			Fp16:                           true,
			TemperatureIncrementOnFallback: 0.2,
			CompressionRatioThreshold:      2.4,
			LogprobThreshold:               -1.0,
			NoSpeechThreshold:              0.6,

			// Output formatting
			HighlightWords:    false,
			SegmentResolution: "sentence",
			PrintProgress:     false,
		}
	}

	// Submit quick transcription job
	job, err := h.quickTranscription.SubmitQuickJob(file, header.Filename, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to submit quick transcription: %v", err)})
		return
	}

	c.JSON(http.StatusOK, job)
}

// @Summary Get quick transcription status
// @Description Get the current status of a quick transcription job
// @Tags transcription
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} transcription.QuickTranscriptionJob
// @Failure 404 {object} map[string]string
// @Router /api/v1/transcription/quick/{id} [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) GetQuickTranscriptionStatus(c *gin.Context) {
	jobID := c.Param("id")

	job, err := h.quickTranscription.GetQuickJob(jobID)
	if err != nil {
		if err.Error() == "job not found" || err.Error() == "job expired" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job status"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// @Summary Download audio from YouTube URL
// @Description Download audio from a YouTube video URL and prepare it for transcription
// @Tags transcription
// @Accept json
// @Produce json
// @Param request body YouTubeDownloadRequest true "YouTube download request"
// @Success 200 {object} models.TranscriptionJob
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/transcription/youtube [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (h *Handler) DownloadFromYouTube(c *gin.Context) {
	var req YouTubeDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate YouTube URL
	if !strings.Contains(req.URL, "youtube.com") && !strings.Contains(req.URL, "youtu.be") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid YouTube URL"})
		return
	}

	// Create upload directory
	uploadDir := h.config.UploadDir
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// Generate unique job ID and filename
	jobID := uuid.New().String()
	filename := fmt.Sprintf("%s.%%(ext)s", jobID)
	filePath := filepath.Join(uploadDir, filename)

	// Get video title if not provided
	var title string
	if req.Title != nil && *req.Title != "" {
		title = *req.Title
	} else {
		// Get title first using standalone yt-dlp
		titleStart := time.Now()
		cmd := exec.Command("yt-dlp", "--get-title", req.URL)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			title = "YouTube Audio"
			logger.Warn("Failed to get YouTube title", "url", req.URL, "error", err.Error(), "duration", time.Since(titleStart))
		} else {
			title = strings.TrimSpace(out.String())
			logger.Info("YouTube title retrieved", "title", title, "duration", time.Since(titleStart))
		}
	}

	// Download audio using yt-dlp in Python environment
	logger.Info("Starting YouTube download", "url", req.URL, "job_id", jobID)
	downloadStart := time.Now()

	// Executing yt-dlp directly (standalone binary)
	ytDlpCmd := exec.Command("yt-dlp",
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", "0", // best quality
		"--output", filePath,
		"--no-playlist",
		req.URL,
	)

	// Execute download and capture stderr for better error messages
	var stderr bytes.Buffer
	ytDlpCmd.Stderr = &stderr

	if err := ytDlpCmd.Run(); err != nil {
		stderrOutput := stderr.String()
		logger.Error("YouTube download failed",
			"url", req.URL,
			"job_id", jobID,
			"error", err.Error(),
			"stderr", stderrOutput,
			"duration", time.Since(downloadStart))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   fmt.Sprintf("Failed to download YouTube audio: %v", err),
			"details": stderrOutput,
		})
		return
	}

	// Find the actual downloaded file (yt-dlp changes the extension)
	pattern := fmt.Sprintf("%s.*", jobID)
	matches, err := filepath.Glob(filepath.Join(uploadDir, pattern))
	if err != nil || len(matches) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Downloaded file not found"})
		return
	}

	actualFilePath := matches[0]

	// Get file size for performance logging
	fileInfo, err := os.Stat(actualFilePath)
	if err == nil {
		fileSizeMB := float64(fileInfo.Size()) / 1024 / 1024
		logger.Info("YouTube download completed",
			"url", req.URL,
			"job_id", jobID,
			"file_path", actualFilePath,
			"file_size_mb", fmt.Sprintf("%.2f", fileSizeMB),
			"duration", time.Since(downloadStart))
	}

	// Create transcription record
	job := models.TranscriptionJob{
		ID:        jobID,
		AudioPath: actualFilePath,
		Status:    models.StatusUploaded,
	}

	// Set title
	if title != "" {
		job.Title = &title
	}

	// Save to database
	if err := h.jobRepo.Create(c.Request.Context(), &job); err != nil {
		// Clean up downloaded file on database error
		os.Remove(actualFilePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transcription record"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// @Summary Get user's default profile
// @Description Get the default transcription profile for the current user
// @Tags profiles
// @Produce json
// @Success 200 {object} models.TranscriptionProfile
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/user/default-profile [get]
func (h *Handler) GetUserDefaultProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get user with default profile ID
	user, err := h.userRepo.FindByID(c.Request.Context(), userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// If user has no default profile set, return the first available profile or no profile
	if user.DefaultProfileID == nil {
		// Try to find a default profile from profiles table
		profile, err := h.profileRepo.FindDefault(c.Request.Context())
		if err == nil {
			c.JSON(http.StatusOK, profile)
			return
		}

		// If no default marked, get first one
		profiles, _, err := h.profileRepo.List(c.Request.Context(), 0, 1)
		if err != nil || len(profiles) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No profiles available"})
			return
		}
		c.JSON(http.StatusOK, profiles[0])
		return
	}

	// Get the user's default profile
	profile, err := h.profileRepo.FindByID(c.Request.Context(), *user.DefaultProfileID)
	if err != nil {
		// Default profile no longer exists, fall back to first available
		profiles, _, err := h.profileRepo.List(c.Request.Context(), 0, 1)
		if err != nil || len(profiles) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No profiles available"})
			return
		}
		c.JSON(http.StatusOK, profiles[0])
		return
	}

	c.JSON(http.StatusOK, profile)
}

// SetUserDefaultProfileRequest represents the request to set user's default profile
type SetUserDefaultProfileRequest struct {
	ProfileID string `json:"profile_id" binding:"required"`
}

// @Summary Set user's default profile
// @Description Set the default transcription profile for the current user
// @Tags profiles
// @Accept json
// @Produce json
// @Param request body SetUserDefaultProfileRequest true "Default profile request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/user/default-profile [post]
func (h *Handler) SetUserDefaultProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req SetUserDefaultProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Verify the profile exists
	_, err := h.profileRepo.FindByID(c.Request.Context(), req.ProfileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}

	// Get user
	user, err := h.userRepo.FindByID(c.Request.Context(), userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Update user's default profile
	user.DefaultProfileID = &req.ProfileID
	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default profile set successfully", "profile_id": req.ProfileID})
}

// UserSettingsResponse represents the user's settings
type UserSettingsResponse struct {
	AutoTranscriptionEnabled bool    `json:"auto_transcription_enabled"`
	DefaultProfileID         *string `json:"default_profile_id,omitempty"`
}

// UpdateUserSettingsRequest represents the request to update user settings
type UpdateUserSettingsRequest struct {
	AutoTranscriptionEnabled *bool `json:"auto_transcription_enabled,omitempty"`
}

// @Summary Get user settings
// @Description Get the current user's settings including auto-transcription preference
// @Tags user
// @Produce json
// @Success 200 {object} UserSettingsResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/user/settings [get]
func (h *Handler) GetUserSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	response := UserSettingsResponse{
		AutoTranscriptionEnabled: user.AutoTranscriptionEnabled,
		DefaultProfileID:         user.DefaultProfileID,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Update user settings
// @Description Update the current user's settings
// @Tags user
// @Accept json
// @Produce json
// @Param request body UpdateUserSettingsRequest true "Settings update request"
// @Success 200 {object} UserSettingsResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/user/settings [put]
func (h *Handler) UpdateUserSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req UpdateUserSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Update fields if provided
	if req.AutoTranscriptionEnabled != nil {
		user.AutoTranscriptionEnabled = *req.AutoTranscriptionEnabled
	}

	// Save updated user
	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
		return
	}

	response := UserSettingsResponse{
		AutoTranscriptionEnabled: user.AutoTranscriptionEnabled,
		DefaultProfileID:         user.DefaultProfileID,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary SSE Events
// @Description Subscribe to server-sent events
// @Tags events
// @Produce text/event-stream
// @Success 200 {string} string "stream"
// @Router /api/v1/events [get]
func (h *Handler) Events(c *gin.Context) {
	h.broadcaster.ServeHTTP(c.Writer, c.Request)
}
