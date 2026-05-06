package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"scriberr/internal/models"
	"scriberr/internal/transcription/asrcontract"
)

type ChunkingMode string

const (
	ChunkingModeFixed    ChunkingMode = "fixed"
	ChunkingModeVAD      ChunkingMode = "vad"
	ChunkingModeProvider ChunkingMode = "provider"
	ChunkingModeNone     ChunkingMode = "none"
)

type planRequest struct {
	Params models.ASRParams
	Steps  []resolvedASRStep
	Models []asrcontract.ModelCard
	Limits PlanLimits
}

type PlanLimits struct {
	MaxChunkSeconds float64
	MaxBatchSize    int
}

type ExecutionPlan struct {
	Steps []PlannedStep
}

type PlannedStep struct {
	Operation  string
	ProviderID string
	Model      string
	Runtime    RuntimePlan
	Chunking   ChunkingPlan
	Batching   BatchingPlan
}

type RuntimePlan struct {
	NumThreads int
}

type ChunkingPlan struct {
	Mode           ChunkingMode
	ProviderOwned  bool
	ChunkSeconds   float64
	OverlapSeconds float64
	VAD            VADPlan
}

type VADPlan struct {
	Threshold         float64
	MinSpeechSeconds  float64
	MinSilenceSeconds float64
	MaxSpeechSeconds  float64
	PaddingSeconds    float64
}

type BatchingPlan struct {
	BatchSize int
}

type PlanBoundary struct {
	Operation  string
	ProviderID string
	Model      string
	ChunkIndex int
	BatchIndex int
	Progress   float64
}

type BoundaryReporter interface {
	ReportPlanBoundary(ctx context.Context, boundary PlanBoundary) error
}

type ExecutionPlanSummary struct {
	Steps []PlannedStepSummary `json:"steps"`
}

type PlannedStepSummary struct {
	Operation      string       `json:"operation"`
	Provider       string       `json:"provider"`
	Model          string       `json:"model"`
	ChunkingMode   ChunkingMode `json:"chunking_mode"`
	ProviderOwned  bool         `json:"provider_owned_chunking,omitempty"`
	ChunkSeconds   float64      `json:"chunk_seconds,omitempty"`
	OverlapSeconds float64      `json:"overlap_seconds,omitempty"`
	BatchSize      int          `json:"batch_size,omitempty"`
	NumThreads     int          `json:"num_threads,omitempty"`
}

func defaultPlanLimits() PlanLimits {
	return PlanLimits{MaxChunkSeconds: 600, MaxBatchSize: 16}
}

func buildExecutionPlan(ctx context.Context, req planRequest) (ExecutionPlan, error) {
	if err := ctx.Err(); err != nil {
		return ExecutionPlan{}, err
	}
	limits := req.Limits
	if limits.MaxChunkSeconds <= 0 {
		limits.MaxChunkSeconds = defaultPlanLimits().MaxChunkSeconds
	}
	if limits.MaxBatchSize <= 0 {
		limits.MaxBatchSize = defaultPlanLimits().MaxBatchSize
	}
	steps := make([]PlannedStep, 0, len(req.Steps))
	for _, step := range req.Steps {
		card, hasCard := modelCardForStep(req.Models, step)
		sourceOptions := step.Options
		if sourceOptions == nil {
			sourceOptions = optionsForStep(req.Params, step)
		}
		chunking, err := planChunking(req.Params, sourceOptions, card, hasCard, limits)
		if err != nil {
			return ExecutionPlan{}, fmt.Errorf("plan %s step %q: %w", step.Kind, step.Model, err)
		}
		batching, err := planBatching(sourceOptions, card, hasCard, limits)
		if err != nil {
			return ExecutionPlan{}, fmt.Errorf("plan %s step %q: %w", step.Kind, step.Model, err)
		}
		steps = append(steps, PlannedStep{
			Operation:  step.Kind,
			ProviderID: step.ProviderID,
			Model:      step.Model,
			Runtime:    RuntimePlan{NumThreads: intOption(sourceOptions, asrcontract.CommonParameterRuntimeNumThreads, 0, intDefault(card.RecommendedDefaults, asrcontract.CommonParameterRuntimeNumThreads))},
			Chunking:   chunking,
			Batching:   batching,
		})
	}
	return ExecutionPlan{Steps: steps}, nil
}

