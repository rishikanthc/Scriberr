package queue

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
)

// TaskQueue manages transcription job processing
type TaskQueue struct {
	workers      int
	jobChannel   chan string
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	processor    JobProcessor
}

// JobProcessor defines the interface for processing jobs
type JobProcessor interface {
	ProcessJob(jobID string) error
}

// NewTaskQueue creates a new task queue
func NewTaskQueue(workers int, processor JobProcessor) *TaskQueue {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &TaskQueue{
		workers:     workers,
		jobChannel:  make(chan string, 100), // Buffer for 100 jobs
		ctx:         ctx,
		cancel:      cancel,
		processor:   processor,
	}
}

// Start starts the task queue workers
func (tq *TaskQueue) Start() {
	log.Printf("Starting task queue with %d workers", tq.workers)
	
	for i := 0; i < tq.workers; i++ {
		tq.wg.Add(1)
		go tq.worker(i)
	}

	// Start the job scanner
	tq.wg.Add(1)
	go tq.jobScanner()
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
	
	log.Printf("Worker %d started", id)
	
	for {
		select {
		case jobID, ok := <-tq.jobChannel:
			if !ok {
				log.Printf("Worker %d stopped", id)
				return
			}
			
			log.Printf("Worker %d processing job %s", id, jobID)
			
			// Update job status to processing
			if err := tq.updateJobStatus(jobID, models.StatusProcessing); err != nil {
				log.Printf("Worker %d: Failed to update job %s status to processing: %v", id, jobID, err)
				continue
			}
			
			// Process the job
			if err := tq.processor.ProcessJob(jobID); err != nil {
				log.Printf("Worker %d: Failed to process job %s: %v", id, jobID, err)
				tq.updateJobStatus(jobID, models.StatusFailed)
				tq.updateJobError(jobID, err.Error())
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

// GetQueueStats returns queue statistics
func (tq *TaskQueue) GetQueueStats() map[string]interface{} {
	var pendingCount, processingCount, completedCount, failedCount int64
	
	database.DB.Model(&models.TranscriptionJob{}).Where("status = ?", models.StatusPending).Count(&pendingCount)
	database.DB.Model(&models.TranscriptionJob{}).Where("status = ?", models.StatusProcessing).Count(&processingCount)
	database.DB.Model(&models.TranscriptionJob{}).Where("status = ?", models.StatusCompleted).Count(&completedCount)
	database.DB.Model(&models.TranscriptionJob{}).Where("status = ?", models.StatusFailed).Count(&failedCount)
	
	return map[string]interface{}{
		"queue_size":       len(tq.jobChannel),
		"queue_capacity":   cap(tq.jobChannel),
		"workers":          tq.workers,
		"pending_jobs":     pendingCount,
		"processing_jobs":  processingCount,
		"completed_jobs":   completedCount,
		"failed_jobs":      failedCount,
	}
}