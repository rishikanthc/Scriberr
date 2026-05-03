package transcription

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Queue interface {
	Enqueue(ctx context.Context, jobID string) error
	Cancel(ctx context.Context, userID uint, jobID string) error
}

type Service struct {
	jobs     repository.JobRepository
	profiles repository.ProfileRepository
	queue    Queue
}

type CreateCommand struct {
	UserID      uint
	FileID      string
	Title       string
	ProfileID   string
	Language    string
	Diarization *bool
}

type SubmitCommand struct {
	UserID      uint
	File        *models.TranscriptionJob
	Title       string
	ProfileID   string
	Language    string
	Diarization *bool
}

type ListOptions = repository.TranscriptionListOptions

type ListCursor = repository.TranscriptionListCursor

var (
	ErrNotFound       = errors.New("transcription not found")
	ErrFileNotFound   = errors.New("file not found")
	ErrInvalidProfile = errors.New("profile_id is invalid")
	ErrStateConflict  = errors.New("transcription state conflict")
)

func NewService(jobs repository.JobRepository, profiles repository.ProfileRepository, queue Queue) *Service {
	return &Service{jobs: jobs, profiles: profiles, queue: queue}
}

func (s *Service) SetQueue(queue Queue) {
	s.queue = queue
}

func (s *Service) Create(ctx context.Context, cmd CreateCommand) (*models.TranscriptionJob, error) {
	if s == nil || s.jobs == nil {
		return nil, fmt.Errorf("transcription service is not configured")
	}
	sourceID := strings.TrimSpace(cmd.FileID)
	source, err := s.jobs.FindFileByIDForUser(ctx, sourceID, cmd.UserID)
	if err != nil {
		return nil, ErrFileNotFound
	}
	return s.createFromSource(ctx, cmd.UserID, source, cmd.Title, cmd.ProfileID, cmd.Language, cmd.Diarization)
}

func (s *Service) Submit(ctx context.Context, cmd SubmitCommand) (*models.TranscriptionJob, error) {
	if s == nil || s.jobs == nil {
		return nil, fmt.Errorf("transcription service is not configured")
	}
	if cmd.File == nil || cmd.File.UserID != cmd.UserID {
		return nil, ErrFileNotFound
	}
	return s.createFromSource(ctx, cmd.UserID, cmd.File, cmd.Title, cmd.ProfileID, cmd.Language, cmd.Diarization)
}

func (s *Service) List(ctx context.Context, userID uint, opts ListOptions) ([]models.TranscriptionJob, error) {
	if s == nil || s.jobs == nil {
		return nil, fmt.Errorf("transcription service is not configured")
	}
	return s.jobs.ListTranscriptionsByUser(ctx, userID, opts)
}

func (s *Service) Get(ctx context.Context, userID uint, id string) (*models.TranscriptionJob, error) {
	if s == nil || s.jobs == nil {
		return nil, fmt.Errorf("transcription service is not configured")
	}
	job, err := s.jobs.FindTranscriptionByIDForUser(ctx, id, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	return job, nil
}

func (s *Service) UpdateTitle(ctx context.Context, userID uint, id string, title string) (*models.TranscriptionJob, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if err := s.jobs.UpdateTranscriptionTitle(ctx, id, userID, title); err != nil {
		return nil, ErrNotFound
	}
	return s.Get(ctx, userID, id)
}

func (s *Service) Delete(ctx context.Context, userID uint, id string) error {
	if s == nil || s.jobs == nil {
		return fmt.Errorf("transcription service is not configured")
	}
	if err := s.jobs.DeleteTranscription(ctx, id, userID); err != nil {
		return ErrNotFound
	}
	return nil
}

func (s *Service) Cancel(ctx context.Context, userID uint, id string) (*models.TranscriptionJob, error) {
	job, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	switch job.Status {
	case models.StatusCompleted, models.StatusFailed, models.StatusStopped, models.StatusCanceled:
		return nil, ErrStateConflict
	}
	if s.queue != nil {
		if err := s.queue.Cancel(ctx, userID, id); err != nil {
			return nil, err
		}
	} else if err := s.jobs.CancelTranscription(ctx, id, time.Now()); err != nil {
		return nil, err
	}
	job.Status = models.StatusStopped
	job.ProgressStage = "stopped"
	return job, nil
}

func (s *Service) Retry(ctx context.Context, userID uint, id string) (*models.TranscriptionJob, error) {
	source, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	sourceFileID := ""
	if source.SourceFileHash != nil {
		sourceFileID = *source.SourceFileHash
	}
	retry := &models.TranscriptionJob{
		ID:             uuid.NewString(),
		UserID:         source.UserID,
		Title:          source.Title,
		Status:         models.StatusPending,
		AudioPath:      source.AudioPath,
		SourceFileName: source.SourceFileName,
		SourceFileHash: &sourceFileID,
		Language:       source.Language,
		Diarization:    source.Diarization,
		Parameters:     source.Parameters,
	}
	if err := s.jobs.Create(ctx, retry); err != nil {
		return nil, err
	}
	if err := s.enqueue(ctx, retry.ID); err != nil {
		return nil, err
	}
	return retry, nil
}

func (s *Service) OpenAudio(ctx context.Context, userID uint, id string) (*os.File, *models.TranscriptionJob, error) {
	job, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}
	file, err := os.Open(job.AudioPath)
	if err != nil {
		return nil, nil, ErrNotFound
	}
	return file, job, nil
}

func (s *Service) createFromSource(ctx context.Context, userID uint, source *models.TranscriptionJob, title string, profileID string, language string, diarization *bool) (*models.TranscriptionJob, error) {
	params, err := s.resolveParams(ctx, userID, profileID)
	if err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" && source.Title != nil {
		title = *source.Title
	}
	sourceFileID := source.ID
	job := &models.TranscriptionJob{
		ID:             uuid.NewString(),
		UserID:         userID,
		Title:          &title,
		Status:         models.StatusPending,
		AudioPath:      source.AudioPath,
		SourceFileName: source.SourceFileName,
		SourceFileHash: &sourceFileID,
		Parameters:     params,
	}
	if language != "" {
		job.Language = &language
		job.Parameters.Language = &language
	}
	if diarization != nil {
		job.Parameters.Diarize = *diarization
	}
	job.Diarization = job.Parameters.Diarize
	if err := s.jobs.Create(ctx, job); err != nil {
		return nil, err
	}
	if err := s.enqueue(ctx, job.ID); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Service) resolveParams(ctx context.Context, userID uint, profileID string) (models.WhisperXParams, error) {
	if s.profiles == nil {
		return models.WhisperXParams{}, nil
	}
	if strings.TrimSpace(profileID) != "" {
		profile, err := s.profiles.FindByIDForUser(ctx, profileID, userID)
		if err != nil {
			return models.WhisperXParams{}, ErrInvalidProfile
		}
		return profile.Parameters, nil
	}
	profile, err := s.profiles.FindDefaultByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.WhisperXParams{}, nil
		}
		return models.WhisperXParams{}, err
	}
	return profile.Parameters, nil
}

func (s *Service) enqueue(ctx context.Context, jobID string) error {
	if s.queue == nil {
		return nil
	}
	return s.queue.Enqueue(ctx, jobID)
}
