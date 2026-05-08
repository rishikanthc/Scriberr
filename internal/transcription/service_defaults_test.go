package transcription

import (
	"context"
	"testing"
	"time"

	"scriberr/internal/models"
	"scriberr/internal/repository"
)

type defaultPipelineJobStore struct {
	created *models.TranscriptionJob
}

func (s *defaultPipelineJobStore) Create(_ context.Context, entity *models.TranscriptionJob) error {
	cp := *entity
	s.created = &cp
	return nil
}

func (s *defaultPipelineJobStore) FindFileByIDForUser(context.Context, string, uint) (*models.TranscriptionJob, error) {
	return nil, ErrFileNotFound
}

func (s *defaultPipelineJobStore) FindTranscriptionByIDForUser(context.Context, string, uint) (*models.TranscriptionJob, error) {
	return nil, ErrNotFound
}

func (s *defaultPipelineJobStore) ListTranscriptionsByUser(context.Context, uint, ListOptions) ([]models.TranscriptionJob, error) {
	return nil, nil
}

func (s *defaultPipelineJobStore) CountStatusesByUser(context.Context, uint) (map[models.JobStatus]int64, error) {
	return nil, nil
}

func (s *defaultPipelineJobStore) CountQueueStatuses(context.Context) ([]models.QueueStatusByUser, error) {
	return nil, nil
}

func (s *defaultPipelineJobStore) UpdateTranscriptionTitle(context.Context, string, uint, string) error {
	return nil
}

func (s *defaultPipelineJobStore) DeleteTranscription(context.Context, string, uint) error {
	return nil
}

func (s *defaultPipelineJobStore) CancelTranscription(context.Context, string, time.Time) error {
	return nil
}

func (s *defaultPipelineJobStore) ListExecutions(context.Context, string) ([]models.TranscriptionJobExecution, error) {
	return nil, nil
}

type missingProfileStore struct{}

func (missingProfileStore) FindByIDForUser(context.Context, string, uint) (*models.TranscriptionProfile, error) {
	return nil, repository.ErrRecordNotFound
}

func (missingProfileStore) FindDefaultByUser(context.Context, uint) (*models.TranscriptionProfile, error) {
	return nil, repository.ErrRecordNotFound
}

func TestServiceSubmitSynthesizesDescriptorResolvedPipelineWithoutProfile(t *testing.T) {
	jobs := &defaultPipelineJobStore{}
	service := NewService(jobs, missingProfileStore{}, nil)
	source := &models.TranscriptionJob{
		ID:             "file-1",
		UserID:         7,
		Status:         models.StatusUploaded,
		AudioPath:      "/tmp/audio.wav",
		SourceFileName: "audio.wav",
	}
	enabled := true

	created, err := service.Submit(context.Background(), SubmitCommand{
		UserID:      7,
		File:        source,
		Diarization: &enabled,
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}
	if created == nil || jobs.created == nil {
		t.Fatal("job was not created")
	}
	steps := jobs.created.Parameters.Pipeline
	if len(steps) != 2 {
		t.Fatalf("pipeline length = %d, want transcription plus diarization: %#v", len(steps), steps)
	}
	if steps[0].Kind != models.ASRStepTranscription || steps[0].Provider != "" || steps[0].Model != "" {
		t.Fatalf("default transcription step should be provider resolved: %#v", steps[0])
	}
	if steps[1].Kind != models.ASRStepDiarization {
		t.Fatalf("diarization step missing: %#v", steps)
	}
	if !jobs.created.Diarization {
		t.Fatalf("Diarization flag was not set")
	}
}
