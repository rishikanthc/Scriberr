package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"scriberr/internal/models"
	"scriberr/pkg/logger"

	"gorm.io/gorm"
)

var (
	ErrQueueStopped  = errors.New("transcription queue is stopped")
	ErrStateConflict = errors.New("transcription state conflict")
)

type QueueService interface {
	Enqueue(ctx context.Context, jobID string) error
	Cancel(ctx context.Context, userID uint, jobID string) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Stats(ctx context.Context, userID uint) (QueueStats, error)
}

type Processor interface {
	Process(ctx context.Context, job *models.TranscriptionJob) (ProcessResult, error)
}

type EventPublisher interface {
	PublishStatus(ctx context.Context, event StatusEvent)
}

type CompletionObserver interface {
	EnqueueForTranscription(ctx context.Context, job *models.TranscriptionJob) error
}

type CompletionObservers []CompletionObserver

func (observers CompletionObservers) EnqueueForTranscription(ctx context.Context, job *models.TranscriptionJob) error {
	var firstErr error
	for _, observer := range observers {
		if observer == nil {
			continue
		}
		if err := observer.EnqueueForTranscription(ctx, job); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

type StatusEvent struct {
	Name     string
	JobID    string
	FileID   string
	UserID   uint
	Stage    string
	Progress float64
	Status   models.JobStatus
}

type ProcessResult struct {
	Status         models.JobStatus
	ExecutionID    string
	TranscriptJSON string
	OutputJSONPath *string
	ErrorMessage   string
	CompletedAt    time.Time
	FailedAt       time.Time
}

type QueueStats struct {
	Queued     int64 `json:"queued"`
	Processing int64 `json:"processing"`
	Completed  int64 `json:"completed"`
	Failed     int64 `json:"failed"`
	Canceled   int64 `json:"canceled"`
	Running    int64 `json:"running"`
}

type Repository interface {
	RecoverOrphanedProcessing(ctx context.Context, now time.Time) (int64, error)
	EnqueueTranscription(ctx context.Context, jobID string, now time.Time) error
	FindByID(ctx context.Context, id interface{}) (*models.TranscriptionJob, error)
	CancelTranscription(ctx context.Context, jobID string, canceledAt time.Time) error
	CountStatusesByUser(ctx context.Context, userID uint) (map[models.JobStatus]int64, error)
	ClaimNextTranscription(ctx context.Context, workerID string, leaseUntil time.Time) (*models.TranscriptionJob, error)
	CompleteTranscription(ctx context.Context, jobID string, transcriptJSON string, outputPath *string, completedAt time.Time) error
	CompleteTranscriptionClaimed(ctx context.Context, jobID, workerID, executionID string, transcriptJSON string, outputPath *string, completedAt time.Time) error
	FailTranscription(ctx context.Context, jobID string, message string, failedAt time.Time) error
	FailTranscriptionClaimed(ctx context.Context, jobID, workerID, executionID string, message string, failedAt time.Time) error
	CancelTranscriptionClaimed(ctx context.Context, jobID, workerID, executionID string, canceledAt time.Time) error
	RenewClaim(ctx context.Context, jobID, workerID string, leaseUntil time.Time) error
}

type Config struct {
	Workers       int
	PollInterval  time.Duration
	LeaseTimeout  time.Duration
	RenewInterval time.Duration
	StopTimeout   time.Duration
	WorkerID      string
}

type Service struct {
	repo       Repository
	processor  Processor
	events     EventPublisher
	completion CompletionObserver
	cfg        Config

	mu      sync.Mutex
	started bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	wake    chan struct{}
	running map[string]runningJob
}

type runningJob struct {
	userID uint
	cancel context.CancelFunc
}

func NewService(repo Repository, processor Processor, cfg Config) *Service {
	cfg = normalizeConfig(cfg)
	return &Service{
		repo:      repo,
		processor: processor,
		cfg:       cfg,
		wake:      make(chan struct{}, 1),
		running:   make(map[string]runningJob),
	}
}

func (s *Service) SetEventPublisher(events EventPublisher) {
	s.events = events
}

func (s *Service) SetCompletionObserver(completion CompletionObserver) {
	s.completion = completion
}

func normalizeConfig(cfg Config) Config {
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
		if cfg.RenewInterval <= 0 {
			cfg.RenewInterval = time.Second
		}
	}
	if cfg.StopTimeout <= 0 {
		cfg.StopTimeout = 30 * time.Second
	}
	if cfg.WorkerID == "" {
		cfg.WorkerID = fmt.Sprintf("worker-%d", time.Now().UnixNano())
	}
	return cfg
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	if s.repo == nil {
		return fmt.Errorf("worker repository is required")
	}
	if s.processor == nil {
		return fmt.Errorf("worker processor is required")
	}
	recovered, err := s.repo.RecoverOrphanedProcessing(ctx, time.Now())
	if err != nil {
		return err
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.started = true
	logger.Info("Starting transcription workers",
		"workers", s.cfg.Workers,
		"poll_interval", s.cfg.PollInterval.String(),
		"lease_timeout", s.cfg.LeaseTimeout.String(),
		"recovered_jobs", recovered,
	)
	for i := 0; i < s.cfg.Workers; i++ {
		workerID := fmt.Sprintf("%s-%d", s.cfg.WorkerID, i)
		s.wg.Add(1)
		go s.workerLoop(workerID)
	}
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	s.started = false
	s.mu.Unlock()

	logger.Info("Stopping transcription workers")
	if cancel != nil {
		cancel()
	}
	s.cancelRunning()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		logger.Info("Transcription workers stopped")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.cfg.StopTimeout):
		return context.DeadlineExceeded
	}
}

