package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"scriberr/internal/account"
	"scriberr/internal/annotations"
	"scriberr/internal/api"
	"scriberr/internal/auth"
	"scriberr/internal/automation"
	chatdomain "scriberr/internal/chat"
	"scriberr/internal/config"
	"scriberr/internal/database"
	filesdomain "scriberr/internal/files"
	"scriberr/internal/llmprovider"
	"scriberr/internal/mediaimport"
	profiledomain "scriberr/internal/profile"
	recordingdomain "scriberr/internal/recording"
	"scriberr/internal/repository"
	"scriberr/internal/summarization"
	"scriberr/internal/tags"
	transcriptiondomain "scriberr/internal/transcription"
	"scriberr/internal/transcription/engineprovider"
	"scriberr/internal/transcription/orchestrator"
	"scriberr/internal/transcription/worker"
	"scriberr/pkg/logger"
)

// App owns the constructed backend graph and its bounded lifecycle.
type App struct {
	Config *config.Config
	Router http.Handler

	queueService       *worker.Service
	summaryService     *summarization.Service
	recordingFinalizer *recordingdomain.FinalizerService
	localProvider      *engineprovider.LocalProvider
}

// Build initializes durable dependencies, repositories, services, API handlers, and routes.
// It does not start background workers or bind an HTTP listener.
func Build(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	logger.Startup("database", "Connecting to database")
	if err := database.Initialize(cfg.DatabasePath); err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	logger.Startup("auth", "Setting up authentication")
	authService := auth.NewAuthService(cfg.JWTSecret)

	logger.Startup("repository", "Initializing repositories")
	jobRepo := repository.NewJobRepository(database.DB)
	annotationRepo := repository.NewAnnotationRepository(database.DB)
	summaryRepo := repository.NewSummaryRepository(database.DB)
	llmConfigRepo := repository.NewLLMConfigRepository(database.DB)
	recordingRepo := repository.NewRecordingRepository(database.DB)
	profileRepo := repository.NewProfileRepository(database.DB)
	tagRepo := repository.NewTagRepository(database.DB)
	userRepo := repository.NewUserRepository(database.DB)
	refreshTokenRepo := repository.NewRefreshTokenRepository(database.DB)
	apiKeyRepo := repository.NewAPIKeyRepository(database.DB)
	chatRepo := repository.NewChatRepository(database.DB)

	logger.Startup("engine", "Initializing local engine provider")
	localProvider, err := engineprovider.NewLocalProvider(cfg.Engine)
	if err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("initialize local engine provider: %w", err)
	}

	providerRegistry, err := engineprovider.NewRegistry(engineprovider.DefaultProviderID, localProvider)
	if err != nil {
		_ = localProvider.Close()
		_ = database.Close()
		return nil, fmt.Errorf("initialize engine provider registry: %w", err)
	}

	processor := &orchestrator.Processor{
		Jobs:      jobRepo,
		Providers: providerRegistry,
		Artifacts: orchestrator.NewLocalTranscriptStore(cfg.TranscriptsDir),
	}
	queueService := worker.NewService(jobRepo, processor, worker.Config{
		Workers:      cfg.Worker.Workers,
		PollInterval: cfg.Worker.PollInterval,
		LeaseTimeout: cfg.Worker.LeaseTimeout,
	})
	summaryService := summarization.NewService(summaryRepo, llmConfigRepo, jobRepo, summarization.Config{})
	chatService := chatdomain.NewService(chatRepo, llmConfigRepo)
	accountService := account.NewService(userRepo, refreshTokenRepo, apiKeyRepo, profileRepo, llmConfigRepo, authService)
	profileService := profiledomain.NewService(profileRepo)
	llmProviderService := llmprovider.NewService(llmConfigRepo, llmprovider.HTTPConnectionTester{})
	fileService := filesdomain.NewService(jobRepo, filesdomain.Config{UploadDir: cfg.UploadDir})
	mediaImportService := mediaimport.NewService(mediaimport.ServiceOptions{
		Repository: jobRepo,
		UploadDir:  cfg.UploadDir,
	})
	transcriptionService := transcriptiondomain.NewService(jobRepo, profileRepo, queueService)
	postFileAutomation := automation.NewService(jobRepo, userRepo, profileRepo, llmConfigRepo, transcriptionService)
	fileService.SetReadyObserver(postFileAutomation)
	annotationService := annotations.NewService(annotationRepo, jobRepo)
	tagService := tags.NewService(tagRepo, jobRepo)
	recordingStorage, err := recordingdomain.NewStorage(cfg.Recordings.Dir)
	if err != nil {
		_ = localProvider.Close()
		_ = database.Close()
		return nil, fmt.Errorf("initialize recording storage: %w", err)
	}
	recordingService := recordingdomain.NewService(recordingRepo, recordingStorage, recordingdomain.Config{
		MaxChunkBytes:    cfg.Recordings.MaxChunkBytes,
		MaxSessionBytes:  cfg.Recordings.MaxSessionBytes,
		MaxDuration:      cfg.Recordings.MaxDuration,
		SessionTTL:       cfg.Recordings.SessionTTL,
		AllowedMimeTypes: cfg.Recordings.AllowedMimeTypes,
	})
	recordingFinalizer := recordingdomain.NewFinalizerService(recordingRepo, jobRepo, profileRepo, recordingStorage, recordingdomain.FFmpegFinalizer{}, recordingdomain.FinalizerConfig{
		Workers:         cfg.Recordings.FinalizerWorkers,
		PollInterval:    cfg.Recordings.FinalizerPollInterval,
		LeaseTimeout:    cfg.Recordings.FinalizerLeaseTimeout,
		CleanupInterval: cfg.Recordings.CleanupInterval,
		FailedRetention: cfg.Recordings.FailedRetention,
	})
	recordingFinalizer.SetTranscriptionEnqueuer(queueService)
	recordingFinalizer.SetFileReadyHandoff(fileService)

	handler := api.NewHandler(cfg, authService, api.HandlerDependencies{
		ReadinessCheck: database.HealthCheck,
		Queue:          queueService,
		ModelRegistry:  providerRegistry,
		Account:        accountService,
		Profiles:       profileService,
		LLMProvider:    llmProviderService,
		Files:          fileService,
		MediaImport:    mediaImportService,
		Annotations:    annotationService,
		Tags:           tagService,
		Recordings:     recordingService,
		Transcriptions: transcriptionService,
		Summaries:      summaryService,
		Chat:           chatService,
		Finalizer:      recordingFinalizer,
	})
	processor.Events = handler
	queueService.SetEventPublisher(handler)
	queueService.SetCompletionObserver(worker.CompletionObservers{summaryService, annotationService})
	summaryService.SetEventPublisher(handler)
	summaryService.SetUserSettingsReader(userRepo)
	postFileAutomation.SetEventPublisher(handler)
	recordingFinalizer.SetEventPublisher(handler)

	return &App{
		Config:             cfg,
		Router:             api.SetupRoutes(handler, authService),
		queueService:       queueService,
		summaryService:     summaryService,
		recordingFinalizer: recordingFinalizer,
		localProvider:      localProvider,
	}, nil
}

