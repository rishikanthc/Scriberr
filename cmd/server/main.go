package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"scriberr/internal/api"
	"scriberr/internal/auth"
	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/queue"
	"scriberr/internal/transcription"

	_ "scriberr/docs" // Import generated docs

	"github.com/gin-gonic/gin"
)

// @title Scriberr API
// @version 1.0
// @description Audio transcription service using WhisperX
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT token with Bearer prefix

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	if err := database.Initialize(cfg.DatabasePath); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	// Initialize authentication service
	authService := auth.NewAuthService(cfg.JWTSecret)

	// Initialize WhisperX service
	whisperXService := transcription.NewWhisperXService(cfg)

	// Initialize quick transcription service
	quickTranscriptionService, err := transcription.NewQuickTranscriptionService(cfg, whisperXService)
	if err != nil {
		log.Fatal("Failed to initialize quick transcription service:", err)
	}

	// Initialize task queue
	taskQueue := queue.NewTaskQueue(2, whisperXService) // 2 workers
	taskQueue.Start()
	defer taskQueue.Stop()

	// Initialize API handlers
	handler := api.NewHandler(cfg, authService, taskQueue, whisperXService, quickTranscriptionService)

	// Set up router
	if cfg.Host != "localhost" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := api.SetupRoutes(handler, authService)

	// Create server
	srv := &http.Server{
		Addr:    cfg.Host + ":" + cfg.Port,
		Handler: router,
	}

	// Initialize default data
	if err := initializeDefaultData(cfg); err != nil {
		log.Printf("Warning: Failed to initialize default data: %v", err)
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s:%s", cfg.Host, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Gracefully shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

// initializeDefaultData creates default API key for development (user registration handled separately)
func initializeDefaultData(cfg *config.Config) error {
	// Create default API key for backward compatibility
	var apiKeyCount int64
	database.DB.Model(&models.APIKey{}).Count(&apiKeyCount)
	
	if apiKeyCount == 0 {
		defaultAPIKey := models.APIKey{
			Key:         cfg.DefaultAPIKey,
			Name:        "Default Development Key",
			Description: strPtr("Default API key for development and testing"),
			IsActive:    true,
		}

		if err := database.DB.Create(&defaultAPIKey).Error; err != nil {
			return err
		}

		log.Printf("Created default API key: %s", cfg.DefaultAPIKey)
	}

	return nil
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}