func (s *Service) Enqueue(ctx context.Context, jobID string) error {
	if !s.isStarted() {
		return ErrQueueStopped
	}
	if err := s.repo.EnqueueTranscription(ctx, jobID, time.Now()); err != nil {
		return err
	}
	logger.Info("Transcription job enqueued", "job_id", jobID)
	s.notify()
	return nil
}

func (s *Service) Cancel(ctx context.Context, userID uint, jobID string) error {
	job, err := s.repo.FindByID(ctx, jobID)
	if err != nil {
		return err
	}
	if job.UserID != userID {
		return gorm.ErrRecordNotFound
	}
	switch job.Status {
	case models.StatusCompleted, models.StatusFailed, models.StatusStopped, models.StatusCanceled:
		return ErrStateConflict
	case models.StatusProcessing:
		if cancel := s.runningCancel(jobID); cancel != nil {
			logger.Info("Canceling running transcription job", "job_id", jobID)
			cancel()
			if err := s.repo.CancelTranscription(ctx, jobID, time.Now()); err != nil {
				return err
			}
			s.publishTerminalStatus(context.Background(), job, models.StatusStopped)
			return nil
		}
	}
	logger.Info("Canceling transcription job", "job_id", jobID, "status", job.Status)
	if err := s.repo.CancelTranscription(ctx, jobID, time.Now()); err != nil {
		return err
	}
	s.publishTerminalStatus(context.Background(), job, models.StatusStopped)
	return nil
}

func (s *Service) Stats(ctx context.Context, userID uint) (QueueStats, error) {
	counts, err := s.repo.CountStatusesByUser(ctx, userID)
	if err != nil {
		return QueueStats{}, err
	}
	stats := QueueStats{
		Queued:     counts[models.StatusPending],
		Processing: counts[models.StatusProcessing],
		Completed:  counts[models.StatusCompleted],
		Failed:     counts[models.StatusFailed],
		Canceled:   counts[models.StatusStopped] + counts[models.StatusCanceled],
		Running:    s.runningCountForUser(userID),
	}
	return stats, nil
}

func (s *Service) workerLoop(workerID string) {
	defer s.wg.Done()
	logger.Info("Transcription worker started", "worker_id", workerID)
	defer logger.Info("Transcription worker stopped", "worker_id", workerID)

	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.wake:
		case <-timer.C:
		}
		if err := s.claimAndProcess(workerID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("Transcription worker claim/process failed", "worker_id", workerID, "error", err)
		}
		timer.Reset(s.cfg.PollInterval)
	}
}

func (s *Service) claimAndProcess(workerID string) error {
	job, err := s.repo.ClaimNextTranscription(s.ctx, workerID, time.Now().Add(s.cfg.LeaseTimeout))
	if err != nil {
		return err
	}
	jobCtx, cancel := context.WithCancel(s.ctx)
	s.setRunning(job.ID, job.UserID, cancel)
	defer func() {
		cancel()
		s.clearRunning(job.ID)
	}()

	renewDone := make(chan struct{})
	go s.renewLease(jobCtx, workerID, job.ID, renewDone)
	result, processErr := s.processor.Process(jobCtx, job)
	close(renewDone)

	if errors.Is(jobCtx.Err(), context.Canceled) || errors.Is(processErr, context.Canceled) {
		if err := s.cancelClaimed(context.Background(), job, workerID, result.ExecutionID, time.Now()); err != nil {
			return err
		}
		s.publishTerminalStatus(context.Background(), job, models.StatusStopped)
		return nil
	}
	if processErr != nil {
		message := processErr.Error()
		if result.ErrorMessage != "" {
			message = result.ErrorMessage
		}
		if err := s.failClaimed(context.Background(), job, workerID, result.ExecutionID, message, nonZeroTime(result.FailedAt)); err != nil {
			return err
		}
		s.publishTerminalStatus(context.Background(), job, models.StatusFailed)
		return nil
	}
	switch result.Status {
	case "", models.StatusCompleted:
		if err := s.completeClaimed(context.Background(), job, workerID, result.ExecutionID, result.TranscriptJSON, result.OutputJSONPath, nonZeroTime(result.CompletedAt)); err != nil {
			return err
		}
		completedJob := *job
		completedJob.Status = models.StatusCompleted
		completedJob.Transcript = &result.TranscriptJSON
		if err := s.enqueueCompletionWork(context.Background(), &completedJob); err != nil {
			logger.Warn("Failed to enqueue transcription completion work", "job_id", job.ID, "error", err)
		}
		s.publishTerminalStatus(context.Background(), job, models.StatusCompleted)
		return nil
	case models.StatusFailed:
		if err := s.failClaimed(context.Background(), job, workerID, result.ExecutionID, result.ErrorMessage, nonZeroTime(result.FailedAt)); err != nil {
			return err
		}
		s.publishTerminalStatus(context.Background(), job, models.StatusFailed)
		return nil
	case models.StatusStopped, models.StatusCanceled:
		if err := s.cancelClaimed(context.Background(), job, workerID, result.ExecutionID, nonZeroTime(result.FailedAt)); err != nil {
			return err
		}
		s.publishTerminalStatus(context.Background(), job, models.StatusStopped)
		return nil
	default:
		return fmt.Errorf("unsupported worker processor result status %q", result.Status)
	}
}

