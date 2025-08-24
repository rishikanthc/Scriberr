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
			auth.POST("/login", handler.Login)
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
	}

	// Set up static file serving for React app
	web.SetupStaticRoutes(router)

	return router
}