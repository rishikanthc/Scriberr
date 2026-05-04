package profile

import (
	"context"
	"errors"
	"strings"

	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/asrcontract"
)

var ErrNotFound = errors.New("profile not found")
var ErrInvalidModel = errors.New("profile model is invalid")

const (
	defaultTranscriptionModel = "whisper-base"
	defaultDiarizationModel   = "diarization-default"
)

type Service struct {
	profiles repository.ProfileRepository
	catalog  ModelCatalog
}

func NewService(profiles repository.ProfileRepository, catalog ...ModelCatalog) *Service {
	modelCatalog := ModelCatalog(defaultModelCatalog())
	if len(catalog) > 0 && catalog[0] != nil {
		modelCatalog = catalog[0]
	}
	return &Service{profiles: profiles, catalog: modelCatalog}
}

func (s *Service) List(ctx context.Context, userID uint) ([]models.TranscriptionProfile, error) {
	items, _, err := s.profiles.ListByUser(ctx, userID, 0, 1000)
	return items, err
}

func (s *Service) Get(ctx context.Context, userID uint, id string) (*models.TranscriptionProfile, error) {
	profile, err := s.profiles.FindByIDForUser(ctx, id, userID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return profile, nil
}

func (s *Service) Create(ctx context.Context, profile *models.TranscriptionProfile) error {
	if err := s.normalizeProfile(ctx, profile); err != nil {
		return err
	}
	return s.profiles.CreateForUser(ctx, profile)
}

func (s *Service) Update(ctx context.Context, profile *models.TranscriptionProfile, defaultChanged bool) error {
	if err := s.normalizeProfile(ctx, profile); err != nil {
		return err
	}
	return s.profiles.UpdateForUser(ctx, profile, defaultChanged)
}

func (s *Service) Delete(ctx context.Context, userID uint, id string) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}
	return s.profiles.DeleteForUser(ctx, id, userID)
}

func (s *Service) SetDefault(ctx context.Context, userID uint, id string) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}
	return s.profiles.SetDefaultForUser(ctx, id, userID)
}

func (s *Service) Exists(ctx context.Context, userID uint, id string) (bool, error) {
	_, err := s.profiles.FindByIDForUser(ctx, id, userID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, repository.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}

func mapNotFound(err error) error {
	if errors.Is(err, repository.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *Service) normalizeProfile(ctx context.Context, profile *models.TranscriptionProfile) error {
	if profile == nil {
		return nil
	}
	model := strings.TrimSpace(profile.Parameters.Model)
	info, err := s.catalog.ResolveTranscriptionModel(ctx, model)
	if err != nil {
		return err
	}
	profile.Parameters.Model = info.ID
	profile.Parameters.ModelFamily = info.Family
	if info.Family == "whisper" {
		profile.Parameters.DecodingMethod = "greedy_search"
	}
	if strings.TrimSpace(profile.Parameters.DiarizeModel) == "" {
		profile.Parameters.DiarizeModel = defaultDiarizationModel
	}
	profile.Parameters.Pipeline = buildDefaultPipeline(profile.Parameters)
	return nil
}

func buildDefaultPipeline(params models.ASRParams) []models.ASRStep {
	pipeline := []models.ASRStep{
		{
			Kind:        models.ASRStepTranscription,
			Provider:    strings.TrimSpace(params.Provider),
			Model:       strings.TrimSpace(params.Model),
			ModelFamily: strings.TrimSpace(params.ModelFamily),
		},
	}
	if params.Diarize {
		pipeline = append(pipeline, models.ASRStep{
			Kind:  models.ASRStepDiarization,
			Model: strings.TrimSpace(params.DiarizeModel),
		})
	}
	return pipeline
}

type ModelInfo struct {
	ID           string
	Family       string
	Capabilities asrcontract.Capabilities
	Default      bool
}

type ModelCatalog interface {
	ResolveTranscriptionModel(ctx context.Context, model string) (ModelInfo, error)
}

type staticModelCatalog map[string]ModelInfo

func defaultModelCatalog() staticModelCatalog {
	models := staticModelCatalog{}
	for _, id := range []string{
		"whisper-tiny",
		"whisper-tiny-en",
		"whisper-base",
		"whisper-base-en",
		"whisper-small",
		"whisper-small-en",
	} {
		models[id] = ModelInfo{
			ID:     id,
			Family: "whisper",
			Capabilities: asrcontract.Capabilities{
				Transcription:  true,
				WordTimestamps: true,
			},
			Default: id == defaultTranscriptionModel,
		}
	}
	for _, id := range []string{"parakeet-v2", "parakeet-v3"} {
		models[id] = ModelInfo{
			ID:     id,
			Family: "nemo_transducer",
			Capabilities: asrcontract.Capabilities{
				Transcription:  true,
				WordTimestamps: true,
			},
		}
	}
	models["canary-180m"] = ModelInfo{
		ID:     "canary-180m",
		Family: "canary",
		Capabilities: asrcontract.Capabilities{
			Transcription:  true,
			WordTimestamps: true,
		},
	}
	return models
}

func (c staticModelCatalog) ResolveTranscriptionModel(ctx context.Context, model string) (ModelInfo, error) {
	if err := ctx.Err(); err != nil {
		return ModelInfo{}, err
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultTranscriptionModel
	}
	info, ok := c[model]
	if !ok || !info.Capabilities.Transcription {
		return ModelInfo{}, ErrInvalidModel
	}
	return info, nil
}
