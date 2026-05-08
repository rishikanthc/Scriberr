package profile

import (
	"context"
	"errors"
	"testing"

	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/asrcontract"
)

type fakeProfileRepository struct {
	created *models.TranscriptionProfile
	updated *models.TranscriptionProfile
}

func (r *fakeProfileRepository) ListByUser(context.Context, uint, int, int) ([]models.TranscriptionProfile, int64, error) {
	return nil, 0, nil
}
func (r *fakeProfileRepository) Create(context.Context, *models.TranscriptionProfile) error {
	return nil
}
func (r *fakeProfileRepository) FindByID(context.Context, interface{}) (*models.TranscriptionProfile, error) {
	return nil, repository.ErrRecordNotFound
}
func (r *fakeProfileRepository) Update(context.Context, *models.TranscriptionProfile) error {
	return nil
}
func (r *fakeProfileRepository) Delete(context.Context, interface{}) error { return nil }
func (r *fakeProfileRepository) List(context.Context, int, int) ([]models.TranscriptionProfile, int64, error) {
	return nil, 0, nil
}
func (r *fakeProfileRepository) FindByIDForUser(context.Context, string, uint) (*models.TranscriptionProfile, error) {
	return nil, repository.ErrRecordNotFound
}
func (r *fakeProfileRepository) FindDefaultByUser(context.Context, uint) (*models.TranscriptionProfile, error) {
	return nil, repository.ErrRecordNotFound
}
func (r *fakeProfileRepository) FindByNameForUser(context.Context, uint, string) (*models.TranscriptionProfile, error) {
	return nil, repository.ErrRecordNotFound
}
func (r *fakeProfileRepository) CreateForUser(ctx context.Context, profile *models.TranscriptionProfile) error {
	cp := *profile
	r.created = &cp
	return nil
}
func (r *fakeProfileRepository) UpdateForUser(ctx context.Context, profile *models.TranscriptionProfile, defaultChanged bool) error {
	cp := *profile
	r.updated = &cp
	return nil
}
func (r *fakeProfileRepository) DeleteForUser(context.Context, string, uint) error     { return nil }
func (r *fakeProfileRepository) SetDefaultForUser(context.Context, string, uint) error { return nil }

type fakeModelCatalog struct {
	models map[string]ModelInfo
	err    error
}

func (c fakeModelCatalog) ResolveTranscriptionModel(ctx context.Context, provider string, model string) (ModelInfo, error) {
	return c.ResolveModel(ctx, provider, model, asrcontract.CapabilityTranscription)
}

func (c fakeModelCatalog) ResolveModel(ctx context.Context, provider string, model string, capability asrcontract.Capability) (ModelInfo, error) {
	if c.err != nil {
		return ModelInfo{}, c.err
	}
	if provider != "" {
		providerModel := provider + "/" + model
		if _, ok := c.models[providerModel]; ok {
			model = providerModel
		}
	}
	if model == "" {
		model = "whisper-base"
	}
	info, ok := c.models[model]
	if !ok {
		return ModelInfo{}, ErrInvalidModel
	}
	if !info.Capabilities.Supports(capability) {
		return ModelInfo{}, ErrInvalidModel
	}
	return info, nil
}

func TestServiceCreateRejectsModelFromDifferentProvider(t *testing.T) {
	service := NewService(&fakeProfileRepository{}, fakeModelCatalog{models: map[string]ModelInfo{
		"remote/parakeet-v2": {
			ID:        "parakeet-v2",
			ModelType: "nemo_transducer",
			Capabilities: asrcontract.Capabilities{
				Transcription: true,
			},
		},
	}})

	err := service.Create(context.Background(), &models.TranscriptionProfile{
		UserID: 1,
		Name:   "Wrong provider",
		Parameters: models.ASRParams{
			Pipeline: []models.ASRStep{{Kind: models.ASRStepTranscription, Provider: "local", Model: "parakeet-v2"}},
		},
	})
	if !errors.Is(err, ErrInvalidPipeline) {
		t.Fatalf("Create error = %v, want ErrInvalidPipeline", err)
	}
}

