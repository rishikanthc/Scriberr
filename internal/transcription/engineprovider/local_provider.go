package engineprovider

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	appconfig "scriberr/internal/config"
	"scriberr/internal/transcription/asrcontract"
	"scriberr/pkg/logger"

	speechengine "scriberr-engine/speech/engine"
	speechproviders "scriberr-engine/speech/providers"
	"scriberr-engine/speech/runtime"
)

type LocalConfig struct {
	CacheDir     string
	Provider     string
	Threads      int
	MaxLoaded    int
	AutoDownload bool
}

type speechEngine interface {
	Inspect(ctx context.Context) (*speechengine.ProviderInfo, error)
	Models(ctx context.Context) ([]speechproviders.ModelDescriptor, error)
	Status(ctx context.Context) (*speechengine.ProviderStatus, error)
	LoadedModels() []speechengine.LoadedModel
	Execute(ctx context.Context, req speechengine.TaskRequest) (*speechengine.TaskResult, error)
	Close() error
}

type modelLoader interface {
	LoadModel(ctx context.Context, modelID string) error
	UnloadModel(modelID string) error
}

type LocalProvider struct {
	id       string
	cfg      LocalConfig
	engine   speechEngine
	provider runtime.Provider
}

func NewLocalProvider(cfg appconfig.EngineConfig) (*LocalProvider, error) {
	return NewLocalProviderFromConfig(LocalConfig{
		CacheDir:     cfg.CacheDir,
		Provider:     cfg.Provider,
		Threads:      cfg.Threads,
		MaxLoaded:    cfg.MaxLoaded,
		AutoDownload: cfg.AutoDownload,
	})
}

func NewLocalProviderFromConfig(cfg LocalConfig) (*LocalProvider, error) {
	provider, err := runtime.ParseProvider(cfg.Provider)
	if err != nil {
		return nil, sanitizeError(err)
	}
	engineCfg := speechengine.Config{
		CacheDir:        cfg.CacheDir,
		Provider:        provider,
		Threads:         cfg.Threads,
		MaxLoaded:       cfg.MaxLoaded,
		AutoDownload:    cfg.AutoDownload,
		AutoDownloadSet: true,
		Logger:          slog.Default(),
	}
	engine, err := speechengine.New(engineCfg)
	if err != nil {
		return nil, sanitizeError(err)
	}
	logger.Info("Engine provider initialized",
		"provider_id", DefaultProviderID,
		"requested_provider", cfg.Provider,
		"cache_dir", cfg.CacheDir,
		"threads", cfg.Threads,
		"max_loaded", cfg.MaxLoaded,
		"auto_download", cfg.AutoDownload,
	)
	return newLocalProviderWithEngine(DefaultProviderID, cfg, provider, engine), nil
}

func newLocalProviderWithEngine(id string, cfg LocalConfig, provider runtime.Provider, engine speechEngine) *LocalProvider {
	if strings.TrimSpace(id) == "" {
		id = DefaultProviderID
	}
	return &LocalProvider{
		id:       id,
		cfg:      cfg,
		engine:   engine,
		provider: provider,
	}
}

func (p *LocalProvider) ID() string {
	return p.id
}

func (p *LocalProvider) Inspect(ctx context.Context) (*asrcontract.ProviderInfo, error) {
	info, err := p.engine.Inspect(ctx)
	if err != nil {
		return nil, sanitizeError(err)
	}
	return providerInfoFromEngine(info, p.id), nil
}

func (p *LocalProvider) Models(ctx context.Context) ([]asrcontract.ModelCard, error) {
	models, err := p.engine.Models(ctx)
	if err != nil {
		return nil, sanitizeError(err)
	}
	out := make([]asrcontract.ModelCard, 0, len(models))
	for _, model := range models {
		out = append(out, modelCardFromEngine(model, p.id))
	}
	return out, nil
}

func (p *LocalProvider) Status(ctx context.Context) (*asrcontract.ProviderStatus, error) {
	status, err := p.engine.Status(ctx)
	if err != nil {
		return nil, sanitizeError(err)
	}
	return providerStatusFromEngine(status), nil
}

