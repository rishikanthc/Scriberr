package annotations

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"scriberr/internal/models"
	"scriberr/internal/repository"

	"gorm.io/gorm"
)

const (
	annotationIDPrefix    = "ann_"
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

type Event struct {
	Name            string
	UserID          uint
	TranscriptionID string
	AnnotationID    string
	Kind            models.AnnotationKind
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

type Service struct {
	annotations repository.AnnotationRepository
	jobs        repository.JobRepository
	events      EventPublisher
}

func NewService(annotations repository.AnnotationRepository, jobs repository.JobRepository) *Service {
	return &Service{annotations: annotations, jobs: jobs}
}

func (s *Service) SetEventPublisher(events EventPublisher) {
	s.events = events
}

func (s *Service) CreateAnnotation(ctx context.Context, req CreateRequest) (*models.TranscriptAnnotation, error) {
	transcriptionID, err := parsePublicTranscriptionID(req.TranscriptionID)
	if err != nil {
		return nil, err
	}
	if err := s.requireTranscription(ctx, transcriptionID, req.UserID); err != nil {
		return nil, err
	}
	annotation, err := buildAnnotation(req.UserID, transcriptionID, req.Kind, req.Content, req.Color, req.Quote, req.Anchor)
	if err != nil {
		return nil, err
	}
	if err := s.annotations.CreateAnnotation(ctx, annotation); err != nil {
		return nil, err
	}
	s.publish(ctx, "annotation.created", annotation)
	return annotation, nil
}

func (s *Service) GetAnnotation(ctx context.Context, userID uint, publicTranscriptionID string, publicAnnotationID string) (*models.TranscriptAnnotation, error) {
	transcriptionID, annotationID, err := parseScopedIDs(publicTranscriptionID, publicAnnotationID)
	if err != nil {
		return nil, err
	}
	if err := s.requireTranscription(ctx, transcriptionID, userID); err != nil {
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
	if err := s.requireTranscription(ctx, transcriptionID, req.UserID); err != nil {
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
	annotations, count, err := s.annotations.ListAnnotationsForTranscription(ctx, req.UserID, transcriptionID, req.Kind, offset, limit)
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
	if err := s.requireTranscription(ctx, transcriptionID, req.UserID); err != nil {
		return nil, err
	}
	annotation, err := s.annotations.FindAnnotationForUser(ctx, req.UserID, transcriptionID, annotationID)
	if err != nil {
		return nil, mapNotFound(err)
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
	if annotation.Kind == models.AnnotationKindNote && annotation.Content == nil {
		return nil, validationError("content is required for notes")
	}
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
	if err := s.requireTranscription(ctx, transcriptionID, userID); err != nil {
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

func PublicAnnotationID(id string) string {
	if id == "" || strings.HasPrefix(id, annotationIDPrefix) {
		return id
	}
	return annotationIDPrefix + id
}

func (s *Service) requireTranscription(ctx context.Context, transcriptionID string, userID uint) error {
	if s == nil || s.annotations == nil || s.jobs == nil {
		return fmt.Errorf("annotation service dependencies are required")
	}
	if userID == 0 {
		return validationError("user_id is required")
	}
	if _, err := s.jobs.FindTranscriptionByIDForUser(ctx, transcriptionID, userID); err != nil {
		return mapNotFound(err)
	}
	return nil
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
		Content:         normalizedContent,
		Color:           normalizeOptionalString(color),
		Quote:           quote,
	}
	applyAnchor(annotation, anchor)
	return annotation, nil
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

func parsePublicTranscriptionID(publicID string) (string, error) {
	return parsePublicID(publicID, transcriptionIDPrefix, "transcription_id")
}

func parsePublicAnnotationID(publicID string) (string, error) {
	return parsePublicID(publicID, annotationIDPrefix, "annotation_id")
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
	})
}
