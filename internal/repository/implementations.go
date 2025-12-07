package repository

import (
	"context"
	"scriberr/internal/models"
	"time"

	"gorm.io/gorm"
)

// UserRepository handles user-specific database operations
type UserRepository interface {
	Repository[models.User]
	FindByUsername(ctx context.Context, username string) (*models.User, error)
}

type userRepository struct {
	*BaseRepository[models.User]
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		BaseRepository: NewBaseRepository[models.User](db),
	}
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// JobRepository handles transcription job operations
type JobRepository interface {
	Repository[models.TranscriptionJob]
	FindWithAssociations(ctx context.Context, id string) (*models.TranscriptionJob, error)
	ListWithParams(ctx context.Context, offset, limit int, sortBy, sortOrder, searchQuery string, updatedAfter *time.Time) ([]models.TranscriptionJob, int64, error)
	ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.TranscriptionJob, int64, error)
	UpdateTranscript(ctx context.Context, jobID string, transcript string) error
	CreateExecution(ctx context.Context, execution *models.TranscriptionJobExecution) error
	UpdateExecution(ctx context.Context, execution *models.TranscriptionJobExecution) error
	DeleteExecutionsByJobID(ctx context.Context, jobID string) error
	DeleteMultiTrackFilesByJobID(ctx context.Context, jobID string) error
}

type jobRepository struct {
	*BaseRepository[models.TranscriptionJob]
}

func NewJobRepository(db *gorm.DB) JobRepository {
	return &jobRepository{
		BaseRepository: NewBaseRepository[models.TranscriptionJob](db),
	}
}

func (r *jobRepository) FindWithAssociations(ctx context.Context, id string) (*models.TranscriptionJob, error) {
	var job models.TranscriptionJob
	err := r.db.WithContext(ctx).
		Preload("MultiTrackFiles").
		Where("id = ?", id).
		First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *jobRepository) ListWithParams(ctx context.Context, offset, limit int, sortBy, sortOrder, searchQuery string, updatedAfter *time.Time) ([]models.TranscriptionJob, int64, error) {
	var jobs []models.TranscriptionJob
	var count int64

	db := r.db.WithContext(ctx).Model(&models.TranscriptionJob{})

	// Handle delta sync if updatedAfter provided
	if updatedAfter != nil {
		db = db.Unscoped().Where("updated_at > ?", *updatedAfter)
	}

	// Apply search filter
	if searchQuery != "" {
		search := "%" + searchQuery + "%"
		db = db.Where("title LIKE ? OR audio_path LIKE ?", search, search)
	}

	// Count total matching records
	if err := db.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	if sortBy != "" {
		if sortOrder == "" {
			sortOrder = "desc"
		}
		db = db.Order(sortBy + " " + sortOrder)
	} else {
		// Default sort
		db = db.Order("created_at desc")
	}

	// Apply pagination
	err := db.Offset(offset).Limit(limit).Find(&jobs).Error
	if err != nil {
		return nil, 0, err
	}

	return jobs, count, nil
}

func (r *jobRepository) ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.TranscriptionJob, int64, error) {
	// Note: Currently TranscriptionJob doesn't have a UserID field in the provided model.
	// Assuming we might need to add it or this is a placeholder for future multi-user support.
	// For now, we'll just return all jobs as the current app seems single-user focused or
	// missing the link.
	// TODO: Add UserID to TranscriptionJob model if multi-user isolation is required.
	return r.List(ctx, offset, limit)
}

func (r *jobRepository) UpdateTranscript(ctx context.Context, jobID string, transcript string) error {
	return r.db.WithContext(ctx).Model(&models.TranscriptionJob{}).
		Where("id = ?", jobID).
		Update("transcript", transcript).Error
}

func (r *jobRepository) CreateExecution(ctx context.Context, execution *models.TranscriptionJobExecution) error {
	return r.db.WithContext(ctx).Create(execution).Error
}

func (r *jobRepository) UpdateExecution(ctx context.Context, execution *models.TranscriptionJobExecution) error {
	return r.db.WithContext(ctx).Save(execution).Error
}

func (r *jobRepository) DeleteExecutionsByJobID(ctx context.Context, jobID string) error {
	return r.db.WithContext(ctx).Where("transcription_job_id = ?", jobID).Delete(&models.TranscriptionJobExecution{}).Error
}

func (r *jobRepository) DeleteMultiTrackFilesByJobID(ctx context.Context, jobID string) error {
	return r.db.WithContext(ctx).Where("transcription_job_id = ?", jobID).Delete(&models.MultiTrackFile{}).Error
}

