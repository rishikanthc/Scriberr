package recording

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/pkg/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MediaFinalizer interface {
	Finalize(ctx context.Context, inputPath, outputPath string) error
}

type TranscriptionEnqueuer interface {
	Enqueue(ctx context.Context, jobID string) error
}

type FinalizerEventPublisher interface {
	PublishRecordingEvent(ctx context.Context, event Event)
	PublishFileEvent(ctx context.Context, name string, payload map[string]any)
	PublishTranscriptionEvent(ctx context.Context, name string, transcriptionID string, payload map[string]any)
}

type FinalizerStorage interface {
	BuildRaw(ctx context.Context, sessionID string, chunkPaths []string) (string, error)
	FinalPath(sessionID string, mimeType string) (string, error)
	RemoveTemporaryArtifacts(sessionID string) error
	RemoveSession(sessionID string) error
}

type FinalizerConfig struct {
	Workers         int
	PollInterval    time.Duration
	LeaseTimeout    time.Duration
	RenewInterval   time.Duration
	StopTimeout     time.Duration
	CleanupInterval time.Duration
	FailedRetention time.Duration
	WorkerID        string
}

type MaintenanceStats struct {
	RecoveredClaims           int64
	ExpiredSessions           int64
	TemporaryArtifactsRemoved int64
	SessionDirsRemoved        int64
}

type FinalizerService struct {
	recordings repository.RecordingRepository
	handoff    repository.RecordingHandoffRepository
	profiles   repository.ProfileRepository
	storage    FinalizerStorage
	media      MediaFinalizer
	queue      TranscriptionEnqueuer
	events     FinalizerEventPublisher
	cfg        FinalizerConfig

	mu      sync.Mutex
	started bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	wake    chan struct{}
}

func NewFinalizerService(recordings repository.RecordingRepository, handoff repository.RecordingHandoffRepository, profiles repository.ProfileRepository, storage FinalizerStorage, media MediaFinalizer, cfg FinalizerConfig) *FinalizerService {
	cfg = normalizeFinalizerConfig(cfg)
	if media == nil {
		media = FFmpegFinalizer{}
	}
	return &FinalizerService{
		recordings: recordings,
		handoff:    handoff,
		profiles:   profiles,
		storage:    storage,
		media:      media,
		cfg:        cfg,
		wake:       make(chan struct{}, 1),
	}
}

func (s *FinalizerService) SetEventPublisher(events FinalizerEventPublisher) {
	s.events = events
}

func (s *FinalizerService) SetTranscriptionEnqueuer(queue TranscriptionEnqueuer) {
	s.queue = queue
}

func normalizeFinalizerConfig(cfg FinalizerConfig) FinalizerConfig {
	if cfg.Workers <= 0 {
		cfg.Workers = 1
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.LeaseTimeout <= 0 {
		cfg.LeaseTimeout = 10 * time.Minute
	}
	if cfg.RenewInterval <= 0 {
		cfg.RenewInterval = cfg.LeaseTimeout / 3
	}
	if cfg.StopTimeout <= 0 {
		cfg.StopTimeout = 30 * time.Second
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 10 * time.Minute
	}
	if cfg.FailedRetention <= 0 {
		cfg.FailedRetention = 24 * time.Hour
	}
	if cfg.WorkerID == "" {
		cfg.WorkerID = fmt.Sprintf("recording-finalizer-%d", time.Now().UnixNano())
	}
	return cfg
}

func (s *FinalizerService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	if s.recordings == nil || s.handoff == nil || s.storage == nil {
		return fmt.Errorf("recording finalizer dependencies are required")
	}
	stats, err := s.RunMaintenance(ctx)
	if err != nil {
		return err
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.started = true
	logger.Info("Starting recording finalizers",
		"workers", s.cfg.Workers,
		"poll_interval", s.cfg.PollInterval.String(),
		"lease_timeout", s.cfg.LeaseTimeout.String(),
		"recovered_sessions", stats.RecoveredClaims,
		"expired_sessions", stats.ExpiredSessions,
		"temporary_artifacts_removed", stats.TemporaryArtifactsRemoved,
		"session_dirs_removed", stats.SessionDirsRemoved,
	)
	for i := 0; i < s.cfg.Workers; i++ {
		workerID := fmt.Sprintf("%s-%d", s.cfg.WorkerID, i)
		s.wg.Add(1)
		go s.workerLoop(workerID)
	}
	s.wg.Add(1)
	go s.maintenanceLoop()
	return nil
}

func (s *FinalizerService) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	s.started = false
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.cfg.StopTimeout):
		return context.DeadlineExceeded
	}
}

