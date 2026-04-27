package worker

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeProcessor struct {
	started chan string
	release chan struct{}
	result  ProcessResult
	err     error
}

type observedStatusEvent struct {
	event    StatusEvent
	dbStatus models.JobStatus
}

type recordingStatusPublisher struct {
	db     *gorm.DB
	events chan observedStatusEvent
}

func (p *recordingStatusPublisher) PublishStatus(ctx context.Context, event StatusEvent) {
	var job models.TranscriptionJob
	_ = p.db.WithContext(ctx).First(&job, "id = ?", event.JobID).Error
	p.events <- observedStatusEvent{event: event, dbStatus: job.Status}
}

func newFakeProcessor() *fakeProcessor {
	return &fakeProcessor{
		started: make(chan string, 10),
		release: make(chan struct{}),
	}
}

func (p *fakeProcessor) Process(ctx context.Context, job *models.TranscriptionJob) (ProcessResult, error) {
	p.started <- job.ID
	select {
	case <-p.release:
		return p.result, p.err
	case <-ctx.Done():
		return ProcessResult{}, ctx.Err()
	}
}

func (p *fakeProcessor) complete() {
	close(p.release)
}

func openWorkerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "scriberr.db"))
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func createWorkerTestUser(t *testing.T, db *gorm.DB, name string) models.User {
	t.Helper()
	user := models.User{Username: name, Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func createWorkerTestJob(t *testing.T, db *gorm.DB, userID uint, id string, status models.JobStatus) models.TranscriptionJob {
	t.Helper()
	title := id
	sourceID := "file-" + id
	job := models.TranscriptionJob{
		ID:             id,
		UserID:         userID,
		Title:          &title,
		Status:         status,
		AudioPath:      filepath.Join(t.TempDir(), id+".wav"),
		SourceFileName: id + ".wav",
		SourceFileHash: &sourceID,
	}
	require.NoError(t, db.Create(&job).Error)
	return job
}

func testConfig() Config {
	return Config{
		Workers:       1,
		PollInterval:  25 * time.Millisecond,
		LeaseTimeout:  150 * time.Millisecond,
		RenewInterval: 25 * time.Millisecond,
		StopTimeout:   time.Second,
		WorkerID:      "test-worker",
	}
}

func waitForJobStatus(t *testing.T, db *gorm.DB, jobID string, status models.JobStatus) models.TranscriptionJob {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var job models.TranscriptionJob
		require.NoError(t, db.First(&job, "id = ?", jobID).Error)
		if job.Status == status {
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
	var job models.TranscriptionJob
	require.NoError(t, db.First(&job, "id = ?", jobID).Error)
	t.Fatalf("job %s status = %s, want %s", jobID, job.Status, status)
	return job
}

func TestServiceEnqueueWakeAndComplete(t *testing.T) {
	db := openWorkerTestDB(t)
	user := createWorkerTestUser(t, db, "worker-user-complete")
	job := createWorkerTestJob(t, db, user.ID, "job-complete", models.StatusUploaded)
	repo := repository.NewJobRepository(db)
	processor := newFakeProcessor()
	processor.result = ProcessResult{TranscriptJSON: `{"text":"done"}`}
	service := NewService(repo, processor, testConfig())

	require.NoError(t, service.Start(context.Background()))
	defer service.Stop(context.Background())
	require.NoError(t, service.Enqueue(context.Background(), job.ID))

	select {
	case startedID := <-processor.started:
		assert.Equal(t, job.ID, startedID)
	case <-time.After(time.Second):
		t.Fatal("processor did not start")
	}
	processor.complete()
	completed := waitForJobStatus(t, db, job.ID, models.StatusCompleted)
	assert.Equal(t, 1.0, completed.Progress)
	assert.Equal(t, "completed", completed.ProgressStage)
	assert.NotNil(t, completed.CompletedAt)
}

func TestServicePublishesTerminalStatusAfterCompletionCommit(t *testing.T) {
	db := openWorkerTestDB(t)
	user := createWorkerTestUser(t, db, "worker-user-terminal-event")
	job := createWorkerTestJob(t, db, user.ID, "job-terminal-event", models.StatusUploaded)
	repo := repository.NewJobRepository(db)
	processor := newFakeProcessor()
	processor.result = ProcessResult{TranscriptJSON: `{"text":"done"}`}
	publisher := &recordingStatusPublisher{db: db, events: make(chan observedStatusEvent, 1)}
	service := NewService(repo, processor, testConfig())
	service.SetEventPublisher(publisher)

	require.NoError(t, service.Start(context.Background()))
	defer service.Stop(context.Background())
	require.NoError(t, service.Enqueue(context.Background(), job.ID))

	select {
	case <-processor.started:
	case <-time.After(time.Second):
		t.Fatal("processor did not start")
	}
	processor.complete()
	waitForJobStatus(t, db, job.ID, models.StatusCompleted)

	select {
	case observed := <-publisher.events:
		assert.Equal(t, "transcription.completed", observed.event.Name)
		assert.Equal(t, models.StatusCompleted, observed.event.Status)
		assert.Equal(t, models.StatusCompleted, observed.dbStatus)
	case <-time.After(time.Second):
		t.Fatal("terminal event was not published")
	}
}

func TestServiceStartRecoversOrphanedProcessingBeforeWorkersClaim(t *testing.T) {
	db := openWorkerTestDB(t)
	user := createWorkerTestUser(t, db, "worker-user-recover")
	job := createWorkerTestJob(t, db, user.ID, "job-recover", models.StatusProcessing)
	repo := repository.NewJobRepository(db)
	processor := newFakeProcessor()
	processor.result = ProcessResult{TranscriptJSON: `{"text":"recovered"}`}
	service := NewService(repo, processor, testConfig())

	require.NoError(t, service.Start(context.Background()))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = service.Stop(ctx)
	})

	require.Eventually(t, func() bool {
		select {
		case id := <-processor.started:
			return id == job.ID
		default:
			return false
		}
	}, 2*time.Second, 10*time.Millisecond)
	processor.complete()

	completed := waitForJobStatus(t, db, job.ID, models.StatusCompleted)
	require.NotNil(t, completed.QueuedAt)
	assert.Equal(t, "completed", completed.ProgressStage)
}

func TestServiceCancelQueued(t *testing.T) {
	db := openWorkerTestDB(t)
	user := createWorkerTestUser(t, db, "worker-user-cancel-queued")
	job := createWorkerTestJob(t, db, user.ID, "job-cancel-queued", models.StatusUploaded)
	repo := repository.NewJobRepository(db)
	service := NewService(repo, newFakeProcessor(), testConfig())

	require.ErrorIs(t, service.Enqueue(context.Background(), job.ID), ErrQueueStopped)
	require.NoError(t, repo.EnqueueTranscription(context.Background(), job.ID, time.Now()))
	require.NoError(t, service.Cancel(context.Background(), user.ID, job.ID))

	stopped := waitForJobStatus(t, db, job.ID, models.StatusStopped)
	assert.Equal(t, "stopped", stopped.ProgressStage)
}

func TestServiceCancelRunning(t *testing.T) {
	db := openWorkerTestDB(t)
	user := createWorkerTestUser(t, db, "worker-user-cancel-running")
	job := createWorkerTestJob(t, db, user.ID, "job-cancel-running", models.StatusUploaded)
	repo := repository.NewJobRepository(db)
	processor := newFakeProcessor()
	service := NewService(repo, processor, testConfig())

	require.NoError(t, service.Start(context.Background()))
	defer service.Stop(context.Background())
	require.NoError(t, service.Enqueue(context.Background(), job.ID))
	select {
	case <-processor.started:
	case <-time.After(time.Second):
		t.Fatal("processor did not start")
	}
	require.NoError(t, service.Cancel(context.Background(), user.ID, job.ID))
	waitForJobStatus(t, db, job.ID, models.StatusStopped)
}

func TestServiceRenewsLeaseWhileProcessing(t *testing.T) {
	db := openWorkerTestDB(t)
	user := createWorkerTestUser(t, db, "worker-user-renew")
	job := createWorkerTestJob(t, db, user.ID, "job-renew-lease", models.StatusUploaded)
	repo := repository.NewJobRepository(db)
	processor := newFakeProcessor()
	service := NewService(repo, processor, testConfig())

	require.NoError(t, service.Start(context.Background()))
	defer service.Stop(context.Background())
	require.NoError(t, service.Enqueue(context.Background(), job.ID))
	select {
	case <-processor.started:
	case <-time.After(time.Second):
		t.Fatal("processor did not start")
	}
	var first models.TranscriptionJob
	require.NoError(t, db.First(&first, "id = ?", job.ID).Error)
	require.NotNil(t, first.ClaimExpiresAt)
	time.Sleep(90 * time.Millisecond)
	var renewed models.TranscriptionJob
	require.NoError(t, db.First(&renewed, "id = ?", job.ID).Error)
	require.NotNil(t, renewed.ClaimExpiresAt)
	assert.True(t, renewed.ClaimExpiresAt.After(*first.ClaimExpiresAt), "claim expiry was not renewed")
	processor.complete()
	waitForJobStatus(t, db, job.ID, models.StatusCompleted)
}

func TestServiceStopCancelsRunningJobs(t *testing.T) {
	db := openWorkerTestDB(t)
	user := createWorkerTestUser(t, db, "worker-user-stop")
	job := createWorkerTestJob(t, db, user.ID, "job-stop", models.StatusUploaded)
	repo := repository.NewJobRepository(db)
	processor := newFakeProcessor()
	service := NewService(repo, processor, testConfig())

	require.NoError(t, service.Start(context.Background()))
	require.NoError(t, service.Enqueue(context.Background(), job.ID))
	select {
	case <-processor.started:
	case <-time.After(time.Second):
		t.Fatal("processor did not start")
	}
	require.NoError(t, service.Stop(context.Background()))
	waitForJobStatus(t, db, job.ID, models.StatusStopped)
}

func TestServiceStatsAndCancelConflict(t *testing.T) {
	db := openWorkerTestDB(t)
	user := createWorkerTestUser(t, db, "worker-user-stats")
	other := createWorkerTestUser(t, db, "worker-user-other")
	repo := repository.NewJobRepository(db)
	createWorkerTestJob(t, db, user.ID, "job-queued", models.StatusPending)
	createWorkerTestJob(t, db, user.ID, "job-processing", models.StatusProcessing)
	completed := createWorkerTestJob(t, db, user.ID, "job-completed", models.StatusCompleted)
	createWorkerTestJob(t, db, user.ID, "job-failed", models.StatusFailed)
	createWorkerTestJob(t, db, other.ID, "job-other", models.StatusPending)
	service := NewService(repo, newFakeProcessor(), testConfig())

	stats, err := service.Stats(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Queued)
	assert.Equal(t, int64(1), stats.Processing)
	assert.Equal(t, int64(1), stats.Completed)
	assert.Equal(t, int64(1), stats.Failed)

	err = service.Cancel(context.Background(), user.ID, completed.ID)
	require.ErrorIs(t, err, ErrStateConflict)
	err = service.Cancel(context.Background(), other.ID, completed.ID)
	require.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}
