package annotations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/orchestrator"

	"gorm.io/gorm"
)

const (
	annotationIDPrefix    = "ann_"
	annotationEntryPrefix = "annent_"
	transcriptionIDPrefix = "tr_"
	defaultListLimit      = 100
	maxListLimit          = 500
)

var (
	ErrNotFound   = errors.New("annotation not found")
	ErrValidation = errors.New("annotation validation failed")
)

type EventPublisher interface {
	PublishAnnotationEvent(ctx context.Context, event Event)
}

type TranscriptionAccessPolicy interface {
	FindAccessibleTranscription(ctx context.Context, userID uint, transcriptionID string) (*models.TranscriptionJob, error)
}

type Event struct {
	Name            string
	UserID          uint
	TranscriptionID string
	AnnotationID    string
	EntryID         string
	Kind            models.AnnotationKind
	Status          string
}

type Anchor struct {
	StartMS   int64
	EndMS     int64
	StartWord *int
	EndWord   *int
	StartChar *int
	EndChar   *int
	TextHash  *string
}

type CreateRequest struct {
	UserID          uint
	TranscriptionID string
	Kind            models.AnnotationKind
	Content         *string
	Color           *string
	Quote           string
	Anchor          Anchor
}

type ListRequest struct {
	UserID          uint
	TranscriptionID string
	Kind            *models.AnnotationKind
	UpdatedAfter    *time.Time
	Offset          int
	Limit           int
}

type UpdateRequest struct {
	UserID          uint
	TranscriptionID string
	AnnotationID    string
	Content         *string
	Color           *string
	Quote           *string
	Anchor          *Anchor
}

type CreateEntryRequest struct {
	UserID          uint
	TranscriptionID string
	AnnotationID    string
	Content         string
}

type UpdateEntryRequest struct {
	UserID          uint
	TranscriptionID string
	AnnotationID    string
	EntryID         string
	Content         string
}

type Service struct {
	annotations repository.AnnotationRepository
	jobs        repository.JobRepository
	access      TranscriptionAccessPolicy
	events      EventPublisher
}

func NewService(annotations repository.AnnotationRepository, jobs repository.JobRepository) *Service {
	return &Service{annotations: annotations, jobs: jobs, access: ownerAccessPolicy{jobs: jobs}}
}

func (s *Service) SetEventPublisher(events EventPublisher) {
	s.events = events
}

func (s *Service) SetTranscriptionAccessPolicy(access TranscriptionAccessPolicy) {
	if access == nil {
		s.access = ownerAccessPolicy{jobs: s.jobs}
		return
	}
	s.access = access
}

func (s *Service) CreateAnnotation(ctx context.Context, req CreateRequest) (*models.TranscriptAnnotation, error) {
	transcriptionID, err := parsePublicTranscriptionID(req.TranscriptionID)
	if err != nil {
		return nil, err
	}
	transcription, err := s.requireTranscription(ctx, transcriptionID, req.UserID)
	if err != nil {
		return nil, err
	}
	annotation, err := buildAnnotation(req.UserID, transcriptionID, req.Kind, req.Content, req.Color, req.Quote, req.Anchor)
	if err != nil {
		return nil, err
	}
	annotation.Status = anchorStatusForTranscription(transcription, annotation)
	if duplicate, err := s.findDuplicateHighlight(ctx, annotation); err != nil {
		return nil, err
	} else if duplicate != nil {
		return duplicate, nil
	}
	if annotation.Kind == models.AnnotationKindNote {
		entry, err := buildAnnotationEntry(req.UserID, annotation.ID, req.Content)
		if err != nil {
			return nil, err
		}
		if err := s.annotations.CreateAnnotationWithEntry(ctx, annotation, entry); err != nil {
			return nil, err
		}
		annotation.Entries = []models.TranscriptAnnotationEntry{*entry}
	} else {
		if err := s.annotations.CreateAnnotation(ctx, annotation); err != nil {
			return nil, err
		}
	}
	s.publish(ctx, "annotation.created", annotation)
	return annotation, nil
}