func (s *FinalizerService) Notify() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *FinalizerService) RunMaintenance(ctx context.Context) (MaintenanceStats, error) {
	var stats MaintenanceStats
	now := time.Now()
	recovered, err := s.recordings.RecoverExpiredFinalizationClaims(ctx, now)
	if err != nil {
		return stats, err
	}
	stats.RecoveredClaims = recovered
	expired, err := s.recordings.ExpireAbandonedSessions(ctx, now)
	if err != nil {
		return stats, err
	}
	stats.ExpiredSessions = expired
	cleanupStats, err := s.cleanupArtifacts(ctx, now)
	if err != nil {
		return stats, err
	}
	stats.TemporaryArtifactsRemoved = cleanupStats.TemporaryArtifactsRemoved
	stats.SessionDirsRemoved = cleanupStats.SessionDirsRemoved
	if stats.RecoveredClaims > 0 || stats.ExpiredSessions > 0 || stats.TemporaryArtifactsRemoved > 0 || stats.SessionDirsRemoved > 0 {
		logger.Info("Recording maintenance completed",
			"recovered_claims", stats.RecoveredClaims,
			"expired_sessions", stats.ExpiredSessions,
			"temporary_artifacts_removed", stats.TemporaryArtifactsRemoved,
			"session_dirs_removed", stats.SessionDirsRemoved,
		)
	}
	return stats, nil
}

func (s *FinalizerService) maintenanceLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.cfg.CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.RunMaintenance(s.ctx); err != nil && !errors.Is(err, context.Canceled) {
				logger.Warn("Recording maintenance failed", "error", err)
			}
		}
	}
}

func (s *FinalizerService) workerLoop(workerID string) {
	defer s.wg.Done()
	for {
		if err := s.claimAndFinalize(workerID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Recording finalizer failed", "worker_id", workerID, "error", err)
		}
		timer := time.NewTimer(s.cfg.PollInterval)
		select {
		case <-s.ctx.Done():
			timer.Stop()
			return
		case <-s.wake:
			timer.Stop()
		case <-timer.C:
		}
	}
}

func (s *FinalizerService) claimAndFinalize(workerID string) error {
	session, err := s.recordings.ClaimNextFinalization(s.ctx, workerID, time.Now().Add(s.cfg.LeaseTimeout))
	if err != nil {
		return err
	}
	s.publishRecording(s.ctx, "recording.finalizing", session)
	jobCtx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	renewDone := make(chan struct{})
	go s.renewLease(jobCtx, workerID, session.ID, renewDone)
	err = s.finalize(jobCtx, workerID, session)
	close(renewDone)
	return err
}

func (s *FinalizerService) finalize(ctx context.Context, workerID string, session *models.RecordingSession) error {
	chunks, err := s.recordings.ListChunks(ctx, session.UserID, session.ID)
	if err != nil {
		return s.fail(ctx, workerID, session, "recording finalization failed", err)
	}
	if err := validateContiguousChunks(session, chunks); err != nil {
		return s.fail(ctx, workerID, session, "recording chunks are incomplete", err)
	}
	chunkPaths := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunkPaths = append(chunkPaths, chunk.Path)
	}
	rawPath, err := s.storage.BuildRaw(ctx, session.ID, chunkPaths)
	if err != nil {
		return s.fail(ctx, workerID, session, "recording finalization failed", err)
	}
	finalPath, err := s.storage.FinalPath(session.ID, session.MimeType)
	if err != nil {
		return s.fail(ctx, workerID, session, "recording finalization failed", err)
	}
	if err := s.media.Finalize(ctx, rawPath, finalPath); err != nil {
		return s.fail(ctx, workerID, session, "recording finalization failed", err)
	}

	file, transcription, err := s.buildHandoffRecords(ctx, session, finalPath)
	if err != nil {
		return s.fail(ctx, workerID, session, "recording finalization failed", err)
	}
	if err := s.handoff.CreateRecordingFileAndTranscription(ctx, file, transcription); err != nil {
		return s.fail(ctx, workerID, session, "recording finalization failed", err)
	}
	var transcriptionID *string
	if transcription != nil {
		transcriptionID = &transcription.ID
		if s.queue != nil {
			if err := s.queue.Enqueue(ctx, transcription.ID); err != nil {
				logger.Warn("Failed to enqueue recording transcription", "recording_id", session.ID, "transcription_id", transcription.ID, "error", err)
			}
		}
	}
	if err := s.recordings.CompleteFinalization(ctx, session.ID, workerID, file.ID, transcriptionID, time.Now()); err != nil {
		return err
	}
	if err := s.storage.RemoveTemporaryArtifacts(session.ID); err != nil {
		logger.Warn("Failed to remove recording temporary artifacts", "recording_id", session.ID, "error", err)
	} else if err := s.recordings.MarkTemporaryArtifactsCleaned(ctx, session.ID, time.Now()); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Warn("Failed to mark recording temporary artifacts cleaned", "recording_id", session.ID, "error", err)
	}
	completed := *session
	completed.Status = models.RecordingStatusReady
	completed.FileID = &file.ID
	completed.TranscriptionID = transcriptionID
	completed.Progress = 1
	completed.ProgressStage = "ready"
	s.publishRecording(ctx, "recording.ready", &completed)
	s.publishFile(ctx, file)
	if transcription != nil {
		s.publishTranscription(ctx, transcription)
	}
	return nil
}

