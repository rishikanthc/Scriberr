package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"scriberr/internal/annotations"
	"scriberr/internal/api"
	"scriberr/internal/auth"
	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/repository"
	"scriberr/internal/summarization"
	"scriberr/internal/transcription/engineprovider"
	"scriberr/internal/transcription/orchestrator"
	"scriberr/internal/transcription/worker"
	"scriberr/pkg/logger"
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

	// Initialize structured logging first
	logger.Init(os.Getenv("LOG_LEVEL"))
	logger.Info("Starting Scriberr", "version", version)

	// Load configuration
	logger.Startup("config", "Loading configuration")
	cfg, err := config.LoadWithError()
	if err != nil {
		logger.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}
	logger.Info("Engine and worker configuration loaded",
		"engine_cache_dir", cfg.Engine.CacheDir,
		"engine_provider", cfg.Engine.Provider,
		"engine_threads", cfg.Engine.Threads,
		"engine_max_loaded", cfg.Engine.MaxLoaded,
		"engine_auto_download", cfg.Engine.AutoDownload,
		"transcription_workers", cfg.Worker.Workers,
		"queue_poll_interval", cfg.Worker.PollInterval.String(),
		"lease_timeout", cfg.Worker.LeaseTimeout.String(),
		"recordings_dir", cfg.Recordings.Dir,
		"recording_max_chunk_bytes", cfg.Recordings.MaxChunkBytes,
		"recording_max_duration", cfg.Recordings.MaxDuration.String(),
		"recording_session_ttl", cfg.Recordings.SessionTTL.String(),
		"recording_finalizer_workers", cfg.Recordings.FinalizerWorkers,
		"recording_finalizer_poll_interval", cfg.Recordings.FinalizerPollInterval.String(),
		"recording_finalizer_lease_timeout", cfg.Recordings.FinalizerLeaseTimeout.String(),
		"recording_allowed_mime_types", cfg.Recordings.AllowedMimeTypes,
	)

	// Initialize database
	logger.Startup("database", "Connecting to database")
	if err := database.Initialize(cfg.DatabasePath); err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Initialize authentication service
	logger.Startup("auth", "Setting up authentication")
	authService := auth.NewAuthService(cfg.JWTSecret)

	// Initialize repositories
	logger.Startup("repository", "Initializing repositories")
	jobRepo := repository.NewJobRepository(database.DB)
	annotationRepo := repository.NewAnnotationRepository(database.DB)
	summaryRepo := repository.NewSummaryRepository(database.DB)
	llmConfigRepo := repository.NewLLMConfigRepository(database.DB)

	// Initialize local engine provider. This must not download models at startup.
	logger.Startup("engine", "Initializing local engine provider")
	localProvider, err := engineprovider.NewLocalProvider(cfg.Engine)
	if err != nil {
		logger.Error("Failed to initialize local engine provider", "error", err)
		os.Exit(1)
	}

	providerRegistry, err := engineprovider.NewRegistry(engineprovider.DefaultProviderID, localProvider)
	if err != nil {
		logger.Error("Failed to initialize engine provider registry", "error", err)
		os.Exit(1)
	}

	processor := &orchestrator.Processor{
		Jobs:      jobRepo,
		Providers: providerRegistry,
		OutputDir: cfg.TranscriptsDir,
	}
	queueService := worker.NewService(jobRepo, processor, worker.Config{
		Workers:      cfg.Worker.Workers,
		PollInterval: cfg.Worker.PollInterval,
		LeaseTimeout: cfg.Worker.LeaseTimeout,
	})
	summaryService := summarization.NewService(summaryRepo, llmConfigRepo, jobRepo, summarization.Config{})
	annotationService := annotations.NewService(annotationRepo, jobRepo)

	// Initialize API handlers
	handler := api.NewHandler(cfg, authService, queueService, providerRegistry, annotationService)
	processor.Events = handler
	queueService.SetEventPublisher(handler)
	queueService.SetCompletionObserver(worker.CompletionObservers{summaryService, annotationService})
	summaryService.SetEventPublisher(handler)
	annotationService.SetEventPublisher(handler)

	// Set up router
	router := api.SetupRoutes(handler, authService)

	// Start durable transcription workers after DB recovery.
	logger.Startup("worker", "Starting durable transcription workers")
	if err := queueService.Start(context.Background()); err != nil {
		logger.Error("Failed to start transcription workers", "error", err)
		os.Exit(1)
	}
	if err := summaryService.Start(context.Background()); err != nil {
		logger.Error("Failed to start summary workers", "error", err)
		os.Exit(1)
	}

	// Create server
	srv := &http.Server{
		Addr:    cfg.Host + ":" + cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Debug("Starting HTTP server", "host", cfg.Host, "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)
	logger.Info("Scriberr is ready",
		"url", fmt.Sprintf("http://%s:%s", cfg.Host, cfg.Port))
	logger.Debug("API documentation available at /swagger/index.html")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Gracefully shutdown the server
	shutdownFailed := false
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		shutdownFailed = true
	}

	if err := queueService.Stop(ctx); err != nil {
		logger.Error("Failed to stop transcription workers", "error", err)
		shutdownFailed = true
	}
	if err := summaryService.Stop(ctx); err != nil {
		logger.Error("Failed to stop summary workers", "error", err)
		shutdownFailed = true
	}

	if err := localProvider.Close(); err != nil {
		logger.Warn("Failed to close local engine provider", "error", err)
	}

	database.Close()
	logger.Info("Server stopped")
	if shutdownFailed {
		os.Exit(1)
	}
}
