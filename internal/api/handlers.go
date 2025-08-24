package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"scriberr/internal/auth"
	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/queue"
	"scriberr/internal/transcription"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Handler contains all the API handlers
type Handler struct {
	config          *config.Config
	authService     *auth.AuthService
	taskQueue       *queue.TaskQueue
	whisperXService *transcription.WhisperXService
}

// NewHandler creates a new handler
func NewHandler(cfg *config.Config, authService *auth.AuthService, taskQueue *queue.TaskQueue, whisperXService *transcription.WhisperXService) *Handler {
	return &Handler{
		config:          cfg,
		authService:     authService,
		taskQueue:       taskQueue,
		whisperXService: whisperXService,
	}
}

// SubmitJobRequest represents the submit job request
type SubmitJobRequest struct {
	Title       *string                   `json:"title,omitempty"`
	Diarization bool                      `json:"diarization"`
	Parameters  models.WhisperXParams     `json:"parameters"`
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
	RequiresRegistration bool `json:"requiresRegistration"`
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
func (h *Handler) UploadAudio(c *gin.Context) {
	// Parse multipart form
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Audio file is required"})
		return
	}
	defer file.Close()

	// Create upload directory
	uploadDir := h.config.UploadDir
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// Generate unique filename
	jobID := uuid.New().String()
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s%s", jobID, ext)
	filePath := filepath.Join(uploadDir, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Create job record with "uploaded" status (not queued for transcription)
	job := models.TranscriptionJob{
		ID:        jobID,
		AudioPath: filePath,
		Status:    models.StatusUploaded, // New status for uploaded but not transcribed
	}

	if title := c.PostForm("title"); title != "" {
		job.Title = &title
	}

	// Save to database
	if err := database.DB.Create(&job).Error; err != nil {
		os.Remove(filePath) // Clean up file
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}

	c.JSON(http.StatusOK, job)
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
func (h *Handler) SubmitJob(c *gin.Context) {
	// Parse multipart form
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Audio file is required"})
		return
	}
	defer file.Close()

	// Create upload directory
	uploadDir := h.config.UploadDir
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// Generate unique filename
	jobID := uuid.New().String()
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s%s", jobID, ext)
	filePath := filepath.Join(uploadDir, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Parse parameters
	diarize := getFormBoolWithDefault(c, "diarize", false)
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

	// Create job
	job := models.TranscriptionJob{
		ID:          jobID,
		AudioPath:   filePath,
		Status:      models.StatusPending,
		Diarization: diarize,
		Parameters:  params,
	}

	if title := c.PostForm("title"); title != "" {
		job.Title = &title
	}

	// Save to database
	if err := database.DB.Create(&job).Error; err != nil {
		os.Remove(filePath) // Clean up file
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
func (h *Handler) GetTranscript(c *gin.Context) {
	jobID := c.Param("id")
	
	var job models.TranscriptionJob
	if err := database.DB.Where("id = ?", jobID).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}

	if job.Status != models.StatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Job not completed, current status: %s", job.Status),
		})
		return
	}

	if job.Transcript == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transcript not available"})
		return
	}

	var transcript interface{}
	if err := json.Unmarshal([]byte(*job.Transcript), &transcript); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse transcript"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id": job.ID,
		"title": job.Title,
		"transcript": transcript,
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
// @Param status query string false "Filter by status"
// @Param q query string false "Search in title and audio filename"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/transcription/list [get]
// @Security ApiKeyAuth
func (h *Handler) ListJobs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")
	search := c.Query("q") // Add search parameter
	
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 1000 {
		limit = 10
	}

	offset := (page - 1) * limit

	query := database.DB.Model(&models.TranscriptionJob{})
	
	// Apply status filter
	if status != "" {
		query = query.Where("status = ?", status)
	}
	
	// Apply search filter - search in title and audio_path
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("title LIKE ? COLLATE NOCASE OR audio_path LIKE ? COLLATE NOCASE", searchPattern, searchPattern)
	}

	var jobs []models.TranscriptionJob
	var total int64
	
	// Count total matching records
	query.Count(&total)
	
	// Apply pagination and ordering
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&jobs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs": jobs,
		"pagination": gin.H{
			"page":   page,
			"limit":  limit,
			"total":  total,
			"pages":  (total + int64(limit) - 1) / int64(limit),
			"search": search, // Include search term in response
		},
	})
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
func (h *Handler) StartTranscription(c *gin.Context) {
	jobID := c.Param("id")
	
	var job models.TranscriptionJob
	if err := database.DB.Where("id = ?", jobID).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}

	// Allow transcription for uploaded, completed, and failed jobs (re-transcription)
	if job.Status != models.StatusUploaded && job.Status != models.StatusCompleted && job.Status != models.StatusFailed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot start transcription: job is currently processing or pending"})
		return
	}

	// Parse transcription parameters from request body
	var requestParams models.WhisperXParams

	// Set defaults
	requestParams = models.WhisperXParams{
		Model:                               "small",
		ModelCacheOnly:                      false,
		Device:                              "cpu",
		DeviceIndex:                         0,
		BatchSize:                           8,
		ComputeType:                         "float32",
		Threads:                             0,
		OutputFormat:                        "all",
		Verbose:                             true,
		Task:                                "transcribe",
		InterpolateMethod:                   "nearest",
		NoAlign:                             false,
		ReturnCharAlignments:                false,
		VadMethod:                           "pyannote",
		VadOnset:                            0.5,
		VadOffset:                           0.363,
		ChunkSize:                           30,
		Diarize:                             false,
		DiarizeModel:                        "pyannote/speaker-diarization-3.1",
		SpeakerEmbeddings:                   false,
		Temperature:                         0,
		BestOf:                              5,
		BeamSize:                            5,
		Patience:                            1.0,
		LengthPenalty:                       1.0,
		SuppressNumerals:                    false,
		ConditionOnPreviousText:             false,
		Fp16:                                true,
		TemperatureIncrementOnFallback:      0.2,
		CompressionRatioThreshold:           2.4,
		LogprobThreshold:                    -1.0,
		NoSpeechThreshold:                   0.6,
		HighlightWords:                      false,
		SegmentResolution:                   "sentence",
		PrintProgress:                       false,
	}

	// Parse request body parameters, overriding defaults
	if err := c.ShouldBindJSON(&requestParams); err != nil {
		// Use defaults if JSON parsing fails
		fmt.Printf("DEBUG: Failed to parse JSON parameters: %v\n", err)
	}
	
	// Debug: log what we received
	fmt.Printf("DEBUG: Parsed parameters for job %s: Diarize=%v, DiarizeModel=%s\n", jobID, requestParams.Diarize, requestParams.DiarizeModel)

	// Update job with parameters
	job.Parameters = requestParams
	job.Diarization = requestParams.Diarize
	job.Status = models.StatusPending
	
	// Clear previous results for re-transcription
	job.Transcript = nil
	job.Summary = nil
	job.ErrorMessage = nil

	// Save updated job
	if err := database.DB.Save(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job"})
		return
	}

	// Enqueue job for transcription
	if err := h.taskQueue.EnqueueJob(jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue job"})
		return
	}

	c.JSON(http.StatusOK, job)
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
func (h *Handler) KillJob(c *gin.Context) {
	jobID := c.Param("id")
	
	var job models.TranscriptionJob
	if err := database.DB.Where("id = ?", jobID).First(&job).Error; err != nil {
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
func (h *Handler) DeleteJob(c *gin.Context) {
	jobID := c.Param("id")
	
	var job models.TranscriptionJob
	if err := database.DB.Where("id = ?", jobID).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}

	// Prevent deletion of jobs that are currently processing
	if job.Status == models.StatusProcessing {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete job that is currently processing"})
		return
	}

	// Delete the audio file from filesystem
	if job.AudioPath != "" {
		if err := os.Remove(job.AudioPath); err != nil && !os.IsNotExist(err) {
			// Log the error but don't fail the request - database cleanup is more important
			fmt.Printf("Warning: Failed to delete audio file %s: %v\n", job.AudioPath, err)
		}
	}

	// Delete any transcript files
	if job.Transcript != nil {
		// Remove transcript directory if it exists (assume it's in data/transcripts)
		transcriptDir := filepath.Join("data", "transcripts", jobID)
		if err := os.RemoveAll(transcriptDir); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: Failed to delete transcript directory %s: %v\n", transcriptDir, err)
		}
	}

	// Delete the job from database
	if err := database.DB.Delete(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete job from database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
}

// @Summary Get transcription record by ID
// @Description Get a specific transcription record by its ID
// @Tags transcription
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} models.TranscriptionJob
// @Failure 404 {object} map[string]string
// @Router /api/v1/transcription/{id} [get]
// @Security ApiKeyAuth
func (h *Handler) GetJobByID(c *gin.Context) {
	jobID := c.Param("id")
	
	var job models.TranscriptionJob
	if err := database.DB.Where("id = ?", jobID).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}

	c.JSON(http.StatusOK, job)
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
	
	var job models.TranscriptionJob
	if err := database.DB.Where("id = ?", jobID).First(&job).Error; err != nil {
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

	// Check if audio file exists
	if job.AudioPath == "" {
		fmt.Printf("DEBUG: Audio path is empty\n")
		c.JSON(http.StatusNotFound, gin.H{"error": "Audio file path not found"})
		return
	}

	// Check if file exists on filesystem
	if _, err := os.Stat(job.AudioPath); os.IsNotExist(err) {
		fmt.Printf("DEBUG: Audio file does not exist on disk: %s\n", job.AudioPath)
		c.JSON(http.StatusNotFound, gin.H{"error": "Audio file not found on disk"})
		return
	}

	fmt.Printf("DEBUG: Audio file exists, serving: %s\n", job.AudioPath)

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

	// Add CORS headers for audio
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET")
	c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, X-API-Key")

	// Serve the audio file
	c.File(job.AudioPath)
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

	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !auth.CheckPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := h.authService.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	response := LoginResponse{
		Token: token,
	}
	response.User.ID = user.ID
	response.User.Username = user.Username

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
	// In a JWT-based system, logout is typically handled client-side
	// by removing the token. For more security, you could maintain
	// a blacklist of tokens on the server side.
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// @Summary Check registration status
// @Description Check if the application requires initial user registration
// @Tags auth
// @Produce json
// @Success 200 {object} RegistrationStatusResponse
// @Router /api/v1/auth/registration-status [get]
func (h *Handler) GetRegistrationStatus(c *gin.Context) {
	var userCount int64
	if err := database.DB.Model(&models.User{}).Count(&userCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check registration status"})
		return
	}

	response := RegistrationStatusResponse{
		RequiresRegistration: userCount == 0,
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
	var userCount int64
	if err := database.DB.Model(&models.User{}).Count(&userCount).Error; err != nil {
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

	if err := database.DB.Create(&user).Error; err != nil {
		if database.DB.Error.Error() == "UNIQUE constraint failed: users.username" {
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

	response := LoginResponse{
		Token: token,
	}
	response.User.ID = user.ID
	response.User.Username = user.Username

	c.JSON(http.StatusCreated, response)
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

	// Get current user
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Verify current password
	if !auth.CheckPassword(req.CurrentPassword, user.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to secure new password"})
		return
	}

	// Update password
	if err := database.DB.Model(&user).Update("password", hashedPassword).Error; err != nil {
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

	// Get current user
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Verify password
	if !auth.CheckPassword(req.Password, user.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password is incorrect"})
		return
	}

	// Check if new username already exists
	var existingUser models.User
	result := database.DB.Where("username = ? AND id != ?", req.NewUsername, userID).First(&existingUser)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	// Update username
	if err := database.DB.Model(&user).Update("username", req.NewUsername).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update username"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Username changed successfully"})
}

// @Summary List API keys
// @Description Get all API keys for the current user (without exposing the actual keys)
// @Tags api-keys
// @Produce json
// @Success 200 {array} APIKeyListResponse
// @Security BearerAuth
// @Router /api/v1/api-keys [get]
func (h *Handler) ListAPIKeys(c *gin.Context) {
	var apiKeys []models.APIKey
	if err := database.DB.Where("is_active = ?", true).Find(&apiKeys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch API keys"})
		return
	}

	var responseKeys []APIKeyListResponse
	for _, key := range apiKeys {
		// Create key preview (show only last 8 characters)
		keyPreview := "••••••••"
		if len(key.Key) >= 8 {
			keyPreview = "••••••••" + key.Key[len(key.Key)-8:]
		}

		responseKeys = append(responseKeys, APIKeyListResponse{
			ID:          key.ID,
			Name:        key.Name,
			Description: func() string {
				if key.Description != nil {
					return *key.Description
				}
				return ""
			}(),
			KeyPreview: keyPreview,
			IsActive:   key.IsActive,
			CreatedAt:  key.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  key.UpdatedAt.Format("2006-01-02 15:04:05"),
			// TODO: Add last_used tracking in future
			LastUsed: "",
		})
	}

	// Return in the format the frontend expects
	c.JSON(http.StatusOK, gin.H{
		"api_keys": responseKeys,
	})
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

	if err := database.DB.Create(&newKey).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	response := CreateAPIKeyResponse{
		ID:          newKey.ID,
		Key:         newKey.Key,
		Name:        newKey.Name,
		Description: req.Description,
	}

	c.JSON(http.StatusCreated, response)
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
	var apiKey models.APIKey
	if err := database.DB.First(&apiKey, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	// Delete the API key (soft delete by setting is_active to false)
	if err := database.DB.Model(&apiKey).Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted successfully"})
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
func (h *Handler) GetSupportedModels(c *gin.Context) {
	models := h.whisperXService.GetSupportedModels()
	languages := h.whisperXService.GetSupportedLanguages()
	
	c.JSON(http.StatusOK, gin.H{
		"models": models,
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
func (h *Handler) ListProfiles(c *gin.Context) {
	var profiles []models.TranscriptionProfile
	if err := database.DB.Order("created_at DESC").Find(&profiles).Error; err != nil {
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
	var existingProfile models.TranscriptionProfile
	if err := database.DB.Where("name = ?", profile.Name).First(&existingProfile).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Profile name already exists"})
		return
	}

	if err := database.DB.Create(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create profile"})
		return
	}

	c.JSON(http.StatusCreated, profile)
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
func (h *Handler) GetProfile(c *gin.Context) {
	profileID := c.Param("id")
	
	var profile models.TranscriptionProfile
	if err := database.DB.Where("id = ?", profileID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profile"})
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
func (h *Handler) UpdateProfile(c *gin.Context) {
	profileID := c.Param("id")
	
	var existingProfile models.TranscriptionProfile
	if err := database.DB.Where("id = ?", profileID).First(&existingProfile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profile"})
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
	var nameCheck models.TranscriptionProfile
	if err := database.DB.Where("name = ? AND id != ?", updatedProfile.Name, profileID).First(&nameCheck).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Profile name already exists"})
		return
	}

	// Update the profile
	updatedProfile.ID = profileID // Ensure ID doesn't change
	if err := database.DB.Save(&updatedProfile).Error; err != nil {
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
func (h *Handler) DeleteProfile(c *gin.Context) {
	profileID := c.Param("id")
	
	var profile models.TranscriptionProfile
	if err := database.DB.Where("id = ?", profileID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profile"})
		return
	}

	if err := database.DB.Delete(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile deleted successfully"})
}

// SetDefaultProfile sets a profile as the default profile
func (h *Handler) SetDefaultProfile(c *gin.Context) {
	profileID := c.Param("id")
	if profileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Profile ID is required"})
		return
	}

	// Find the profile
	var profile models.TranscriptionProfile
	if err := database.DB.Where("id = ?", profileID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profile"})
		return
	}

	// Set this profile as default (the BeforeSave hook will handle unsetting other defaults)
	profile.IsDefault = true
	if err := database.DB.Save(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default profile set successfully", "profile": profile})
}