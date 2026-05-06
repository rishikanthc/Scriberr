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
	caps       []engineprovider.ModelCapability
	modelCards []asrcontract.ModelCard
	transcribe *engineprovider.TranscriptionResult
	diarize    *engineprovider.DiarizationResult
	speakerID  *asrcontract.SpeakerIDResult
	transErr   error
	diarizeErr error
	speakerErr error
	progress   []asrcontract.ProviderProgress
	transReq   engineprovider.TranscriptionRequest
	diarizeReq engineprovider.DiarizationRequest
	speakerReq asrcontract.SpeakerIDRequest
}

func (p *fakeProvider) ID() string { return p.id }
func (p *fakeProvider) Inspect(context.Context) (*asrcontract.ProviderInfo, error) {
	return &asrcontract.ProviderInfo{ContractVersion: asrcontract.ContractVersionV1}, nil
}
func (p *fakeProvider) Models(context.Context) ([]asrcontract.ModelCard, error) {
	return p.modelCards, nil
}
func (p *fakeProvider) Status(context.Context) (*asrcontract.ProviderStatus, error) {
	return &asrcontract.ProviderStatus{State: asrcontract.ProviderStateIdle}, nil
}
func (p *fakeProvider) LoadModel(context.Context, asrcontract.LoadModelRequest) error     { return nil }
func (p *fakeProvider) UnloadModel(context.Context, asrcontract.UnloadModelRequest) error { return nil }
func (p *fakeProvider) LoadedModels(context.Context) ([]asrcontract.LoadedModel, error) {
	return nil, nil
}
func (p *fakeProvider) Capabilities(context.Context) ([]engineprovider.ModelCapability, error) {
	if p.caps != nil {
		return p.caps, nil
	}
	return []engineprovider.ModelCapability{
		{ID: "whisper-base", Provider: p.id, Installed: true, Default: true, Capabilities: []string{"transcription"}},
		{ID: "whisper-base-en", Provider: p.id, Installed: true, Capabilities: []string{"transcription"}},
		{ID: "custom-transcriber", Provider: p.id, Installed: true, Capabilities: []string{"transcription"}},
		{ID: "remote-model", Provider: p.id, Installed: true, Capabilities: []string{"transcription"}},
		{ID: "diarization-default", Provider: p.id, Installed: true, Default: true, Capabilities: []string{"diarization"}},
		{ID: "custom-diarizer", Provider: p.id, Installed: true, Capabilities: []string{"diarization"}},
		{ID: "speaker-id-default", Provider: p.id, Installed: true, Capabilities: []string{"speaker_identification"}},
	}, nil
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
func (p *fakeProvider) IdentifySpeakers(ctx context.Context, req asrcontract.SpeakerIDRequest) (*asrcontract.SpeakerIDResult, error) {
	p.speakerReq = req
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.speakerID != nil || p.speakerErr != nil {
		return p.speakerID, p.speakerErr
	}
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

func createOrchestratorJob(t *testing.T, db *gorm.DB, audioPath string, params models.ASRParams) models.TranscriptionJob {
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
		Diarization:    testHasASRStep(params.Pipeline, models.ASRStepDiarization),
	}
	require.NoError(t, db.Create(&job).Error)
	return job
}

func testHasASRStep(steps []models.ASRStep, kind string) bool {
	for _, step := range steps {
		if step.Kind == kind {
			return true
		}
	}
	return false
}

func TestProcessorCreatesExecutionAndReturnsCanonicalTranscript(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, audioPath, models.ASRParams{
		Pipeline: []models.ASRStep{
			{
				Kind:        models.ASRStepTranscription,
				Model:       "custom-transcriber",
				ModelFamily: "whisper",
				Options: map[string]any{
					"task":                                  "translate",
					asrcontract.CommonParameterChunkingMode: "vad",
					asrcontract.CommonParameterChunkingChunkSeconds: float64(24),
					asrcontract.CommonParameterBatchingBatchSize:    1,
				},
			},
			{Kind: models.ASRStepDiarization, Model: "custom-diarizer"},
		},
	})
	provider := &fakeProvider{
		id: "local",
		progress: []asrcontract.ProviderProgress{{
			Stage:     asrcontract.StageTranscribing,
			Operation: asrcontract.OperationTranscription,
			Model:     "custom-transcriber",
			Timestamp: time.Now(),
		}},
		transcribe: &engineprovider.TranscriptionResult{
			Text:     "Hello there.",
			Language: "en",
			ModelID:  "custom-transcriber",
			EngineID: "local",
			Words: []engineprovider.TranscriptWord{
				{Start: 0, End: 0.4, Word: "Hello"},
				{Start: 0.5, End: 0.9, Word: "there"},
			},
			Metadata: map[string]any{
				"chunking_mode": "vad",
				"plan": map[string]any{
					"chunking_mode": "vad",
					"chunk_count":   float64(2),
				},
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
	assert.Equal(t, "translate", provider.transReq.Parameters["task"])
	assert.Equal(t, "vad", provider.transReq.Parameters[asrcontract.CommonParameterChunkingMode])
	assert.Equal(t, float64(24), provider.transReq.Parameters[asrcontract.CommonParameterChunkingChunkSeconds])
	assert.Equal(t, 1, provider.transReq.Parameters[asrcontract.CommonParameterBatchingBatchSize])

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
	assert.Contains(t, executions[0].ConfigJSON, `"provider_metadata"`)
	assert.Contains(t, executions[0].ConfigJSON, `"chunking_mode":"vad"`)

	assertEventStages(t, events.events, []string{"preparing", "transcribing", "merging", "saving", "completed"})
}

func TestProcessorPassesProviderSpecificChunkingOptionsWithoutBackendPlanning(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, audioPath, models.ASRParams{
		Pipeline: []models.ASRStep{{
			Kind:     models.ASRStepTranscription,
			Provider: "remote",
			Model:    "provider-chunker",
			Options:  map[string]any{asrcontract.CommonParameterChunkingMode: "fixed"},
		}},
	})
	provider := &fakeProvider{
		id: "remote",
		caps: []engineprovider.ModelCapability{{
			ID:           "provider-chunker",
			Provider:     "remote",
			Installed:    true,
			Capabilities: []string{"transcription"},
		}},
		modelCards: []asrcontract.ModelCard{{
			ID:       "provider-chunker",
			Provider: "remote",
			Capabilities: asrcontract.Capabilities{
				Transcription: true,
			},
			Chunking: &asrcontract.ChunkingCapabilities{
				SupportsEngineChunking:   false,
				SupportsProviderChunking: true,
				PreferredMode:            "provider",
			},
		}},
		transcribe: &engineprovider.TranscriptionResult{Text: "provider owns validation"},
	}
	registry, err := engineprovider.NewRegistry("remote", provider)
	require.NoError(t, err)
	processor := &Processor{
		Jobs:      repository.NewJobRepository(db),
		Providers: registry,
		OutputDir: t.TempDir(),
	}

	result, err := processor.Process(context.Background(), &job)

	require.NoError(t, err)
	assert.Equal(t, models.StatusCompleted, result.Status)
	assert.Equal(t, "fixed", provider.transReq.Parameters[asrcontract.CommonParameterChunkingMode])
	var executions []models.TranscriptionJobExecution
	require.NoError(t, db.Where("transcription_id = ?", job.ID).Find(&executions).Error)
	assert.Len(t, executions, 1)
}

func TestProcessorChainsDiarizationAcrossProviders(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, audioPath, models.ASRParams{
		Pipeline: []models.ASRStep{
			{Kind: models.ASRStepTranscription, Provider: "local", Model: "local-transcriber", ModelFamily: "nemo_transducer"},
			{Kind: models.ASRStepDiarization, Provider: "remote-diarizer", Model: "remote-diarizer-model"},
		},
	})
	local := &fakeProvider{
		id: "local",
		caps: []engineprovider.ModelCapability{{
			ID:           "local-transcriber",
			Provider:     "local",
			Installed:    true,
			Capabilities: []string{"transcription"},
		}},
		transcribe: &engineprovider.TranscriptionResult{
			Text: "Hello.",
			Words: []engineprovider.TranscriptWord{
				{Start: 0, End: 0.4, Word: "Hello"},
			},
			ModelID:  "local-transcriber",
			EngineID: "local",
		},
	}
	remote := &fakeProvider{
		id: "remote-diarizer",
		caps: []engineprovider.ModelCapability{{
			ID:           "remote-diarizer-model",
			Provider:     "remote-diarizer",
			Installed:    true,
			Capabilities: []string{"diarization"},
		}},
		diarize: &engineprovider.DiarizationResult{
			ModelID:  "remote-diarizer-model",
			EngineID: "remote-diarizer",
			Segments: []engineprovider.DiarizationSegment{
				{Start: 0, End: 1, Speaker: "remote-speaker"},
			},
		},
	}
	registry, err := engineprovider.NewRegistry("local", local, remote)
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
	assert.Equal(t, job.ID, local.transReq.JobID)
	assert.Empty(t, local.diarizeReq.JobID)
	assert.Equal(t, job.ID, remote.diarizeReq.JobID)
	assert.Empty(t, remote.transReq.JobID)
	assert.Equal(t, "local-transcriber", local.transReq.ModelID)
	assert.Equal(t, "remote-diarizer-model", remote.diarizeReq.ModelID)
	assert.Contains(t, result.TranscriptJSON, `"speaker":"SPEAKER_00"`)

	var executions []models.TranscriptionJobExecution
	require.NoError(t, db.Where("transcription_id = ?", job.ID).Find(&executions).Error)
	require.Len(t, executions, 1)
	assert.Contains(t, executions[0].ConfigJSON, `"provider":"local"`)
	assert.Contains(t, executions[0].ConfigJSON, `"provider":"remote-diarizer"`)
	assert.NotContains(t, executions[0].ConfigJSON, audioPath)
}

func TestProcessorRejectsUnsupportedPipelineStep(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, audioPath, models.ASRParams{
		Pipeline: []models.ASRStep{{Kind: "summarization", Provider: "local", Model: "bad-step"}},
	})
	provider := &fakeProvider{id: "local"}
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
	assert.Contains(t, result.ErrorMessage, "unsupported ASR pipeline step")
	assert.Empty(t, provider.transReq.JobID)
}

