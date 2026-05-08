package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"scriberr/internal/auth"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/transcription/asrcontract"
	"scriberr/internal/transcription/engineprovider"
	"scriberr/internal/transcription/worker"

	"github.com/stretchr/testify/require"
)

type fakeQueueService struct {
	mu         sync.Mutex
	enqueued   []string
	canceled   []string
	stats      worker.QueueStats
	adminStats worker.AdminQueueStats
	err        error
	cancelErr  error
}

func (q *fakeQueueService) Enqueue(ctx context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enqueued = append(q.enqueued, jobID)
	return q.err
}
func (q *fakeQueueService) Cancel(ctx context.Context, userID uint, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.canceled = append(q.canceled, jobID)
	return q.cancelErr
}
func (q *fakeQueueService) Start(context.Context) error { return nil }
func (q *fakeQueueService) Stop(context.Context) error  { return nil }
func (q *fakeQueueService) Stats(context.Context, uint) (worker.QueueStats, error) {
	return q.stats, q.err
}
func (q *fakeQueueService) AdminStats(context.Context) (worker.AdminQueueStats, error) {
	if q.adminStats.ByUser == nil {
		return worker.AdminQueueStats{QueueStats: q.stats}, q.err
	}
	return q.adminStats, q.err
}

func setTestQueueService(s *authTestServer, queue *fakeQueueService) {
	s.handler.queueService = queue
	if s.handler.transcriptions != nil {
		s.handler.transcriptions.SetQueue(queue)
	}
}

type fakeCapabilityProvider struct {
	models []asrcontract.ModelCard
}

func (p fakeCapabilityProvider) ID() string { return "local" }
func (p fakeCapabilityProvider) Inspect(context.Context) (*asrcontract.ProviderInfo, error) {
	return &asrcontract.ProviderInfo{ContractVersion: asrcontract.ContractVersionV1}, nil
}
func (p fakeCapabilityProvider) Models(context.Context) ([]asrcontract.ModelCard, error) {
	return p.models, nil
}
func (p fakeCapabilityProvider) Status(context.Context) (*asrcontract.ProviderStatus, error) {
	return &asrcontract.ProviderStatus{State: asrcontract.ProviderStateIdle}, nil
}
func (p fakeCapabilityProvider) LoadModel(context.Context, asrcontract.LoadModelRequest) error {
	return nil
}
func (p fakeCapabilityProvider) UnloadModel(context.Context, asrcontract.UnloadModelRequest) error {
	return nil
}
func (p fakeCapabilityProvider) LoadedModels(context.Context) ([]asrcontract.LoadedModel, error) {
	return nil, nil
}
func (p fakeCapabilityProvider) ExecuteTask(context.Context, engineprovider.TaskRequest) (*engineprovider.TaskResult, error) {
	return nil, nil
}
func (p fakeCapabilityProvider) Close() error { return nil }

func testASRModel(provider, id, name, modelType string, isDefault bool, capabilities ...asrcontract.Capability) asrcontract.ModelCard {
	return asrcontract.ModelCard{
		ID:           id,
		DisplayName:  name,
		Provider:     provider,
		ModelType:    modelType,
		Installed:    true,
		Default:      isDefault,
		Capabilities: testASRCapabilities(capabilities...),
	}
}

func testASRCapabilities(capabilities ...asrcontract.Capability) asrcontract.Capabilities {
	out := asrcontract.Capabilities{Extensions: map[string]bool{}}
	for _, capability := range capabilities {
		switch capability {
		case asrcontract.CapabilityTranscription:
			out.Transcription = true
		case asrcontract.CapabilityDiarization:
			out.Diarization = true
		case asrcontract.CapabilitySpeakerIdentification:
			out.SpeakerIdentification = true
		case asrcontract.CapabilityWordTimestamps:
			out.WordTimestamps = true
		default:
			out.Extensions[string(capability)] = true
		}
	}
	if len(out.Extensions) == 0 {
		out.Extensions = nil
	}
	return out
}

