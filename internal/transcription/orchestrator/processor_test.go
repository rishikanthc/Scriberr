package orchestrator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/asrcontract"
	"scriberr/internal/transcription/engineprovider"
	"scriberr/internal/transcription/preprocess"

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
	progress   []asrcontract.ProviderProgress
	transReq   engineprovider.TranscriptionRequest
	diarizeReq engineprovider.DiarizationRequest
}

func (p *fakeProvider) ID() string { return p.id }
func (p *fakeProvider) Inspect(context.Context) (*asrcontract.ProviderInfo, error) {
	return &asrcontract.ProviderInfo{ContractVersion: asrcontract.ContractVersionV1}, nil
}
func (p *fakeProvider) Models(context.Context) ([]asrcontract.ModelCard, error) { return nil, nil }
func (p *fakeProvider) Status(context.Context) (*asrcontract.ProviderStatus, error) {
	return &asrcontract.ProviderStatus{State: asrcontract.ProviderStateIdle}, nil
}
func (p *fakeProvider) LoadModel(context.Context, asrcontract.LoadModelRequest) error     { return nil }
func (p *fakeProvider) UnloadModel(context.Context, asrcontract.UnloadModelRequest) error { return nil }
func (p *fakeProvider) LoadedModels(context.Context) ([]asrcontract.LoadedModel, error) {
	return nil, nil
}
func (p *fakeProvider) Capabilities(context.Context) ([]engineprovider.ModelCapability, error) {
	return nil, nil
}
func (p *fakeProvider) Prepare(context.Context) error { return nil }
func (p *fakeProvider) Transcribe(ctx context.Context, req engineprovider.TranscriptionRequest) (*engineprovider.TranscriptionResult, error) {
	p.transReq = req
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, event := range p.progress {
		req.Progress.Report(ctx, event)
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
func (p *fakeProvider) IdentifySpeakers(context.Context, asrcontract.SpeakerIDRequest) (*asrcontract.SpeakerIDResult, error) {
	return nil, asrcontract.NewProviderError(asrcontract.CodeUnsupportedOperation, "speaker identification is not supported", false)
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
	assert.Equal(t, audioPath, provider.transReq.AudioPath)
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
	assert.NotContains(t, executions[0].ConfigJSON, audioPath)
	assert.Contains(t, executions[0].ConfigJSON, `"operation":"transcription"`)
	assert.Contains(t, executions[0].ConfigJSON, `"operation":"diarization"`)

	assertEventStages(t, events.events, []string{"preparing", "transcribing", "diarizing", "merging", "saving", "completed"})
}

func TestProcessorPassesPreprocessedAudioToProvider(t *testing.T) {
	db := openOrchestratorTestDB(t)
	sourcePath := filepath.Join(t.TempDir(), "source.wav")
	require.NoError(t, os.WriteFile(sourcePath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, sourcePath, models.WhisperXParams{Diarize: true})
	provider := &fakeProvider{
		id: "local",
		transcribe: &engineprovider.TranscriptionResult{
			Text: "Hello.",
		},
		diarize: &engineprovider.DiarizationResult{},
	}
	registry, err := engineprovider.NewRegistry("local", provider)
	require.NoError(t, err)
	processor := &Processor{
		Jobs:      repository.NewJobRepository(db),
		Providers: registry,
		Audio: preprocess.NewLocalPreprocessor(preprocess.Config{
			Dir:               t.TempDir(),
			ProviderMountRoot: "/provider-input/audio",
			FFmpegPath:        fakeFFmpegForOrchestrator(t),
		}),
		OutputDir: t.TempDir(),
	}

	result, err := processor.Process(context.Background(), &job)

	require.NoError(t, err)
	require.Equal(t, models.StatusCompleted, result.Status)
	assert.Equal(t, "/provider-input/audio/file-orchestrator.wav", provider.transReq.AudioPath)
	assert.Equal(t, provider.transReq.AudioPath, provider.diarizeReq.AudioPath)
	assert.NotEqual(t, sourcePath, provider.transReq.AudioPath)
}

func TestProcessorPersistsProviderProgress(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, audioPath, models.WhisperXParams{})
	progress := 0.31
	provider := &fakeProvider{
		id: "local",
		transcribe: &engineprovider.TranscriptionResult{
			Text:     "Hello there.",
			Language: "en",
		},
		progress: []asrcontract.ProviderProgress{{
			Stage:     asrcontract.StageLoadingModel,
			Progress:  &progress,
			Message:   "loading /tmp/private/model api_key=secret",
			Operation: asrcontract.OperationTranscription,
			Model:     "whisper-base",
			Timestamp: time.Now(),
		}},
	}
	registry, err := engineprovider.NewRegistry("local", provider)
	require.NoError(t, err)
	events := &recordingEvents{}
	processor := &Processor{
		Jobs:      repository.NewJobRepository(db),
		Providers: registry,
		Events:    events,
		OutputDir: t.TempDir(),
	}

	result, err := processor.Process(context.Background(), &job)

	require.NoError(t, err)
	require.Equal(t, models.StatusCompleted, result.Status)
	var stored models.TranscriptionJob
	require.NoError(t, db.First(&stored, "id = ?", job.ID).Error)
	require.InDelta(t, 0.95, stored.Progress, 0.001)
	assert.Equal(t, "saving", stored.ProgressStage)
	require.GreaterOrEqual(t, len(events.events), 3)
	assert.Equal(t, "loading_model", events.events[2].Stage)
	assert.InDelta(t, 0.31, events.events[2].Progress, 0.001)
	assert.NotContains(t, events.events[2].Stage, "/tmp/private")
}

func TestLocalTranscriptStoreWritesPathSafeArtifact(t *testing.T) {
	outputDir := t.TempDir()
	store := NewLocalTranscriptStore(outputDir)

	outputPath, err := store.SaveTranscriptJSON(context.Background(), "job-artifact", []byte(`{"text":"hello"}`))

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(outputDir, "job-artifact", "transcript.json"), outputPath)
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.JSONEq(t, `{"text":"hello"}`, string(data))
}

func TestLocalTranscriptStoreRejectsTraversalJobID(t *testing.T) {
	outputDir := t.TempDir()
	store := NewLocalTranscriptStore(outputDir)

	_, err := store.SaveTranscriptJSON(context.Background(), "../escape", []byte(`{}`))

	require.Error(t, err)
	assert.NoFileExists(t, filepath.Join(outputDir, "..", "escape", "transcript.json"))
}

func TestLocalTranscriptStoreRequiresOutputDir(t *testing.T) {
	store := NewLocalTranscriptStore("")

	_, err := store.SaveTranscriptJSON(context.Background(), "job-artifact", []byte(`{}`))

	require.Error(t, err)
}

func TestLocalExecutionLogStoreReadsOnlyWithinRoot(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "job", "execution.log")
	require.NoError(t, os.MkdirAll(filepath.Dir(logPath), 0o755))
	require.NoError(t, os.WriteFile(logPath, []byte("hello\n"), 0o600))
	store := NewLocalExecutionLogStore(root)

	text, err := store.ReadExecutionLog(context.Background(), models.TranscriptionJobExecution{LogsPath: &logPath})

	require.NoError(t, err)
	assert.Equal(t, "hello\n", text)

	outside := filepath.Join(t.TempDir(), "execution.log")
	require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o600))
	_, err = store.ReadExecutionLog(context.Background(), models.TranscriptionJobExecution{LogsPath: &outside})
	require.Error(t, err)
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

