package api

import (
	"context"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"scriberr/internal/annotations"
	"scriberr/internal/auth"
	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/repository"
	"scriberr/internal/summarization"
	"scriberr/internal/transcription/engineprovider"
	"scriberr/internal/transcription/orchestrator"
	"scriberr/internal/transcription/worker"
	"scriberr/internal/web"
	"scriberr/pkg/logger"
	"scriberr/pkg/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	config          *config.Config
	authService     *auth.AuthService
	readinessCheck  func() error
	idempotency     *idempotencyStore
	events          *eventBroker
	youtubeImporter YouTubeImporter
	mediaExtractor  MediaExtractor
	eventHeartbeat  time.Duration
	asyncJobs       sync.WaitGroup
	maxUploadBytes  int64
	queueService    worker.QueueService
	modelRegistry   engineprovider.Registry
	annotations     *annotations.Service
}

func NewHandler(cfg *config.Config, authService *auth.AuthService, services ...any) *Handler {
	if cfg == nil {
		cfg = &config.Config{}
	}
	handler := &Handler{
		config:          cfg,
		authService:     authService,
		readinessCheck:  database.HealthCheck,
		idempotency:     newIdempotencyStore(),
		events:          newEventBroker(),
		youtubeImporter: ytDLPImporter{},
		mediaExtractor:  ffmpegMediaExtractor{},
		eventHeartbeat:  25 * time.Second,
		maxUploadBytes:  defaultMaxUploadSizeBytes,
	}
	for _, service := range services {
		switch value := service.(type) {
		case worker.QueueService:
			handler.queueService = value
		case engineprovider.Registry:
			handler.modelRegistry = value
		case *annotations.Service:
			handler.annotations = value
		}
	}
	if handler.annotations == nil && database.DB != nil {
		handler.annotations = annotations.NewService(repository.NewAnnotationRepository(database.DB), repository.NewJobRepository(database.DB))
		handler.annotations.SetEventPublisher(handler)
	}
	return handler
}

func (h *Handler) Publish(_ context.Context, event orchestrator.ProgressEvent) {
	h.publishTranscriptionStatus(event.Name, event.JobID, event.FileID, string(event.Status), event.Progress, event.Stage)
}

func (h *Handler) PublishStatus(_ context.Context, event worker.StatusEvent) {
	h.publishTranscriptionStatus(event.Name, event.JobID, event.FileID, string(event.Status), event.Progress, event.Stage)
}

func (h *Handler) PublishSummaryStatus(_ context.Context, event summarization.StatusEvent) {
	if h == nil {
		return
	}
	payload := gin.H{
		"id":                   event.SummaryID,
		"transcription_id":     "tr_" + event.TranscriptionID,
		"status":               event.Status,
		"transcript_truncated": event.Truncated,
	}
	if event.WidgetRunID != "" {
		payload["widget_run_id"] = event.WidgetRunID
		payload["widget_id"] = event.WidgetID
		payload["context_truncated"] = event.Truncated
	}
	h.publishTranscriptionEvent(event.Name, "tr_"+event.TranscriptionID, payload)
	h.publishEvent(event.Name, payload)
}

func (h *Handler) PublishAnnotationEvent(_ context.Context, event annotations.Event) {
	if h == nil {
		return
	}
	payload := gin.H{
		"id":               event.AnnotationID,
		"transcription_id": event.TranscriptionID,
		"kind":             string(event.Kind),
		"status":           event.Status,
	}
	if event.EntryID != "" {
		payload["entry_id"] = event.EntryID
	}
	h.publishTranscriptionEvent(event.Name, event.TranscriptionID, payload)
	h.publishEvent(event.Name, payload)
}

func (h *Handler) publishTranscriptionStatus(name, jobID, fileID, status string, progress float64, stage string) {
	if h == nil {
		return
	}
	payload := gin.H{
		"id":       "tr_" + jobID,
		"status":   status,
		"progress": progress,
		"stage":    stage,
	}
	if fileID != "" {
		payload["file_id"] = fileID
	}
	h.publishTranscriptionEvent(name, "tr_"+jobID, payload)
}