// APIKeyRepository handles API key operations
type APIKeyRepository interface {
	Repository[models.APIKey]
	FindByKey(ctx context.Context, key string) (*models.APIKey, error)
	ListActive(ctx context.Context) ([]models.APIKey, error)
	Revoke(ctx context.Context, id uint) error
}

type apiKeyRepository struct {
	*BaseRepository[models.APIKey]
}

func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{
		BaseRepository: NewBaseRepository[models.APIKey](db),
	}
}

func (r *apiKeyRepository) FindByKey(ctx context.Context, key string) (*models.APIKey, error) {
	var apiKey models.APIKey
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&apiKey).Error
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) ListActive(ctx context.Context) ([]models.APIKey, error) {
	var apiKeys []models.APIKey
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&apiKeys).Error
	if err != nil {
		return nil, err
	}
	return apiKeys, nil
}

func (r *apiKeyRepository) Revoke(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.APIKey{}).Where("id = ?", id).Update("is_active", false).Error
}

// ProfileRepository handles transcription profile operations
type ProfileRepository interface {
	Repository[models.TranscriptionProfile]
	FindDefault(ctx context.Context) (*models.TranscriptionProfile, error)
}

type profileRepository struct {
	*BaseRepository[models.TranscriptionProfile]
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{
		BaseRepository: NewBaseRepository[models.TranscriptionProfile](db),
	}
}