func TestASRModelCatalogEndpointFiltersCapabilities(t *testing.T) {
	s := newAuthTestServer(t)
	registry, err := engineprovider.NewRegistry("local", fakeCapabilityProvider{models: []asrcontract.ModelCard{
		{
			ID:          "parakeet-v2",
			DisplayName: "NVIDIA Parakeet TDT v2",
			Provider:    "local",
			ModelType:   "nemo_transducer",
			Installed:   true,
			Capabilities: testASRCapabilities(
				asrcontract.CapabilityTranscription,
				asrcontract.CapabilityWordTimestamps,
				asrcontract.CapabilitySegmentTimestamps,
			),
			ParameterSchema: asrcontract.ParameterSchema{
				{Key: asrcontract.CommonParameterChunkingMode, Type: asrcontract.ParameterTypeEnum, Scope: asrcontract.ParameterScopeChunking, Options: []asrcontract.ParameterOption{{Value: "fixed", Label: "Fixed"}}},
				{Key: "sherpa.model_type", Type: asrcontract.ParameterTypeString, Scope: asrcontract.ParameterScopeModel, Default: "nemo_transducer"},
			},
			RecommendedDefaults: map[string]any{"sherpa.model_type": "nemo_transducer"},
		},
		{
			ID:          "parakeet-v3",
			DisplayName: "NVIDIA Parakeet TDT v3",
			Provider:    "local",
			ModelType:   "nemo_transducer",
			Installed:   true,
			Capabilities: testASRCapabilities(
				asrcontract.CapabilityTranscription,
				asrcontract.CapabilityWordTimestamps,
				asrcontract.CapabilitySegmentTimestamps,
			),
			ParameterSchema: asrcontract.ParameterSchema{
				{Key: asrcontract.CommonParameterChunkingMode, Type: asrcontract.ParameterTypeEnum, Scope: asrcontract.ParameterScopeChunking, Options: []asrcontract.ParameterOption{{Value: "fixed", Label: "Fixed"}}},
				{Key: "sherpa.model_type", Type: asrcontract.ParameterTypeString, Scope: asrcontract.ParameterScopeModel, Default: "nemo_transducer"},
			},
			RecommendedDefaults: map[string]any{"sherpa.model_type": "nemo_transducer"},
		},
		{
			ID:          "diarization-default",
			DisplayName: "Pyannote + 3D-Speaker Diarization",
			Provider:    "local",
			ModelType:   "diarization",
			Installed:   true,
			Capabilities: testASRCapabilities(
				asrcontract.CapabilityDiarization,
			),
			ParameterSchema: asrcontract.ParameterSchema{
				{Key: "diarization.num_clusters", Type: asrcontract.ParameterTypeInteger, Scope: asrcontract.ParameterScopeModel, Default: float64(0)},
			},
		},
		{
			ID:           "speaker-id-default",
			DisplayName:  "Speaker ID",
			Provider:     "local",
			ModelType:    "speaker_identification",
			Installed:    true,
			Capabilities: testASRCapabilities(asrcontract.CapabilitySpeakerIdentification),
		},
	}})
	require.NoError(t, err)
	s.handler.modelRegistry = registry
	token := registerForFileTests(t, s)

	resp, body := s.request(t, http.MethodGet, "/api/v1/models", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.ElementsMatch(t, []string{"parakeet-v2", "parakeet-v3", "diarization-default", "speaker-id-default"}, modelIDsFromResponse(body))

	resp, body = s.request(t, http.MethodGet, "/api/v1/models?capability=transcription", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.ElementsMatch(t, []string{"parakeet-v2", "parakeet-v3"}, modelIDsFromResponse(body))
	requireModelSchemaKey(t, body, "parakeet-v2", "sherpa.model_type")
	requireModelSchemaKey(t, body, "parakeet-v3", "sherpa.model_type")

	resp, body = s.request(t, http.MethodGet, "/api/v1/models?capability=diarization", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.ElementsMatch(t, []string{"diarization-default"}, modelIDsFromResponse(body))
	requireModelSchemaKey(t, body, "diarization-default", "diarization.num_clusters")

	resp, body = s.request(t, http.MethodGet, "/api/v1/models?capability=transcription,diarization", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.ElementsMatch(t, []string{"parakeet-v2", "parakeet-v3", "diarization-default"}, modelIDsFromResponse(body))

	resp, body = s.request(t, http.MethodGet, "/api/v1/models?capability=unknown", nil, token, "")
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	require.Equal(t, "VALIDATION_ERROR", body["error"].(map[string]any)["code"])
}

func modelIDsFromResponse(body map[string]any) []string {
	items := body["items"].([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		model := item.(map[string]any)
		out = append(out, model["id"].(string))
	}
	return out
}

func requireModelSchemaKey(t *testing.T, body map[string]any, modelID string, key string) {
	t.Helper()
	items := body["items"].([]any)
	for _, item := range items {
		model := item.(map[string]any)
		if model["id"] != modelID {
			continue
		}
		schema := model["parameter_schema"].([]any)
		for _, rawParameter := range schema {
			parameter := rawParameter.(map[string]any)
			if parameter["key"] == key {
				return
			}
		}
		require.Failf(t, "missing schema key", "model %q missing parameter %q", modelID, key)
	}
	require.Failf(t, "missing model", "model %q not found", modelID)
}

type fakeAdminASRProvider struct {
	mu         sync.Mutex
	id         string
	status     asrcontract.ProviderStatus
	models     []asrcontract.ModelCard
	loaded     []asrcontract.LoadedModel
	loads      []asrcontract.LoadModelRequest
	unloads    []asrcontract.UnloadModelRequest
	loadErr    error
	unloadErr  error
	statusErr  error
	inspectErr error
	modelsErr  error
	loadedErr  error
}

func (p *fakeAdminASRProvider) ID() string { return p.id }
func (p *fakeAdminASRProvider) Inspect(context.Context) (*asrcontract.ProviderInfo, error) {
	if p.inspectErr != nil {
		return nil, p.inspectErr
	}
	return &asrcontract.ProviderInfo{
		ContractVersion: asrcontract.ContractVersionV1,
		Provider:        asrcontract.ProviderIdentity{ID: p.id, Name: "Fake ASR"},
		Runtime:         asrcontract.RuntimeInfo{DeviceBackends: []string{"cpu"}, MaxConcurrentJobs: 1},
	}, nil
}
func (p *fakeAdminASRProvider) Models(context.Context) ([]asrcontract.ModelCard, error) {
	if p.modelsErr != nil {
		return nil, p.modelsErr
	}
	return p.models, nil
}
func (p *fakeAdminASRProvider) Status(context.Context) (*asrcontract.ProviderStatus, error) {
	if p.statusErr != nil {
		return nil, p.statusErr
	}
	status := p.status
	if status.State == "" {
		status.State = asrcontract.ProviderStateIdle
	}
	return &status, nil
}
func (p *fakeAdminASRProvider) LoadModel(_ context.Context, req asrcontract.LoadModelRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.loads = append(p.loads, req)
	return p.loadErr
}
func (p *fakeAdminASRProvider) UnloadModel(_ context.Context, req asrcontract.UnloadModelRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.unloads = append(p.unloads, req)
	return p.unloadErr
}
func (p *fakeAdminASRProvider) LoadedModels(context.Context) ([]asrcontract.LoadedModel, error) {
	if p.loadedErr != nil {
		return nil, p.loadedErr
	}
	return p.loaded, nil
}
func (p *fakeAdminASRProvider) ExecuteTask(context.Context, engineprovider.TaskRequest) (*engineprovider.TaskResult, error) {
	return nil, nil
}
func (p *fakeAdminASRProvider) Close() error { return nil }

func TestCreateSubmitRetryUseQueueService(t *testing.T) {
	s := newAuthTestServer(t)
	queue := &fakeQueueService{}
	setTestQueueService(s, queue)
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{
		"file_id": fileID,
		"title":   "Queued by service",
	}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	firstID := strings.TrimPrefix(body["id"].(string), "tr_")

	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+body["id"].(string)+":retry", nil, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	retryID := strings.TrimPrefix(body["id"].(string), "tr_")

	require.Len(t, queue.enqueued, 2)
	require.Equal(t, firstID, queue.enqueued[0])
	require.Equal(t, retryID, queue.enqueued[1])
}

func TestCreateReturnsServiceUnavailableWhenQueueStopped(t *testing.T) {
	s := newAuthTestServer(t)
	setTestQueueService(s, &fakeQueueService{err: worker.ErrQueueStopped})
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{"file_id": fileID}, token, "")

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	require.Equal(t, "SERVICE_UNAVAILABLE", body["error"].(map[string]any)["code"])

	var count int64
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).
		Where("source_file_hash IS NOT NULL").
		Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestRetryPreservesNewJobWhenQueueStopped(t *testing.T) {
	s := newAuthTestServer(t)
	queue := &fakeQueueService{}
	setTestQueueService(s, queue)
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{"file_id": fileID}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	transcriptionID := body["id"].(string)

	queue.err = worker.ErrQueueStopped
	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+":retry", nil, token, "")
	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	require.Equal(t, "SERVICE_UNAVAILABLE", body["error"].(map[string]any)["code"])

	var count int64
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).
		Where("source_file_hash IS NOT NULL").
		Count(&count).Error)
	require.Equal(t, int64(2), count)
}

func TestCancelUsesQueueServiceAndMapsConflict(t *testing.T) {
	s := newAuthTestServer(t)
	queue := &fakeQueueService{cancelErr: worker.ErrStateConflict}
	setTestQueueService(s, queue)
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)
	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{"file_id": fileID}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	transcriptionID := body["id"].(string)

	resp, body = s.request(t, http.MethodPost, "/api/v1/transcriptions/"+transcriptionID+":cancel", nil, token, "")

	require.Equal(t, http.StatusConflict, resp.Code)
	require.Equal(t, "CONFLICT", body["error"].(map[string]any)["code"])
	require.Equal(t, strings.TrimPrefix(transcriptionID, "tr_"), queue.canceled[0])
}

