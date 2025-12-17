package tests

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"testing"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/queue"
	"scriberr/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockJobProcessor for testing
type MockJobProcessor struct {
	mock.Mock
	processDelay time.Duration
}

func (m *MockJobProcessor) ProcessJob(ctx context.Context, jobID string) error {
	return m.ProcessJobWithProcess(ctx, jobID, func(*exec.Cmd) {})
}

func (m *MockJobProcessor) ProcessJobWithProcess(ctx context.Context, jobID string, registerProcess func(*exec.Cmd)) error {
	args := m.Called(ctx, jobID)

	// Call registerProcess with a mock command to simulate real behavior
	if registerProcess != nil {
		registerProcess(&exec.Cmd{})
	}

	// Simulate processing time if delay is set
	if m.processDelay > 0 {
		select {
		case <-time.After(m.processDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return args.Error(0)
}

type QueueTestSuite struct {
	suite.Suite
	helper  *TestHelper
	jobRepo repository.JobRepository
}

func (suite *QueueTestSuite) SetupSuite() {
	suite.helper = NewTestHelper(suite.T(), "queue_test.db")
	suite.jobRepo = repository.NewJobRepository(suite.helper.DB)
}

func (suite *QueueTestSuite) TearDownSuite() {
	suite.helper.Cleanup()
}

func (suite *QueueTestSuite) SetupTest() {
	suite.helper.ResetDB(suite.T())
}

// Test queue creation
func (suite *QueueTestSuite) TestNewTaskQueue() {
	mockProcessor := &MockJobProcessor{}

	tq := queue.NewTaskQueue(2, mockProcessor, suite.jobRepo)

	assert.NotNil(suite.T(), tq)

	// Test queue stats before starting
	stats := tq.GetQueueStats()
	assert.Equal(suite.T(), 2, stats["current_workers"])
	assert.Equal(suite.T(), 0, stats["queue_size"])
	assert.Equal(suite.T(), 200, stats["queue_capacity"])
}

// Test enqueuing jobs
func (suite *QueueTestSuite) TestEnqueueJob() {
	mockProcessor := &MockJobProcessor{}
	tq := queue.NewTaskQueue(1, mockProcessor, suite.jobRepo)

	// Test successful enqueue
	err := tq.EnqueueJob("test-job-1")
	assert.NoError(suite.T(), err)

	// Test queue stats after enqueue
	stats := tq.GetQueueStats()
	assert.Equal(suite.T(), 1, stats["queue_size"])
}

// Test job processing
func (suite *QueueTestSuite) TestJobProcessing() {
	// Create test job in database first
	job := suite.helper.CreateTestTranscriptionJob(suite.T(), "Test Job Processing")

	mockProcessor := &MockJobProcessor{}
	mockProcessor.On("ProcessJobWithProcess", mock.Anything, job.ID).Return(nil)

	tq := queue.NewTaskQueue(1, mockProcessor, suite.jobRepo)

	// Start the queue
	tq.Start()
	defer tq.Stop()

	// Enqueue the job
	err := tq.EnqueueJob(job.ID)
	assert.NoError(suite.T(), err)

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Verify the job was processed
	mockProcessor.AssertCalled(suite.T(), "ProcessJobWithProcess", mock.Anything, job.ID, mock.Anything)

	// Check job status in database
	updatedJob, err := tq.GetJobStatus(job.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.StatusCompleted, updatedJob.Status)
}

// Test job processing failure
func (suite *QueueTestSuite) TestJobProcessingFailure() {
	mockProcessor := &MockJobProcessor{}
	mockProcessor.On("ProcessJobWithProcess", mock.Anything, mock.Anything).Return(assert.AnError)

	// Create test job in database
	job := suite.helper.CreateTestTranscriptionJob(suite.T(), "Test Job Failure")

	tq := queue.NewTaskQueue(1, mockProcessor, suite.jobRepo)

	// Start the queue
	tq.Start()
	defer tq.Stop()

	// Enqueue the job
	err := tq.EnqueueJob(job.ID)
	assert.NoError(suite.T(), err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify the job was processed
	mockProcessor.AssertCalled(suite.T(), "ProcessJobWithProcess", mock.Anything, job.ID, mock.Anything)

	// Check job status in database
	updatedJob, err := tq.GetJobStatus(job.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.StatusFailed, updatedJob.Status)
	assert.NotNil(suite.T(), updatedJob.ErrorMessage)
}

// Test job cancellation
func (suite *QueueTestSuite) TestJobCancellation() {
	mockProcessor := &MockJobProcessor{}
	// Set a delay so we have time to cancel
	mockProcessor.processDelay = 500 * time.Millisecond
	mockProcessor.On("ProcessJobWithProcess", mock.Anything, mock.Anything).Return(context.Canceled)

	// Create test job in database
	job := suite.helper.CreateTestTranscriptionJob(suite.T(), "Test Job Cancellation")

	tq := queue.NewTaskQueue(1, mockProcessor, suite.jobRepo)

	// Start the queue
	tq.Start()
	defer tq.Stop()

	// Enqueue the job
	err := tq.EnqueueJob(job.ID)
	assert.NoError(suite.T(), err)

	// Wait a bit for job to start processing
	time.Sleep(50 * time.Millisecond)

	// Verify job is running
	assert.True(suite.T(), tq.IsJobRunning(job.ID))

	// Cancel the job
	err = tq.KillJob(job.ID)
	assert.NoError(suite.T(), err)

	// Wait for cancellation to complete using Eventually
	assert.Eventually(suite.T(), func() bool {
		return !tq.IsJobRunning(job.ID)
	}, 2*time.Second, 100*time.Millisecond, "Job should stop running after cancellation")

	// Check job status in database with Eventually
	assert.Eventually(suite.T(), func() bool {
		updatedJob, err := tq.GetJobStatus(job.ID)
		return err == nil && updatedJob.Status == models.StatusFailed
	}, 2*time.Second, 100*time.Millisecond, "Job status should update to failed")
}

// Test killing non-running job
func (suite *QueueTestSuite) TestKillNonRunningJob() {
	mockProcessor := &MockJobProcessor{}
	tq := queue.NewTaskQueue(1, mockProcessor, suite.jobRepo)

	err := tq.KillJob("non-existent-job")
	assert.Error(suite.T(), err)
	// The error message depends on whether it was found in DB or not
	// Since we didn't create it in DB, it returns "not found"
	assert.Contains(suite.T(), err.Error(), "not found")
}

// Test queue stats
func (suite *QueueTestSuite) TestGetQueueStats() {
	mockProcessor := &MockJobProcessor{}
	tq := queue.NewTaskQueue(3, mockProcessor, suite.jobRepo)

	// Create test jobs with different statuses
	suite.helper.CreateTestTranscriptionJob(suite.T(), "Pending Job")

	processingJob := suite.helper.CreateTestTranscriptionJob(suite.T(), "Processing Job")
	processingJob.Status = models.StatusProcessing
	// Update in database would be done here in real implementation

	completedJob := suite.helper.CreateTestTranscriptionJob(suite.T(), "Completed Job")
	completedJob.Status = models.StatusCompleted

	failedJob := suite.helper.CreateTestTranscriptionJob(suite.T(), "Failed Job")
	failedJob.Status = models.StatusFailed

	stats := tq.GetQueueStats()

	assert.Equal(suite.T(), 3, stats["current_workers"])
	assert.Equal(suite.T(), 0, stats["queue_size"]) // No jobs in queue buffer
	assert.Equal(suite.T(), 200, stats["queue_capacity"])

	// Note: The actual counts depend on what's in the database
	assert.Contains(suite.T(), stats, "pending_jobs")
	assert.Contains(suite.T(), stats, "processing_jobs")
	assert.Contains(suite.T(), stats, "completed_jobs")
	assert.Contains(suite.T(), stats, "failed_jobs")
}

// Test multiple workers
func (suite *QueueTestSuite) TestMultipleWorkers() {
	mockProcessor := &MockJobProcessor{}
	// Add some delay to see concurrent processing
	mockProcessor.processDelay = 100 * time.Millisecond
	mockProcessor.On("ProcessJobWithProcess", mock.Anything, mock.Anything).Return(nil)

	// Create multiple test jobs
	jobs := make([]*models.TranscriptionJob, 5)
	for i := 0; i < 5; i++ {
		jobs[i] = suite.helper.CreateTestTranscriptionJob(suite.T(), fmt.Sprintf("Concurrent Job %d", i))
	}

	tq := queue.NewTaskQueue(3, mockProcessor, suite.jobRepo) // 3 workers

	// Start the queue
	tq.Start()
	defer tq.Stop()

	// Enqueue all jobs
	for _, job := range jobs {
		err := tq.EnqueueJob(job.ID)
		assert.NoError(suite.T(), err)
	}

	// Wait for all jobs to complete
	time.Sleep(300 * time.Millisecond)

	// Verify all jobs were processed
	for _, job := range jobs {
		mockProcessor.AssertCalled(suite.T(), "ProcessJobWithProcess", mock.Anything, job.ID, mock.Anything)
	}
}

// Test queue shutdown
func (suite *QueueTestSuite) TestQueueShutdown() {
	mockProcessor := &MockJobProcessor{}
	mockProcessor.On("ProcessJobWithProcess", mock.Anything, mock.Anything).Return(nil)

	tq := queue.NewTaskQueue(2, mockProcessor, suite.jobRepo)

	// Start and then stop
	tq.Start()

	// Enqueue a job
	job := suite.helper.CreateTestTranscriptionJob(suite.T(), "Shutdown Test Job")
	err := tq.EnqueueJob(job.ID)
	assert.NoError(suite.T(), err)

	// Stop the queue
	tq.Stop()

	// Try to enqueue after shutdown (should fail)
	err = tq.EnqueueJob("after-shutdown-job")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "shutting down")
}

// Test queue overflow
func (suite *QueueTestSuite) TestQueueOverflow() {
	mockProcessor := &MockJobProcessor{}
	// Make jobs take a long time so they don't get processed
	mockProcessor.processDelay = 5 * time.Second
	mockProcessor.On("ProcessJobWithProcess", mock.Anything, mock.Anything).Return(nil)

	tq := queue.NewTaskQueue(1, mockProcessor, suite.jobRepo)

	// Fill up the queue (capacity is 200)
	for i := 0; i < 200; i++ {
		err := tq.EnqueueJob(fmt.Sprintf("job-%d", i))
		assert.NoError(suite.T(), err)
	}

	// The 201st job should fail
	err := tq.EnqueueJob("overflow-job")
	if assert.Error(suite.T(), err, "Expected error for queue overflow") {
		assert.Contains(suite.T(), err.Error(), "queue is full")
	}
}

// Test job status retrieval
func (suite *QueueTestSuite) TestGetJobStatus() {
	mockProcessor := &MockJobProcessor{}
	tq := queue.NewTaskQueue(1, mockProcessor, suite.jobRepo)

	// Create a test job
	job := suite.helper.CreateTestTranscriptionJob(suite.T(), "Status Test Job")

	// Get job status
	retrievedJob, err := tq.GetJobStatus(job.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), job.ID, retrievedJob.ID)
	assert.Equal(suite.T(), models.StatusPending, retrievedJob.Status)

	// Test non-existent job
	_, err = tq.GetJobStatus("non-existent-job")
	assert.Error(suite.T(), err)
}

// Test job running check
func (suite *QueueTestSuite) TestIsJobRunning() {
	mockProcessor := &MockJobProcessor{}
	mockProcessor.processDelay = 200 * time.Millisecond
	mockProcessor.On("ProcessJobWithProcess", mock.Anything, mock.Anything).Return(nil)

	job := suite.helper.CreateTestTranscriptionJob(suite.T(), "Running Check Job")

	tq := queue.NewTaskQueue(1, mockProcessor, suite.jobRepo)
	tq.Start()
	defer tq.Stop()

	// Job should not be running initially
	assert.False(suite.T(), tq.IsJobRunning(job.ID))

	// Enqueue the job
	err := tq.EnqueueJob(job.ID)
	assert.NoError(suite.T(), err)

	// Wait a bit for job to start
	time.Sleep(50 * time.Millisecond)

	// Job should be running now
	assert.True(suite.T(), tq.IsJobRunning(job.ID))

	// Wait for job to complete using Eventually
	assert.Eventually(suite.T(), func() bool {
		return !tq.IsJobRunning(job.ID)
	}, 2*time.Second, 100*time.Millisecond, "Job should stop running after completion")
}

// Test concurrent access safety
func (suite *QueueTestSuite) TestConcurrentAccess() {
	mockProcessor := &MockJobProcessor{}
	mockProcessor.On("ProcessJobWithProcess", mock.Anything, mock.Anything).Return(nil)

	tq := queue.NewTaskQueue(5, mockProcessor, suite.jobRepo)
	tq.Start()
	defer tq.Stop()

	var wg sync.WaitGroup
	numGoroutines := 10
	jobsPerGoroutine := 5

	// Concurrently enqueue jobs
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < jobsPerGoroutine; j++ {
				jobID := fmt.Sprintf("concurrent-job-%d-%d", goroutineID, j)
				job := suite.helper.CreateTestTranscriptionJob(suite.T(), fmt.Sprintf("Concurrent Job %d-%d", goroutineID, j))
				job.ID = jobID

				err := tq.EnqueueJob(jobID)
				// Some enqueues might fail if queue fills up, but shouldn't panic
				if err != nil && !assert.Contains(suite.T(), err.Error(), "queue is full") {
					assert.NoError(suite.T(), err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Wait for processing to complete
	time.Sleep(500 * time.Millisecond)

	// Check that we can still get stats without panicking
	stats := tq.GetQueueStats()
	assert.NotNil(suite.T(), stats)
}

func TestQueueTestSuite(t *testing.T) {
	suite.Run(t, new(QueueTestSuite))
}

// Test ResetZombieJobs
func (suite *QueueTestSuite) TestResetZombieJobs() {
	mockProcessor := &MockJobProcessor{}
	tq := queue.NewTaskQueue(1, mockProcessor, suite.jobRepo)

	// Create a "zombie" job (one that is processing in DB but not running)
	job := suite.helper.CreateTestTranscriptionJob(suite.T(), "Zombie Job")
	// Manually set status to processing
	err := suite.helper.DB.Model(&models.TranscriptionJob{}).Where("id = ?", job.ID).Update("status", models.StatusProcessing).Error
	assert.NoError(suite.T(), err)

	// Verify it is processing
	var checkJob models.TranscriptionJob
	suite.helper.DB.First(&checkJob, "id = ?", job.ID)
	assert.Equal(suite.T(), models.StatusProcessing, checkJob.Status)

	// Run ResetZombieJobs
	tq.ResetZombieJobs()

	// Verify it is now failed
	var updatedJob models.TranscriptionJob
	suite.helper.DB.First(&updatedJob, "id = ?", job.ID)
	assert.Equal(suite.T(), models.StatusFailed, updatedJob.Status)
	assert.Contains(suite.T(), *updatedJob.ErrorMessage, "interrupted by server restart")
}