func (p *LocalProvider) LoadModel(ctx context.Context, req asrcontract.LoadModelRequest) error {
	if loader, ok := p.engine.(modelLoader); ok {
		return sanitizeError(loader.LoadModel(ctx, req.Model))
	}
	return asrcontract.NewProviderError(asrcontract.CodeUnsupportedOperation, "local engine model loading is not available", false)
}

func (p *LocalProvider) UnloadModel(ctx context.Context, req asrcontract.UnloadModelRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if loader, ok := p.engine.(modelLoader); ok {
		return sanitizeError(loader.UnloadModel(req.Model))
	}
	return asrcontract.NewProviderError(asrcontract.CodeUnsupportedOperation, "local engine model unloading is not available", false)
}

func (p *LocalProvider) LoadedModels(ctx context.Context) ([]asrcontract.LoadedModel, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	models := p.engine.LoadedModels()
	out := make([]asrcontract.LoadedModel, 0, len(models))
	for _, model := range models {
		out = append(out, loadedModelFromEngine(model))
	}
	return out, nil
}

func (p *LocalProvider) ExecuteTask(ctx context.Context, req TaskRequest) (*TaskResult, error) {
	switch req.Operation {
	case asrcontract.OperationTranscription:
		result, err := p.executeTranscription(ctx, req)
		if err != nil {
			return nil, err
		}
		return &TaskResult{Operation: req.Operation, ModelID: result.ModelID, EngineID: result.EngineID, Result: result, Metadata: result.Metadata}, nil
	case asrcontract.OperationDiarization:
		result, err := p.executeDiarization(ctx, req)
		if err != nil {
			return nil, err
		}
		return &TaskResult{Operation: req.Operation, ModelID: result.ModelID, EngineID: result.EngineID, Result: result}, nil
	case asrcontract.OperationSpeakerIdentification:
		return nil, asrcontract.NewProviderError(asrcontract.CodeUnsupportedOperation, "local provider does not support speaker identification", false)
	default:
		return nil, asrcontract.NewProviderError(asrcontract.CodeUnsupportedOperation, "local provider task is not supported", false)
	}
}

func (p *LocalProvider) executeTranscription(ctx context.Context, req TaskRequest) (*TranscriptionResult, error) {
	modelID := strings.TrimSpace(req.ModelID)
	engineReq := speechengine.TaskRequest{
		RequestID:  req.JobID,
		Task:       speechproviders.TaskTranscription,
		ModelID:    modelID,
		AudioPath:  req.AudioPath,
		Parameters: copyParameters(req.Parameters),
		Progress:   localProgressSink{downstream: req.Progress},
	}
	taskResult, err := p.engine.Execute(ctx, engineReq)
	if err != nil {
		return nil, sanitizeError(err)
	}
	if taskResult == nil {
		return nil, sanitizeErrorf("local engine returned no transcription task result")
	}
	out, _ := taskResult.Result.(*speechengine.TranscriptionResult)
	if out == nil {
		return nil, sanitizeErrorf("local engine returned no transcription result")
	}
	modelID = defaultString(out.Plan.ModelID, modelID)
	if modelID == "" {
		modelID = p.defaultModelID(ctx, asrcontract.CapabilityTranscription)
	}
	words := make([]TranscriptWord, 0, len(out.Words))
	for _, word := range out.Words {
		words = append(words, TranscriptWord{
			Start: word.StartSec,
			End:   word.EndSec,
			Word:  word.Text,
		})
	}
	segments := make([]TranscriptSegment, 0, len(out.Segments))
	for idx, segment := range out.Segments {
		segments = append(segments, TranscriptSegment{
			ID:    fmt.Sprintf("seg_%04d", idx),
			Start: segment.StartSec,
			End:   segment.EndSec,
			Text:  segment.Text,
		})
	}
	return &TranscriptionResult{
		Text:     out.Text,
		Language: out.Language,
		Words:    words,
		Segments: segments,
		ModelID:  modelID,
		EngineID: p.id,
		Metadata: localTranscriptionMetadata(modelID, out),
	}, nil
}

