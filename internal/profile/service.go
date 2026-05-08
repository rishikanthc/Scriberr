package profile

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/asrcontract"
)

var ErrNotFound = errors.New("profile not found")
var ErrInvalidModel = errors.New("profile model is invalid")
var ErrInvalidPipeline = errors.New("profile pipeline is invalid")

type Service struct {
	profiles repository.ProfileRepository
	catalog  ModelCatalog
}

func NewService(profiles repository.ProfileRepository, catalog ...ModelCatalog) *Service {
	var modelCatalog ModelCatalog
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
	params := profile.Parameters
	pipeline, err := s.normalizePipeline(ctx, params.Pipeline)
	if err != nil {
		return err
	}
	params.Pipeline = pipeline
	profile.Parameters = params
	return nil
}

func (s *Service) normalizePipeline(ctx context.Context, steps []models.ASRStep) ([]models.ASRStep, error) {
	if len(steps) == 0 {
		return nil, fmt.Errorf("%w: pipeline must contain at least one step", ErrInvalidPipeline)
	}
	if len(steps) > 8 {
		return nil, fmt.Errorf("%w: pipeline cannot contain more than 8 steps", ErrInvalidPipeline)
	}
	out := make([]models.ASRStep, 0, len(steps))
	var transcriptionCount int
	for i, step := range steps {
		kind := strings.TrimSpace(step.Kind)
		if kind == "" {
			return nil, fmt.Errorf("%w: step %d kind is required", ErrInvalidPipeline, i)
		}
		capability, err := pipelineStepCapability(kind)
		if err != nil {
			return nil, err
		}
		provider := strings.TrimSpace(step.Provider)
		model := strings.TrimSpace(step.Model)
		if s.catalog == nil {
			return nil, fmt.Errorf("%w: model catalog is required", ErrInvalidPipeline)
		}
		info, err := s.catalog.ResolveModel(ctx, provider, model, capability)
		if err != nil {
			return nil, fmt.Errorf("%w: step %d model is invalid", ErrInvalidPipeline, i)
		}
		if kind == models.ASRStepTranscription {
			transcriptionCount++
			if i != 0 {
				return nil, fmt.Errorf("%w: transcription step must be first", ErrInvalidPipeline)
			}
		}
		options, err := validateStepOptions(info.ParameterSchema, step.Options)
		if err != nil {
			return nil, fmt.Errorf("%w: step %d options are invalid", ErrInvalidPipeline, i)
		}
		out = append(out, models.ASRStep{
			Kind:        kind,
			Provider:    provider,
			Model:       info.ID,
			ModelFamily: info.ModelType,
			Options:     options,
		})
	}
	if transcriptionCount != 1 {
		return nil, fmt.Errorf("%w: pipeline must contain exactly one transcription step", ErrInvalidPipeline)
	}
	return out, nil
}

func pipelineStepCapability(kind string) (asrcontract.Capability, error) {
	switch kind {
	case models.ASRStepTranscription:
		return asrcontract.CapabilityTranscription, nil
	case models.ASRStepDiarization:
		return asrcontract.CapabilityDiarization, nil
	case models.ASRStepSpeakerIdentification:
		return asrcontract.CapabilitySpeakerIdentification, nil
	default:
		return "", fmt.Errorf("%w: unsupported step kind %q", ErrInvalidPipeline, kind)
	}
}

func validateStepOptions(schema asrcontract.ParameterSchema, options map[string]any) (map[string]any, error) {
	sanitized := sanitizeStepOptions(schema, options)
	if len(sanitized) == 0 || len(schema) == 0 {
		return sanitized, nil
	}
	values, err := asrcontract.ValidateParameterValues(schema, sanitized)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func sanitizeStepOptions(schema asrcontract.ParameterSchema, options map[string]any) map[string]any {
	if len(options) == 0 {
		return nil
	}
	declared := make(map[string]struct{}, len(schema))
	for _, parameter := range schema {
		key := strings.TrimSpace(parameter.Key)
		if key != "" {
			declared[key] = struct{}{}
		}
	}
	out := make(map[string]any, len(options))
	for key, value := range options {
		if len(out) >= 32 {
			break
		}
		key = strings.TrimSpace(key)
		lower := strings.ToLower(key)
		if key == "" || len(key) > 64 || strings.Contains(lower, "token") || strings.Contains(lower, "api_key") || strings.Contains(lower, "apikey") {
			continue
		}
		if len(declared) == 0 && (strings.Contains(lower, "path") || strings.Contains(lower, "url")) {
			continue
		}
		if sanitized, ok := sanitizeOptionValue(value); ok {
			out[key] = sanitized
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func sanitizeOptionValue(value any) (any, bool) {
	switch typed := value.(type) {
	case nil:
		return nil, true
	case bool, int, int32, int64, float32, float64:
		return typed, true
	case string:
		typed = strings.TrimSpace(typed)
		if len(typed) > 512 {
			typed = typed[:512]
		}
		return typed, true
	case []any:
		if len(typed) > 16 {
			typed = typed[:16]
		}
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			value, ok := sanitizeOptionValue(item)
			if ok {
				out = append(out, value)
			}
		}
		return out, true
	default:
		return nil, false
	}
}

type ModelInfo struct {
	ID              string
	ModelType       string
	Capabilities    asrcontract.Capabilities
	Default         bool
	ParameterSchema asrcontract.ParameterSchema
}

type ModelCatalog interface {
	ResolveTranscriptionModel(ctx context.Context, provider string, model string) (ModelInfo, error)
	ResolveModel(ctx context.Context, provider string, model string, capability asrcontract.Capability) (ModelInfo, error)
}
