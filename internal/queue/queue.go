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

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/pkg/logger"
)

// RunningJob tracks both context cancellation and OS process
type RunningJob struct {
	Cancel  context.CancelFunc
	Process *exec.Cmd
}

// TaskQueue manages transcription job processing
type TaskQueue struct {
	minWorkers    int
	maxWorkers    int
	currentWorkers int64 // Use atomic for thread-safe access
	jobChannel    chan string
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	processor     JobProcessor
	runningJobs   map[string]*RunningJob
	jobsMutex     sync.RWMutex
	workerMutex   sync.Mutex
	autoScale     bool
	lastScaleTime time.Time
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
	} else if numCPU <= 4 {
		return 1, 3
	} else if numCPU <= 8 {
		return 2, 4
	} else {
		return 2, 6 // Cap at 6 for very high CPU systems
	}
}

// NewTaskQueue creates a new task queue with auto-scaling capabilities
func NewTaskQueue(legacyWorkers int, processor JobProcessor) *TaskQueue {
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
	}
}

// Start starts the task queue workers
func (tq *TaskQueue) Start() {
	workers := int(atomic.LoadInt64(&tq.currentWorkers))
	logger.Info("Starting task queue", 
		"workers", workers, 
		"min_workers", tq.minWorkers, 
		"max_workers", tq.maxWorkers, 
		"auto_scale", tq.autoScale)

	// Start initial workers
	for i := 0; i < workers; i++ {
		tq.wg.Add(1)
		go tq.worker(i)
	}

	// Start the job scanner
	tq.wg.Add(1)
	go tq.jobScanner()

	// Start auto-scaling monitor if enabled
	if tq.autoScale {
		tq.wg.Add(1)
		go tq.autoScaler()
	}
}

// Stop stops the task queue
func (tq *TaskQueue) Stop() {
	log.Println("Stopping task queue...")
	tq.cancel()
	close(tq.jobChannel)
	tq.wg.Wait()
	log.Println("Task queue stopped")
}

// EnqueueJob adds a job to the queue
func (tq *TaskQueue) EnqueueJob(jobID string) error {
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

	logger.Info("Worker started", "worker_id", id)

	for {
		select {
		case jobID, ok := <-tq.jobChannel:
			if !ok {
				logger.Info("Worker stopped", "worker_id", id)
				return
			}

			logger.WorkerInfo(id, jobID, "start")

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
					log.Printf("Worker %d: Job %s was cancelled", id, jobID)
					tq.updateJobStatus(jobID, models.StatusFailed)
					tq.updateJobError(jobID, "Job was cancelled by user")
				} else {
					log.Printf("Worker %d: Failed to process job %s: %v", id, jobID, err)
					tq.updateJobStatus(jobID, models.StatusFailed)
					tq.updateJobError(jobID, err.Error())
				}
			} else {
				log.Printf("Worker %d: Successfully processed job %s", id, jobID)
				tq.updateJobStatus(jobID, models.StatusCompleted)
			}

		case <-tq.ctx.Done():
			log.Printf("Worker %d stopped due to context cancellation", id)
			return
		}
	}
}

// jobScanner scans for pending jobs and adds them to the queue
func (tq *TaskQueue) jobScanner() {
	defer tq.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // Scan every 10 seconds
	defer ticker.Stop()

	log.Println("Job scanner started")

	for {
		select {
		case <-ticker.C:
			tq.scanPendingJobs()
		case <-tq.ctx.Done():
			log.Println("Job scanner stopped")
			return
		}
	}
}

// scanPendingJobs finds pending jobs and enqueues them
func (tq *TaskQueue) scanPendingJobs() {
	var jobs []models.TranscriptionJob

	if err := database.DB.Where("status = ?", models.StatusPending).Find(&jobs).Error; err != nil {
		log.Printf("Failed to scan pending jobs: %v", err)
		return
	}

	for _, job := range jobs {
		select {
		case tq.jobChannel <- job.ID:
			log.Printf("Enqueued pending job %s", job.ID)
		default:
			log.Printf("Queue is full, skipping job %s", job.ID)
			break
		}
	}
}

// KillJob aggressively terminates a running job
func (tq *TaskQueue) KillJob(jobID string) error {
	tq.jobsMutex.Lock()
	defer tq.jobsMutex.Unlock()

	runningJob, exists := tq.runningJobs[jobID]
	if !exists {
		return fmt.Errorf("job %s is not currently running", jobID)
	}

	log.Printf("Aggressively killing job %s", jobID)

	// Check if this is a multi-track job and handle accordingly
	if mtProcessor, ok := tq.processor.(MultiTrackJobProcessor); ok && mtProcessor.IsMultiTrackJob(jobID) {
		log.Printf("Terminating multi-track job %s", jobID)
		
		// Terminate all individual track jobs
		if err := mtProcessor.TerminateMultiTrackJob(jobID); err != nil {
			log.Printf("Failed to terminate multi-track job %s: %v", jobID, err)
		}
	}

	// First, try to kill the OS process group (or process on non-Unix)
	if runningJob.Process != nil && runningJob.Process.Process != nil {
		log.Printf("Attempting to terminate process tree for PID %d (job %s)", runningJob.Process.Process.Pid, jobID)
		if err := killProcessTree(runningJob.Process.Process); err != nil {
			log.Printf("Failed to terminate process tree for job %s: %v, trying direct kill()", jobID, err)
			_ = runningJob.Process.Process.Kill()
		}
	}

	// Also cancel the context for cleanup
	runningJob.Cancel()

	// Immediately update job status without waiting for process to finish
	go func() {
		tq.updateJobStatus(jobID, models.StatusFailed)
		tq.updateJobError(jobID, "Job was forcefully terminated by user")
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
	return database.DB.Model(&models.TranscriptionJob{}).
		Where("id = ?", jobID).
		Update("status", status).Error
}

// updateJobError updates the error message of a job
func (tq *TaskQueue) updateJobError(jobID string, errorMsg string) error {
	return database.DB.Model(&models.TranscriptionJob{}).
		Where("id = ?", jobID).
		Update("error_message", errorMsg).Error
}

// GetJobStatus gets the status of a job
func (tq *TaskQueue) GetJobStatus(jobID string) (*models.TranscriptionJob, error) {
	var job models.TranscriptionJob
	err := database.DB.Where("id = ?", jobID).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
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
	var pendingCount, processingCount, completedCount, failedCount int64

	database.DB.Model(&models.TranscriptionJob{}).Where("status = ?", models.StatusPending).Count(&pendingCount)
	database.DB.Model(&models.TranscriptionJob{}).Where("status = ?", models.StatusProcessing).Count(&processingCount)
	database.DB.Model(&models.TranscriptionJob{}).Where("status = ?", models.StatusCompleted).Count(&completedCount)
	database.DB.Model(&models.TranscriptionJob{}).Where("status = ?", models.StatusFailed).Count(&failedCount)

	tq.jobsMutex.RLock()
	runningJobsCount := len(tq.runningJobs)
	tq.jobsMutex.RUnlock()

	return map[string]interface{}{
		"queue_size":       len(tq.jobChannel),
		"queue_capacity":   cap(tq.jobChannel),
		"current_workers":  int(atomic.LoadInt64(&tq.currentWorkers)),
		"min_workers":      tq.minWorkers,
		"max_workers":      tq.maxWorkers,
		"auto_scale":       tq.autoScale,
		"running_jobs":     runningJobsCount,
		"pending_jobs":     pendingCount,
		"processing_jobs":  processingCount,
		"completed_jobs":   completedCount,
		"failed_jobs":      failedCount,
	}
}