func localTranscriptionMetadata(modelID string, out *speechengine.TranscriptionResult) map[string]any {
	metadata := map[string]any{
		"model": modelID,
	}
	if out == nil {
		return metadata
	}
	if out.Metrics.AudioDurationSec > 0 ||
		out.Metrics.DecodeDuration > 0 ||
		out.Metrics.ChunkCount > 0 ||
		out.Metrics.BatchSize > 0 ||
		out.Metrics.HypothesisWords > 0 {
		metadata["metrics"] = out.Metrics
	}
	metadata["plan"] = out.Plan
	return metadata
}

func (p *LocalProvider) defaultModelID(ctx context.Context, capability asrcontract.Capability) string {
	models, err := p.Models(ctx)
	if err != nil {
		return ""
	}
	for _, model := range models {
		if model.Default && model.Supports(capability) {
			return model.ID
		}
	}
	return ""
}

func copyParameters(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (p *LocalProvider) executeDiarization(ctx context.Context, req TaskRequest) (*DiarizationResult, error) {
	modelID := strings.TrimSpace(req.ModelID)
	engineReq := speechengine.TaskRequest{
		RequestID:  req.JobID,
		Task:       speechproviders.TaskDiarization,
		ModelID:    modelID,
		AudioPath:  req.AudioPath,
		Parameters: copyParameters(req.Parameters),
		Progress:   localProgressSink{downstream: req.Progress},
	}
	taskResult, err := p.engine.Execute(ctx, engineReq)
	if err != nil {
		return nil, sanitizeError(err)
	}
	if taskResult == nil {
		return nil, sanitizeErrorf("local engine returned no diarization task result")
	}
	out, _ := taskResult.Result.(*speechengine.DiarizationResult)
	if out == nil {
		return nil, sanitizeErrorf("local engine returned no diarization result")
	}
	if modelID == "" {
		modelID = p.defaultModelID(ctx, asrcontract.CapabilityDiarization)
	}
	segments := make([]DiarizationSegment, 0, len(out.SpeakerSegments))
	for _, segment := range out.SpeakerSegments {
		segments = append(segments, DiarizationSegment{
			Start:   segment.StartSec,
			End:     segment.EndSec,
			Speaker: segment.SpeakerID,
		})
	}
	return &DiarizationResult{
		Segments: segments,
		ModelID:  modelID,
		EngineID: p.id,
	}, nil
}

func (p *LocalProvider) Close() error {
	if p.engine == nil {
		return nil
	}
	if err := p.engine.Close(); err != nil {
		return sanitizeError(err)
	}
	return nil
}

type localProgressSink struct {
	downstream ProgressSink
}

func (s localProgressSink) Report(ctx context.Context, progress speechengine.Progress) {
	if s.downstream == nil {
		return
	}
	s.downstream.Report(ctx, asrcontract.ProviderProgress{
		Stage:     asrcontract.Stage(progress.Stage),
		Progress:  progress.Progress,
		Message:   progress.Message,
		Operation: asrcontract.Operation(progress.Operation),
		Model:     progress.Model,
		Timestamp: progress.Timestamp,
	})
}

func providerInfoFromEngine(info *speechengine.ProviderInfo, providerID string) *asrcontract.ProviderInfo {
	if info == nil {
		return nil
	}
	return &asrcontract.ProviderInfo{
		ContractVersion: asrcontract.ContractVersionV1,
		Provider: asrcontract.ProviderIdentity{
			ID:      providerID,
			Name:    info.Provider.Name,
			Version: info.Provider.Version,
			Vendor:  info.Provider.Vendor,
		},
		Runtime: asrcontract.RuntimeInfo{
			DeviceBackends:       append([]string(nil), info.Runtime.DeviceBackends...),
			ActiveBackend:        info.Runtime.ActiveBackend,
			SupportsConcurrent:   info.Runtime.SupportsConcurrent,
			MaxConcurrentJobs:    info.Runtime.MaxConcurrentJobs,
			ProviderCapabilities: providerCapabilitiesFromEngine(info.Runtime.ProviderCapabilities),
		},
		AudioInput: asrcontract.AudioInputSpec{
			RequiredSampleRate: info.AudioInput.RequiredSampleRate,
			RequiredChannels:   info.AudioInput.RequiredChannels,
			Formats:            append([]string(nil), info.AudioInput.Formats...),
			PathMode:           asrcontract.PathMode(info.AudioInput.PathMode),
		},
	}
}

func modelCardFromEngine(descriptor speechproviders.ModelDescriptor, providerID string) asrcontract.ModelCard {
	capabilities := capabilitiesFromDescriptor(descriptor)
	return asrcontract.ModelCard{
		ID:                   descriptor.ID,
		DisplayName:          descriptor.DisplayName,
		Provider:             providerID,
		ModelType:            descriptor.ModelType,
		Version:              descriptor.Version,
		Installed:            descriptor.Installed,
		Loaded:               descriptor.Loaded,
		Default:              descriptor.Default,
		Tasks:                tasksFromDescriptor(descriptor.Tasks),
		Languages:            languageIDsFromDescriptor(descriptor.Languages),
		LanguageSupport:      languageSupportFromDescriptor(descriptor.Languages),
		Capabilities:         capabilities,
		ResourceRequirements: resourceRequirementsFromDescriptor(descriptor.Runtime),
		Chunking:             chunkingCapabilitiesFromDescriptor(descriptor.Chunking, descriptor.Runtime, capabilities),
		Dependencies:         dependencyRequirementsFromDescriptor(descriptor.Dependencies),
		Artifacts:            artifactRequirementsFromDescriptor(descriptor.Artifacts),
		ParameterSchema:      parameterSchemaFromDescriptor(descriptor.Parameters),
		RecommendedDefaults:  copyRecommendedDefaults(descriptor.RecommendedDefaults),
		License:              descriptor.License,
	}
}

func languageSupportFromDescriptor(languages []speechproviders.LanguageSupport) *asrcontract.LanguageSupport {
	ids := make([]string, 0, len(languages))
	mode := ""
	for _, language := range languages {
		if strings.TrimSpace(language.ID) != "" {
			ids = append(ids, language.ID)
		}
		if mode == "" && strings.TrimSpace(string(language.Mode)) != "" {
			mode = string(language.Mode)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	if mode == "" {
		mode = "fixed"
	}
	return &asrcontract.LanguageSupport{Languages: ids, Mode: mode}
}

func languageIDsFromDescriptor(languages []speechproviders.LanguageSupport) []string {
	out := make([]string, 0, len(languages))
	for _, language := range languages {
		if strings.TrimSpace(language.ID) != "" {
			out = append(out, language.ID)
		}
	}
	return out
}

func tasksFromDescriptor(tasks []speechproviders.TaskDescriptor) []asrcontract.Task {
	out := make([]asrcontract.Task, 0, len(tasks))
	for _, task := range tasks {
		switch task.Kind {
		case speechproviders.TaskTranscription:
			out = append(out, asrcontract.TaskTranscribe)
		case speechproviders.TaskTranslation:
			out = append(out, asrcontract.TaskTranslate)
		default:
			out = append(out, asrcontract.Task(task.Kind))
		}
	}
	return out
}

func resourceRequirementsFromDescriptor(runtime speechproviders.RuntimeCapabilities) asrcontract.ResourceRequirements {
	backends := make([]string, 0, len(runtime.Backends))
	for _, backend := range runtime.Backends {
		backends = append(backends, string(backend))
	}
	return asrcontract.ResourceRequirements{Backends: backends}
}

func chunkingCapabilitiesFromDescriptor(chunking speechproviders.ChunkingCapabilities, runtime speechproviders.RuntimeCapabilities, capabilities asrcontract.Capabilities) *asrcontract.ChunkingCapabilities {
	if !capabilities.Transcription {
		return nil
	}
	if !chunking.SupportsEngineChunking && !chunking.SupportsProviderChunking && chunking.PreferredMode == "" {
		return nil
	}
	out := &asrcontract.ChunkingCapabilities{
		SupportsEngineChunking:   chunking.SupportsEngineChunking,
		SupportsProviderChunking: chunking.SupportsProviderChunking,
		PreferredMode:            string(chunking.PreferredMode),
		SupportsBatching:         runtime.SupportsBatching,
	}
	if chunking.RecommendedChunkSeconds > 0 {
		out.RecommendedChunkSeconds = ptrFloat64(chunking.RecommendedChunkSeconds)
	}
	if chunking.MaxChunkSeconds > 0 {
		out.MaxChunkSeconds = ptrFloat64(chunking.MaxChunkSeconds)
	}
	if runtime.RecommendedBatchSize > 0 {
		out.RecommendedBatchSize = ptrInt(runtime.RecommendedBatchSize)
	}
	if runtime.MaxBatchSize > 0 {
		out.MaxBatchSize = ptrInt(runtime.MaxBatchSize)
	}
	return out
}

func parameterSchemaFromDescriptor(parameters []speechproviders.ParameterDescriptor) asrcontract.ParameterSchema {
	if len(parameters) == 0 {
		return nil
	}
	out := make(asrcontract.ParameterSchema, 0, len(parameters))
	for _, parameter := range parameters {
		out = append(out, asrcontract.ParameterDescriptor{
			Key:             parameter.Key,
			Label:           parameter.Label,
			Type:            asrcontract.ParameterType(parameter.Type),
			Default:         parameter.Default,
			Min:             cloneFloat64(parameter.Min),
			Max:             cloneFloat64(parameter.Max),
			Step:            cloneFloat64(parameter.Step),
			Options:         parameterOptionsFromDescriptor(parameter.Options),
			Scope:           asrcontract.ParameterScope(parameter.Scope),
			Required:        parameter.Required,
			Advanced:        parameter.Advanced,
			ReadOnly:        parameter.ReadOnly,
			RequiresReload:  parameter.RequiresReload,
			ExposeInSummary: parameter.ExposeInSummary,
			VisibleWhen:     activationRulesFromDescriptor(parameter.VisibleWhen),
		})
	}
	return out
}

func dependencyRequirementsFromDescriptor(dependencies []speechproviders.DependencyRequirement) []asrcontract.DependencyRequirement {
	out := make([]asrcontract.DependencyRequirement, 0, len(dependencies))
	for _, dependency := range dependencies {
		out = append(out, asrcontract.DependencyRequirement{
			ID:          dependency.ID,
			Required:    dependency.Required,
			Description: dependency.Description,
			Activation:  activationRulesFromDescriptor(dependency.Activation),
		})
	}
	return out
}

func artifactRequirementsFromDescriptor(artifacts []speechproviders.ArtifactRequirement) []asrcontract.ArtifactRequirement {
	out := make([]asrcontract.ArtifactRequirement, 0, len(artifacts))
	for _, artifact := range artifacts {
		out = append(out, asrcontract.ArtifactRequirement{
			Key:             artifact.Key,
			Required:        artifact.Required,
			ExternalWeights: artifact.ExternalWeights,
			Description:     artifact.Description,
		})
	}
	return out
}

func activationRulesFromDescriptor(rules []speechproviders.ActivationRule) []asrcontract.ActivationRule {
	out := make([]asrcontract.ActivationRule, 0, len(rules))
	for _, rule := range rules {
		out = append(out, asrcontract.ActivationRule{
			Parameter: rule.Parameter,
			Operator:  asrcontract.ActivationOperator(rule.Operator),
			Value:     rule.Value,
		})
	}
	return out
}

func parameterOptionsFromDescriptor(options []speechproviders.ParameterOption) []asrcontract.ParameterOption {
	if len(options) == 0 {
		return nil
	}
	out := make([]asrcontract.ParameterOption, 0, len(options))
	for _, option := range options {
		out = append(out, asrcontract.ParameterOption{Value: option.Value, Label: option.Label})
	}
	return out
}

func copyRecommendedDefaults(defaults map[string]any) map[string]any {
	if len(defaults) == 0 {
		return nil
	}
	out := make(map[string]any, len(defaults))
	for key, value := range defaults {
		out[key] = value
	}
	return out
}

func cloneFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func ptrFloat64(value float64) *float64 { return &value }

func ptrInt(value int) *int { return &value }

func providerStatusFromEngine(status *speechengine.ProviderStatus) *asrcontract.ProviderStatus {
	if status == nil {
		return nil
	}
	loaded := make([]asrcontract.LoadedModel, 0, len(status.LoadedModels))
	for _, model := range status.LoadedModels {
		loaded = append(loaded, loadedModelFromEngine(model))
	}
	return &asrcontract.ProviderStatus{
		State:        asrcontract.ProviderState(status.State),
		ActiveJob:    activeJobFromEngine(status.ActiveJob),
		LoadedModels: loaded,
		Capacity: asrcontract.ProviderCapacity{
			MaxConcurrentJobs: status.Capacity.MaxConcurrentJobs,
			AvailableSlots:    status.Capacity.AvailableSlots,
		},
	}
}

func activeJobFromEngine(job *speechengine.ActiveJob) *asrcontract.ActiveJob {
	if job == nil {
		return nil
	}
	return &asrcontract.ActiveJob{
		ID:        job.ID,
		Operation: asrcontract.Operation(job.Operation),
		Model:     job.Model,
		Stage:     asrcontract.Stage(job.Stage),
		Progress:  job.Progress,
	}
}

func loadedModelFromEngine(model speechengine.LoadedModel) asrcontract.LoadedModel {
	return asrcontract.LoadedModel{
		ID:             model.ID,
		ResourceKind:   model.ResourceKind,
		ResourceRole:   model.ResourceRole,
		RuntimeBackend: model.RuntimeBackend,
		Threads:        model.Threads,
		ReloadKey:      model.ReloadKey,
		LoadedAt:       model.LoadedAt,
	}
}

func providerCapabilitiesFromEngine(capabilities []speechproviders.TaskKind) []asrcontract.Capability {
	out := make([]asrcontract.Capability, 0, len(capabilities))
	for _, capability := range capabilities {
		switch capability {
		case speechproviders.TaskTranscription:
			out = append(out, asrcontract.CapabilityTranscription)
		case speechproviders.TaskDiarization:
			out = append(out, asrcontract.CapabilityDiarization)
		case speechproviders.TaskSpeakerIdentification:
			out = append(out, asrcontract.CapabilitySpeakerIdentification)
		case speechproviders.TaskTranslation:
			out = append(out, asrcontract.CapabilityTranslation)
		default:
			out = append(out, asrcontract.Capability(capability))
		}
	}
	return out
}

func capabilitiesFromDescriptor(descriptor speechproviders.ModelDescriptor) asrcontract.Capabilities {
	capabilities := asrcontract.Capabilities{
		WordTimestamps:    descriptor.Output.WordTimestamps,
		SegmentTimestamps: descriptor.Output.SegmentTimestamps,
		TokenTimestamps:   descriptor.Output.TokenTimestamps,
		LanguageDetection: descriptor.Output.LanguageSpans,
		SpeakerEmbeddings: descriptor.Output.SpeakerLabels,
		Translation:       descriptor.Output.Translation,
	}
	for _, task := range descriptor.Tasks {
		switch task.Kind {
		case speechproviders.TaskTranscription:
			capabilities.Transcription = true
		case speechproviders.TaskDiarization:
			capabilities.Diarization = true
		case speechproviders.TaskSpeakerIdentification:
			capabilities.SpeakerIdentification = true
		case speechproviders.TaskTranslation:
			capabilities.Translation = true
		}
	}
	return capabilities
}