func (s *FinalizerService) buildHandoffRecords(ctx context.Context, session *models.RecordingSession, finalPath string) (*models.TranscriptionJob, *models.TranscriptionJob, error) {
	title := "Recording"
	if session.Title != nil && strings.TrimSpace(*session.Title) != "" {
		title = strings.TrimSpace(*session.Title)
	}
	file := &models.TranscriptionJob{
		ID:               uuid.NewString(),
		UserID:           session.UserID,
		Title:            &title,
		Status:           models.StatusUploaded,
		AudioPath:        finalPath,
		SourceFileName:   "recording-" + session.ID + filepath.Ext(finalPath),
		SourceDurationMs: session.DurationMs,
	}
	if !session.AutoTranscribe {
		return file, nil, nil
	}
	params, err := s.resolveParams(ctx, session)
	if err != nil {
		return nil, nil, err
	}
	applyRecordingOptions(&params, session.TranscriptionOptionsJSON)
	fileID := file.ID
	transcription := &models.TranscriptionJob{
		ID:               uuid.NewString(),
		UserID:           session.UserID,
		Title:            &title,
		Status:           models.StatusUploaded,
		AudioPath:        finalPath,
		SourceFileName:   file.SourceFileName,
		SourceFileHash:   &fileID,
		SourceDurationMs: session.DurationMs,
		Language:         params.Language,
		Diarization:      params.Diarize,
		Parameters:       params,
	}
	return file, transcription, nil
}

func (s *FinalizerService) resolveParams(ctx context.Context, session *models.RecordingSession) (models.WhisperXParams, error) {
	if s.profiles == nil {
		return models.WhisperXParams{}, nil
	}
	if session.ProfileID != nil && *session.ProfileID != "" {
		profile, err := s.profiles.FindByIDForUser(ctx, *session.ProfileID, session.UserID)
		if err != nil {
			return models.WhisperXParams{}, err
		}
		return profile.Parameters, nil
	}
	profile, err := s.profiles.FindDefaultByUser(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.WhisperXParams{}, nil
		}
		return models.WhisperXParams{}, err
	}
	return profile.Parameters, nil
}

type recordingOptions struct {
	Language    string `json:"language"`
	Diarization *bool  `json:"diarization"`
}

func applyRecordingOptions(params *models.WhisperXParams, raw string) {
	var opts recordingOptions
	if err := json.Unmarshal([]byte(raw), &opts); err != nil {
		return
	}
	if opts.Language != "" {
		params.Language = &opts.Language
	}
	if opts.Diarization != nil {
		params.Diarize = *opts.Diarization
	}
}

func validateContiguousChunks(session *models.RecordingSession, chunks []models.RecordingChunk) error {
	if session.ExpectedFinalIndex == nil {
		return fmt.Errorf("expected final chunk index is required")
	}
	expectedCount := *session.ExpectedFinalIndex + 1
	if expectedCount <= 0 || len(chunks) != expectedCount {
		return fmt.Errorf("expected %d chunks, got %d", expectedCount, len(chunks))
	}
	for i, chunk := range chunks {
		if chunk.ChunkIndex != i {
			return fmt.Errorf("missing recording chunk %d", i)
		}
	}
	return nil
}