func TestServiceCreateNormalizesProfileModelFromCatalog(t *testing.T) {
	repo := &fakeProfileRepository{}
	service := NewService(repo, fakeModelCatalog{models: map[string]ModelInfo{
		"parakeet-v2": {
			ID:        "parakeet-v2",
			ModelType: "nemo_transducer",
			Capabilities: asrcontract.Capabilities{
				Transcription: true,
			},
		},
	}})

	err := service.Create(context.Background(), &models.TranscriptionProfile{
		UserID: 1,
		Name:   "Parakeet",
		Parameters: models.ASRParams{
			Pipeline: []models.ASRStep{{Kind: models.ASRStepTranscription, Model: "parakeet-v2"}},
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("profile was not created")
	}
	if len(repo.created.Parameters.Pipeline) != 1 {
		t.Fatalf("profile pipeline length = %d, want 1", len(repo.created.Parameters.Pipeline))
	}
	step := repo.created.Parameters.Pipeline[0]
	if step.Kind != models.ASRStepTranscription || step.Model != "parakeet-v2" || step.ModelFamily != "nemo_transducer" {
		t.Fatalf("profile pipeline was not normalized: %#v", step)
	}
}

func TestServiceCreateRejectsUnknownModel(t *testing.T) {
	service := NewService(&fakeProfileRepository{}, fakeModelCatalog{models: map[string]ModelInfo{}})

	err := service.Create(context.Background(), &models.TranscriptionProfile{
		UserID: 1,
		Name:   "Invalid",
		Parameters: models.ASRParams{
			Pipeline: []models.ASRStep{{Kind: models.ASRStepTranscription, Model: "large-v3"}},
		},
	})
	if !errors.Is(err, ErrInvalidPipeline) {
		t.Fatalf("Create error = %v, want ErrInvalidPipeline", err)
	}
}

func TestServiceCreateRejectsMissingPipeline(t *testing.T) {
	service := NewService(&fakeProfileRepository{}, fakeModelCatalog{models: map[string]ModelInfo{}})

	err := service.Create(context.Background(), &models.TranscriptionProfile{
		UserID:     1,
		Name:       "Missing pipeline",
		Parameters: models.ASRParams{},
	})
	if !errors.Is(err, ErrInvalidPipeline) {
		t.Fatalf("Create error = %v, want ErrInvalidPipeline", err)
	}
}

func TestServiceCreatePersistsMultiStepPipelineAndSanitizesOptions(t *testing.T) {
	repo := &fakeProfileRepository{}
	service := NewService(repo, fakeModelCatalog{models: map[string]ModelInfo{
		"parakeet-v2": {
			ID:        "parakeet-v2",
			ModelType: "nemo_transducer",
			Capabilities: asrcontract.Capabilities{
				Transcription: true,
			},
		},
		"diarization-default": {
			ID:        "diarization-default",
			ModelType: "diarization",
			Capabilities: asrcontract.Capabilities{
				Diarization: true,
			},
		},
	}})

	err := service.Create(context.Background(), &models.TranscriptionProfile{
		UserID: 1,
		Name:   "Pipeline",
		Parameters: models.ASRParams{
			Pipeline: []models.ASRStep{
				{Kind: models.ASRStepTranscription, Provider: "local", Model: "parakeet-v2", Options: map[string]any{"beam": float64(4), "api_key": "secret"}},
				{Kind: models.ASRStepDiarization, Provider: "remote-diarizer", Model: "diarization-default", Options: map[string]any{"threshold": float64(0.7), "audio_path": "/secret.wav"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("profile was not created")
	}
	if len(repo.created.Parameters.Pipeline) != 2 {
		t.Fatalf("pipeline length = %d", len(repo.created.Parameters.Pipeline))
	}
	if _, ok := repo.created.Parameters.Pipeline[0].Options["api_key"]; ok {
		t.Fatalf("sensitive option was not removed: %#v", repo.created.Parameters.Pipeline[0].Options)
	}
	if _, ok := repo.created.Parameters.Pipeline[1].Options["audio_path"]; ok {
		t.Fatalf("path option was not removed: %#v", repo.created.Parameters.Pipeline[1].Options)
	}
	if repo.created.Parameters.Pipeline[1].Provider != "remote-diarizer" {
		t.Fatalf("provider not preserved: %#v", repo.created.Parameters.Pipeline[1])
	}
}

func TestServiceCreateValidatesStepOptionsFromModelSchema(t *testing.T) {
	repo := &fakeProfileRepository{}
	service := NewService(repo, fakeModelCatalog{models: map[string]ModelInfo{
		"schema-model": {
			ID:        "schema-model",
			ModelType: "whisper",
			Capabilities: asrcontract.Capabilities{
				Transcription: true,
			},
			ParameterSchema: asrcontract.ParameterSchema{
				{
					Key:     asrcontract.CommonParameterDecodingMethod,
					Label:   "Decoding",
					Type:    asrcontract.ParameterTypeEnum,
					Default: "greedy_search",
					Options: []asrcontract.ParameterOption{
						{Value: "greedy_search", Label: "Greedy"},
						{Value: "modified_beam_search", Label: "Beam"},
					},
					Scope: asrcontract.ParameterScopeDecoding,
				},
				{
					Key:   "sherpa.whisper.tail_paddings",
					Label: "Tail paddings",
					Type:  asrcontract.ParameterTypeInteger,
					Min:   floatPtr(-1),
					Max:   floatPtr(16),
					Scope: asrcontract.ParameterScopeDecoding,
				},
				{
					Key:   "speaker.embedding_path",
					Label: "Speaker embedding",
					Type:  asrcontract.ParameterTypePathRef,
					Scope: asrcontract.ParameterScopeOutput,
				},
				{
					Key:      "sherpa.model_type",
					Label:    "Sherpa model type",
					Type:     asrcontract.ParameterTypeString,
					Default:  "nemo_transducer",
					Scope:    asrcontract.ParameterScopeModel,
					ReadOnly: true,
				},
			},
		},
	}})

	err := service.Create(context.Background(), &models.TranscriptionProfile{
		UserID: 1,
		Name:   "Schema",
		Parameters: models.ASRParams{
			Pipeline: []models.ASRStep{{
				Kind:  models.ASRStepTranscription,
				Model: "schema-model",
				Options: map[string]any{
					asrcontract.CommonParameterDecodingMethod: "modified_beam_search",
					"sherpa.whisper.tail_paddings":            float64(4),
					"speaker.embedding_path":                  "speaker-a",
				},
			}},
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if got := repo.created.Parameters.Pipeline[0].Options["sherpa.whisper.tail_paddings"]; got != int64(4) {
		t.Fatalf("schema option was not normalized: %#v", repo.created.Parameters.Pipeline[0].Options)
	}
	if got := repo.created.Parameters.Pipeline[0].Options["speaker.embedding_path"]; got != "speaker-a" {
		t.Fatalf("declared path_ref option was not preserved: %#v", repo.created.Parameters.Pipeline[0].Options)
	}

	err = service.Create(context.Background(), &models.TranscriptionProfile{
		UserID: 1,
		Name:   "Invalid schema option",
		Parameters: models.ASRParams{
			Pipeline: []models.ASRStep{{
				Kind:    models.ASRStepTranscription,
				Model:   "schema-model",
				Options: map[string]any{asrcontract.CommonParameterDecodingMethod: "unsupported"},
			}},
		},
	})
	if !errors.Is(err, ErrInvalidPipeline) {
		t.Fatalf("Create error = %v, want ErrInvalidPipeline", err)
	}

	err = service.Create(context.Background(), &models.TranscriptionProfile{
		UserID: 1,
		Name:   "Read-only default",
		Parameters: models.ASRParams{
			Pipeline: []models.ASRStep{{
				Kind:    models.ASRStepTranscription,
				Model:   "schema-model",
				Options: map[string]any{"sherpa.model_type": "nemo_transducer"},
			}},
		},
	})
	if err != nil {
		t.Fatalf("Create with default read-only value returned error: %v", err)
	}

	err = service.Create(context.Background(), &models.TranscriptionProfile{
		UserID: 1,
		Name:   "Changed read-only",
		Parameters: models.ASRParams{
			Pipeline: []models.ASRStep{{
				Kind:    models.ASRStepTranscription,
				Model:   "schema-model",
				Options: map[string]any{"sherpa.model_type": "whisper"},
			}},
		},
	})
	if !errors.Is(err, ErrInvalidPipeline) {
		t.Fatalf("Create error = %v, want ErrInvalidPipeline", err)
	}
}

func floatPtr(v float64) *float64 { return &v }