func (p ExecutionPlan) Summary() ExecutionPlanSummary {
	out := ExecutionPlanSummary{Steps: make([]PlannedStepSummary, 0, len(p.Steps))}
	for _, step := range p.Steps {
		out.Steps = append(out.Steps, PlannedStepSummary{
			Operation:      step.Operation,
			Provider:       step.ProviderID,
			Model:          step.Model,
			ChunkingMode:   step.Chunking.Mode,
			ProviderOwned:  step.Chunking.ProviderOwned,
			ChunkSeconds:   step.Chunking.ChunkSeconds,
			OverlapSeconds: step.Chunking.OverlapSeconds,
			BatchSize:      step.Batching.BatchSize,
			NumThreads:     step.Runtime.NumThreads,
		})
	}
	return out
}

func (p ExecutionPlan) ReportBoundary(ctx context.Context, operation string, reporter BoundaryReporter) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if reporter == nil {
		return nil
	}
	for _, step := range p.Steps {
		if step.Operation != operation {
			continue
		}
		return reporter.ReportPlanBoundary(ctx, PlanBoundary{
			Operation:  step.Operation,
			ProviderID: step.ProviderID,
			Model:      step.Model,
			ChunkIndex: 0,
			BatchIndex: 0,
			Progress:   boundaryProgress(operation),
		})
	}
	return nil
}

func boundaryProgress(operation string) float64 {
	switch operation {
	case models.ASRStepTranscription:
		return 0.20
	case models.ASRStepDiarization:
		return 0.70
	case models.ASRStepSpeakerIdentification:
		return 0.78
	default:
		return 0
	}
}

func planChunking(params models.ASRParams, options map[string]any, card asrcontract.ModelCard, hasCard bool, limits PlanLimits) (ChunkingPlan, error) {
	explicitMode := hasOption(options, asrcontract.CommonParameterChunkingMode)
	mode := ChunkingMode(stringOption(options, asrcontract.CommonParameterChunkingMode, "", stringDefault(card.RecommendedDefaults, asrcontract.CommonParameterChunkingMode)))
	if strings.TrimSpace(string(mode)) == "" {
		mode = preferredChunkingMode(card, hasCard)
	}
	if mode == "" {
		mode = ChunkingModeNone
	}
	if !validChunkingMode(mode) {
		return ChunkingPlan{}, fmt.Errorf("chunking mode %q is invalid", mode)
	}
	capabilities := card.Chunking
	if hasCard && capabilities != nil && !capabilities.SupportsEngineChunking {
		if capabilities.SupportsProviderChunking && !explicitMode {
			mode = ChunkingModeProvider
		} else if mode != ChunkingModeProvider && mode != ChunkingModeNone {
			return ChunkingPlan{}, fmt.Errorf("chunking mode %q is not supported by model", mode)
		}
	}
	providerOwned := mode == ChunkingModeProvider
	chunkSeconds := floatOption(options, asrcontract.CommonParameterChunkingChunkSeconds, 0, floatDefault(card.RecommendedDefaults, asrcontract.CommonParameterChunkingChunkSeconds))
	if chunkSeconds <= 0 && capabilities != nil && capabilities.RecommendedChunkSeconds != nil {
		chunkSeconds = *capabilities.RecommendedChunkSeconds
	}
	if mode == ChunkingModeNone || providerOwned {
		chunkSeconds = 0
	}
	if chunkSeconds > limits.MaxChunkSeconds {
		return ChunkingPlan{}, fmt.Errorf("chunk seconds %.2f exceeds limit %.2f", chunkSeconds, limits.MaxChunkSeconds)
	}
	if capabilities != nil && capabilities.MaxChunkSeconds != nil && chunkSeconds > *capabilities.MaxChunkSeconds {
		return ChunkingPlan{}, fmt.Errorf("chunk seconds %.2f exceeds model limit %.2f", chunkSeconds, *capabilities.MaxChunkSeconds)
	}
	return ChunkingPlan{
		Mode:           mode,
		ProviderOwned:  providerOwned,
		ChunkSeconds:   chunkSeconds,
		OverlapSeconds: floatOption(options, asrcontract.CommonParameterChunkingOverlapSeconds, 0, 0),
		VAD: VADPlan{
			Threshold:         floatOption(options, asrcontract.CommonParameterVADThreshold, 0, floatDefault(card.RecommendedDefaults, asrcontract.CommonParameterVADThreshold)),
			MinSpeechSeconds:  floatOption(options, asrcontract.CommonParameterVADMinSpeechSeconds, 0, 0),
			MinSilenceSeconds: floatOption(options, asrcontract.CommonParameterVADMinSilenceSeconds, 0, 0),
			MaxSpeechSeconds:  floatOption(options, asrcontract.CommonParameterVADMaxSpeechSeconds, 0, 0),
			PaddingSeconds:    floatOption(options, asrcontract.CommonParameterVADPaddingSeconds, 0, 0),
		},
	}, nil
}

