package recording

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"strings"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/repository"

	"gorm.io/gorm"
)

const (
	recordingIDPrefix = "rec_"
	defaultListLimit  = 50
	maxListLimit      = 200
	maxTitleLength    = 240
)

var (
	ErrNotFound   = errors.New("recording not found")
	ErrValidation = errors.New("recording validation failed")
	ErrConflict   = errors.New("recording state conflict")

	errChunkTooLarge = errors.New("recording chunk too large")
)

type ChunkStorage interface {
	WriteChunk(ctx context.Context, sessionID string, chunkIndex int, mimeType string, source io.Reader) (string, int64, error)
	RemoveChunk(sessionID string, chunkIndex int, mimeType string) error
}

type EventPublisher interface {
	PublishRecordingEvent(ctx context.Context, event Event)
}

type Event struct {
	Name            string
	UserID          uint
	RecordingID     string
	Status          models.RecordingStatus
	Stage           string
	Progress        float64
	FileID          string
	TranscriptionID string
}

type Config struct {
	MaxChunkBytes    int64
	MaxSessionBytes  int64
	MaxDuration      time.Duration
	SessionTTL       time.Duration
	AllowedMimeTypes []string
}

type CreateSessionRequest struct {
	UserID                   uint
	Title                    string
	SourceKind               string
	MimeType                 string
	Codec                    *string
	ChunkDurationMs          *int64
	AutoTranscribe           bool
	ProfileID                *string
	TranscriptionOptionsJSON string
}

type AppendChunkRequest struct {
	UserID      uint
	RecordingID string
	ChunkIndex  int
	MimeType    string
	SHA256      *string
	DurationMs  *int64
	Body        io.Reader
}

type ChunkResult struct {
	Session        *models.RecordingSession
	Chunk          *models.RecordingChunk
	AlreadyStored  bool
	ReceivedChunks int
	ReceivedBytes  int64
}

type StopSessionRequest struct {
	UserID          uint
	RecordingID     string
	FinalChunkIndex int
	DurationMs      *int64
	AutoTranscribe  *bool
}

type ListSessionsRequest struct {
	UserID uint
	Offset int
	Limit  int
}

type Service struct {
	repo    repository.RecordingRepository
	storage ChunkStorage
	events  EventPublisher
	cfg     Config
	now     func() time.Time
}

func NewService(repo repository.RecordingRepository, storage ChunkStorage, cfg Config) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		cfg:     normalizeConfig(cfg),
		now:     time.Now,
	}
}

func (s *Service) SetEventPublisher(events EventPublisher) {
	s.events = events
}

func normalizeConfig(cfg Config) Config {
	if cfg.MaxChunkBytes <= 0 {
		cfg.MaxChunkBytes = 25 << 20
	}
	if cfg.MaxSessionBytes <= 0 {
		cfg.MaxSessionBytes = 2 << 30
	}
	if cfg.MaxDuration <= 0 {
		cfg.MaxDuration = 8 * time.Hour
	}
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 12 * time.Hour
	}
	if len(cfg.AllowedMimeTypes) == 0 {
		cfg.AllowedMimeTypes = []string{"audio/webm;codecs=opus", "audio/webm"}
	}
	return cfg
}

func (s *Service) CreateSession(ctx context.Context, req CreateSessionRequest) (*models.RecordingSession, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("recording service repository is required")
	}
	title := strings.Join(strings.Fields(strings.TrimSpace(req.Title)), " ")
	if len(title) > maxTitleLength {
		return nil, validationError("title is too long")
	}
	mimeType, err := s.validateMimeType(req.MimeType)
	if err != nil {
		return nil, err
	}
	sourceKind := models.RecordingSourceKind(strings.TrimSpace(req.SourceKind))
	if sourceKind == "" {
		sourceKind = models.RecordingSourceKindMicrophone
	}
	if !validSourceKind(sourceKind) {
		return nil, validationError("source_kind is invalid")
	}
	if req.ChunkDurationMs != nil && *req.ChunkDurationMs <= 0 {
		return nil, validationError("chunk_duration_ms must be greater than zero")
	}
	optionsJSON := strings.TrimSpace(req.TranscriptionOptionsJSON)
	if optionsJSON == "" {
		optionsJSON = "{}"
	}
	if !json.Valid([]byte(optionsJSON)) {
		return nil, validationError("transcription options must be valid JSON")
	}
	now := s.now()
	expiresAt := now.Add(s.cfg.SessionTTL)
	session := &models.RecordingSession{
		UserID:                   req.UserID,
		Title:                    optionalString(title),
		Status:                   models.RecordingStatusRecording,
		SourceKind:               sourceKind,
		MimeType:                 mimeType,
		Codec:                    normalizeOptionalString(req.Codec),
		ChunkDurationMs:          req.ChunkDurationMs,
		AutoTranscribe:           req.AutoTranscribe,
		ProfileID:                normalizeOptionalString(req.ProfileID),
		TranscriptionOptionsJSON: optionsJSON,
		StartedAt:                now,
		ExpiresAt:                &expiresAt,
		ProgressStage:            "recording",
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	s.publish(ctx, "recording.created", session)
	return session, nil
}

