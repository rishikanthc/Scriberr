package api

import (
	"scriberr/internal/auth"
	"scriberr/internal/web"
	"scriberr/pkg/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRoutes sets up all API routes
func SetupRoutes(handler *Handler, authService *auth.AuthService) *gin.Engine {
	// Create Gin router
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
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

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.GET("/registration-status", handler.GetRegistrationStatus)
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/logout", handler.Logout)
			
			// Account management routes (require authentication)
			authProtected := auth.Group("")
			authProtected.Use(middleware.AuthMiddleware(authService))
			{
				authProtected.POST("/change-password", handler.ChangePassword)
				authProtected.POST("/change-username", handler.ChangeUsername)
			}
		}

		// API Key management routes (require authentication)
		apiKeys := v1.Group("/api-keys")
		apiKeys.Use(middleware.AuthMiddleware(authService))
		{
			apiKeys.GET("/", handler.ListAPIKeys)
			apiKeys.POST("/", handler.CreateAPIKey)
			apiKeys.DELETE("/:id", handler.DeleteAPIKey)
		}

		// Transcription routes (require authentication)
		transcription := v1.Group("/transcription")
		transcription.Use(middleware.AuthMiddleware(authService))
		{
			transcription.POST("/upload", handler.UploadAudio)
			transcription.POST("/submit", handler.SubmitJob)
			transcription.POST("/:id/start", handler.StartTranscription)
			transcription.POST("/:id/kill", handler.KillJob)
			transcription.GET("/:id/status", handler.GetJobStatus)
			transcription.GET("/:id/transcript", handler.GetTranscript)
			transcription.GET("/:id/audio", handler.GetAudioFile)
			transcription.GET("/:id", handler.GetJobByID)
			transcription.DELETE("/:id", handler.DeleteJob)
			transcription.GET("/list", handler.ListJobs)
			transcription.GET("/models", handler.GetSupportedModels)
			// Notes for a transcription
			transcription.GET("/:id/notes", handler.ListNotes)
			transcription.POST("/:id/notes", handler.CreateNote)
			
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
	}

	// Set up static file serving for React app
	web.SetupStaticRoutes(router, authService)

	return router
}
