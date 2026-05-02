package tags

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"scriberr/internal/models"
	"scriberr/internal/repository"

	"gorm.io/gorm"
)

const (
	tagIDPrefix           = "tag_"
	transcriptionIDPrefix = "tr_"
	defaultListLimit      = 100
	maxListLimit          = 500
	maxTagsPerRequest     = 50
	maxTagNameLength      = 120
)

var (
	ErrNotFound   = errors.New("tag not found")
	ErrValidation = errors.New("tag validation failed")
	ErrConflict   = errors.New("tag conflict")

	hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

type EventPublisher interface {
	PublishTagEvent(ctx context.Context, event Event)
}

type Event struct {
	Name            string
	UserID          uint
	TagID           string
	TranscriptionID string
}

type CreateRequest struct {
	UserID      uint
	Name        string
	Color       *string
	Description *string
	WhenToUse   *string
}

type UpdateRequest struct {
	UserID      uint
	TagID       string
	Name        *string
	Color       *string
	Description *string
	WhenToUse   *string
}

type ListRequest struct {
	UserID uint
	Search string
	Offset int
	Limit  int
}

type TranscriptionTagRequest struct {
	UserID          uint
	TranscriptionID string
	TagID           string
}

type ReplaceTranscriptionTagsRequest struct {
	UserID          uint
	TranscriptionID string
	TagIDs          []string
}

type FilterRequest struct {
	UserID   uint
	TagRefs  []string
	MatchAll bool
}

type Service struct {
	tags   repository.TagRepository
	jobs   repository.JobRepository
	events EventPublisher
}

func NewService(tags repository.TagRepository, jobs repository.JobRepository) *Service {
	return &Service{tags: tags, jobs: jobs}
}

func (s *Service) SetEventPublisher(events EventPublisher) {
	s.events = events
}

func (s *Service) CreateTag(ctx context.Context, req CreateRequest) (*models.AudioTag, error) {
	tag, err := buildTag(req.UserID, req.Name, req.Color, req.Description, req.WhenToUse)
	if err != nil {
		return nil, err
	}
	if existing, err := s.tags.FindTagForUserByNormalizedName(ctx, req.UserID, tag.NormalizedName); err == nil && existing != nil {
		return nil, ErrConflict
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err := s.tags.CreateTag(ctx, tag); err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	s.publish(ctx, "tag.created", tag, "")
	return tag, nil
}

func (s *Service) GetTag(ctx context.Context, userID uint, publicTagID string) (*models.AudioTag, error) {
	tagID, err := parsePublicTagID(publicTagID)
	if err != nil {
		return nil, err
	}
	tag, err := s.tags.FindTagForUser(ctx, userID, tagID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return tag, nil
}

func (s *Service) ListTags(ctx context.Context, req ListRequest) ([]models.AudioTag, int64, error) {
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
	return s.tags.ListTagsForUser(ctx, req.UserID, models.NormalizeAudioTagName(req.Search), offset, limit)
}

func (s *Service) UpdateTag(ctx context.Context, req UpdateRequest) (*models.AudioTag, error) {
	tagID, err := parsePublicTagID(req.TagID)
	if err != nil {
		return nil, err
	}
	tag, err := s.tags.FindTagForUser(ctx, req.UserID, tagID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	if req.Name != nil {
		name, normalized, err := validateTagName(*req.Name)
		if err != nil {
			return nil, err
		}
		if normalized != tag.NormalizedName {
			if existing, err := s.tags.FindTagForUserByNormalizedName(ctx, req.UserID, normalized); err == nil && existing.ID != tag.ID {
				return nil, ErrConflict
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
		}
		tag.Name = name
		tag.NormalizedName = normalized
	}
	if req.Color != nil {
		color, err := normalizeColor(req.Color)
		if err != nil {
			return nil, err
		}
		tag.Color = color
	}
	if req.Description != nil {
		tag.Description = normalizeOptionalString(req.Description)
	}
	if req.WhenToUse != nil {
		tag.WhenToUse = normalizeOptionalString(req.WhenToUse)
	}
	if err := s.tags.UpdateTag(ctx, tag); err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrConflict
		}
		return nil, mapNotFound(err)
	}
	updated, err := s.tags.FindTagForUser(ctx, req.UserID, tagID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	s.publish(ctx, "tag.updated", updated, "")
	return updated, nil
}

func (s *Service) DeleteTag(ctx context.Context, userID uint, publicTagID string) error {
	tagID, err := parsePublicTagID(publicTagID)
	if err != nil {
		return err
	}
	tag, err := s.tags.FindTagForUser(ctx, userID, tagID)
	if err != nil {
		return mapNotFound(err)
	}
	if err := s.tags.SoftDeleteTag(ctx, userID, tagID); err != nil {
		return mapNotFound(err)
	}
	s.publish(ctx, "tag.deleted", tag, "")
	return nil
}

func (s *Service) ListTranscriptionTags(ctx context.Context, userID uint, publicTranscriptionID string) ([]models.AudioTag, error) {
	transcriptionID, err := parsePublicTranscriptionID(publicTranscriptionID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireTranscription(ctx, userID, transcriptionID); err != nil {
		return nil, err
	}
	return s.tags.ListTagsForTranscription(ctx, userID, transcriptionID)
}

func (s *Service) ReplaceTranscriptionTags(ctx context.Context, req ReplaceTranscriptionTagsRequest) ([]models.AudioTag, error) {
	transcriptionID, err := parsePublicTranscriptionID(req.TranscriptionID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireTranscription(ctx, req.UserID, transcriptionID); err != nil {
		return nil, err
	}
	tagIDs, err := s.resolveTagIDs(ctx, req.UserID, req.TagIDs)
	if err != nil {
		return nil, err
	}
	if err := s.tags.ReplaceTagsForTranscription(ctx, req.UserID, transcriptionID, tagIDs); err != nil {
		return nil, err
	}
	s.publishTranscriptionTagsUpdated(ctx, req.UserID, transcriptionID)
	return s.tags.ListTagsForTranscription(ctx, req.UserID, transcriptionID)
}

func (s *Service) AddTagToTranscription(ctx context.Context, req TranscriptionTagRequest) ([]models.AudioTag, error) {
	transcriptionID, tagID, err := s.parseTranscriptionTagRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.tags.AddTagToTranscription(ctx, req.UserID, transcriptionID, tagID); err != nil {
		if isUniqueConstraintError(err) {
			return s.tags.ListTagsForTranscription(ctx, req.UserID, transcriptionID)
		}
		return nil, err
	}
	s.publishTranscriptionTagsUpdated(ctx, req.UserID, transcriptionID)
	return s.tags.ListTagsForTranscription(ctx, req.UserID, transcriptionID)
}

func (s *Service) RemoveTagFromTranscription(ctx context.Context, req TranscriptionTagRequest) error {
	transcriptionID, tagID, err := s.parseTranscriptionTagRequest(ctx, req)
	if err != nil {
		return err
	}
	if err := s.tags.RemoveTagFromTranscription(ctx, req.UserID, transcriptionID, tagID); err != nil {
		return err
	}
	s.publishTranscriptionTagsUpdated(ctx, req.UserID, transcriptionID)
	return nil
}

func (s *Service) TranscriptionIDsByTags(ctx context.Context, req FilterRequest) ([]string, error) {
	tagIDs, err := s.resolveTagIDs(ctx, req.UserID, req.TagRefs)
	if err != nil {
		return nil, err
	}
	return s.tags.ListTranscriptionIDsByTags(ctx, req.UserID, tagIDs, req.MatchAll)
}

func (s *Service) parseTranscriptionTagRequest(ctx context.Context, req TranscriptionTagRequest) (string, string, error) {
	transcriptionID, err := parsePublicTranscriptionID(req.TranscriptionID)
	if err != nil {
		return "", "", err
	}
	if _, err := s.requireTranscription(ctx, req.UserID, transcriptionID); err != nil {
		return "", "", err
	}
	tagID, err := parsePublicTagID(req.TagID)
	if err != nil {
		return "", "", err
	}
	if _, err := s.tags.FindTagForUser(ctx, req.UserID, tagID); err != nil {
		return "", "", mapNotFound(err)
	}
	return transcriptionID, tagID, nil
}

func (s *Service) resolveTagIDs(ctx context.Context, userID uint, refs []string) ([]string, error) {
	if len(refs) > maxTagsPerRequest {
		return nil, validationError("too many tags")
	}
	seen := make(map[string]struct{}, len(refs))
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		trimmed := strings.TrimSpace(ref)
		if trimmed == "" {
			continue
		}
		var tag *models.AudioTag
		var err error
		if strings.HasPrefix(trimmed, tagIDPrefix) {
			tagID, parseErr := parsePublicTagID(trimmed)
			if parseErr != nil {
				return nil, parseErr
			}
			tag, err = s.tags.FindTagForUser(ctx, userID, tagID)
		} else {
			normalized := models.NormalizeAudioTagName(trimmed)
			if normalized == "" {
				return nil, validationError("tag is invalid")
			}
			tag, err = s.tags.FindTagForUserByNormalizedName(ctx, userID, normalized)
		}
		if err != nil {
			return nil, mapNotFound(err)
		}
		if _, ok := seen[tag.ID]; ok {
			continue
		}
		seen[tag.ID] = struct{}{}
		ids = append(ids, tag.ID)
	}
	if len(ids) == 0 {
		return nil, validationError("at least one tag is required")
	}
	return ids, nil
}

func (s *Service) requireTranscription(ctx context.Context, userID uint, transcriptionID string) (*models.TranscriptionJob, error) {
	job, err := s.jobs.FindTranscriptionByIDForUser(ctx, transcriptionID, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	return job, nil
}

func (s *Service) publish(ctx context.Context, name string, tag *models.AudioTag, transcriptionID string) {
	if s.events == nil || tag == nil {
		return
	}
	publicTranscriptionID := ""
	if transcriptionID != "" {
		publicTranscriptionID = transcriptionIDPrefix + transcriptionID
	}
	s.events.PublishTagEvent(ctx, Event{
		Name:            name,
		UserID:          tag.UserID,
		TagID:           PublicTagID(tag.ID),
		TranscriptionID: publicTranscriptionID,
	})
}

func (s *Service) publishTranscriptionTagsUpdated(ctx context.Context, userID uint, transcriptionID string) {
	if s.events == nil {
		return
	}
	s.events.PublishTagEvent(ctx, Event{
		Name:            "transcription.tags.updated",
		UserID:          userID,
		TranscriptionID: transcriptionIDPrefix + transcriptionID,
	})
}

func buildTag(userID uint, rawName string, rawColor *string, rawDescription *string, rawWhenToUse *string) (*models.AudioTag, error) {
	name, normalized, err := validateTagName(rawName)
	if err != nil {
		return nil, err
	}
	color, err := normalizeColor(rawColor)
	if err != nil {
		return nil, err
	}
	return &models.AudioTag{
		UserID:         userID,
		Name:           name,
		NormalizedName: normalized,
		Color:          color,
		Description:    normalizeOptionalString(rawDescription),
		WhenToUse:      normalizeOptionalString(rawWhenToUse),
	}, nil
}

func validateTagName(raw string) (string, string, error) {
	name := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if name == "" {
		return "", "", validationError("name is required")
	}
	if len(name) > maxTagNameLength {
		return "", "", validationError("name is too long")
	}
	normalized := models.NormalizeAudioTagName(name)
	if normalized == "" {
		return "", "", validationError("name is invalid")
	}
	return name, normalized, nil
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

func normalizeColor(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if !hexColorPattern.MatchString(trimmed) {
		return nil, validationError("color is invalid")
	}
	return &trimmed, nil
}

func parsePublicTagID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, tagIDPrefix) {
		return "", validationError("tag_id is invalid")
	}
	id := strings.TrimPrefix(trimmed, tagIDPrefix)
	if id == "" || strings.ContainsAny(id, " \t\r\n/") {
		return "", validationError("tag_id is invalid")
	}
	return id, nil
}

func parsePublicTranscriptionID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, transcriptionIDPrefix) {
		return "", validationError("transcription_id is invalid")
	}
	id := strings.TrimPrefix(trimmed, transcriptionIDPrefix)
	if id == "" || strings.ContainsAny(id, " \t\r\n/") {
		return "", validationError("transcription_id is invalid")
	}
	return id, nil
}

func PublicTagID(id string) string {
	if strings.HasPrefix(id, tagIDPrefix) {
		return id
	}
	return tagIDPrefix + id
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
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "duplicated key")
}
