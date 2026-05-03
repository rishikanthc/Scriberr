package chat

import (
	"context"
	"errors"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/repository"

	"gorm.io/gorm"
)

type Service struct {
	repo       repository.ChatRepository
	llmConfigs repository.LLMConfigRepository
	llmFactory LLMClientFactory
}

func NewService(repo repository.ChatRepository, llmConfigs repository.LLMConfigRepository) *Service {
	return &Service{repo: repo, llmConfigs: llmConfigs, llmFactory: ClientForConfig}
}

func (s *Service) ActiveLLMConfig(ctx context.Context, userID uint) (*models.LLMConfig, error) {
	config, err := s.llmConfigs.GetActiveByUser(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProviderMissing
	}
	return config, err
}

func (s *Service) CreateSession(ctx context.Context, session *models.ChatSession) error {
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return chatNotFound(err)
	}
	_, err := NewContextBuilder(s.repo, ApproxTokenEstimator{}).AddParentSource(ctx, session.UserID, session.ID)
	return chatNotFound(err)
}

func (s *Service) ListSessions(ctx context.Context, userID uint, parentID string) ([]models.ChatSession, error) {
	sessions, _, err := s.repo.ListSessionsForTranscription(ctx, userID, parentID, 0, 100)
	return sessions, err
}

func (s *Service) GetSession(ctx context.Context, userID uint, sessionID string) (*models.ChatSession, error) {
	session, err := s.repo.FindSessionForUser(ctx, userID, sessionID)
	if err != nil {
		return nil, chatNotFound(err)
	}
	return session, nil
}

func (s *Service) UpdateSession(ctx context.Context, session *models.ChatSession) error {
	return chatNotFound(s.repo.UpdateSession(ctx, session))
}

func (s *Service) DeleteSession(ctx context.Context, userID uint, sessionID string) error {
	return chatNotFound(s.repo.DeleteSession(ctx, userID, sessionID))
}

func (s *Service) ListMessages(ctx context.Context, userID uint, sessionID string, limit int) ([]models.ChatMessage, error) {
	messages, _, err := s.repo.ListMessages(ctx, userID, sessionID, 0, limit)
	return messages, err
}

func (s *Service) ListContextSources(ctx context.Context, userID uint, sessionID string, enabledOnly bool) ([]models.ChatContextSource, error) {
	return s.repo.ListContextSources(ctx, userID, sessionID, enabledOnly)
}

func (s *Service) AddTranscriptSource(ctx context.Context, userID uint, sessionID string, transcriptionID string) (*models.ChatContextSource, error) {
	mutation, err := NewContextBuilder(s.repo, ApproxTokenEstimator{}).AddTranscriptSource(ctx, userID, sessionID, transcriptionID, models.ChatContextSourceKindTranscript)
	if err != nil {
		return nil, chatNotFound(err)
	}
	return mutation.Source, nil
}

func (s *Service) SetContextSourceEnabled(ctx context.Context, userID uint, sessionID string, sourceID string, enabled bool) error {
	return chatNotFound(s.repo.SetContextSourceEnabled(ctx, userID, sessionID, sourceID, enabled))
}

func (s *Service) FindContextSource(ctx context.Context, userID uint, sessionID string, sourceID string) (*models.ChatContextSource, error) {
	source, err := s.repo.FindContextSourceForUser(ctx, userID, sessionID, sourceID)
	if err != nil {
		return nil, chatNotFound(err)
	}
	return source, nil
}

func (s *Service) DeleteContextSource(ctx context.Context, userID uint, sessionID string, sourceID string) error {
	return chatNotFound(s.repo.DeleteContextSource(ctx, userID, sessionID, sourceID))
}

func (s *Service) CreateMessage(ctx context.Context, message *models.ChatMessage) error {
	return s.repo.CreateMessage(ctx, message)
}

func (s *Service) UpdateMessage(ctx context.Context, message *models.ChatMessage) error {
	return s.repo.UpdateMessage(ctx, message)
}

func (s *Service) CreateGenerationRun(ctx context.Context, run *models.ChatGenerationRun) error {
	return s.repo.CreateGenerationRun(ctx, run)
}

func (s *Service) FindGenerationRun(ctx context.Context, userID uint, runID string) (*models.ChatGenerationRun, error) {
	run, err := s.repo.FindGenerationRunForUser(ctx, userID, runID)
	if err != nil {
		return nil, chatNotFound(err)
	}
	return run, nil
}

func (s *Service) UpdateGenerationRunStatus(ctx context.Context, userID uint, runID string, status models.ChatGenerationRunStatus, at time.Time, errorMessage *string) error {
	return chatNotFound(s.repo.UpdateGenerationRunStatus(ctx, userID, runID, status, at, errorMessage))
}

func (s *Service) BuildContext(ctx context.Context, userID uint, sessionID string, window int) (string, error) {
	built, err := NewContextBuilder(s.repo, ApproxTokenEstimator{}).Build(ctx, userID, sessionID, BuildOptions{Budget: ContextBudget{ContextWindow: window, ReservedResponse: 512, ReservedChat: 1024, SafetyMarginTokens: 128}})
	return built.Content, err
}

func chatNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
