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

	"scriberr/internal/app"
	"scriberr/internal/config"
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

	application, err := app.Build(cfg)
	if err != nil {
		logger.Error("Failed to build application", "error", err)
		os.Exit(1)
	}

	if err := application.Start(context.Background()); err != nil {
		logger.Error("Failed to start application", "error", err)
		os.Exit(1)
	}

	srv := application.Server()

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

	if err := application.Shutdown(ctx); err != nil {
		logger.Error("Failed to stop application", "error", err)
		shutdownFailed = true
	}
	logger.Info("Server stopped")
	if shutdownFailed {
		os.Exit(1)
	}
}
