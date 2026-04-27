package orchestrator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/engineprovider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeProvider struct {
	id         string
	transcribe *engineprovider.TranscriptionResult
	diarize    *engineprovider.DiarizationResult
	transErr   error
	diarizeErr error
	transReq   engineprovider.TranscriptionRequest
	diarizeReq engineprovider.DiarizationRequest
}

func (p *fakeProvider) ID() string { return p.id }
func (p *fakeProvider) Capabilities(context.Context) ([]engineprovider.ModelCapability, error) {
	return nil, nil
}
func (p *fakeProvider) Prepare(context.Context) error { return nil }
func (p *fakeProvider) Transcribe(ctx context.Context, req engineprovider.TranscriptionRequest) (*engineprovider.TranscriptionResult, error) {
	p.transReq = req
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return p.transcribe, p.transErr
}
func (p *fakeProvider) Diarize(ctx context.Context, req engineprovider.DiarizationRequest) (*engineprovider.DiarizationResult, error) {
	p.diarizeReq = req
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return p.diarize, p.diarizeErr
}
func (p *fakeProvider) Close() error { return nil }

type recordingEvents struct {
	events []ProgressEvent
}

func (e *recordingEvents) Publish(ctx context.Context, event ProgressEvent) {
	e.events = append(e.events, event)
}

func openOrchestratorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "scriberr.db"))
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func createOrchestratorJob(t *testing.T, db *gorm.DB, audioPath string, params models.WhisperXParams) models.TranscriptionJob {
	t.Helper()
	user := models.User{Username: "orchestrator-user-" + time.Now().Format("150405.000000000"), Password: "pw"}
	require.NoError(t, db.Create(&user).Error)
	sourceID := "file-orchestrator"
	job := models.TranscriptionJob{
		ID:             "job-orchestrator",
		UserID:         user.ID,
		Status:         models.StatusProcessing,
		AudioPath:      audioPath,
		SourceFileName: filepath.Base(audioPath),
		SourceFileHash: &sourceID,
		Parameters:     params,
		Diarization:    params.Diarize,
	}
	require.NoError(t, db.Create(&job).Error)
	return job
}

func TestProcessorCreatesExecutionAndReturnsCanonicalTranscript(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, audioPath, models.WhisperXParams{
		Model:            "custom-transcriber",
		Task:             "translate",
		ChunkingStrategy: "vad",
		ChunkSize:        24,
		Diarize:          true,
		DiarizeModel:     "custom-diarizer",
	})
	provider := &fakeProvider{
		id: "local",
		transcribe: &engineprovider.TranscriptionResult{
			Text:     "Hello there.",
			Language: "en",
			ModelID:  "custom-transcriber",
			EngineID: "local",
			Words: []engineprovider.TranscriptWord{
				{Start: 0, End: 0.4, Word: "Hello"},
				{Start: 0.5, End: 0.9, Word: "there"},
			},
		},
		diarize: &engineprovider.DiarizationResult{
			ModelID:  "custom-diarizer",
			EngineID: "local",
			Segments: []engineprovider.DiarizationSegment{
				{Start: 0, End: 1, Speaker: "raw-speaker"},
			},
		},
	}
	registry, err := engineprovider.NewRegistry("local", provider)
	require.NoError(t, err)
	events := &recordingEvents{}
	outputDir := t.TempDir()
	processor := &Processor{
		Jobs:      repository.NewJobRepository(db),
		Providers: registry,
		Events:    events,
		OutputDir: outputDir,
	}

	result, err := processor.Process(context.Background(), &job)

	require.NoError(t, err)
	assert.Equal(t, models.StatusCompleted, result.Status)
	require.NotNil(t, result.OutputJSONPath)
	assert.Equal(t, filepath.Join(outputDir, job.ID, "transcript.json"), *result.OutputJSONPath)
	assert.Contains(t, result.TranscriptJSON, `"words":[`)
	assert.Contains(t, result.TranscriptJSON, `"speaker":"SPEAKER_00"`)
	assert.FileExists(t, *result.OutputJSONPath)
	assert.Equal(t, "custom-transcriber", provider.transReq.ModelID)
	assert.Equal(t, "custom-diarizer", provider.diarizeReq.ModelID)
	assert.Equal(t, "translate", provider.transReq.Task)
	assert.Equal(t, "vad", provider.transReq.Chunking)
	assert.Equal(t, float64(24), provider.transReq.ChunkDurationSec)

	var executions []models.TranscriptionJobExecution
	require.NoError(t, db.Where("transcription_id = ?", job.ID).Find(&executions).Error)
	require.Len(t, executions, 1)
	assert.Equal(t, models.StatusProcessing, executions[0].Status)
	assert.Equal(t, "local", executions[0].Provider)
	assert.Equal(t, "custom-transcriber", executions[0].ModelName)
	assert.NotContains(t, executions[0].RequestJSON, audioPath)

	assertEventStages(t, events.events, []string{"preparing", "transcribing", "diarizing", "merging", "saving", "completed"})
}

func TestProcessorNormalizesUnsupportedWhisperDecodingMethod(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, audioPath, models.WhisperXParams{
		ModelFamily:    "whisper",
		Model:          "whisper-base-en",
		DecodingMethod: "modified_beam_search",
	})
	provider := &fakeProvider{
		id: "local",
		transcribe: &engineprovider.TranscriptionResult{
			Text:     "Hello there.",
			Language: "en",
			ModelID:  "whisper-base-en",
			EngineID: "local",
		},
	}
	registry, err := engineprovider.NewRegistry("local", provider)
	require.NoError(t, err)
	processor := &Processor{
		Jobs:      repository.NewJobRepository(db),
		Providers: registry,
		Events:    &recordingEvents{},
		OutputDir: t.TempDir(),
	}

	result, err := processor.Process(context.Background(), &job)

	require.NoError(t, err)
	require.Equal(t, models.StatusCompleted, result.Status)
	require.Equal(t, "greedy_search", provider.transReq.DecodingMethod)
}

func TestProcessorReturnsSanitizedFailure(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, audioPath, models.WhisperXParams{})
	provider := &fakeProvider{
		id:       "local",
		transErr: errors.New("open /tmp/private/model.bin failed api_key=secret-value"),
	}
	registry, err := engineprovider.NewRegistry("local", provider)
	require.NoError(t, err)
	processor := &Processor{
		Jobs:      repository.NewJobRepository(db),
		Providers: registry,
		OutputDir: t.TempDir(),
	}

	result, err := processor.Process(context.Background(), &job)

	require.Error(t, err)
	assert.Equal(t, models.StatusFailed, result.Status)
	assert.NotContains(t, result.ErrorMessage, "/tmp/private")
	assert.NotContains(t, result.ErrorMessage, "secret-value")
	assert.Contains(t, result.ErrorMessage, "[redacted-path]")
	assert.Contains(t, result.ErrorMessage, "api_key=[redacted]")
}

func TestProcessorCancellationReturnsCanceled(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, audioPath, models.WhisperXParams{})
	provider := &fakeProvider{id: "local"}
	registry, err := engineprovider.NewRegistry("local", provider)
	require.NoError(t, err)
	processor := &Processor{
		Jobs:      repository.NewJobRepository(db),
		Providers: registry,
		OutputDir: t.TempDir(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := processor.Process(ctx, &job)

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, models.StatusCanceled, result.Status)
}

func assertEventStages(t *testing.T, events []ProgressEvent, stages []string) {
	t.Helper()
	require.Len(t, events, len(stages))
	for i, stage := range stages {
		assert.Equal(t, stage, events[i].Stage)
	}
}