func (s *Service) GetSession(ctx context.Context, userID uint, publicRecordingID string) (*models.RecordingSession, error) {
	sessionID, err := ParsePublicID(publicRecordingID)
	if err != nil {
		return nil, err
	}
	session, err := s.repo.FindSessionForUser(ctx, userID, sessionID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return session, nil
}

func (s *Service) ListSessions(ctx context.Context, req ListSessionsRequest) ([]models.RecordingSession, int64, error) {
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	limit := req.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	return s.repo.ListSessionsForUser(ctx, req.UserID, offset, limit)
}

func (s *Service) AppendChunk(ctx context.Context, req AppendChunkRequest) (*ChunkResult, error) {
	if s == nil || s.repo == nil || s.storage == nil {
		return nil, fmt.Errorf("recording service dependencies are required")
	}
	sessionID, err := ParsePublicID(req.RecordingID)
	if err != nil {
		return nil, err
	}
	if req.ChunkIndex < 0 {
		return nil, validationError("chunk_index cannot be negative")
	}
	if req.Body == nil {
		return nil, validationError("chunk body is required")
	}
	mimeType, err := s.validateMimeType(req.MimeType)
	if err != nil {
		return nil, err
	}
	session, err := s.repo.FindSessionForUser(ctx, req.UserID, sessionID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	if session.Status != models.RecordingStatusRecording {
		return nil, ErrConflict
	}
	if session.ExpiresAt != nil && !s.now().Before(*session.ExpiresAt) {
		return nil, ErrConflict
	}
	if s.now().Sub(session.StartedAt) > s.cfg.MaxDuration {
		return nil, validationError("recording duration exceeds maximum")
	}
	if normalizeMediaType(session.MimeType) != normalizeMediaType(mimeType) {
		return nil, validationError("chunk mime_type does not match recording session")
	}
	if req.DurationMs != nil {
		if *req.DurationMs < 0 {
			return nil, validationError("chunk duration_ms cannot be negative")
		}
		if time.Duration(*req.DurationMs)*time.Millisecond > s.cfg.MaxDuration {
			return nil, validationError("chunk duration exceeds maximum")
		}
	}
	if session.ReceivedBytes >= s.cfg.MaxSessionBytes {
		return nil, validationError("recording size exceeds maximum")
	}
	expectedSHA, err := normalizeSHA256(req.SHA256)
	if err != nil {
		return nil, err
	}
	if existing, err := s.repo.FindChunk(ctx, req.UserID, sessionID, req.ChunkIndex); err == nil {
		if chunkMatches(existing, expectedSHA, -1) {
			return &ChunkResult{Session: session, Chunk: existing, AlreadyStored: true, ReceivedChunks: session.ReceivedChunks, ReceivedBytes: session.ReceivedBytes}, nil
		}
		return nil, ErrConflict
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hasher := sha256.New()
	limited := &limitedReader{reader: io.TeeReader(req.Body, hasher), remaining: s.cfg.MaxChunkBytes + 1}
	path, size, err := s.storage.WriteChunk(ctx, sessionID, req.ChunkIndex, mimeType, limited)
	if err != nil {
		if errors.Is(err, errChunkTooLarge) {
			return nil, validationError("chunk is too large")
		}
		if errors.Is(err, ErrArtifactExists) {
			return nil, ErrConflict
		}
		return nil, err
	}
	if size > s.cfg.MaxChunkBytes || limited.exceeded {
		_ = s.storage.RemoveChunk(sessionID, req.ChunkIndex, mimeType)
		return nil, validationError("chunk is too large")
	}
	if session.ReceivedBytes+size > s.cfg.MaxSessionBytes {
		_ = s.storage.RemoveChunk(sessionID, req.ChunkIndex, mimeType)
		return nil, validationError("recording size exceeds maximum")
	}
	actualSHA := hex.EncodeToString(hasher.Sum(nil))
	if expectedSHA != nil && *expectedSHA != actualSHA {
		_ = s.storage.RemoveChunk(sessionID, req.ChunkIndex, mimeType)
		return nil, validationError("chunk checksum does not match")
	}
	chunk := &models.RecordingChunk{
		UserID:     req.UserID,
		SessionID:  sessionID,
		ChunkIndex: req.ChunkIndex,
		Path:       path,
		MimeType:   mimeType,
		SHA256:     &actualSHA,
		SizeBytes:  size,
		DurationMs: req.DurationMs,
		ReceivedAt: s.now(),
	}
	if err := s.repo.AddChunk(ctx, chunk); err != nil {
		_ = s.storage.RemoveChunk(sessionID, req.ChunkIndex, mimeType)
		if isUniqueConstraintError(err) {
			if existing, findErr := s.repo.FindChunk(ctx, req.UserID, sessionID, req.ChunkIndex); findErr == nil && chunkMatches(existing, &actualSHA, size) {
				return &ChunkResult{Session: session, Chunk: existing, AlreadyStored: true, ReceivedChunks: session.ReceivedChunks, ReceivedBytes: session.ReceivedBytes}, nil
			}
			return nil, ErrConflict
		}
		if errors.Is(err, gorm.ErrInvalidData) {
			return nil, ErrConflict
		}
		return nil, err
	}
	updated, err := s.repo.FindSessionForUser(ctx, req.UserID, sessionID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	s.publish(ctx, "recording.chunk.stored", updated)
	return &ChunkResult{Session: updated, Chunk: chunk, ReceivedChunks: updated.ReceivedChunks, ReceivedBytes: updated.ReceivedBytes}, nil
}

func (s *Service) StopSession(ctx context.Context, req StopSessionRequest) (*models.RecordingSession, error) {
	sessionID, err := ParsePublicID(req.RecordingID)
	if err != nil {
		return nil, err
	}
	if req.FinalChunkIndex < 0 {
		return nil, validationError("final_chunk_index cannot be negative")
	}
	if req.DurationMs != nil {
		if *req.DurationMs < 0 {
			return nil, validationError("duration_ms cannot be negative")
		}
		if time.Duration(*req.DurationMs)*time.Millisecond > s.cfg.MaxDuration {
			return nil, validationError("recording duration exceeds maximum")
		}
	}
	session, err := s.repo.FindSessionForUser(ctx, req.UserID, sessionID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	switch session.Status {
	case models.RecordingStatusRecording, models.RecordingStatusFailed:
	case models.RecordingStatusStopping, models.RecordingStatusFinalizing, models.RecordingStatusReady:
		if session.ExpectedFinalIndex != nil && *session.ExpectedFinalIndex == req.FinalChunkIndex {
			return session, nil
		}
		return nil, ErrConflict
	default:
		return nil, ErrConflict
	}
	autoTranscribe := session.AutoTranscribe
	if req.AutoTranscribe != nil {
		autoTranscribe = *req.AutoTranscribe
	}
	if err := s.repo.MarkStopping(ctx, req.UserID, sessionID, req.FinalChunkIndex, req.DurationMs, autoTranscribe, s.now()); err != nil {
		return nil, mapNotFound(err)
	}
	updated, err := s.repo.FindSessionForUser(ctx, req.UserID, sessionID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	s.publish(ctx, "recording.stopping", updated)
	return updated, nil
}

func (s *Service) CancelSession(ctx context.Context, userID uint, publicRecordingID string) error {
	sessionID, err := ParsePublicID(publicRecordingID)
	if err != nil {
		return err
	}
	session, err := s.repo.FindSessionForUser(ctx, userID, sessionID)
	if err != nil {
		return mapNotFound(err)
	}
	if session.Status == models.RecordingStatusCanceled {
		return nil
	}
	if session.Status == models.RecordingStatusReady || session.Status == models.RecordingStatusExpired {
		return ErrConflict
	}
	if err := s.repo.CancelSession(ctx, userID, sessionID, s.now()); err != nil {
		return mapNotFound(err)
	}
	session.Status = models.RecordingStatusCanceled
	session.ProgressStage = "canceled"
	s.publish(ctx, "recording.canceled", session)
	return nil
}

func (s *Service) RetryFinalization(ctx context.Context, userID uint, publicRecordingID string) (*models.RecordingSession, error) {
	sessionID, err := ParsePublicID(publicRecordingID)
	if err != nil {
		return nil, err
	}
	session, err := s.repo.FindSessionForUser(ctx, userID, sessionID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	if session.Status != models.RecordingStatusFailed || session.ExpectedFinalIndex == nil {
		return nil, ErrConflict
	}
	if err := s.repo.MarkStopping(ctx, userID, sessionID, *session.ExpectedFinalIndex, session.DurationMs, session.AutoTranscribe, s.now()); err != nil {
		return nil, mapNotFound(err)
	}
	updated, err := s.repo.FindSessionForUser(ctx, userID, sessionID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	s.publish(ctx, "recording.stopping", updated)
	return updated, nil
}

func PublicID(id string) string {
	if strings.HasPrefix(id, recordingIDPrefix) {
		return id
	}
	return recordingIDPrefix + id
}

func ParsePublicID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, recordingIDPrefix) {
		return "", validationError("recording_id is invalid")
	}
	id := strings.TrimPrefix(trimmed, recordingIDPrefix)
	if id == "" || strings.ContainsAny(id, " \t\r\n/\\") {
		return "", validationError("recording_id is invalid")
	}
	return id, nil
}

func (s *Service) validateMimeType(raw string) (string, error) {
	normalized := normalizeMediaType(raw)
	if normalized == "" {
		return "", validationError("mime_type is required")
	}
	for _, allowed := range s.cfg.AllowedMimeTypes {
		if normalizeMediaType(allowed) == normalized {
			return strings.TrimSpace(raw), nil
		}
	}
	return "", validationError("mime_type is not allowed")
}

func normalizeMediaType(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	base, params, err := mime.ParseMediaType(raw)
	if err != nil {
		return strings.ToLower(raw)
	}
	if codec := strings.TrimSpace(params["codecs"]); codec != "" {
		return strings.ToLower(base) + ";codecs=" + strings.ToLower(codec)
	}
	return strings.ToLower(base)
}

func validSourceKind(kind models.RecordingSourceKind) bool {
	switch kind {
	case models.RecordingSourceKindMicrophone, models.RecordingSourceKindTab, models.RecordingSourceKindSystem:
		return true
	default:
		return false
	}
}

func normalizeSHA256(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.ToLower(strings.TrimSpace(*value))
	if len(trimmed) != 64 {
		return nil, validationError("chunk checksum is invalid")
	}
	for _, r := range trimmed {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return nil, validationError("chunk checksum is invalid")
		}
	}
	return &trimmed, nil
}

func chunkMatches(chunk *models.RecordingChunk, sha *string, size int64) bool {
	if chunk == nil {
		return false
	}
	if size >= 0 && chunk.SizeBytes != size {
		return false
	}
	if sha != nil {
		return chunk.SHA256 != nil && strings.EqualFold(*chunk.SHA256, *sha)
	}
	return true
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func validationError(message string) error {
	return fmt.Errorf("%w: %s", ErrValidation, message)
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "duplicated key") || strings.Contains(msg, "constraint failed")
}

func (s *Service) publish(ctx context.Context, name string, session *models.RecordingSession) {
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

type limitedReader struct {
	reader    io.Reader
	remaining int64
	exceeded  bool
}

func (r *limitedReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		r.exceeded = true
		return 0, errChunkTooLarge
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.reader.Read(p)
	r.remaining -= int64(n)
	if r.remaining <= 0 && err == nil {
		r.exceeded = true
		return n, errChunkTooLarge
	}
	return n, err
}