func (s *Service) findDuplicateHighlight(ctx context.Context, annotation *models.TranscriptAnnotation) (*models.TranscriptAnnotation, error) {
	if annotation.Kind != models.AnnotationKindHighlight || annotation.Status != models.AnnotationStatusActive {
		return nil, nil
	}
	kind := models.AnnotationKindHighlight
	items, _, err := s.annotations.ListAnnotationsForTranscription(ctx, annotation.UserID, annotation.TranscriptionID, &kind, nil, 0, maxListLimit)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Status == models.AnnotationStatusActive && sameHighlightAnchor(&items[i], annotation) {
			return &items[i], nil
		}
	}
	return nil, nil
}

func (s *Service) GetAnnotation(ctx context.Context, userID uint, publicTranscriptionID string, publicAnnotationID string) (*models.TranscriptAnnotation, error) {
	transcriptionID, annotationID, err := parseScopedIDs(publicTranscriptionID, publicAnnotationID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireTranscription(ctx, transcriptionID, userID); err != nil {
		return nil, err
	}
	annotation, err := s.annotations.FindAnnotationForUser(ctx, userID, transcriptionID, annotationID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return annotation, nil
}

func (s *Service) ListAnnotations(ctx context.Context, req ListRequest) ([]models.TranscriptAnnotation, int64, error) {
	transcriptionID, err := parsePublicTranscriptionID(req.TranscriptionID)
	if err != nil {
		return nil, 0, err
	}
	if req.Kind != nil && !validKind(*req.Kind) {
		return nil, 0, validationError("kind is invalid")
	}
	if _, err := s.requireTranscription(ctx, transcriptionID, req.UserID); err != nil {
		return nil, 0, err
	}
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
	annotations, count, err := s.annotations.ListAnnotationsForTranscription(ctx, req.UserID, transcriptionID, req.Kind, req.UpdatedAfter, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	return annotations, count, nil
}

func (s *Service) UpdateAnnotation(ctx context.Context, req UpdateRequest) (*models.TranscriptAnnotation, error) {
	transcriptionID, annotationID, err := parseScopedIDs(req.TranscriptionID, req.AnnotationID)
	if err != nil {
		return nil, err
	}
	transcription, err := s.requireTranscription(ctx, transcriptionID, req.UserID)
	if err != nil {
		return nil, err
	}
	annotation, err := s.annotations.FindAnnotationForUser(ctx, req.UserID, transcriptionID, annotationID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	if req.Content != nil && annotation.Kind == models.AnnotationKindNote {
		return nil, validationError("note content belongs to entries")
	}
	if req.Content != nil {
		annotation.Content = normalizeOptionalString(req.Content)
	}
	if req.Color != nil {
		annotation.Color = normalizeOptionalString(req.Color)
	}
	if req.Quote != nil {
		quote := strings.TrimSpace(*req.Quote)
		if quote == "" {
			return nil, validationError("quote is required")
		}
		annotation.Quote = quote
	}
	if req.Anchor != nil {
		if err := validateAnchor(*req.Anchor); err != nil {
			return nil, err
		}
		applyAnchor(annotation, *req.Anchor)
	}
	annotation.Status = anchorStatusForTranscription(transcription, annotation)
	if err := s.annotations.UpdateAnnotation(ctx, annotation); err != nil {
		return nil, mapNotFound(err)
	}
	updated, err := s.annotations.FindAnnotationForUser(ctx, req.UserID, transcriptionID, annotationID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	s.publish(ctx, "annotation.updated", updated)
	return updated, nil
}

func (s *Service) DeleteAnnotation(ctx context.Context, userID uint, publicTranscriptionID string, publicAnnotationID string) error {
	transcriptionID, annotationID, err := parseScopedIDs(publicTranscriptionID, publicAnnotationID)
	if err != nil {
		return err
	}
	if _, err := s.requireTranscription(ctx, transcriptionID, userID); err != nil {
		return err
	}
	annotation, err := s.annotations.FindAnnotationForUser(ctx, userID, transcriptionID, annotationID)
	if err != nil {
		return mapNotFound(err)
	}
	if err := s.annotations.SoftDeleteAnnotation(ctx, userID, transcriptionID, annotationID); err != nil {
		return mapNotFound(err)
	}
	s.publish(ctx, "annotation.deleted", annotation)
	return nil
}

func (s *Service) CreateAnnotationEntry(ctx context.Context, req CreateEntryRequest) (*models.TranscriptAnnotationEntry, *models.TranscriptAnnotation, error) {
	transcriptionID, annotationID, err := parseScopedIDs(req.TranscriptionID, req.AnnotationID)
	if err != nil {
		return nil, nil, err
	}
	if _, err := s.requireTranscription(ctx, transcriptionID, req.UserID); err != nil {
		return nil, nil, err
	}
	annotation, err := s.annotations.FindAnnotationForUser(ctx, req.UserID, transcriptionID, annotationID)
	if err != nil {
		return nil, nil, mapNotFound(err)
	}
	if annotation.Kind != models.AnnotationKindNote {
		return nil, nil, validationError("annotation must be a note")
	}
	entry, err := buildAnnotationEntry(req.UserID, annotationID, &req.Content)
	if err != nil {
		return nil, nil, err
	}
	if err := s.annotations.CreateAnnotationEntry(ctx, entry); err != nil {
		return nil, nil, err
	}
	s.publishEntry(ctx, "annotation.entry.created", annotation, entry)
	return entry, annotation, nil
}

func (s *Service) UpdateAnnotationEntry(ctx context.Context, req UpdateEntryRequest) (*models.TranscriptAnnotationEntry, *models.TranscriptAnnotation, error) {
	transcriptionID, annotationID, entryID, err := parseEntryScopedIDs(req.TranscriptionID, req.AnnotationID, req.EntryID)
	if err != nil {
		return nil, nil, err
	}
	if _, err := s.requireTranscription(ctx, transcriptionID, req.UserID); err != nil {
		return nil, nil, err
	}
	annotation, err := s.annotations.FindAnnotationForUser(ctx, req.UserID, transcriptionID, annotationID)
	if err != nil {
		return nil, nil, mapNotFound(err)
	}
	if annotation.Kind != models.AnnotationKindNote {
		return nil, nil, validationError("annotation must be a note")
	}
	entry, err := s.annotations.FindAnnotationEntryForUser(ctx, req.UserID, annotationID, entryID)
	if err != nil {
		return nil, nil, mapNotFound(err)
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, nil, validationError("content is required")
	}
	entry.Content = content
	if err := s.annotations.UpdateAnnotationEntry(ctx, entry); err != nil {
		return nil, nil, mapNotFound(err)
	}
	updated, err := s.annotations.FindAnnotationEntryForUser(ctx, req.UserID, annotationID, entryID)
	if err != nil {
		return nil, nil, mapNotFound(err)
	}
	s.publishEntry(ctx, "annotation.entry.updated", annotation, updated)
	return updated, annotation, nil
}

func (s *Service) DeleteAnnotationEntry(ctx context.Context, userID uint, publicTranscriptionID string, publicAnnotationID string, publicEntryID string) error {
	transcriptionID, annotationID, entryID, err := parseEntryScopedIDs(publicTranscriptionID, publicAnnotationID, publicEntryID)
	if err != nil {
		return err
	}
	if _, err := s.requireTranscription(ctx, transcriptionID, userID); err != nil {
		return err
	}
	annotation, err := s.annotations.FindAnnotationForUser(ctx, userID, transcriptionID, annotationID)
	if err != nil {
		return mapNotFound(err)
	}
	if annotation.Kind != models.AnnotationKindNote {
		return validationError("annotation must be a note")
	}
	entry, err := s.annotations.FindAnnotationEntryForUser(ctx, userID, annotationID, entryID)
	if err != nil {
		return mapNotFound(err)
	}
	if err := s.annotations.SoftDeleteAnnotationEntry(ctx, userID, annotationID, entryID); err != nil {
		return mapNotFound(err)
	}
	s.publishEntry(ctx, "annotation.entry.deleted", annotation, entry)
	return nil
}

func PublicAnnotationID(id string) string {
	if id == "" || strings.HasPrefix(id, annotationIDPrefix) {
		return id
	}
	return annotationIDPrefix + id
}

func PublicAnnotationEntryID(id string) string {
	if id == "" || strings.HasPrefix(id, annotationEntryPrefix) {
		return id
	}
	return annotationEntryPrefix + id
}

func (s *Service) RefreshAnchorStatusesForTranscription(ctx context.Context, userID uint, publicTranscriptionID string) error {
	transcriptionID, err := parsePublicTranscriptionID(publicTranscriptionID)
	if err != nil {
		return err
	}
	transcription, err := s.requireTranscription(ctx, transcriptionID, userID)
	if err != nil {
		return err
	}
	offset := 0
	for {
		items, _, err := s.annotations.ListAnnotationsForTranscription(ctx, userID, transcriptionID, nil, nil, offset, maxListLimit)
		if err != nil {
			return err
		}
		for i := range items {
			status := anchorStatusForTranscription(transcription, &items[i])
			if status != items[i].Status {
				if err := s.annotations.UpdateAnnotationStatus(ctx, userID, transcriptionID, items[i].ID, status); err != nil {
					return mapNotFound(err)
				}
				items[i].Status = status
				s.publish(ctx, "annotation.updated", &items[i])
			}
		}
		if len(items) < maxListLimit {
			break
		}
		offset += len(items)
	}
	return nil
}

func (s *Service) EnqueueForTranscription(ctx context.Context, job *models.TranscriptionJob) error {
	if job == nil {
		return nil
	}
	return s.RefreshAnchorStatusesForTranscription(ctx, job.UserID, transcriptionIDPrefix+job.ID)
}

func (s *Service) requireTranscription(ctx context.Context, transcriptionID string, userID uint) (*models.TranscriptionJob, error) {
	if s == nil || s.annotations == nil || s.jobs == nil || s.access == nil {
		return nil, fmt.Errorf("annotation service dependencies are required")
	}
	if userID == 0 {
		return nil, validationError("user_id is required")
	}
	transcription, err := s.access.FindAccessibleTranscription(ctx, userID, transcriptionID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return transcription, nil
}

type ownerAccessPolicy struct {
	jobs repository.JobRepository
}

func (p ownerAccessPolicy) FindAccessibleTranscription(ctx context.Context, userID uint, transcriptionID string) (*models.TranscriptionJob, error) {
	if p.jobs == nil {
		return nil, fmt.Errorf("job repository is required")
	}
	return p.jobs.FindTranscriptionByIDForUser(ctx, transcriptionID, userID)
}

func buildAnnotation(userID uint, transcriptionID string, kind models.AnnotationKind, content *string, color *string, quote string, anchor Anchor) (*models.TranscriptAnnotation, error) {
	if userID == 0 {
		return nil, validationError("user_id is required")
	}
	if !validKind(kind) {
		return nil, validationError("kind is invalid")
	}
	quote = strings.TrimSpace(quote)
	if quote == "" {
		return nil, validationError("quote is required")
	}
	if err := validateAnchor(anchor); err != nil {
		return nil, err
	}
	normalizedContent := normalizeOptionalString(content)
	if kind == models.AnnotationKindNote && normalizedContent == nil {
		return nil, validationError("content is required for notes")
	}
	annotation := &models.TranscriptAnnotation{
		UserID:          userID,
		TranscriptionID: transcriptionID,
		Kind:            kind,
		Color:           normalizeOptionalString(color),
		Quote:           quote,
	}
	if kind == models.AnnotationKindHighlight {
		annotation.Content = normalizedContent
	}
	applyAnchor(annotation, anchor)
	return annotation, nil
}

func buildAnnotationEntry(userID uint, annotationID string, content *string) (*models.TranscriptAnnotationEntry, error) {
	normalizedContent := normalizeOptionalString(content)
	if normalizedContent == nil {
		return nil, validationError("content is required")
	}
	return &models.TranscriptAnnotationEntry{
		UserID:       userID,
		AnnotationID: annotationID,
		Content:      *normalizedContent,
	}, nil
}

func applyAnchor(annotation *models.TranscriptAnnotation, anchor Anchor) {
	annotation.AnchorStartMS = anchor.StartMS
	annotation.AnchorEndMS = anchor.EndMS
	annotation.AnchorStartWord = anchor.StartWord
	annotation.AnchorEndWord = anchor.EndWord
	annotation.AnchorStartChar = anchor.StartChar
	annotation.AnchorEndChar = anchor.EndChar
	annotation.AnchorTextHash = normalizeOptionalString(anchor.TextHash)
}

func validateAnchor(anchor Anchor) error {
	if anchor.StartMS < 0 || anchor.EndMS < 0 {
		return validationError("anchor timestamps must be non-negative")
	}
	if anchor.EndMS < anchor.StartMS {
		return validationError("anchor end must be greater than or equal to start")
	}
	if anchor.StartWord != nil && *anchor.StartWord < 0 || anchor.EndWord != nil && *anchor.EndWord < 0 {
		return validationError("anchor word indexes must be non-negative")
	}
	if anchor.StartChar != nil && *anchor.StartChar < 0 || anchor.EndChar != nil && *anchor.EndChar < 0 {
		return validationError("anchor character offsets must be non-negative")
	}
	if anchor.StartWord != nil && anchor.EndWord != nil && *anchor.EndWord < *anchor.StartWord {
		return validationError("anchor end word must be greater than or equal to start word")
	}
	if anchor.StartChar != nil && anchor.EndChar != nil && *anchor.EndChar < *anchor.StartChar {
		return validationError("anchor end character must be greater than or equal to start character")
	}
	return nil
}

func anchorStatusForTranscription(transcription *models.TranscriptionJob, annotation *models.TranscriptAnnotation) string {
	if annotation == nil || annotation.AnchorTextHash == nil || strings.TrimSpace(*annotation.AnchorTextHash) == "" {
		return models.AnnotationStatusActive
	}
	actual, ok := anchoredText(transcription, annotation)
	if !ok {
		return models.AnnotationStatusStale
	}
	if anchorTextHashMatches(*annotation.AnchorTextHash, actual) {
		return models.AnnotationStatusActive
	}
	return models.AnnotationStatusStale
}

func anchoredText(transcription *models.TranscriptionJob, annotation *models.TranscriptAnnotation) (string, bool) {
	if transcription == nil || transcription.Transcript == nil {
		return annotation.Quote, strings.TrimSpace(annotation.Quote) != ""
	}
	transcript, err := orchestrator.ParseStoredTranscript(*transcription.Transcript)
	if err != nil {
		return annotation.Quote, strings.TrimSpace(annotation.Quote) != ""
	}
	if annotation.AnchorStartChar != nil && annotation.AnchorEndChar != nil {
		if text, ok := charRange(transcript.Text, *annotation.AnchorStartChar, *annotation.AnchorEndChar); ok {
			return text, true
		}
	}
	if annotation.AnchorStartWord != nil && annotation.AnchorEndWord != nil {
		if text, ok := wordRange(transcript.Words, *annotation.AnchorStartWord, *annotation.AnchorEndWord); ok {
			return text, true
		}
	}
	if strings.TrimSpace(annotation.Quote) != "" {
		return annotation.Quote, true
	}
	return "", false
}

func charRange(text string, start int, end int) (string, bool) {
	runes := []rune(text)
	if start < 0 || end < start || end > len(runes) {
		return "", false
	}
	return string(runes[start:end]), true
}

func wordRange(words []orchestrator.CanonicalWord, start int, end int) (string, bool) {
	if start < 0 || end < start || end >= len(words) {
		return "", false
	}
	values := make([]string, 0, end-start+1)
	for _, word := range words[start : end+1] {
		if value := strings.TrimSpace(word.Word); value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return "", false
	}
	return strings.Join(values, " "), true
}

func anchorTextHashMatches(expected string, actual string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return true
	}
	return expected == HashAnchorText(actual) || expected == hashRawText(actual)
}

func sameHighlightAnchor(existing *models.TranscriptAnnotation, next *models.TranscriptAnnotation) bool {
	if existing == nil || next == nil {
		return false
	}
	if existing.Kind != next.Kind || existing.UserID != next.UserID || existing.TranscriptionID != next.TranscriptionID {
		return false
	}
	if existing.AnchorStartChar != nil && existing.AnchorEndChar != nil && next.AnchorStartChar != nil && next.AnchorEndChar != nil {
		return *existing.AnchorStartChar == *next.AnchorStartChar && *existing.AnchorEndChar == *next.AnchorEndChar
	}
	if existing.AnchorStartWord != nil && existing.AnchorEndWord != nil && next.AnchorStartWord != nil && next.AnchorEndWord != nil {
		return *existing.AnchorStartWord == *next.AnchorStartWord && *existing.AnchorEndWord == *next.AnchorEndWord
	}
	return existing.AnchorStartMS == next.AnchorStartMS &&
		existing.AnchorEndMS == next.AnchorEndMS &&
		normalizeHashText(existing.Quote) == normalizeHashText(next.Quote)
}

func HashAnchorText(text string) string {
	return hashRawText(normalizeHashText(text))
}

func hashRawText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func normalizeHashText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func parseScopedIDs(publicTranscriptionID string, publicAnnotationID string) (string, string, error) {
	transcriptionID, err := parsePublicTranscriptionID(publicTranscriptionID)
	if err != nil {
		return "", "", err
	}
	annotationID, err := parsePublicAnnotationID(publicAnnotationID)
	if err != nil {
		return "", "", err
	}
	return transcriptionID, annotationID, nil
}

func parseEntryScopedIDs(publicTranscriptionID string, publicAnnotationID string, publicEntryID string) (string, string, string, error) {
	transcriptionID, annotationID, err := parseScopedIDs(publicTranscriptionID, publicAnnotationID)
	if err != nil {
		return "", "", "", err
	}
	entryID, err := parsePublicAnnotationEntryID(publicEntryID)
	if err != nil {
		return "", "", "", err
	}
	return transcriptionID, annotationID, entryID, nil
}

func parsePublicTranscriptionID(publicID string) (string, error) {
	return parsePublicID(publicID, transcriptionIDPrefix, "transcription_id")
}

func parsePublicAnnotationID(publicID string) (string, error) {
	return parsePublicID(publicID, annotationIDPrefix, "annotation_id")
}

func parsePublicAnnotationEntryID(publicID string) (string, error) {
	return parsePublicID(publicID, annotationEntryPrefix, "entry_id")
}

func parsePublicID(publicID string, prefix string, field string) (string, error) {
	id := strings.TrimSpace(publicID)
	if !strings.HasPrefix(id, prefix) || len(id) == len(prefix) {
		return "", validationError(field + " is invalid")
	}
	return strings.TrimPrefix(id, prefix), nil
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

func validKind(kind models.AnnotationKind) bool {
	switch kind {
	case models.AnnotationKindHighlight, models.AnnotationKindNote:
		return true
	default:
		return false
	}
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

func (s *Service) publish(ctx context.Context, name string, annotation *models.TranscriptAnnotation) {
	if s.events == nil || annotation == nil {
		return
	}
	s.events.PublishAnnotationEvent(ctx, Event{
		Name:            name,
		UserID:          annotation.UserID,
		TranscriptionID: "tr_" + annotation.TranscriptionID,
		AnnotationID:    PublicAnnotationID(annotation.ID),
		Kind:            annotation.Kind,
		Status:          annotation.Status,
	})
}

func (s *Service) publishEntry(ctx context.Context, name string, annotation *models.TranscriptAnnotation, entry *models.TranscriptAnnotationEntry) {
	if s.events == nil || annotation == nil || entry == nil {
		return
	}
	s.events.PublishAnnotationEvent(ctx, Event{
		Name:            name,
		UserID:          annotation.UserID,
		TranscriptionID: "tr_" + annotation.TranscriptionID,
		AnnotationID:    PublicAnnotationID(annotation.ID),
		EntryID:         PublicAnnotationEntryID(entry.ID),
		Kind:            annotation.Kind,
		Status:          annotation.Status,
	})
}
