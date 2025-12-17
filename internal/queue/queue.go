package queue

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/pkg/logger"
)

// RunningJob tracks both context cancellation and OS process
type RunningJob struct {
	Cancel  context.CancelFunc
	Process *exec.Cmd
}

// TaskQueue manages transcription job processing
type TaskQueue struct {
	minWorkers     int
	maxWorkers     int
	currentWorkers int64 // Use atomic for thread-safe access
	jobChannel     chan string
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	processor      JobProcessor
	runningJobs    map[string]*RunningJob
	jobsMutex      sync.RWMutex
	autoScale      bool
	lastScaleTime  time.Time
	jobRepo        repository.JobRepository
}

// JobProcessor defines the interface for processing jobs
type JobProcessor interface {
	ProcessJob(ctx context.Context, jobID string) error
	ProcessJobWithProcess(ctx context.Context, jobID string, registerProcess func(*exec.Cmd)) error
}

// MultiTrackJobProcessor extends JobProcessor with multi-track specific methods
type MultiTrackJobProcessor interface {
	JobProcessor
	TerminateMultiTrackJob(jobID string) error
	IsMultiTrackJob(jobID string) bool
}

// getOptimalWorkerCount calculates optimal worker count based on system resources
func getOptimalWorkerCount() (min, max int) {
	numCPU := runtime.NumCPU()

	// Check for environment variable override
	if workerStr := os.Getenv("QUEUE_WORKERS"); workerStr != "" {
		if workers, err := strconv.Atoi(workerStr); err == nil && workers > 0 {
			return workers, workers // Fixed worker count
		}
	}

	// For transcription workloads, we typically want fewer workers than CPUs
	// since each job is CPU and I/O intensive
	if numCPU <= 2 {
		return 1, 2
	}
	if numCPU <= 4 {
		return 1, 3
	}
	if numCPU <= 8 {
		return 2, 4
	}
	return 2, 6 // Cap at 6 for very high CPU systems
}

// NewTaskQueue creates a new task queue with auto-scaling capabilities
func NewTaskQueue(legacyWorkers int, processor JobProcessor, jobRepo repository.JobRepository) *TaskQueue {
	ctx, cancel := context.WithCancel(context.Background())

	// Calculate optimal worker counts, fallback to legacy parameter
	min, max := getOptimalWorkerCount()
	if legacyWorkers > 0 {
		min = legacyWorkers
		max = legacyWorkers
	}

	// Check if auto-scaling should be enabled
	autoScale := os.Getenv("QUEUE_AUTO_SCALE") != "false"
	if min == max {
		autoScale = false // Disable auto-scaling if min == max
	}

	return &TaskQueue{
		minWorkers:     min,
		maxWorkers:     max,
		currentWorkers: int64(min),
		jobChannel:     make(chan string, 200), // Increased buffer for better throughput
		ctx:            ctx,
		cancel:         cancel,
		processor:      processor,
		runningJobs:    make(map[string]*RunningJob),
		autoScale:      autoScale,
		lastScaleTime:  time.Now(),
		jobRepo:        jobRepo,
	}
}

// Start starts the task queue workers
func (tq *TaskQueue) Start() {
	workers := int(atomic.LoadInt64(&tq.currentWorkers))
	logger.Debug("Starting task queue",
		"workers", workers,
		"min_workers", tq.minWorkers,
		"max_workers", tq.maxWorkers,
		"auto_scale", tq.autoScale)

	// Reset any zombie jobs from previous runs synchronously before starting workers
	tq.ResetZombieJobs()

	// One-time recovery: enqueue any pending jobs left from previous server run
	// This is NOT a polling mechanism - it only runs once at startup
	tq.recoverPendingJobs()

	// Start initial workers
	for i := 0; i < workers; i++ {
		tq.wg.Add(1)
		go tq.worker(i)
	}

	// Start auto-scaling monitor if enabled
	if tq.autoScale {
		tq.wg.Add(1)
		go tq.autoScaler()
	}
}