func (s *Service) completeClaimed(ctx context.Context, job *models.TranscriptionJob, workerID, executionID, transcriptJSON string, outputPath *string, completedAt time.Time) error {
	if executionID == "" {
		return s.repo.CompleteTranscription(ctx, job.ID, transcriptJSON, outputPath, completedAt)
	}
	return s.repo.CompleteTranscriptionClaimed(ctx, job.ID, workerID, executionID, transcriptJSON, outputPath, completedAt)
}

func (s *Service) failClaimed(ctx context.Context, job *models.TranscriptionJob, workerID, executionID, message string, failedAt time.Time) error {
	if executionID == "" {
		return s.repo.FailTranscription(ctx, job.ID, message, failedAt)
	}
	return s.repo.FailTranscriptionClaimed(ctx, job.ID, workerID, executionID, message, failedAt)
}

func (s *Service) cancelClaimed(ctx context.Context, job *models.TranscriptionJob, workerID, executionID string, canceledAt time.Time) error {
	if executionID == "" {
		return s.repo.CancelTranscription(ctx, job.ID, canceledAt)
	}
	return s.repo.CancelTranscriptionClaimed(ctx, job.ID, workerID, executionID, canceledAt)
}

func (s *Service) enqueueCompletionWork(ctx context.Context, job *models.TranscriptionJob) error {
	if s.completion == nil {
		return nil
	}
	return s.completion.EnqueueForTranscription(ctx, job)
}

func (s *Service) publishTerminalStatus(ctx context.Context, job *models.TranscriptionJob, status models.JobStatus) {
	if s.events == nil || job == nil {
		return
	}
	stage := string(status)
	name := "transcription.progress"
	progress := job.Progress
	switch status {
	case models.StatusCompleted:
		name = "transcription.completed"
		stage = "completed"
		progress = 1
	case models.StatusFailed:
		name = "transcription.failed"
		stage = "failed"
	case models.StatusStopped, models.StatusCanceled:
		name = "transcription.stopped"
		stage = "stopped"
		status = models.StatusStopped
	}
	s.events.PublishStatus(ctx, StatusEvent{
		Name:     name,
		JobID:    job.ID,
		FileID:   fileIDForJob(job),
		UserID:   job.UserID,
		Stage:    stage,
		Progress: progress,
		Status:   status,
	})
}

func fileIDForJob(job *models.TranscriptionJob) string {
	if job == nil {
		return ""
	}
	if job.SourceFileHash != nil && *job.SourceFileHash != "" {
		return "file_" + *job.SourceFileHash
	}
	return "file_" + job.ID
}

func (s *Service) renewLease(ctx context.Context, workerID string, jobID string, done <-chan struct{}) {
	ticker := time.NewTicker(s.cfg.RenewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			if err := s.repo.RenewClaim(ctx, jobID, workerID, time.Now().Add(s.cfg.LeaseTimeout)); err != nil {
				logger.Warn("Failed to renew transcription job lease", "worker_id", workerID, "job_id", jobID, "error", err)
			}
		}
	}
}

func (s *Service) notify() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Service) isStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

func (s *Service) setRunning(jobID string, userID uint, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running[jobID] = runningJob{userID: userID, cancel: cancel}
}

func (s *Service) clearRunning(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, jobID)
}

func (s *Service) runningCancel(jobID string) context.CancelFunc {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running[jobID].cancel
}

func (s *Service) cancelRunning() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, job := range s.running {
		job.cancel()
	}
}

func (s *Service) runningCountForUser(userID uint) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var count int64
	for _, job := range s.running {
		if job.userID == userID {
			count++
		}
	}
	return count
}

func nonZeroTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now()
	}
	return value
}