func (s *FinalizerService) fail(ctx context.Context, workerID string, session *models.RecordingSession, publicMessage string, cause error) error {
	_ = s.recordings.FailFinalization(context.Background(), session.ID, workerID, publicMessage, time.Now())
	failed := *session
	failed.Status = models.RecordingStatusFailed
	failed.ProgressStage = "failed"
	failed.LastError = &publicMessage
	s.publishRecording(context.Background(), "recording.failed", &failed)
	return cause
}

func (s *FinalizerService) cleanupArtifacts(ctx context.Context, now time.Time) (MaintenanceStats, error) {
	var stats MaintenanceStats
	candidates, err := s.recordings.ListArtifactCleanupCandidates(ctx, now, s.cfg.FailedRetention, 100)
	if err != nil {
		return stats, err
	}
	for _, session := range candidates {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		removeErr := s.removeArtifactsForSession(session)
		if removeErr != nil {
			logger.Warn("Failed to clean recording artifacts", "recording_id", session.ID, "status", session.Status, "error", removeErr)
			continue
		}
		if err := s.recordings.MarkTemporaryArtifactsCleaned(ctx, session.ID, now); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return stats, err
		}
		switch session.Status {
		case models.RecordingStatusReady:
			stats.TemporaryArtifactsRemoved++
		default:
			stats.SessionDirsRemoved++
		}
	}
	return stats, nil
}

func (s *FinalizerService) removeArtifactsForSession(session models.RecordingSession) error {
	switch session.Status {
	case models.RecordingStatusReady:
		return s.storage.RemoveTemporaryArtifacts(session.ID)
	case models.RecordingStatusCanceled, models.RecordingStatusExpired, models.RecordingStatusFailed:
		return s.storage.RemoveSession(session.ID)
	default:
		return nil
	}
}

func (s *FinalizerService) renewLease(ctx context.Context, workerID, sessionID string, done <-chan struct{}) {
	ticker := time.NewTicker(s.cfg.RenewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			if err := s.recordings.RenewFinalizationClaim(ctx, sessionID, workerID, time.Now().Add(s.cfg.LeaseTimeout)); err != nil {
				logger.Warn("Failed to renew recording finalizer claim", "session_id", sessionID, "worker_id", workerID, "error", err)
			}
		}
	}
}

func (s *FinalizerService) publishRecording(ctx context.Context, name string, session *models.RecordingSession) {
	if s.events == nil || session == nil {
		return
	}
	event := Event{
		Name:        name,
		UserID:      session.UserID,
		RecordingID: PublicID(session.ID),
		Status:      session.Status,
		Stage:       session.ProgressStage,
		Progress:    session.Progress,
	}
	if session.FileID != nil {
		event.FileID = "file_" + *session.FileID
	}
	if session.TranscriptionID != nil {
		event.TranscriptionID = "tr_" + *session.TranscriptionID
	}
	s.events.PublishRecordingEvent(ctx, event)
}

func (s *FinalizerService) publishFile(ctx context.Context, file *models.TranscriptionJob) {
	if s.events == nil || file == nil {
		return
	}
	s.events.PublishFileEvent(ctx, "file.ready", map[string]any{"id": "file_" + file.ID, "kind": "audio", "status": "ready"})
}

func (s *FinalizerService) publishTranscription(ctx context.Context, transcription *models.TranscriptionJob) {
	if s.events == nil || transcription == nil {
		return
	}
	s.events.PublishTranscriptionEvent(ctx, "transcription.created", "tr_"+transcription.ID, map[string]any{"id": "tr_" + transcription.ID, "file_id": fileIDForTranscription(transcription), "status": string(transcription.Status)})
}

func fileIDForTranscription(job *models.TranscriptionJob) string {
	if job.SourceFileHash != nil && *job.SourceFileHash != "" {
		return "file_" + *job.SourceFileHash
	}
	return "file_" + job.ID
}

type FFmpegFinalizer struct{}

func (FFmpegFinalizer) Finalize(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", inputPath, "-vn", "-c:a", "copy", outputPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg finalization failed: %s", sanitizeFinalizerOutput(string(output)))
	}
	return nil
}

func sanitizeFinalizerOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return "ffmpeg failed"
	}
	if len(output) > 200 {
		output = output[:200]
	}
	return strings.ReplaceAll(output, "\n", " ")
}