// Stop stops the task queue
func (tq *TaskQueue) Stop() {
	logger.Debug("Stopping task queue")
	logger.Debug("Stopping task queue")
	tq.cancel()
	// Do not close jobChannel here as it causes panics in EnqueueJob
	// The channel will be garbage collected when the queue is no longer referenced
	tq.wg.Wait()
	logger.Debug("Task queue stopped")
}

// EnqueueJob adds a job to the queue
func (tq *TaskQueue) EnqueueJob(jobID string) error {
	// Check if queue is already shut down
	select {
	case <-tq.ctx.Done():
		return fmt.Errorf("queue is shutting down")
	default:
	}

	select {
	case tq.jobChannel <- jobID:
		return nil
	case <-tq.ctx.Done():
		return fmt.Errorf("queue is shutting down")
	default:
		return fmt.Errorf("queue is full")
	}
}

// worker processes jobs from the channel
func (tq *TaskQueue) worker(id int) {
	defer tq.wg.Done()

	logger.Debug("Worker started", "worker_id", id)

	for {
		select {
		case jobID, ok := <-tq.jobChannel:
			if !ok {
				logger.Debug("Worker stopped", "worker_id", id)
				return
			}

			logger.WorkerOperation(id, jobID, "start")

			// Update job status to processing
			if err := tq.updateJobStatus(jobID, models.StatusProcessing); err != nil {
				logger.Error("Failed to update job status", "worker_id", id, "job_id", jobID, "error", err)
				continue
			}

			// Create context for this job and track it
			jobCtx, jobCancel := context.WithCancel(tq.ctx)
			runningJob := &RunningJob{
				Cancel:  jobCancel,
				Process: nil, // Will be set by registerProcess callback
			}

			tq.jobsMutex.Lock()
			tq.runningJobs[jobID] = runningJob
			tq.jobsMutex.Unlock()

			// Register process callback
			registerProcess := func(cmd *exec.Cmd) {
				tq.jobsMutex.Lock()
				if job, exists := tq.runningJobs[jobID]; exists {
					job.Process = cmd
				}
				tq.jobsMutex.Unlock()
			}

			// Process the job with process registration
			err := tq.processor.ProcessJobWithProcess(jobCtx, jobID, registerProcess)

			// Remove job from running jobs
			tq.jobsMutex.Lock()
			delete(tq.runningJobs, jobID)
			tq.jobsMutex.Unlock()

			// Handle result
			if err != nil {
				if jobCtx.Err() == context.Canceled {
					logger.Info("Job cancelled", "worker_id", id, "job_id", jobID)
					if err := tq.updateJobStatus(jobID, models.StatusFailed); err != nil {
						logger.Error("Failed to update job status", "job_id", jobID, "error", err)
					}
					if err := tq.updateJobError(jobID, "Job was cancelled by user"); err != nil {
						logger.Error("Failed to update job error", "job_id", jobID, "error", err)
					}
				} else {
					logger.Error("Job processing failed", "worker_id", id, "job_id", jobID, "error", err)
					if err := tq.updateJobStatus(jobID, models.StatusFailed); err != nil {
						logger.Error("Failed to update job status", "job_id", jobID, "error", err)
					}
					if err := tq.updateJobError(jobID, err.Error()); err != nil {
						logger.Error("Failed to update job error", "job_id", jobID, "error", err)
					}
				}
			} else {
				logger.Debug("Job processed successfully", "worker_id", id, "job_id", jobID)
				if err := tq.updateJobStatus(jobID, models.StatusCompleted); err != nil {
					logger.Error("Failed to update job status", "job_id", jobID, "error", err)
				}
			}

		case <-tq.ctx.Done():
			logger.Debug("Worker stopped", "worker_id", id, "reason", "context_cancelled")
			return
		}
	}
}

