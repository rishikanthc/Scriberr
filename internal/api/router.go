package api

import (
	"scriberr/internal/auth"
	"scriberr/internal/web"
	"scriberr/pkg/logger"
	"scriberr/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up all API routes
func SetupRoutes(handler *Handler, authService *auth.AuthService) *gin.Engine {
	// Suppress all GIN debug output
	gin.SetMode(gin.ReleaseMode)
	logger.SetGinOutput()

	// Create Gin router without default middleware
	router := gin.New()

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Add custom logger middleware
	router.Use(logger.GinLogger())

	// Add compression middleware first for maximum benefit
	router.Use(middleware.CompressionMiddleware())

	// Add CORS middleware (uses config from handler)
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Determine allowed origin based on config
		allowOrigin := "*"
		if handler.config.IsProduction() && len(handler.config.AllowedOrigins) > 0 {
			// In production, validate against configured origins
			allowOrigin = ""
			for _, allowed := range handler.config.AllowedOrigins {
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
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint (no auth required)
	router.GET("/health", handler.HealthCheck)

	// CLI install script alias (root level for easier access)
	router.GET("/install.sh", handler.GetInstallScript)
	router.GET("/install-cli.sh", handler.GetInstallScript)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.GET("/registration-status", handler.GetRegistrationStatus)
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/refresh", handler.Refresh)
			auth.POST("/logout", handler.Logout)

			// Account management routes (require authentication)
			authProtected := auth.Group("")
			// Account management must require JWT (API keys do not represent a user)
			authProtected.Use(middleware.JWTOnlyMiddleware(authService))
			{
				authProtected.POST("/change-password", handler.ChangePassword)
				authProtected.POST("/change-username", handler.ChangeUsername)

				// CLI Authentication routes
				cliAuth := authProtected.Group("/cli")
				{
					cliAuth.GET("/authorize", handler.AuthorizeCLI)
					cliAuth.POST("/authorize", handler.ConfirmCLIAuthorization)
				}
			}
		}

		// Public CLI routes (no auth required to download, script handles auth)
		cliPublic := v1.Group("/cli")
		{
			cliPublic.GET("/download", handler.DownloadCLIBinary)
			cliPublic.GET("/install", handler.GetInstallScript)
		}
		// API Key management routes (require authentication)
		apiKeys := v1.Group("/api-keys")
		// API key management restricted to JWT-authenticated users
		apiKeys.Use(middleware.JWTOnlyMiddleware(authService))
		{
			apiKeys.GET("/", handler.ListAPIKeys)
			apiKeys.POST("/", handler.CreateAPIKey)
			apiKeys.DELETE("/:id", handler.DeleteAPIKey)
		}

		// Transcription routes (require authentication)
		transcription := v1.Group("/transcription")
		transcription.Use(middleware.AuthMiddleware(authService))
		{
			// File upload routes - disable compression for these
			uploadRoutes := transcription.Group("")
			uploadRoutes.Use(middleware.NoCompressionMiddleware())
			{
				uploadRoutes.POST("/upload", handler.UploadAudio)
				uploadRoutes.POST("/upload-video", handler.UploadVideo)
				uploadRoutes.POST("/upload-multitrack", handler.UploadMultiTrack)
				uploadRoutes.GET("/:id/audio", handler.GetAudioFile) // Audio streaming shouldn't be compressed
			}

			// Regular API routes with compression
			transcription.POST("/youtube", handler.DownloadFromYouTube)
			transcription.POST("/submit", handler.SubmitJob)
			transcription.POST("/:id/start", handler.StartTranscription)
			transcription.POST("/:id/kill", handler.KillJob)
			transcription.GET("/:id/logs", handler.GetJobLogs)
			transcription.GET("/:id/status", handler.GetJobStatus)
			transcription.GET("/:id/transcript", handler.GetTranscript)
			transcription.GET("/:id/execution", handler.GetJobExecutionData)
			transcription.GET("/:id/merge-status", handler.GetMergeStatus)
			transcription.GET("/:id/track-progress", handler.GetTrackProgress)
			transcription.PUT("/:id/title", handler.UpdateTranscriptionTitle)
			transcription.GET("/:id/summary", handler.GetSummaryForTranscription)
			transcription.GET("/:id", handler.GetTranscriptionJob)
			transcription.DELETE("/:id", handler.DeleteTranscriptionJob)
			transcription.GET("/list", handler.ListTranscriptionJobs)
			transcription.GET("/models", handler.GetSupportedModels)
			// Notes for a transcription
			transcription.GET("/:id/notes", handler.ListNotes)
			transcription.POST("/:id/notes", handler.CreateNote)

			// Speaker mappings for a transcription
			transcription.GET("/:id/speakers", handler.GetSpeakerMappings)
			transcription.POST("/:id/speakers", handler.UpdateSpeakerMappings)

			// Quick transcription endpoints
			transcription.POST("/quick", handler.SubmitQuickTranscription)
			transcription.GET("/quick/:id", handler.GetQuickTranscriptionStatus)
		}

		// Profile routes (require authentication)
		profiles := v1.Group("/profiles")
		profiles.Use(middleware.AuthMiddleware(authService))
		{
			profiles.GET("/", handler.ListProfiles)
			profiles.POST("/", handler.CreateProfile)
			profiles.GET("/:id", handler.GetProfile)
			profiles.PUT("/:id", handler.UpdateProfile)
			profiles.DELETE("/:id", handler.DeleteProfile)
			profiles.POST("/:id/set-default", handler.SetDefaultProfile)
		}

		// User routes (require authentication)
		user := v1.Group("/user")
		user.Use(middleware.JWTOnlyMiddleware(authService))
		{
			user.GET("/default-profile", handler.GetUserDefaultProfile)
			user.POST("/default-profile", handler.SetUserDefaultProfile)
			user.GET("/settings", handler.GetUserSettings)
			user.PUT("/settings", handler.UpdateUserSettings)
		}

		// Admin routes (require authentication)
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware(authService))
		{
			queue := admin.Group("/queue")
			{
				queue.GET("/stats", handler.GetQueueStats)
			}
		}

		// LLM configuration routes (require authentication)
		llm := v1.Group("/llm")
		llm.Use(middleware.AuthMiddleware(authService))
		{
			llm.GET("/config", handler.GetLLMConfig)
			llm.POST("/config", handler.SaveLLMConfig)
		}

		// Summarization templates routes (require authentication)
		summaries := v1.Group("/summaries")
		summaries.Use(middleware.AuthMiddleware(authService))
		{
			summaries.GET("/", handler.ListSummaryTemplates)
			summaries.POST("/", handler.CreateSummaryTemplate)
			summaries.GET("/:id", handler.GetSummaryTemplate)
			summaries.PUT("/:id", handler.UpdateSummaryTemplate)
			summaries.DELETE("/:id", handler.DeleteSummaryTemplate)
			summaries.GET("/settings", handler.GetSummarySettings)
			summaries.POST("/settings", handler.SaveSummarySettings)
		}

		// Chat routes (require authentication)
		chat := v1.Group("/chat")
		chat.Use(middleware.AuthMiddleware(authService))
		{
			chat.GET("/models", handler.GetChatModels)
			chat.POST("/sessions", handler.CreateChatSession)
			chat.GET("/transcriptions/:transcription_id/sessions", handler.GetChatSessions)
			chat.GET("/sessions/:session_id", handler.GetChatSession)
			chat.POST("/sessions/:session_id/messages", handler.SendChatMessage)
			chat.PUT("/sessions/:session_id/title", handler.UpdateChatSessionTitle)
			chat.POST("/sessions/:session_id/title/auto", handler.AutoGenerateChatTitle)
			chat.DELETE("/sessions/:session_id", handler.DeleteChatSession)
		}

		// Notes routes (require authentication)
		notes := v1.Group("/notes")
		notes.Use(middleware.AuthMiddleware(authService))
		{
			notes.GET("/:note_id", handler.GetNote)
			notes.PUT("/:note_id", handler.UpdateNote)
			notes.DELETE("/:note_id", handler.DeleteNote)
		}

		// Summarization route (require authentication)
		summarize := v1.Group("/summarize")
		summarize.Use(middleware.AuthMiddleware(authService))
		{
			summarize.POST("/", handler.Summarize)
		}

		// CSV Batch processing routes (require authentication)
		csvBatch := v1.Group("/csv-batch")
		csvBatch.Use(middleware.AuthMiddleware(authService))
		{
			csvBatchHandler := NewCSVBatchHandler(handler, handler.GetCSVBatchProcessor())

			// Upload routes - disable compression for uploads
			uploadRoutes := csvBatch.Group("")
			uploadRoutes.Use(middleware.NoCompressionMiddleware())
			{
				uploadRoutes.POST("/upload", csvBatchHandler.UploadCSV)
			}

			// Regular routes
			csvBatch.GET("", csvBatchHandler.ListBatches)
			csvBatch.GET("/:id/status", csvBatchHandler.GetBatchStatus)
			csvBatch.GET("/:id/rows", csvBatchHandler.GetBatchRows)
			csvBatch.POST("/:id/start", csvBatchHandler.StartBatch)
			csvBatch.POST("/:id/stop", csvBatchHandler.StopBatch)
			csvBatch.DELETE("/:id", csvBatchHandler.DeleteBatch)
			csvBatch.GET("/:id/output/:row_id", csvBatchHandler.DownloadOutput)
			csvBatch.GET("/:id/outputs", csvBatchHandler.ListOutputs)
		// Config routes (require authentication)
		config := v1.Group("/config")
		config.Use(middleware.AuthMiddleware(authService))
		{
			config.POST("/openai/validate", handler.ValidateOpenAIKey)
		}

		// SSE Events (require authentication)
		events := v1.Group("/events")
		events.Use(middleware.AuthMiddleware(authService))
		{
			events.GET("/", handler.Events)
		}
	}

	// Set up static file serving for React app
	web.SetupStaticRoutes(router, authService)

	return router
}