func TestTranscriptExecutionsLogsModelsAndStatsUseEngineServices(t *testing.T) {
	s := newAuthTestServer(t)
	queue := &fakeQueueService{stats: worker.QueueStats{Queued: 2, Processing: 1, Completed: 3, Failed: 4, Canceled: 5, Running: 1}}
	setTestQueueService(s, queue)
	registry, err := engineprovider.NewRegistry("local", fakeCapabilityProvider{models: []asrcontract.ModelCard{
		testASRModel("local", "whisper-base", "Whisper Base", "whisper", true, asrcontract.CapabilityTranscription, asrcontract.CapabilityWordTimestamps),
	}})
	require.NoError(t, err)
	s.handler.modelRegistry = registry

	token := registerForFileTests(t, s)
	userID := firstUserID(t)
	fileID, _ := createUploadedFileForTranscription(t, s, token)
	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{"file_id": fileID}, token, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	transcriptionID := body["id"].(string)
	jobID := strings.TrimPrefix(transcriptionID, "tr_")

	now := time.Now().UTC().Truncate(time.Millisecond)
	transcript := `{"text":"hello","segments":[{"id":"seg_000001","start":0,"end":1,"speaker":"SPEAKER_00","text":"hello"}]}`
	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":          models.StatusCompleted,
		"transcript_text": transcript,
		"progress":        1.0,
		"progress_stage":  "completed",
		"started_at":      now.Add(-time.Minute),
		"completed_at":    now,
	}).Error)
	errorMessage := "failed at /tmp/private/model.bin api_key=secret-value"
	require.NoError(t, database.DB.Create(&models.TranscriptionJobExecution{
		TranscriptionJobID: jobID,
		UserID:             userID,
		Status:             models.StatusFailed,
		Provider:           "local",
		ModelName:          "whisper-base",
		StartedAt:          now.Add(-time.Minute),
		FailedAt:           &now,
		ErrorMessage:       &errorMessage,
	}).Error)

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, float64(1), body["progress"])
	require.Equal(t, "completed", body["progress_stage"])
	require.NotNil(t, body["started_at"])
	require.NotNil(t, body["completed_at"])

	require.NoError(t, database.DB.Model(&models.TranscriptionJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":     models.StatusFailed,
		"last_error": errorMessage,
	}).Error)
	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID, nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.NotContains(t, body["error"], "/tmp/private")
	require.NotContains(t, body["error"], "secret-value")
	require.Contains(t, body["error"], "[redacted-path]")

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/transcript", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "hello", body["text"])
	require.Empty(t, body["words"])
	require.Len(t, body["segments"].([]any), 1)

	resp, body = s.request(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/executions", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	execution := body["items"].([]any)[0].(map[string]any)
	require.Equal(t, "local", execution["provider"])
	require.Equal(t, "whisper-base", execution["model"])
	require.NotContains(t, execution["error"], "/tmp/private")
	require.NotContains(t, execution["error"], "secret-value")

	resp, rawLogs := s.rawRequest(t, http.MethodGet, "/api/v1/transcriptions/"+transcriptionID+"/logs", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.NotContains(t, rawLogs, "/tmp/private")
	require.NotContains(t, rawLogs, "secret-value")
	require.Contains(t, rawLogs, "[redacted-path]")
	require.Contains(t, rawLogs, "\nfailed_at=")

	resp, body = s.request(t, http.MethodGet, "/api/v1/models/transcription", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	model := body["items"].([]any)[0].(map[string]any)
	require.Equal(t, "whisper-base", model["id"])
	require.Equal(t, true, model["installed"])
	require.Equal(t, true, model["default"])
	require.Equal(t, true, model["capabilities"].(map[string]any)["transcription"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/admin/queue", nil, token, "")
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, float64(2), body["queued"])
	require.Equal(t, float64(1), body["running"])
}

func TestQueueServiceErrorDoesNotLeakInternals(t *testing.T) {
	s := newAuthTestServer(t)
	setTestQueueService(s, &fakeQueueService{err: errors.New("open /tmp/private/socket token=secret failed")})
	token := registerForFileTests(t, s)
	fileID, _ := createUploadedFileForTranscription(t, s, token)

	resp, body := s.request(t, http.MethodPost, "/api/v1/transcriptions", map[string]any{"file_id": fileID}, token, "")

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	message := body["error"].(map[string]any)["message"].(string)
	require.NotContains(t, message, "/tmp/private")
	require.NotContains(t, message, "secret")
}

func TestAdminASRProviderDiagnosticsAndModelCommands(t *testing.T) {
	s := newAuthTestServer(t)
	adminToken := registerForFileTests(t, s)
	loadedAt := time.Now().UTC().Truncate(time.Millisecond)
	memory := 512
	progress := 0.42
	provider := &fakeAdminASRProvider{
		id: "local",
		status: asrcontract.ProviderStatus{
			State: asrcontract.ProviderStateBusy,
			ActiveJob: &asrcontract.ActiveJob{
				ID:        "job-/tmp/private/audio.wav",
				Operation: asrcontract.OperationTranscription,
				Model:     "whisper-base api_key=secret",
				Stage:     asrcontract.StageRunning,
				Progress:  &progress,
			},
			Capacity: asrcontract.ProviderCapacity{MaxConcurrentJobs: 1},
		},
		models: []asrcontract.ModelCard{{
			ID:          "whisper-base",
			DisplayName: "Whisper Base",
			Provider:    "local",
			ModelType:   "whisper",
			Installed:   true,
			Loaded:      true,
			Default:     true,
			SourceURL:   "file:///tmp/private/model.bin?api_key=secret",
			Capabilities: asrcontract.Capabilities{
				Transcription:  true,
				WordTimestamps: true,
			},
		}},
		loaded: []asrcontract.LoadedModel{{ID: "whisper-base", LoadedAt: &loadedAt, MemoryMB: &memory}},
	}
	registry, err := engineprovider.NewRegistry("local", provider)
	require.NoError(t, err)
	s.handler.modelRegistry = registry

	resp, body := s.request(t, http.MethodGet, "/api/v1/admin/asr-providers", nil, adminToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	item := body["items"].([]any)[0].(map[string]any)
	require.Equal(t, "local", item["id"])
	status := item["status"].(map[string]any)
	activeJob := status["active_job"].(map[string]any)
	require.NotContains(t, activeJob["id"], "/tmp/private")
	require.NotContains(t, activeJob["model"], "secret")

	resp, body = s.request(t, http.MethodGet, "/api/v1/admin/asr-providers/local", nil, adminToken, "")
	require.Equal(t, http.StatusOK, resp.Code)
	model := body["models"].([]any)[0].(map[string]any)
	require.Equal(t, "whisper-base", model["id"])
	require.NotContains(t, model, "source_url")
	require.Len(t, body["loaded_models"].([]any), 1)

	resp, body = s.request(t, http.MethodPost, "/api/v1/admin/asr-providers/local/models/load", map[string]any{
		"model":       "whisper-base",
		"operation":   "transcription",
		"load_policy": "require",
	}, adminToken, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	require.Equal(t, "loading", body["status"])
	require.Len(t, provider.loads, 1)
	require.Equal(t, asrcontract.OperationTranscription, provider.loads[0].Operation)
	require.Equal(t, asrcontract.LoadPolicyRequire, provider.loads[0].LoadPolicy)

	resp, body = s.request(t, http.MethodPost, "/api/v1/admin/asr-providers/local/models/unload", map[string]any{
		"model": "whisper-base",
		"force": true,
	}, adminToken, "")
	require.Equal(t, http.StatusAccepted, resp.Code)
	require.Equal(t, "unloading", body["status"])
	require.Len(t, provider.unloads, 1)
	require.True(t, provider.unloads[0].Force)
}

func TestAdminASRProviderRoutesAreAdminOnlyAndMapProviderErrors(t *testing.T) {
	s := newAuthTestServer(t)
	adminToken := registerForFileTests(t, s)
	nonAdmin := models.User{Username: "asr-member", Password: "pw", Role: "user"}
	require.NoError(t, database.DB.Create(&nonAdmin).Error)
	nonAdminToken, err := auth.NewAuthService("test-secret").GenerateToken(&nonAdmin)
	require.NoError(t, err)
	provider := &fakeAdminASRProvider{
		id:      "local",
		loadErr: asrcontract.NewProviderError(asrcontract.CodeProviderBusy, "busy at /tmp/private/model.bin token=secret", true),
	}
	registry, err := engineprovider.NewRegistry("local", provider)
	require.NoError(t, err)
	s.handler.modelRegistry = registry

	resp, body := s.request(t, http.MethodGet, "/api/v1/admin/asr-providers", nil, nonAdminToken, "")
	require.Equal(t, http.StatusForbidden, resp.Code)
	require.Equal(t, "FORBIDDEN", body["error"].(map[string]any)["code"])

	resp, body = s.request(t, http.MethodGet, "/api/v1/admin/asr-providers/missing", nil, adminToken, "")
	require.Equal(t, http.StatusNotFound, resp.Code)

	resp, body = s.request(t, http.MethodPost, "/api/v1/admin/asr-providers/local/models/load", map[string]any{
		"model": "whisper-base",
	}, adminToken, "")
	require.Equal(t, http.StatusConflict, resp.Code)
	errBody := body["error"].(map[string]any)
	require.Equal(t, string(asrcontract.CodeProviderBusy), errBody["code"])
	require.NotContains(t, errBody["message"], "/tmp/private")
	require.NotContains(t, errBody["message"], "secret")
}
