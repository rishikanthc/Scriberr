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

func (c fakeModelCatalog) ResolveTranscriptionModel(ctx context.Context, model string) (ModelInfo, error) {
	if c.err != nil {
		return ModelInfo{}, c.err
	}
	if model == "" {
		model = "whisper-base"
	}
	info, ok := c.models[model]
	if !ok {
		return ModelInfo{}, ErrInvalidModel
	}
	return info, nil
}

func TestServiceCreateNormalizesProfileModelFromCatalog(t *testing.T) {
	repo := &fakeProfileRepository{}
	service := NewService(repo, fakeModelCatalog{models: map[string]ModelInfo{
		"parakeet-v2": {
			ID:     "parakeet-v2",
			Family: "nemo_transducer",
			Capabilities: asrcontract.Capabilities{
				Transcription: true,
			},
		},
	}})

	err := service.Create(context.Background(), &models.TranscriptionProfile{
		UserID: 1,
		Name:   "Parakeet",
		Parameters: models.WhisperXParams{
			Model: "parakeet-v2",
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("profile was not created")
	}
	if repo.created.Parameters.Model != "parakeet-v2" || repo.created.Parameters.ModelFamily != "nemo_transducer" {
		t.Fatalf("profile parameters were not normalized: %#v", repo.created.Parameters)
	}
}

func TestServiceCreateRejectsUnknownModel(t *testing.T) {
	service := NewService(&fakeProfileRepository{}, fakeModelCatalog{models: map[string]ModelInfo{}})

	err := service.Create(context.Background(), &models.TranscriptionProfile{
		UserID: 1,
		Name:   "Invalid",
		Parameters: models.WhisperXParams{
			Model: "large-v3",
		},
	})
	if !errors.Is(err, ErrInvalidModel) {
		t.Fatalf("Create error = %v, want ErrInvalidModel", err)
	}
}