func (r *profileRepository) FindDefault(ctx context.Context) (*models.TranscriptionProfile, error) {
	var profile models.TranscriptionProfile
	err := r.db.WithContext(ctx).Where("is_default = ?", true).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// LLMConfigRepository handles LLM configuration operations
type LLMConfigRepository interface {
	Repository[models.LLMConfig]
	GetActive(ctx context.Context) (*models.LLMConfig, error)
}

type llmConfigRepository struct {
	*BaseRepository[models.LLMConfig]
}

func NewLLMConfigRepository(db *gorm.DB) LLMConfigRepository {
	return &llmConfigRepository{
		BaseRepository: NewBaseRepository[models.LLMConfig](db),
	}
}

func (r *llmConfigRepository) GetActive(ctx context.Context) (*models.LLMConfig, error) {
	var config models.LLMConfig
	err := r.db.WithContext(ctx).Where("is_active = ?", true).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// SummaryRepository handles summary templates and settings
type SummaryRepository interface {
	Repository[models.SummaryTemplate]
	GetSettings(ctx context.Context) (*models.SummarySetting, error)
	SaveSettings(ctx context.Context, settings *models.SummarySetting) error
	SaveSummary(ctx context.Context, summary *models.Summary) error
	GetLatestSummary(ctx context.Context, transcriptionID string) (*models.Summary, error)
	DeleteByTranscriptionID(ctx context.Context, transcriptionID string) error
}

type summaryRepository struct {
	*BaseRepository[models.SummaryTemplate]
}

func NewSummaryRepository(db *gorm.DB) SummaryRepository {
	return &summaryRepository{
		BaseRepository: NewBaseRepository[models.SummaryTemplate](db),
	}
}

func (r *summaryRepository) GetSettings(ctx context.Context) (*models.SummarySetting, error) {
	var settings models.SummarySetting
	// Assuming singleton settings or per-user (but currently model might not have user_id)
	// If it's a singleton table:
	err := r.db.WithContext(ctx).First(&settings).Error
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

func (r *summaryRepository) SaveSettings(ctx context.Context, settings *models.SummarySetting) error {
	return r.db.WithContext(ctx).Save(settings).Error
}

func (r *summaryRepository) SaveSummary(ctx context.Context, summary *models.Summary) error {
	return r.db.WithContext(ctx).Create(summary).Error
}

func (r *summaryRepository) GetLatestSummary(ctx context.Context, transcriptionID string) (*models.Summary, error) {
	var summary models.Summary
	err := r.db.WithContext(ctx).Where("transcription_id = ?", transcriptionID).Order("created_at DESC").First(&summary).Error
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

func (r *summaryRepository) DeleteByTranscriptionID(ctx context.Context, transcriptionID string) error {
	return r.db.WithContext(ctx).Where("transcription_id = ?", transcriptionID).Delete(&models.Summary{}).Error
}

// ChatRepository handles chat sessions and messages
type ChatRepository interface {
	Repository[models.ChatSession]
	GetSessionWithMessages(ctx context.Context, id string) (*models.ChatSession, error)
	GetSessionWithTranscription(ctx context.Context, id string) (*models.ChatSession, error)
	AddMessage(ctx context.Context, message *models.ChatMessage) error
	ListByJob(ctx context.Context, jobID string) ([]models.ChatSession, error)
	DeleteSession(ctx context.Context, id string) error
	GetMessages(ctx context.Context, sessionID string, limit int) ([]models.ChatMessage, error)
	DeleteByJobID(ctx context.Context, jobID string) error
}

type chatRepository struct {
	*BaseRepository[models.ChatSession]
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{
		BaseRepository: NewBaseRepository[models.ChatSession](db),
	}
}

func (r *chatRepository) GetSessionWithMessages(ctx context.Context, id string) (*models.ChatSession, error) {
	var session models.ChatSession
	err := r.db.WithContext(ctx).Preload("Messages").Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *chatRepository) GetSessionWithTranscription(ctx context.Context, id string) (*models.ChatSession, error) {
	var session models.ChatSession
	err := r.db.WithContext(ctx).Preload("Transcription").Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *chatRepository) AddMessage(ctx context.Context, message *models.ChatMessage) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *chatRepository) ListByJob(ctx context.Context, jobID string) ([]models.ChatSession, error) {
	var sessions []models.ChatSession
	err := r.db.WithContext(ctx).Where("transcription_id = ?", jobID).Order("created_at DESC").Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *chatRepository) DeleteSession(ctx context.Context, id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete messages first
		if err := tx.Where("chat_session_id = ?", id).Delete(&models.ChatMessage{}).Error; err != nil {
			return err
		}
		// Delete session
		if err := tx.Delete(&models.ChatSession{}, "id = ?", id).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *chatRepository) DeleteByJobID(ctx context.Context, jobID string) error {
	// Find all sessions for this job
	var sessions []models.ChatSession
	if err := r.db.WithContext(ctx).Where("transcription_id = ?", jobID).Find(&sessions).Error; err != nil {
		return err
	}

	// Delete each session (which deletes messages)
	for _, session := range sessions {
		if err := r.DeleteSession(ctx, session.ID); err != nil {
			return err
		}
	}
	return nil
}

func (r *chatRepository) GetMessages(ctx context.Context, sessionID string, limit int) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	query := r.db.WithContext(ctx).Where("chat_session_id = ?", sessionID).Order("created_at ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// NoteRepository handles notes
type NoteRepository interface {
	Repository[models.Note]
	ListByJob(ctx context.Context, jobID string) ([]models.Note, error)
	DeleteByTranscriptionID(ctx context.Context, transcriptionID string) error
}

type noteRepository struct {
	*BaseRepository[models.Note]
}

func NewNoteRepository(db *gorm.DB) NoteRepository {
	return &noteRepository{
		BaseRepository: NewBaseRepository[models.Note](db),
	}
}

func (r *noteRepository) ListByJob(ctx context.Context, jobID string) ([]models.Note, error) {
	var notes []models.Note
	err := r.db.WithContext(ctx).Where("transcription_id = ?", jobID).Order("created_at DESC").Find(&notes).Error
	if err != nil {
		return nil, err
	}
	return notes, nil
}

func (r *noteRepository) DeleteByTranscriptionID(ctx context.Context, transcriptionID string) error {
	return r.db.WithContext(ctx).Where("transcription_id = ?", transcriptionID).Delete(&models.Note{}).Error
}

// SpeakerMappingRepository handles speaker mappings
type SpeakerMappingRepository interface {
	Repository[models.SpeakerMapping]
	ListByJob(ctx context.Context, jobID string) ([]models.SpeakerMapping, error)
	UpdateMappings(ctx context.Context, jobID string, mappings []models.SpeakerMapping) error
	DeleteByJobID(ctx context.Context, jobID string) error
}

type speakerMappingRepository struct {
	*BaseRepository[models.SpeakerMapping]
}

func NewSpeakerMappingRepository(db *gorm.DB) SpeakerMappingRepository {
	return &speakerMappingRepository{
		BaseRepository: NewBaseRepository[models.SpeakerMapping](db),
	}
}

func (r *speakerMappingRepository) ListByJob(ctx context.Context, jobID string) ([]models.SpeakerMapping, error) {
	var mappings []models.SpeakerMapping
	err := r.db.WithContext(ctx).Where("transcription_job_id = ?", jobID).Find(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *speakerMappingRepository) DeleteByJobID(ctx context.Context, jobID string) error {
	return r.db.WithContext(ctx).Where("transcription_job_id = ?", jobID).Delete(&models.SpeakerMapping{}).Error
}

func (r *speakerMappingRepository) UpdateMappings(ctx context.Context, jobID string, mappings []models.SpeakerMapping) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing mappings for this job
		if err := tx.Where("transcription_job_id = ?", jobID).Delete(&models.SpeakerMapping{}).Error; err != nil {
			return err
		}

		// Create new mappings
		if len(mappings) > 0 {
			if err := tx.Create(&mappings).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