func planBatching(options map[string]any, card asrcontract.ModelCard, hasCard bool, limits PlanLimits) (BatchingPlan, error) {
	batchSize := intOption(options, asrcontract.CommonParameterBatchingBatchSize, 0, intDefault(card.RecommendedDefaults, asrcontract.CommonParameterBatchingBatchSize))
	if batchSize <= 0 && hasCard && card.Chunking != nil && card.Chunking.RecommendedBatchSize != nil {
		batchSize = *card.Chunking.RecommendedBatchSize
	}
	if batchSize <= 0 {
		batchSize = 1
	}
	if batchSize > limits.MaxBatchSize {
		return BatchingPlan{}, fmt.Errorf("batch size %d exceeds limit %d", batchSize, limits.MaxBatchSize)
	}
	if hasCard && card.Chunking != nil {
		if !card.Chunking.SupportsBatching && batchSize > 1 {
			return BatchingPlan{}, fmt.Errorf("batch size %d is not supported by model", batchSize)
		}
		if card.Chunking.MaxBatchSize != nil && batchSize > *card.Chunking.MaxBatchSize {
			return BatchingPlan{}, fmt.Errorf("batch size %d exceeds model limit %d", batchSize, *card.Chunking.MaxBatchSize)
		}
	}
	return BatchingPlan{BatchSize: batchSize}, nil
}

func modelCardForStep(cards []asrcontract.ModelCard, step resolvedASRStep) (asrcontract.ModelCard, bool) {
	for _, card := range cards {
		if card.ID == step.Model && (card.Provider == "" || card.Provider == step.ProviderID) {
			return card, true
		}
	}
	return asrcontract.ModelCard{}, false
}

func optionsForStep(params models.ASRParams, step resolvedASRStep) map[string]any {
	for _, candidate := range params.Pipeline {
		if strings.TrimSpace(candidate.Kind) == step.Kind && strings.TrimSpace(candidate.Model) == step.Model {
			return candidate.Options
		}
	}
	return nil
}

func preferredChunkingMode(card asrcontract.ModelCard, hasCard bool) ChunkingMode {
	if hasCard && card.Chunking != nil {
		if strings.TrimSpace(card.Chunking.PreferredMode) != "" {
			return ChunkingMode(card.Chunking.PreferredMode)
		}
		if card.Chunking.SupportsProviderChunking && !card.Chunking.SupportsEngineChunking {
			return ChunkingModeProvider
		}
		if card.Chunking.SupportsEngineChunking {
			return ChunkingModeFixed
		}
	}
	return ChunkingModeNone
}

func validChunkingMode(mode ChunkingMode) bool {
	switch mode {
	case ChunkingModeFixed, ChunkingModeVAD, ChunkingModeProvider, ChunkingModeNone:
		return true
	default:
		return false
	}
}

func hasOption(options map[string]any, key string) bool {
	_, ok := options[key]
	return ok
}

func stringOption(options map[string]any, key, legacy, fallback string) string {
	if value, ok := options[key]; ok {
		if typed, ok := value.(string); ok {
			return strings.TrimSpace(typed)
		}
	}
	if strings.TrimSpace(legacy) != "" {
		return strings.TrimSpace(legacy)
	}
	return strings.TrimSpace(fallback)
}

func intOption(options map[string]any, key string, legacy, fallback int) int {
	if value, ok := options[key]; ok {
		if number, ok := numberValue(value); ok {
			return int(number)
		}
	}
	if legacy > 0 {
		return legacy
	}
	return fallback
}

func floatOption(options map[string]any, key string, legacy, fallback float64) float64 {
	if value, ok := options[key]; ok {
		if number, ok := numberValue(value); ok {
			return number
		}
	}
	if legacy > 0 {
		return legacy
	}
	return fallback
}

func stringDefault(defaults map[string]any, key string) string {
	if value, ok := defaults[key]; ok {
		if typed, ok := value.(string); ok {
			return typed
		}
	}
	return ""
}

func intDefault(defaults map[string]any, key string) int {
	if value, ok := defaults[key]; ok {
		if number, ok := numberValue(value); ok {
			return int(number)
		}
	}
	return 0
}

func floatDefault(defaults map[string]any, key string) float64 {
	if value, ok := defaults[key]; ok {
		if number, ok := numberValue(value); ok {
			return number
		}
	}
	return 0
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	default:
		return 0, false
	}
}