// KillJob aggressively terminates a running job
func (tq *TaskQueue) KillJob(jobID string) error {
	tq.jobsMutex.Lock()
	defer tq.jobsMutex.Unlock()

	runningJob, exists := tq.runningJobs[jobID]
	if !exists {
		// If job is not in memory but exists in DB as processing, it's a zombie
		// We should still mark it as failed in DB
		logger.Warn("Job not found in running jobs map, checking DB status", "job_id", jobID)

		job, err := tq.jobRepo.FindByID(context.Background(), jobID)
		if err != nil {
			return fmt.Errorf("job %s not found: %v", jobID, err)
		}

		if job.Status == models.StatusProcessing {
			logger.Info("Found zombie job in DB, marking as failed", "job_id", jobID)
			if err := tq.updateJobStatus(jobID, models.StatusFailed); err != nil {
				logger.Error("Failed to update zombie job status", "job_id", jobID, "error", err)
			}
			if err := tq.updateJobError(jobID, "Job was forcefully terminated by user (zombie process)"); err != nil {
				logger.Error("Failed to update zombie job error", "job_id", jobID, "error", err)
			}
			return nil
		}

		return fmt.Errorf("job %s is not currently running", jobID)
	}

	logger.Info("Killing job", "job_id", jobID)

	// Check if this is a multi-track job and handle accordingly
	if mtProcessor, ok := tq.processor.(MultiTrackJobProcessor); ok && mtProcessor.IsMultiTrackJob(jobID) {
		logger.Debug("Terminating multi-track job", "job_id", jobID)

		// Terminate all individual track jobs
		if err := mtProcessor.TerminateMultiTrackJob(jobID); err != nil {
			logger.Error("Failed to terminate multi-track job", "job_id", jobID, "error", err)
		}
	}

	// First, try to kill the OS process group (or process on non-Unix)
	if runningJob.Process != nil && runningJob.Process.Process != nil {
		logger.Debug("Terminating process tree", "pid", runningJob.Process.Process.Pid, "job_id", jobID)
		if err := killProcessTree(runningJob.Process.Process); err != nil {
			log.Printf("Failed to terminate process tree for job %s: %v, trying direct kill()", jobID, err)
			_ = runningJob.Process.Process.Kill()
		}
	}

	// Also cancel the context for cleanup
	runningJob.Cancel()

	// Immediately update job status without waiting for process to finish
	go func() {
		_ = tq.updateJobStatus(jobID, models.StatusFailed)
		_ = tq.updateJobError(jobID, "Job was forcefully terminated by user")
	}()

	return nil
}

// IsJobRunning checks if a job is currently being processed
func (tq *TaskQueue) IsJobRunning(jobID string) bool {
	tq.jobsMutex.RLock()
	defer tq.jobsMutex.RUnlock()

	_, exists := tq.runningJobs[jobID]
	return exists
}

// updateJobStatus updates the status of a job
func (tq *TaskQueue) updateJobStatus(jobID string, status models.JobStatus) error {
	return tq.jobRepo.UpdateStatus(context.Background(), jobID, status)
}

// updateJobError updates the error message of a job
func (tq *TaskQueue) updateJobError(jobID string, errorMsg string) error {
	return tq.jobRepo.UpdateError(context.Background(), jobID, errorMsg)
}

// GetJobStatus gets the status of a job
func (tq *TaskQueue) GetJobStatus(jobID string) (*models.TranscriptionJob, error) {
	return tq.jobRepo.FindByID(context.Background(), jobID)
}

// autoScaler monitors queue load and adjusts worker count
func (tq *TaskQueue) autoScaler() {
	defer tq.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	log.Println("Auto-scaler started")

	for {
		select {
		case <-ticker.C:
			tq.checkAndScale()
		case <-tq.ctx.Done():
			log.Println("Auto-scaler stopped")
			return
		}
	}
}