func (a *App) Server() *http.Server {
	return &http.Server{
		Addr:    a.Config.Host + ":" + a.Config.Port,
		Handler: a.Router,
	}
}

func (a *App) Start(ctx context.Context) error {
	logger.Startup("worker", "Starting durable transcription workers")
	if err := a.queueService.Start(ctx); err != nil {
		return fmt.Errorf("start transcription workers: %w", err)
	}
	if err := a.summaryService.Start(ctx); err != nil {
		_ = a.queueService.Stop(ctx)
		return fmt.Errorf("start summary workers: %w", err)
	}
	if err := a.recordingFinalizer.Start(ctx); err != nil {
		_ = a.summaryService.Stop(ctx)
		_ = a.queueService.Stop(ctx)
		return fmt.Errorf("start recording finalizers: %w", err)
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	var shutdownErrs []error
	if err := a.queueService.Stop(ctx); err != nil {
		shutdownErrs = append(shutdownErrs, fmt.Errorf("stop transcription workers: %w", err))
	}
	if err := a.summaryService.Stop(ctx); err != nil {
		shutdownErrs = append(shutdownErrs, fmt.Errorf("stop summary workers: %w", err))
	}
	if err := a.recordingFinalizer.Stop(ctx); err != nil {
		shutdownErrs = append(shutdownErrs, fmt.Errorf("stop recording finalizers: %w", err))
	}
	if err := a.localProvider.Close(); err != nil {
		logger.Warn("Failed to close local engine provider", "error", err)
	}
	if err := database.Close(); err != nil {
		shutdownErrs = append(shutdownErrs, fmt.Errorf("close database: %w", err))
	}
	return errors.Join(shutdownErrs...)
}