func TestProcessorFailsWhenPipelineStepProviderUnavailable(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, audioPath, models.ASRParams{
		Pipeline: []models.ASRStep{{Kind: models.ASRStepTranscription, Provider: "missing", Model: "remote-model"}},
	})
	provider := &fakeProvider{id: "local"}
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
	assert.Contains(t, result.ErrorMessage, "missing")
	assert.Empty(t, provider.transReq.JobID)
}

func TestProcessorExecutesSpeakerIdentificationStep(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, audioPath, models.ASRParams{
		Pipeline: []models.ASRStep{
			{Kind: models.ASRStepTranscription, Provider: "local", Model: "whisper-base"},
			{Kind: models.ASRStepSpeakerIdentification, Provider: "speaker-provider", Model: "speaker-id-default"},
		},
	})
	local := &fakeProvider{
		id: "local",
		transcribe: &engineprovider.TranscriptionResult{
			Text: "Hello.",
		},
	}
	speakerProvider := &fakeProvider{
		id: "speaker-provider",
		caps: []engineprovider.ModelCapability{{
			ID:           "speaker-id-default",
			Provider:     "speaker-provider",
			Installed:    true,
			Capabilities: []string{"speaker_identification"},
		}},
		speakerID: &asrcontract.SpeakerIDResult{
			Model:    "speaker-id-default",
			Speakers: []asrcontract.SpeakerIdentity{{Speaker: "SPEAKER_00", Label: "Ada"}},
		},
	}
	registry, err := engineprovider.NewRegistry("local", local, speakerProvider)
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
	assert.Equal(t, job.ID, speakerProvider.speakerReq.RequestID)
	assert.Equal(t, "speaker-id-default", speakerProvider.speakerReq.Model)
	assert.Empty(t, speakerProvider.transReq.JobID)
	assertEventStages(t, events.events, []string{"preparing", "merging", "saving", "completed"})
}

