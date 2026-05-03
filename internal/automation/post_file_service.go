package automation

import (
	"context"
	"errors"
	"strings"

	filesdomain "scriberr/internal/files"
	"scriberr/internal/models"
	transcriptiondomain "scriberr/internal/transcription"

	"gorm.io/gorm"
)

type FileRepository interface {
	FindReadyFileByID(ctx context.Context, id string) (*models.TranscriptionJob, error)
	CountTranscriptionsBySourceFile(ctx context.Context, userID uint, fileID string) (int64, error)
}

type UserRepository interface {
	FindAutomationUserByID(ctx context.Context, userID uint) (*models.User, error)
}

type ProfileRepository interface {
	FindDefaultByUser(ctx context.Context, userID uint) (*models.TranscriptionProfile, error)
}

type LLMConfigRepository interface {
	GetActiveByUser(ctx context.Context, userID uint) (*models.LLMConfig, error)
}

type TranscriptionCreator interface {
	Create(ctx context.Context, cmd transcriptiondomain.CreateCommand) (*models.TranscriptionJob, error)
}

type EventPublisher interface {
	PublishTranscriptionEvent(ctx context.Context, name string, transcriptionID string, payload map[string]any)
}

type Service struct {
	files          FileRepository
	users          UserRepository
	profiles       ProfileRepository
	llmConfigs     LLMConfigRepository
	transcriptions TranscriptionCreator
	events         EventPublisher
}

func NewService(files FileRepository, users UserRepository, profiles ProfileRepository, llmConfigs LLMConfigRepository, transcriptions TranscriptionCreator) *Service {
	return &Service{
		files:          files,
		users:          users,
		profiles:       profiles,
		llmConfigs:     llmConfigs,
		transcriptions: transcriptions,
	}
}

func (s *Service) SetEventPublisher(events EventPublisher) {
	s.events = events
}

func (s *Service) FileReady(ctx context.Context, event filesdomain.ReadyEvent) error {
	if s == nil || s.files == nil || s.users == nil {
		return nil
	}
	if event.FileID == "" || (event.Kind != "" && event.Kind != "audio" && event.Kind != "youtube") {
		return nil
	}
	file, err := s.files.FindReadyFileByID(ctx, event.FileID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if file == nil || file.SourceFileHash != nil || file.Status != models.StatusUploaded {
		return nil
	}
	user, err := s.users.FindAutomationUserByID(ctx, file.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}
	if user.AutoTranscriptionEnabled {
		if err := s.autoTranscribe(ctx, file, user); err != nil {
			return err
		}
	}
	if user.AutoRenameEnabled {
		_, err := s.smallLLMReady(ctx, user.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) autoTranscribe(ctx context.Context, file *models.TranscriptionJob, user *models.User) error {
	if s.profiles == nil || s.transcriptions == nil {
		return nil
	}
	if _, err := s.profiles.FindDefaultByUser(ctx, user.ID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	count, err := s.files.CountTranscriptionsBySourceFile(ctx, user.ID, file.ID)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	transcription, err := s.transcriptions.Create(ctx, transcriptiondomain.CreateCommand{
		UserID: user.ID,
		FileID: file.ID,
	})
	if errors.Is(err, transcriptiondomain.ErrInvalidProfile) || errors.Is(err, transcriptiondomain.ErrFileNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	s.publishCreated(ctx, transcription)
	return nil
}

func (s *Service) smallLLMReady(ctx context.Context, userID uint) (bool, error) {
	if s.llmConfigs == nil {
		return false, nil
	}
	config, err := s.llmConfigs.GetActiveByUser(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return config != nil &&
		strings.TrimSpace(config.Provider) != "" &&
		config.BaseURL != nil &&
		strings.TrimSpace(*config.BaseURL) != "" &&
		config.SmallModel != nil &&
		strings.TrimSpace(*config.SmallModel) != "", nil
}

func (s *Service) publishCreated(ctx context.Context, job *models.TranscriptionJob) {
	if s.events == nil || job == nil {
		return
	}
	id := "tr_" + job.ID
	payload := map[string]any{
		"id":      id,
		"user_id": job.UserID,
		"file_id": fileIDForTranscription(job),
		"status":  string(job.Status),
	}
	s.events.PublishTranscriptionEvent(ctx, "transcription.created", id, payload)
}

func fileIDForTranscription(job *models.TranscriptionJob) string {
	if job.SourceFileHash != nil && *job.SourceFileHash != "" {
		return "file_" + *job.SourceFileHash
	}
	return "file_" + job.ID
}