// checkAndScale evaluates current load and adjusts worker count
func (tq *TaskQueue) checkAndScale() {
	// Prevent too frequent scaling
	if time.Since(tq.lastScaleTime) < 1*time.Minute {
		return
	}

	queueSize := len(tq.jobChannel)
	currentWorkers := int(atomic.LoadInt64(&tq.currentWorkers))

	tq.jobsMutex.RLock()
	runningJobsCount := len(tq.runningJobs)
	tq.jobsMutex.RUnlock()

	// Scale up if queue is building up and we have capacity
	if queueSize > 10 && currentWorkers < tq.maxWorkers {
		newWorkerCount := currentWorkers + 1
		log.Printf("Scaling up workers: %d -> %d (queue size: %d)", currentWorkers, newWorkerCount, queueSize)

		atomic.StoreInt64(&tq.currentWorkers, int64(newWorkerCount))
		tq.wg.Add(1)
		go tq.worker(newWorkerCount - 1)
		tq.lastScaleTime = time.Now()

		// Scale down if queue is empty and minimal jobs running
	} else if queueSize == 0 && runningJobsCount <= 1 && currentWorkers > tq.minWorkers {
		newWorkerCount := currentWorkers - 1
		log.Printf("Scaling down workers: %d -> %d (queue size: %d, running: %d)",
			currentWorkers, newWorkerCount, queueSize, runningJobsCount)

		atomic.StoreInt64(&tq.currentWorkers, int64(newWorkerCount))
		tq.lastScaleTime = time.Now()

		// Note: We don't actively stop workers here. They will naturally exit
		// when no more jobs are available and the queue empties.
	}
}

// GetQueueStats returns queue statistics
func (tq *TaskQueue) GetQueueStats() map[string]interface{} {
	ctx := context.Background()
	pendingCount, _ := tq.jobRepo.CountByStatus(ctx, models.StatusPending)
	processingCount, _ := tq.jobRepo.CountByStatus(ctx, models.StatusProcessing)
	completedCount, _ := tq.jobRepo.CountByStatus(ctx, models.StatusCompleted)
	failedCount, _ := tq.jobRepo.CountByStatus(ctx, models.StatusFailed)

	tq.jobsMutex.RLock()
	runningJobsCount := len(tq.runningJobs)
	tq.jobsMutex.RUnlock()

	return map[string]interface{}{
		"queue_size":      len(tq.jobChannel),
		"queue_capacity":  cap(tq.jobChannel),
		"current_workers": int(atomic.LoadInt64(&tq.currentWorkers)),
		"min_workers":     tq.minWorkers,
		"max_workers":     tq.maxWorkers,
		"auto_scale":      tq.autoScale,
		"running_jobs":    runningJobsCount,
		"pending_jobs":    pendingCount,
		"processing_jobs": processingCount,
		"completed_jobs":  completedCount,
		"failed_jobs":     failedCount,
	}
}

// ResetZombieJobs finds jobs stuck in processing state from previous runs and marks them as failed
func (tq *TaskQueue) ResetZombieJobs() {
	// Find all jobs with status "processing"
	zombieJobs, err := tq.jobRepo.FindByStatus(context.Background(), models.StatusProcessing)
	if err != nil {
		logger.Error("Failed to scan for zombie jobs", "error", err)
		return
	}

	if len(zombieJobs) == 0 {
		return
	}

	logger.Info("Found zombie jobs from previous run", "count", len(zombieJobs))

	for _, job := range zombieJobs {
		logger.Info("Resetting zombie job", "job_id", job.ID)

		// Mark as failed
		if err := tq.updateJobStatus(job.ID, models.StatusFailed); err != nil {
			logger.Error("Failed to update zombie job status", "job_id", job.ID, "error", err)
			continue
		}

		// Update error message
		if err := tq.updateJobError(job.ID, "Job interrupted by server restart"); err != nil {
			logger.Error("Failed to update zombie job error message", "job_id", job.ID, "error", err)
		}
	}
}

// recoverPendingJobs enqueues pending jobs from previous server runs
// This runs ONCE at startup, not repeatedly like the old scanner
func (tq *TaskQueue) recoverPendingJobs() {
	pendingJobs, err := tq.jobRepo.FindByStatus(context.Background(), models.StatusPending)
	if err != nil {
		logger.Error("Failed to scan for pending jobs during startup recovery", "error", err)
		return
	}

	if len(pendingJobs) == 0 {
		return
	}

	logger.Info("Recovering pending jobs from previous server run", "count", len(pendingJobs))

	for _, job := range pendingJobs {
		select {
		case tq.jobChannel <- job.ID:
			logger.Debug("Recovered pending job", "job_id", job.ID)
		default:
			logger.Warn("Queue full during startup recovery, job will remain pending", "job_id", job.ID)
		}
	}
}
