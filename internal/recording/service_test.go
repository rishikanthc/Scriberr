package recording

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

type recordingEvents struct {
	events []Event
}

func (p *recordingEvents) PublishRecordingEvent(_ context.Context, event Event) {
	p.events = append(p.events, event)
}

func openRecordingServiceTest(t *testing.T) (*Service, *recordingEvents, models.User) {
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
	user := models.User{Username: "recording-service-user", Password: "pw"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user returned error: %v", err)
	}
	storage, err := NewStorage(filepath.Join(t.TempDir(), "recordings"))
	if err != nil {
		t.Fatalf("NewStorage returned error: %v", err)
	}
	service := NewService(repository.NewRecordingRepository(db), storage, Config{
		MaxChunkBytes:    8,
		MaxDuration:      time.Hour,
		SessionTTL:       time.Hour,
		AllowedMimeTypes: []string{"audio/webm;codecs=opus", "audio/webm"},
	})
	service.now = func() time.Time { return time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC) }
	events := &recordingEvents{}
	service.SetEventPublisher(events)
	return service, events, user
}

func TestServiceCreateAppendStopAndCancelRecording(t *testing.T) {
	service, events, user := openRecordingServiceTest(t)
	ctx := context.Background()

	session, err := service.CreateSession(ctx, CreateSessionRequest{
		UserID:          user.ID,
		Title:           " Team   sync ",
		SourceKind:      "microphone",
		MimeType:        "audio/webm;codecs=opus",
		ChunkDurationMs: int64Ptr(3000),
	})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if session.Title == nil || *session.Title != "Team sync" {
		t.Fatalf("title = %#v", session.Title)
	}
	if session.ExpiresAt == nil || !session.ExpiresAt.Equal(service.now().Add(time.Hour)) {
		t.Fatalf("expires_at = %v", session.ExpiresAt)
	}

	checksum := sha256Hex("chunk")
	result, err := service.AppendChunk(ctx, AppendChunkRequest{
		UserID:      user.ID,
		RecordingID: PublicID(session.ID),
		ChunkIndex:  0,
		MimeType:    "audio/webm;codecs=opus",
		SHA256:      &checksum,
		DurationMs:  int64Ptr(3000),
		Body:        strings.NewReader("chunk"),
	})
	if err != nil {
		t.Fatalf("AppendChunk returned error: %v", err)
	}
	if result.AlreadyStored {
		t.Fatal("first AppendChunk marked chunk as already stored")
	}
	if result.ReceivedChunks != 1 || result.ReceivedBytes != 5 {
		t.Fatalf("received chunks/bytes = %d/%d", result.ReceivedChunks, result.ReceivedBytes)
	}
	if _, err := os.Stat(result.Chunk.Path); err != nil {
		t.Fatalf("chunk file missing: %v", err)
	}

	retry, err := service.AppendChunk(ctx, AppendChunkRequest{
		UserID:      user.ID,
		RecordingID: PublicID(session.ID),
		ChunkIndex:  0,
		MimeType:    "audio/webm;codecs=opus",
		SHA256:      &checksum,
		DurationMs:  int64Ptr(3000),
		Body:        strings.NewReader("chunk"),
	})
	if err != nil {
		t.Fatalf("retry AppendChunk returned error: %v", err)
	}
	if !retry.AlreadyStored {
		t.Fatal("retry AppendChunk did not report already stored")
	}

	duration := int64(3000)
	stopped, err := service.StopSession(ctx, StopSessionRequest{
		UserID:          user.ID,
		RecordingID:     PublicID(session.ID),
		FinalChunkIndex: 0,
		DurationMs:      &duration,
	})
	if err != nil {
		t.Fatalf("StopSession returned error: %v", err)
	}
	if stopped.Status != models.RecordingStatusStopping {
		t.Fatalf("status = %s", stopped.Status)
	}

	if err := service.CancelSession(ctx, user.ID, PublicID(session.ID)); err != nil {
		t.Fatalf("CancelSession returned error: %v", err)
	}

	if len(events.events) < 4 {
		t.Fatalf("events = %#v", events.events)
	}
	if events.events[0].Name != "recording.created" {
		t.Fatalf("first event = %s", events.events[0].Name)
	}
	if events.events[1].Name != "recording.chunk.stored" {
		t.Fatalf("second event = %s", events.events[1].Name)
	}
}

