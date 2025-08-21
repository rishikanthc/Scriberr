package api

import (
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
	params := models.WhisperXParams{
		Model:       getFormValueWithDefault(c, "model", "base"),
		BatchSize:   getFormIntWithDefault(c, "batch_size", 16),
		ComputeType: getFormValueWithDefault(c, "compute_type", "int8"),
		Device:      getFormValueWithDefault(c, "device", "cpu"),
		VadFilter:   getFormBoolWithDefault(c, "vad_filter", false),
		VadOnset:    getFormFloatWithDefault(c, "vad_onset", 0.500),
		VadOffset:   getFormFloatWithDefault(c, "vad_offset", 0.363),
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

	// Create job
	job := models.TranscriptionJob{
		ID:          jobID,
		AudioPath:   filePath,
		Status:      models.StatusPending,
		Diarization: getFormBoolWithDefault(c, "diarization", false),
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
// @Description Get a list of all transcription jobs
// @Tags transcription
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param status query string false "Filter by status"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/transcription/list [get]
// @Security ApiKeyAuth
func (h *Handler) ListJobs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")
	
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	query := database.DB.Model(&models.TranscriptionJob{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var jobs []models.TranscriptionJob
	var total int64
	
	query.Count(&total)
	
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&jobs).Error; err != nil {
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