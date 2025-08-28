package tests

import (
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/queue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockJobProcessor is a mock implementation of JobProcessor
type MockJobProcessor struct {
	mock.Mock
}

func (m *MockJobProcessor) ProcessJob(jobID string) error {
	args := m.Called(jobID)
	return args.Error(0)
}

func TestTaskQueue_NewTaskQueue(t *testing.T) {
	mockProcessor := &MockJobProcessor{}
	tq := queue.NewTaskQueue(2, mockProcessor)
	
	assert.NotNil(t, tq)
}

func TestTaskQueue_EnqueueJob(t *testing.T) {
	mockProcessor := &MockJobProcessor{}
	tq := queue.NewTaskQueue(1, mockProcessor)
	
	jobID := "test-job-123"
	err := tq.EnqueueJob(jobID)
	assert.NoError(t, err)
}

func TestTaskQueue_GetQueueStats(t *testing.T) {
	// Initialize test database
	database.Initialize("test_queue.db")
	defer func() {
		database.Close()
		// Note: In a real test environment, you'd clean up the test DB
	}()
	
	// Create some test jobs
	testJobs := []models.TranscriptionJob{
		{
			ID:        "pending-1",
			Status:    models.StatusPending,
			AudioPath: "test1.mp3",
		},
		{
			ID:        "processing-1",
			Status:    models.StatusProcessing,
			AudioPath: "test2.mp3",
		},
		{
			ID:        "completed-1",
			Status:    models.StatusCompleted,
			AudioPath: "test3.mp3",
		},
		{
			ID:        "failed-1",
			Status:    models.StatusFailed,
			AudioPath: "test4.mp3",
		},
	}
	
	for _, job := range testJobs {
		database.DB.Create(&job)
	}
	
	mockProcessor := &MockJobProcessor{}
	tq := queue.NewTaskQueue(2, mockProcessor)
	
	stats := tq.GetQueueStats()
	
	assert.Contains(t, stats, "queue_size")
	assert.Contains(t, stats, "queue_capacity")
	assert.Contains(t, stats, "workers")
	assert.Contains(t, stats, "pending_jobs")
	assert.Contains(t, stats, "processing_jobs")
	assert.Contains(t, stats, "completed_jobs")
	assert.Contains(t, stats, "failed_jobs")
	
	assert.Equal(t, 2, stats["workers"])
	assert.Equal(t, int64(1), stats["pending_jobs"])
	assert.Equal(t, int64(1), stats["processing_jobs"])
	assert.Equal(t, int64(1), stats["completed_jobs"])
	assert.Equal(t, int64(1), stats["failed_jobs"])
}

func TestTaskQueue_GetJobStatus(t *testing.T) {
	database.Initialize("test_queue_status.db")
	defer func() {
		database.Close()
	}()
	
	// Create test job
	testJob := models.TranscriptionJob{
		ID:        "status-test-job",
		Status:    models.StatusPending,
		AudioPath: "test.mp3",
		Title:     stringPtr("Test Job"),
	}
	database.DB.Create(&testJob)
	
	mockProcessor := &MockJobProcessor{}
	tq := queue.NewTaskQueue(1, mockProcessor)
	
	// Test getting existing job
	job, err := tq.GetJobStatus("status-test-job")
	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, "status-test-job", job.ID)
	assert.Equal(t, models.StatusPending, job.Status)
	
	// Test getting non-existent job
	job, err = tq.GetJobStatus("non-existent-job")
	assert.Error(t, err)
	assert.Nil(t, job)
}

func TestTaskQueue_StartAndStop(t *testing.T) {
	mockProcessor := &MockJobProcessor{}
	tq := queue.NewTaskQueue(1, mockProcessor)
	
	// Start the queue
	tq.Start()
	
	// Wait a bit to ensure workers are running
	time.Sleep(100 * time.Millisecond)
	
	// Stop the queue
	done := make(chan bool)
	go func() {
		tq.Stop()
		done <- true
	}()
	
	// Ensure stop completes within reasonable time
	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("TaskQueue.Stop() took too long")
	}
}

func TestTaskQueue_ProcessJob(t *testing.T) {
	database.Initialize("test_queue_process.db")
	defer func() {
		database.Close()
	}()
	
	// Create test job
	testJob := models.TranscriptionJob{
		ID:        "process-test-job",
		Status:    models.StatusPending,
		AudioPath: "test.mp3",
	}
	database.DB.Create(&testJob)
	
	mockProcessor := &MockJobProcessor{}
	mockProcessor.On("ProcessJob", "process-test-job").Return(nil)
	
	tq := queue.NewTaskQueue(1, mockProcessor)
	tq.Start()
	
	// Enqueue the job
	err := tq.EnqueueJob("process-test-job")
	assert.NoError(t, err)
	
	// Wait for processing to complete
	time.Sleep(500 * time.Millisecond)
	
	tq.Stop()
	
	// Verify the mock was called
	mockProcessor.AssertExpectations(t)
	
	// Verify job status was updated
	var updatedJob models.TranscriptionJob
	database.DB.Where("id = ?", "process-test-job").First(&updatedJob)
	assert.Equal(t, models.StatusCompleted, updatedJob.Status)
}

func TestTaskQueue_ProcessJobWithError(t *testing.T) {
	database.Initialize("test_queue_error.db")
	defer func() {
		database.Close()
	}()
	
	// Create test job
	testJob := models.TranscriptionJob{
		ID:        "error-test-job",
		Status:    models.StatusPending,
		AudioPath: "test.mp3",
	}
	database.DB.Create(&testJob)
	
	mockProcessor := &MockJobProcessor{}
	mockProcessor.On("ProcessJob", "error-test-job").Return(assert.AnError)
	
	tq := queue.NewTaskQueue(1, mockProcessor)
	tq.Start()
	
	// Enqueue the job
	err := tq.EnqueueJob("error-test-job")
	assert.NoError(t, err)
	
	// Wait for processing to complete
	time.Sleep(500 * time.Millisecond)
	
	tq.Stop()
	
	// Verify the mock was called
	mockProcessor.AssertExpectations(t)
	
	// Verify job status was updated to failed
	var updatedJob models.TranscriptionJob
	database.DB.Where("id = ?", "error-test-job").First(&updatedJob)
	assert.Equal(t, models.StatusFailed, updatedJob.Status)
	assert.NotNil(t, updatedJob.ErrorMessage)
}