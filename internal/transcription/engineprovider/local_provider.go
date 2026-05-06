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
	Models(ctx context.Context) ([]speechengine.ModelCard, error)
	Status(ctx context.Context) (*speechengine.ProviderStatus, error)
	LoadedModels() []speechengine.LoadedModel
	Transcribe(ctx context.Context, req speechengine.TranscriptionRequest) (*speechengine.TranscriptionResult, error)
	Diarize(ctx context.Context, req speechengine.DiarizationRequest) (*speechengine.DiarizationResult, error)
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
		out = append(out, modelCardFromEngine(model, p.id, p.cfg, p.provider))
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

func (p *LocalProvider) Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error) {
	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		modelID = DefaultTranscriptionModel
	}
	engineReq := speechengine.TranscriptionRequest{
		RequestID:  req.JobID,
		ModelID:    modelID,
		AudioPath:  req.AudioPath,
		Parameters: copyParameters(req.Parameters),
		Progress:   localProgressSink{downstream: req.Progress},
	}
	out, err := p.engine.Transcribe(ctx, engineReq)
	if err != nil {
		return nil, sanitizeError(err)
	}
	if out == nil {
		return nil, sanitizeErrorf("local engine returned no transcription result")
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

func (p *LocalProvider) Diarize(ctx context.Context, req DiarizationRequest) (*DiarizationResult, error) {
	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		modelID = DefaultDiarizationModel
	}
	engineReq := speechengine.DiarizationRequest{
		RequestID:  req.JobID,
		ModelID:    modelID,
		AudioPath:  req.AudioPath,
		Parameters: copyParameters(req.Parameters),
		Progress:   localProgressSink{downstream: req.Progress},
	}
	out, err := p.engine.Diarize(ctx, engineReq)
	if err != nil {
		return nil, sanitizeError(err)
	}
	if out == nil {
		return nil, sanitizeErrorf("local engine returned no diarization result")
	}
	segments := make([]DiarizationSegment, 0, len(out.Segments))
	for _, segment := range out.Segments {
		segments = append(segments, DiarizationSegment{
			Start:   segment.Start,
			End:     segment.End,
			Speaker: fmt.Sprintf("SPEAKER_%02d", segment.Speaker),
		})
	}
	return &DiarizationResult{
		Segments: segments,
		ModelID:  modelID,
		EngineID: p.id,
	}, nil
}

