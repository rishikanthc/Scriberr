package main

import (
	"context"
	"flag"
	"fmt"
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
	"scriberr/internal/queue"
	"scriberr/internal/transcription"
	"scriberr/pkg/logger"

	_ "scriberr/api-docs" // Import generated Swagger docs

	"github.com/gin-gonic/gin"
)

// Version information (set by GoReleaser)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
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
	// Handle version flag
	var showVersion = flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Scriberr %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
		os.Exit(0)
	}

	log.Println("🚀 Scriberr starting up...")
	
	// Load configuration
	log.Println("📋 Loading configuration...")
	cfg := config.Load()

	// Initialize structured logging
	log.Println("📝 Initializing logging system...")
	logger.Init(os.Getenv("LOG_LEVEL"))
	logger.Info("Starting Scriberr", "version", version, "commit", commit)

	// Initialize database
	log.Println("🗄️  Initializing database connection...")
	if err := database.Initialize(cfg.DatabasePath); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()
	log.Println("✅ Database connection established")

	// Initialize authentication service
	log.Println("🔐 Setting up authentication service...")
	authService := auth.NewAuthService(cfg.JWTSecret)
	log.Println("✅ Authentication service ready")

	// Initialize WhisperX service
	log.Println("🎤 Initializing WhisperX transcription service...")
	whisperXService := transcription.NewWhisperXService(cfg)
	
	// Bootstrap embedded Python environment (pyproject + diarization script)
	log.Println("🐍 Setting up Python environment and dependencies...")
	if err := whisperXService.InitEmbeddedPythonEnv(); err != nil {
		log.Fatalf("Failed to initialize Python env: %v", err)
	}
	log.Println("✅ Python environment ready")

	// Initialize quick transcription service
	log.Println("⚡ Initializing quick transcription service...")
	quickTranscriptionService, err := transcription.NewQuickTranscriptionService(cfg, whisperXService)
	if err != nil {
		log.Fatal("Failed to initialize quick transcription service:", err)
	}
	log.Println("✅ Quick transcription service ready")

	// Initialize task queue
	log.Println("📋 Starting background task queue...")
	taskQueue := queue.NewTaskQueue(2, whisperXService) // 2 workers
	taskQueue.Start()
	defer taskQueue.Stop()
	log.Println("✅ Task queue started with 2 workers")

	// Initialize API handlers
	log.Println("🔧 Setting up API handlers...")
	handler := api.NewHandler(cfg, authService, taskQueue, whisperXService, quickTranscriptionService)

	// Set up router
	log.Println("🛤️  Configuring routes...")
	if cfg.Host != "localhost" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := api.SetupRoutes(handler, authService)
	log.Println("✅ Routes configured")

	// Create server
	srv := &http.Server{
		Addr:    cfg.Host + ":" + cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🌐 Starting HTTP server on %s:%s", cfg.Host, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()
	
	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)
	log.Printf("🎉 Scriberr is now running! Server listening on http://%s:%s", cfg.Host, cfg.Port)
	log.Println("💡 Visit /swagger/index.html for API documentation")
	log.Println("🛑 Press Ctrl+C to stop the server")

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