func TestProcessorUsesExplicitEngineProviderSelection(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	engineID := "remote"
	job := createOrchestratorJob(t, db, audioPath, models.WhisperXParams{Model: "remote-model"})
	job.EngineID = &engineID
	require.NoError(t, db.Model(&models.TranscriptionJob{}).Where("id = ?", job.ID).Update("engine_id", engineID).Error)
	local := &fakeProvider{
		id: "local",
		transcribe: &engineprovider.TranscriptionResult{
			Text: "local",
		},
	}
	remote := &fakeProvider{
		id: "remote",
		transcribe: &engineprovider.TranscriptionResult{
			Text:    "remote",
			ModelID: "remote-model",
		},
	}
	registry, err := engineprovider.NewRegistry("local", local, remote)
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
	assert.Empty(t, local.transReq.JobID)
	assert.Equal(t, job.ID, remote.transReq.JobID)
	assert.Equal(t, "remote-model", remote.transReq.ModelID)
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
	assert.Equal(t, models.StatusStopped, result.Status)
}

func assertEventStages(t *testing.T, events []ProgressEvent, stages []string) {
	t.Helper()
	require.Len(t, events, len(stages))
	for i, stage := range stages {
		assert.Equal(t, stage, events[i].Stage)
	}
}

func fakeFFmpegForOrchestrator(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake ffmpeg shell script is unix-only")
	}
	path := filepath.Join(t.TempDir(), "ffmpeg")
	script := "#!/bin/sh\nset -eu\nout=\"\"\nfor arg in \"$@\"; do out=\"$arg\"; done\nprintf 'normalized audio\\n' > \"$out\"\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o700))
	return path
}
