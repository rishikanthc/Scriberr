package recording

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"

	"gorm.io/gorm"
)

type fakeMediaFinalizer struct {
	err error
}

func (f fakeMediaFinalizer) Finalize(_ context.Context, inputPath, outputPath string) error {
	if f.err != nil {
		return f.err
	}
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0o600)
}

type fakeEnqueuer struct {
	ids []string
	err error
}

func (q *fakeEnqueuer) Enqueue(_ context.Context, jobID string) error {
	q.ids = append(q.ids, jobID)
	return q.err
}

type finalizerEvents struct {
	recordings     []Event
	files          []map[string]any
	transcriptions []map[string]any
}

func (e *finalizerEvents) PublishRecordingEvent(_ context.Context, event Event) {
	e.recordings = append(e.recordings, event)
}

func (e *finalizerEvents) PublishFileEvent(_ context.Context, _ string, payload map[string]any) {
	e.files = append(e.files, payload)
}

func (e *finalizerEvents) PublishTranscriptionEvent(_ context.Context, _ string, _ string, payload map[string]any) {
	e.transcriptions = append(e.transcriptions, payload)
}

func openFinalizerTest(t *testing.T) (*gorm.DB, repository.RecordingRepository, repository.JobRepository, repository.ProfileRepository, *Storage, models.User) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "scriberr.db"))
	if err != nil {
		t.Fatalf("database.Open returned error: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("database.Migrate returned error: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	user := models.User{Username: "finalizer-user", Password: "pw"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user returned error: %v", err)
	}
	storage, err := NewStorage(filepath.Join(t.TempDir(), "recordings"))
	if err != nil {
		t.Fatalf("NewStorage returned error: %v", err)
	}
	recordings := repository.NewRecordingRepository(db)
	jobs := repository.NewJobRepository(db)
	profiles := repository.NewProfileRepository(db)
	return db, recordings, jobs, profiles, storage, user
}

func createStoppedRecordingWithChunks(t *testing.T, repo repository.RecordingRepository, storage *Storage, user models.User, autoTranscribe bool) *models.RecordingSession {
	t.Helper()
	ctx := context.Background()
	title := "Recorded sync"
	session := &models.RecordingSession{UserID: user.ID, Title: &title, MimeType: "audio/webm;codecs=opus", AutoTranscribe: autoTranscribe}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	for i, body := range []string{"aaa", "bbb"} {
		path, size, err := storage.WriteChunk(ctx, session.ID, i, session.MimeType, strings.NewReader(body))
		if err != nil {
			t.Fatalf("WriteChunk returned error: %v", err)
		}
		if err := repo.AddChunk(ctx, &models.RecordingChunk{UserID: user.ID, SessionID: session.ID, ChunkIndex: i, Path: path, MimeType: session.MimeType, SizeBytes: size}); err != nil {
			t.Fatalf("AddChunk returned error: %v", err)
		}
	}
	if err := repo.MarkStopping(ctx, user.ID, session.ID, 1, int64Ptr(6000), autoTranscribe, time.Now()); err != nil {
		t.Fatalf("MarkStopping returned error: %v", err)
	}
	return session
}

func TestFinalizerCreatesSingleFileAndRemovesTemporaryArtifacts(t *testing.T) {
	db, recordings, jobs, profiles, storage, user := openFinalizerTest(t)
	session := createStoppedRecordingWithChunks(t, recordings, storage, user, false)
	events := &finalizerEvents{}
	service := NewFinalizerService(recordings, jobs, profiles, storage, fakeMediaFinalizer{}, FinalizerConfig{})
	service.SetEventPublisher(events)

	claimed, err := recordings.ClaimNextFinalization(context.Background(), "worker-a", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("ClaimNextFinalization returned error: %v", err)
	}
	if err := service.finalize(context.Background(), "worker-a", claimed); err != nil {
		t.Fatalf("finalize returned error: %v", err)
	}

	updated, err := recordings.FindSessionForUser(context.Background(), user.ID, session.ID)
	if err != nil {
		t.Fatalf("FindSessionForUser returned error: %v", err)
	}
	if updated.Status != models.RecordingStatusReady {
		t.Fatalf("status = %s", updated.Status)
	}
	if updated.FileID == nil {
		t.Fatal("file_id was not set")
	}
	var file models.TranscriptionJob
	if err := db.First(&file, "id = ?", *updated.FileID).Error; err != nil {
		t.Fatalf("final file row missing: %v", err)
	}
	data, err := os.ReadFile(file.AudioPath)
	if err != nil {
		t.Fatalf("final audio missing: %v", err)
	}
	if string(data) != "aaabbb" {
		t.Fatalf("final audio = %q", string(data))
	}
	chunkDir, err := storage.ChunkDir(session.ID)
	if err != nil {
		t.Fatalf("ChunkDir returned error: %v", err)
	}
	if _, err := os.Stat(chunkDir); !os.IsNotExist(err) {
		t.Fatalf("chunk dir still exists or unexpected error: %v", err)
	}
	rawPath, _ := storage.RawPath(session.ID)
	if _, err := os.Stat(rawPath); !os.IsNotExist(err) {
		t.Fatalf("raw file still exists or unexpected error: %v", err)
	}
	if len(events.files) != 1 {
		t.Fatalf("file events = %#v", events.files)
	}
}

func TestFinalizerFailsMissingChunkWithoutCleanup(t *testing.T) {
	_, recordings, jobs, profiles, storage, user := openFinalizerTest(t)
	session := &models.RecordingSession{UserID: user.ID, MimeType: "audio/webm;codecs=opus"}
	if err := recordings.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	path, size, err := storage.WriteChunk(context.Background(), session.ID, 0, session.MimeType, strings.NewReader("aaa"))
	if err != nil {
		t.Fatalf("WriteChunk returned error: %v", err)
	}
	if err := recordings.AddChunk(context.Background(), &models.RecordingChunk{UserID: user.ID, SessionID: session.ID, ChunkIndex: 0, Path: path, MimeType: session.MimeType, SizeBytes: size}); err != nil {
		t.Fatalf("AddChunk returned error: %v", err)
	}
	if err := recordings.MarkStopping(context.Background(), user.ID, session.ID, 1, nil, false, time.Now()); err != nil {
		t.Fatalf("MarkStopping returned error: %v", err)
	}
	claimed, err := recordings.ClaimNextFinalization(context.Background(), "worker-a", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("ClaimNextFinalization returned error: %v", err)
	}
	service := NewFinalizerService(recordings, jobs, profiles, storage, fakeMediaFinalizer{}, FinalizerConfig{})
	if err := service.finalize(context.Background(), "worker-a", claimed); err == nil {
		t.Fatal("finalize returned nil error for missing chunk")
	}
	updated, err := recordings.FindSessionForUser(context.Background(), user.ID, session.ID)
	if err != nil {
		t.Fatalf("FindSessionForUser returned error: %v", err)
	}
	if updated.Status != models.RecordingStatusFailed {
		t.Fatalf("status = %s", updated.Status)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("chunk should remain for retry: %v", err)
	}
}

func TestFinalizerCreatesAndEnqueuesAutoTranscription(t *testing.T) {
	db, recordings, jobs, profiles, storage, user := openFinalizerTest(t)
	language := "en"
	profile := models.TranscriptionProfile{
		UserID:    user.ID,
		Name:      "Default",
		IsDefault: true,
		Parameters: models.WhisperXParams{
			Language: &language,
			Diarize:  true,
		},
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("create profile returned error: %v", err)
	}
	session := createStoppedRecordingWithChunks(t, recordings, storage, user, true)
	queue := &fakeEnqueuer{}
	events := &finalizerEvents{}
	service := NewFinalizerService(recordings, jobs, profiles, storage, fakeMediaFinalizer{}, FinalizerConfig{})
	service.SetTranscriptionEnqueuer(queue)
	service.SetEventPublisher(events)

	claimed, err := recordings.ClaimNextFinalization(context.Background(), "worker-a", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("ClaimNextFinalization returned error: %v", err)
	}
	if err := service.finalize(context.Background(), "worker-a", claimed); err != nil {
		t.Fatalf("finalize returned error: %v", err)
	}

	updated, err := recordings.FindSessionForUser(context.Background(), user.ID, session.ID)
	if err != nil {
		t.Fatalf("FindSessionForUser returned error: %v", err)
	}
	if updated.TranscriptionID == nil {
		t.Fatal("transcription_id was not set")
	}
	if len(queue.ids) != 1 || queue.ids[0] != *updated.TranscriptionID {
		t.Fatalf("queued ids = %#v", queue.ids)
	}
	var transcription models.TranscriptionJob
	if err := db.First(&transcription, "id = ?", *updated.TranscriptionID).Error; err != nil {
		t.Fatalf("transcription row missing: %v", err)
	}
	if transcription.SourceFileHash == nil || *transcription.SourceFileHash != *updated.FileID {
		t.Fatalf("source file hash = %#v file=%#v", transcription.SourceFileHash, updated.FileID)
	}
	if transcription.Language == nil || *transcription.Language != "en" || !transcription.Diarization {
		t.Fatalf("profile params not applied: language=%#v diarization=%v", transcription.Language, transcription.Diarization)
	}
	if len(events.transcriptions) != 1 {
		t.Fatalf("transcription events = %#v", events.transcriptions)
	}
}

func TestFinalizerMaintenanceExpiresRecoversAndCleansArtifacts(t *testing.T) {
	db, recordings, jobs, profiles, storage, user := openFinalizerTest(t)
	now := time.Now().Truncate(time.Millisecond)

	expired := &models.RecordingSession{
		UserID:    user.ID,
		MimeType:  "audio/webm;codecs=opus",
		ExpiresAt: timePtr(now.Add(-time.Minute)),
	}
	if err := recordings.CreateSession(context.Background(), expired); err != nil {
		t.Fatalf("CreateSession expired returned error: %v", err)
	}
	expiredDir, err := storage.SessionDir(expired.ID)
	if err != nil {
		t.Fatalf("SessionDir returned error: %v", err)
	}
	if err := os.MkdirAll(expiredDir, 0o755); err != nil {
		t.Fatalf("MkdirAll expired returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(expiredDir, "orphan.tmp"), []byte("tmp"), 0o600); err != nil {
		t.Fatalf("WriteFile expired returned error: %v", err)
	}

	ready := &models.RecordingSession{
		UserID:   user.ID,
		MimeType: "audio/webm;codecs=opus",
		Status:   models.RecordingStatusReady,
	}
	if err := recordings.CreateSession(context.Background(), ready); err != nil {
		t.Fatalf("CreateSession ready returned error: %v", err)
	}
	chunkDir, err := storage.ChunkDir(ready.ID)
	if err != nil {
		t.Fatalf("ChunkDir returned error: %v", err)
	}
	if err := os.MkdirAll(chunkDir, 0o755); err != nil {
		t.Fatalf("MkdirAll chunks returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chunkDir, "000000.webm"), []byte("chunk"), 0o600); err != nil {
		t.Fatalf("WriteFile chunk returned error: %v", err)
	}
	rawPath, err := storage.RawPath(ready.ID)
	if err != nil {
		t.Fatalf("RawPath returned error: %v", err)
	}
	if err := os.WriteFile(rawPath, []byte("raw"), 0o600); err != nil {
		t.Fatalf("WriteFile raw returned error: %v", err)
	}
	finalPath, err := storage.FinalPath(ready.ID, ready.MimeType)
	if err != nil {
		t.Fatalf("FinalPath returned error: %v", err)
	}
	if err := os.WriteFile(finalPath, []byte("final"), 0o600); err != nil {
		t.Fatalf("WriteFile final returned error: %v", err)
	}

	stopping := &models.RecordingSession{UserID: user.ID, MimeType: "audio/webm;codecs=opus"}
	if err := recordings.CreateSession(context.Background(), stopping); err != nil {
		t.Fatalf("CreateSession stopping returned error: %v", err)
	}
	if err := recordings.MarkStopping(context.Background(), user.ID, stopping.ID, 0, nil, false, now); err != nil {
		t.Fatalf("MarkStopping returned error: %v", err)
	}
	if _, err := recordings.ClaimNextFinalization(context.Background(), "worker-a", now.Add(-time.Minute)); err != nil {
		t.Fatalf("ClaimNextFinalization returned error: %v", err)
	}

	service := NewFinalizerService(recordings, jobs, profiles, storage, fakeMediaFinalizer{}, FinalizerConfig{FailedRetention: time.Hour})
	stats, err := service.RunMaintenance(context.Background())
	if err != nil {
		t.Fatalf("RunMaintenance returned error: %v", err)
	}
	if stats.ExpiredSessions != 1 {
		t.Fatalf("ExpiredSessions = %d", stats.ExpiredSessions)
	}
	if stats.RecoveredClaims != 1 {
		t.Fatalf("RecoveredClaims = %d", stats.RecoveredClaims)
	}
	if stats.TemporaryArtifactsRemoved != 1 {
		t.Fatalf("TemporaryArtifactsRemoved = %d", stats.TemporaryArtifactsRemoved)
	}
	if stats.SessionDirsRemoved != 1 {
		t.Fatalf("SessionDirsRemoved = %d", stats.SessionDirsRemoved)
	}

	recovered, err := recordings.FindSessionForUser(context.Background(), user.ID, stopping.ID)
	if err != nil {
		t.Fatalf("FindSessionForUser recovered returned error: %v", err)
	}
	if recovered.Status != models.RecordingStatusStopping || recovered.ClaimedBy != nil {
		t.Fatalf("recovered session status=%s claimed_by=%#v", recovered.Status, recovered.ClaimedBy)
	}
	if _, err := os.Stat(expiredDir); !os.IsNotExist(err) {
		t.Fatalf("expired session dir still exists or unexpected error: %v", err)
	}
	if _, err := os.Stat(finalPath); err != nil {
		t.Fatalf("final artifact should remain: %v", err)
	}
	if _, err := os.Stat(chunkDir); !os.IsNotExist(err) {
		t.Fatalf("ready chunk dir still exists or unexpected error: %v", err)
	}
	if _, err := os.Stat(rawPath); !os.IsNotExist(err) {
		t.Fatalf("ready raw artifact still exists or unexpected error: %v", err)
	}

	var readyReloaded models.RecordingSession
	if err := db.First(&readyReloaded, "id = ?", ready.ID).Error; err != nil {
		t.Fatalf("reload ready returned error: %v", err)
	}
	if readyReloaded.TemporaryArtifactsCleanedAt == nil {
		t.Fatal("ready temporary_artifacts_cleaned_at was not set")
	}
}

func TestFinalizerStartStopIsGracefulWithoutWork(t *testing.T) {
	_, recordings, jobs, profiles, storage, _ := openFinalizerTest(t)
	service := NewFinalizerService(recordings, jobs, profiles, storage, fakeMediaFinalizer{}, FinalizerConfig{
		PollInterval:    time.Hour,
		CleanupInterval: time.Hour,
		StopTimeout:     time.Second,
	})

	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := service.Stop(stopCtx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