func (p *LocalProvider) IdentifySpeakers(ctx context.Context, req asrcontract.SpeakerIDRequest) (*asrcontract.SpeakerIDResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, asrcontract.NewProviderError(asrcontract.CodeUnsupportedOperation, "local provider does not support speaker identification", false)
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

func modelCardFromEngine(model speechengine.ModelCard, providerID string, cfg LocalConfig, provider runtime.Provider) asrcontract.ModelCard {
	capabilities := capabilitiesFromEngine(model.Capabilities)
	descriptor := model.Descriptor
	if strings.TrimSpace(descriptor.ID) == "" {
		return fallbackModelCardFromEngine(model, providerID, capabilities)
	}
	return asrcontract.ModelCard{
		ID:                   firstNonEmpty(descriptor.ID, model.ID),
		DisplayName:          firstNonEmpty(descriptor.DisplayName, model.DisplayName),
		Provider:             providerID,
		Family:               firstNonEmpty(descriptor.Family, model.Family),
		Version:              firstNonEmpty(descriptor.Version, model.Version),
		Installed:            model.Installed,
		Loaded:               model.Loaded,
		Default:              model.Default,
		Tasks:                tasksFromDescriptor(descriptor.Tasks, model.Tasks),
		Languages:            languageIDsFromDescriptor(descriptor.Languages, model.Languages),
		LanguageSupport:      languageSupportFromDescriptor(descriptor.Languages, model.Languages),
		Capabilities:         capabilities,
		ResourceRequirements: resourceRequirementsFromDescriptor(descriptor.Runtime, model.ResourceRequirements),
		Chunking:             chunkingCapabilitiesFromDescriptor(descriptor.Chunking, descriptor.Runtime, capabilities),
		ParameterSchema:      parameterSchemaFromDescriptor(descriptor.Parameters),
		RecommendedDefaults:  copyRecommendedDefaults(descriptor.RecommendedDefaults),
		Extensions:           descriptorExtensionsFromEngine(descriptor),
	}
}

func fallbackModelCardFromEngine(model speechengine.ModelCard, providerID string, capabilities asrcontract.Capabilities) asrcontract.ModelCard {
	return asrcontract.ModelCard{
		ID:                   model.ID,
		DisplayName:          model.DisplayName,
		Provider:             providerID,
		Family:               model.Family,
		Version:              model.Version,
		Installed:            model.Installed,
		Loaded:               model.Loaded,
		Default:              model.Default,
		Tasks:                tasksFromEngine(model.Tasks),
		Languages:            append([]string(nil), model.Languages...),
		LanguageSupport:      fallbackLanguageSupport(model.Languages),
		Capabilities:         capabilities,
		ResourceRequirements: resourceRequirementsFromEngine(model.ResourceRequirements),
	}
}

func fallbackLanguageSupport(languages []string) *asrcontract.LanguageSupport {
	if len(languages) == 0 {
		return nil
	}
	mode := "fixed"
	if len(languages) > 1 {
		mode = "configurable"
	}
	return &asrcontract.LanguageSupport{Languages: append([]string(nil), languages...), Mode: mode}
}

func languageSupportFromDescriptor(languages []speechproviders.LanguageSupport, fallback []string) *asrcontract.LanguageSupport {
	if len(languages) == 0 {
		return fallbackLanguageSupport(fallback)
	}
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

func languageIDsFromDescriptor(languages []speechproviders.LanguageSupport, fallback []string) []string {
	if len(languages) == 0 {
		return append([]string(nil), fallback...)
	}
	out := make([]string, 0, len(languages))
	for _, language := range languages {
		if strings.TrimSpace(language.ID) != "" {
			out = append(out, language.ID)
		}
	}
	return out
}

func tasksFromDescriptor(tasks []speechproviders.TaskDescriptor, fallback []speechengine.Task) []asrcontract.Task {
	if len(tasks) == 0 {
		return tasksFromEngine(fallback)
	}
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

func resourceRequirementsFromDescriptor(runtime speechproviders.RuntimeCapabilities, fallback speechengine.ResourceRequirements) asrcontract.ResourceRequirements {
	if len(runtime.Backends) == 0 {
		return resourceRequirementsFromEngine(fallback)
	}
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
			Key:            parameter.Key,
			Label:          parameter.Label,
			Type:           asrcontract.ParameterType(parameter.Type),
			Default:        parameter.Default,
			Min:            cloneFloat64(parameter.Min),
			Max:            cloneFloat64(parameter.Max),
			Step:           cloneFloat64(parameter.Step),
			Options:        parameterOptionsFromDescriptor(parameter.Options),
			Scope:          asrcontract.ParameterScope(parameter.Scope),
			Advanced:       parameter.Advanced,
			RequiresReload: parameter.RequiresReload,
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

func descriptorExtensionsFromEngine(descriptor speechproviders.ModelDescriptor) map[string]any {
	extensions := map[string]any{}
	if descriptor.License != "" {
		extensions["license"] = descriptor.License
	}
	if len(descriptor.Artifacts) > 0 {
		artifacts := make([]map[string]any, 0, len(descriptor.Artifacts))
		for _, artifact := range descriptor.Artifacts {
			artifacts = append(artifacts, map[string]any{
				"key":              artifact.Key,
				"required":         artifact.Required,
				"external_weights": artifact.ExternalWeights,
				"description":      artifact.Description,
			})
		}
		extensions["artifacts"] = artifacts
	}
	if len(extensions) == 0 {
		return nil
	}
	return extensions
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
		ID:       model.ID,
		LoadedAt: model.LoadedAt,
	}
}

func providerCapabilitiesFromEngine(capabilities []speechengine.Capability) []asrcontract.Capability {
	out := make([]asrcontract.Capability, 0, len(capabilities))
	for _, capability := range capabilities {
		out = append(out, asrcontract.Capability(capability))
	}
	return out
}

func tasksFromEngine(tasks []speechengine.Task) []asrcontract.Task {
	out := make([]asrcontract.Task, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, asrcontract.Task(task))
	}
	return out
}

func capabilitiesFromEngine(capabilities speechengine.Capabilities) asrcontract.Capabilities {
	return asrcontract.Capabilities{
		Transcription:     capabilities.Transcription,
		Diarization:       capabilities.Diarization,
		WordTimestamps:    capabilities.WordTimestamps,
		SegmentTimestamps: capabilities.SegmentTimestamps,
		TokenTimestamps:   capabilities.TokenTimestamps,
		LanguageDetection: capabilities.LanguageDetection,
		SpeakerEmbeddings: capabilities.SpeakerEmbeddings,
	}
}

func resourceRequirementsFromEngine(requirements speechengine.ResourceRequirements) asrcontract.ResourceRequirements {
	return asrcontract.ResourceRequirements{
		Backends: append([]string(nil), requirements.Backends...),
	}
}