func TestServiceAppendChunkRejectsConflictingRetry(t *testing.T) {
	service, _, user := openRecordingServiceTest(t)
	ctx := context.Background()
	session, err := service.CreateSession(ctx, CreateSessionRequest{UserID: user.ID, MimeType: "audio/webm;codecs=opus"})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	firstChecksum := sha256Hex("chunk")
	if _, err := service.AppendChunk(ctx, AppendChunkRequest{UserID: user.ID, RecordingID: PublicID(session.ID), ChunkIndex: 0, MimeType: session.MimeType, SHA256: &firstChecksum, Body: strings.NewReader("chunk")}); err != nil {
		t.Fatalf("AppendChunk returned error: %v", err)
	}
	secondChecksum := sha256Hex("other")
	_, err = service.AppendChunk(ctx, AppendChunkRequest{UserID: user.ID, RecordingID: PublicID(session.ID), ChunkIndex: 0, MimeType: session.MimeType, SHA256: &secondChecksum, Body: strings.NewReader("other")})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("conflicting retry err = %v, want ErrConflict", err)
	}
}

func TestServiceAppendChunkValidationAndCleanup(t *testing.T) {
	service, _, user := openRecordingServiceTest(t)
	ctx := context.Background()
	session, err := service.CreateSession(ctx, CreateSessionRequest{UserID: user.ID, MimeType: "audio/webm;codecs=opus"})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}

	_, err = service.AppendChunk(ctx, AppendChunkRequest{UserID: user.ID, RecordingID: PublicID(session.ID), ChunkIndex: 0, MimeType: "audio/webm;codecs=opus", Body: strings.NewReader("too-large")})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("oversized chunk err = %v, want ErrValidation", err)
	}
	if _, err := service.repo.FindChunk(ctx, user.ID, session.ID, 0); !errors.Is(err, gorm.ErrRecordNotFound) && err == nil {
		t.Fatalf("oversized chunk metadata was stored")
	}

	badChecksum := strings.Repeat("0", 64)
	_, err = service.AppendChunk(ctx, AppendChunkRequest{UserID: user.ID, RecordingID: PublicID(session.ID), ChunkIndex: 1, MimeType: "audio/webm;codecs=opus", SHA256: &badChecksum, Body: strings.NewReader("chunk")})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("bad checksum err = %v, want ErrValidation", err)
	}
}

func TestServiceRejectsInvalidCreateAndStopRequests(t *testing.T) {
	service, _, user := openRecordingServiceTest(t)
	ctx := context.Background()
	_, err := service.CreateSession(ctx, CreateSessionRequest{UserID: user.ID, MimeType: "video/webm"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid mime err = %v", err)
	}

	session, err := service.CreateSession(ctx, CreateSessionRequest{UserID: user.ID, MimeType: "audio/webm;codecs=opus"})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	tooLong := int64((2 * time.Hour) / time.Millisecond)
	_, err = service.StopSession(ctx, StopSessionRequest{UserID: user.ID, RecordingID: PublicID(session.ID), FinalChunkIndex: 0, DurationMs: &tooLong})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("too-long stop err = %v", err)
	}
}

func TestServiceRetryFinalizationRequiresFailedSession(t *testing.T) {
	service, _, user := openRecordingServiceTest(t)
	ctx := context.Background()
	session, err := service.CreateSession(ctx, CreateSessionRequest{UserID: user.ID, MimeType: "audio/webm;codecs=opus"})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	_, err = service.RetryFinalization(ctx, user.ID, PublicID(session.ID))
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("RetryFinalization err = %v, want ErrConflict", err)
	}
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func int64Ptr(value int64) *int64 {
	return &value
}