func SetupRoutes(handler *Handler, _ *auth.AuthService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	logger.SetGinOutput()

	router := gin.New()
	router.Use(recoveryMiddleware())
	router.Use(requestIDMiddleware())
	router.Use(logger.GinLogger())
	router.Use(middleware.CompressionMiddleware())
	router.Use(corsMiddleware(handler.config))

	router.GET("/health", handler.health)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", handler.health)
		v1.GET("/ready", handler.ready)

		authRoutes := v1.Group("/auth")
		{
			authRoutes.GET("/registration-status", handler.registrationStatus)
			authRoutes.POST("/register", handler.register)
			authRoutes.POST("/login", handler.login)
			authRoutes.POST("/refresh", handler.refresh)
			authRoutes.POST("/logout", handler.logout)

			protected := authRoutes.Group("")
			protected.Use(handler.jwtRequired())
			{
				protected.GET("/me", handler.me)
				protected.POST("/change-password", handler.changePassword)
				protected.POST("/change-username", handler.changeUsername)
			}
		}

		apiKeys := v1.Group("/api-keys")
		apiKeys.Use(handler.jwtRequired())
		{
			apiKeys.GET("", handler.listAPIKeys)
			apiKeys.POST("", handler.idempotencyMiddleware(), handler.createAPIKey)
			apiKeys.DELETE("/:id", handler.deleteAPIKey)
		}

		files := v1.Group("/files")
		files.Use(handler.authRequired())
		{
			files.POST("", handler.idempotencyMiddleware(), handler.uploadFile)
			files.GET("", handler.listFiles)
			files.GET("/:id", handler.getFile)
			files.PATCH("/:id", handler.updateFile)
			files.DELETE("/:id", handler.deleteFile)
			files.GET("/:id/audio", handler.streamFileAudio)
		}
		transcriptions := v1.Group("/transcriptions")
		transcriptions.Use(handler.authRequired())
		{
			transcriptions.POST("", handler.idempotencyMiddleware(), handler.createTranscription)
			transcriptions.GET("", handler.listTranscriptions)
			transcriptions.GET("/:id", handler.getTranscription)
			transcriptions.PATCH("/:id", handler.updateTranscription)
			transcriptions.DELETE("/:id", handler.deleteTranscription)
			transcriptions.GET("/:id/transcript", handler.getTranscript)
			transcriptions.GET("/:id/annotations", handler.listAnnotations)
			transcriptions.POST("/:id/annotations", handler.idempotencyMiddleware(), handler.createAnnotation)
			transcriptions.GET("/:id/annotations/:annotation_id", handler.getAnnotation)
			transcriptions.PATCH("/:id/annotations/:annotation_id", handler.updateAnnotation)
			transcriptions.DELETE("/:id/annotations/:annotation_id", handler.deleteAnnotation)
			transcriptions.POST("/:id/annotations/:annotation_id/entries", handler.idempotencyMiddleware(), handler.createAnnotationEntry)
			transcriptions.PATCH("/:id/annotations/:annotation_id/entries/:entry_id", handler.updateAnnotationEntry)
			transcriptions.DELETE("/:id/annotations/:annotation_id/entries/:entry_id", handler.deleteAnnotationEntry)
			transcriptions.GET("/:id/summary", handler.getTranscriptionSummary)
			transcriptions.GET("/:id/summary/widgets", handler.listTranscriptionSummaryWidgets)
			transcriptions.GET("/:id/audio", handler.streamTranscriptionAudio)
			transcriptions.GET("/:id/events", handler.streamTranscriptionEvents)
			transcriptions.GET("/:id/logs", handler.getTranscriptionLogs)
			transcriptions.GET("/:id/executions", handler.getTranscriptionExecutions)
		}

		profiles := v1.Group("/profiles")
		profiles.Use(handler.authRequired())
		{
			profiles.GET("", handler.listProfiles)
			profiles.POST("", handler.idempotencyMiddleware(), handler.createProfile)
			profiles.GET("/:id", handler.getProfile)
			profiles.PATCH("/:id", handler.updateProfile)
			profiles.DELETE("/:id", handler.deleteProfile)
			profiles.POST("/:idAction", handler.idempotencyMiddleware(), handler.profileCommand)
		}

		settings := v1.Group("/settings")
		settings.Use(handler.authRequired())
		{
			settings.GET("", handler.getSettings)
			settings.PATCH("", handler.updateSettings)
			settings.GET("/llm-provider", handler.getLLMProvider)
			settings.PUT("/llm-provider", handler.updateLLMProvider)
			settings.GET("/summary-widgets", handler.listSummaryWidgets)
			settings.POST("/summary-widgets", handler.idempotencyMiddleware(), handler.createSummaryWidget)
			settings.PATCH("/summary-widgets/:id", handler.updateSummaryWidget)
			settings.DELETE("/summary-widgets/:id", handler.deleteSummaryWidget)
		}

		v1.GET("/events", handler.authRequired(), handler.streamEvents)
		v1.GET("/models/transcription", handler.authRequired(), handler.listTranscriptionModels)
		v1.GET("/admin/queue", handler.authRequired(), handler.queueStats)
	}

	web.SetupStaticRoutes(router, handler.authService)
	router.NoRoute(func(c *gin.Context) {
		if handler.handleCommandRoute(c) {
			return
		}
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "API endpoint not found", nil)
			return
		}
		cleanPath := strings.TrimPrefix(path.Clean(c.Request.URL.Path), "/")
		if strings.Contains(cleanPath, "..") {
			c.Status(http.StatusForbidden)
			return
		}
		if cleanPath != "" && strings.Contains(path.Base(cleanPath), ".") {
			web.GetStaticHandler().ServeHTTP(c.Writer, c.Request)
			return
		}
		indexHTML, err := web.GetIndexHTML()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error loading page")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})

	return router
}