func TestProcessorPassesPreprocessedAudioToProvider(t *testing.T) {
	db := openOrchestratorTestDB(t)
	sourcePath := filepath.Join(t.TempDir(), "source.wav")
	require.NoError(t, os.WriteFile(sourcePath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, sourcePath, models.ASRParams{
		Pipeline: []models.ASRStep{
			{Kind: models.ASRStepTranscription, Model: engineprovider.DefaultTranscriptionModel},
			{Kind: models.ASRStepDiarization, Model: engineprovider.DefaultDiarizationModel},
		},
	})
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
	job := createOrchestratorJob(t, db, audioPath, models.ASRParams{})
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
	event := requireEventStage(t, events.events, "loading_model")
	assert.InDelta(t, 0.31, event.Progress, 0.001)
	assert.NotContains(t, event.Stage, "/tmp/private")
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

func TestProcessorPassesPipelineOptionsToProvider(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	job := createOrchestratorJob(t, db, audioPath, models.ASRParams{
		Pipeline: []models.ASRStep{
			{
				Kind:        models.ASRStepTranscription,
				Model:       "whisper-base-en",
				ModelFamily: "whisper",
				Options: map[string]any{
					asrcontract.CommonParameterDecodingMethod: "modified_beam_search",
				},
			},
		},
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
	require.Equal(t, "modified_beam_search", provider.transReq.Parameters[asrcontract.CommonParameterDecodingMethod])
}

func TestProcessorUsesExplicitEngineProviderSelection(t *testing.T) {
	db := openOrchestratorTestDB(t)
	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake wav"), 0o600))
	engineID := "remote"
	job := createOrchestratorJob(t, db, audioPath, models.ASRParams{
		Pipeline: []models.ASRStep{
			{Kind: models.ASRStepTranscription, Provider: "remote", Model: "remote-model"},
		},
	})
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
	job := createOrchestratorJob(t, db, audioPath, models.ASRParams{})
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
	job := createOrchestratorJob(t, db, audioPath, models.ASRParams{})
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

func requireEventStage(t *testing.T, events []ProgressEvent, stage string) ProgressEvent {
	t.Helper()
	for _, event := range events {
		if event.Stage == stage {
			return event
		}
	}
	t.Fatalf("stage %q not found in %#v", stage, events)
	return ProgressEvent{}
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
