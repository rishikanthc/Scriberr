package orchestrator

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"scriberr/internal/models"
	"scriberr/internal/transcription/asrcontract"

	"github.com/stretchr/testify/require"
)

func TestBuildExecutionPlanDefaultsParakeetFixedChunking(t *testing.T) {
	plan, err := buildExecutionPlan(context.Background(), planRequest{
		Params: models.ASRParams{},
		Steps: []resolvedASRStep{{
			Kind:        models.ASRStepTranscription,
			ProviderID:  "local",
			Model:       "parakeet-v3",
			ModelFamily: "nemo_transducer",
		}},
		Models: []asrcontract.ModelCard{parakeetPlanCard()},
		Limits: defaultPlanLimits(),
	})

	require.NoError(t, err)
	require.Len(t, plan.Steps, 1)
	step := plan.Steps[0]
	require.Equal(t, ChunkingModeFixed, step.Chunking.Mode)
	require.Equal(t, 30.0, step.Chunking.ChunkSeconds)
	require.Equal(t, 1, step.Batching.BatchSize)
	require.Equal(t, 4, step.Runtime.NumThreads)
	require.False(t, step.Chunking.ProviderOwned)
}

func TestBuildExecutionPlanValidatesUserOverrides(t *testing.T) {
	_, err := buildExecutionPlan(context.Background(), planRequest{
		Params: models.ASRParams{
			Pipeline: []models.ASRStep{{
				Kind:    models.ASRStepTranscription,
				Model:   "parakeet-v3",
				Options: map[string]any{asrcontract.CommonParameterBatchingBatchSize: float64(8)},
			}},
		},
		Steps: []resolvedASRStep{{
			Kind:       models.ASRStepTranscription,
			ProviderID: "local",
			Model:      "parakeet-v3",
		}},
		Models: []asrcontract.ModelCard{parakeetPlanCard()},
		Limits: defaultPlanLimits(),
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "batch size")
}

func TestBuildExecutionPlanAllowsProviderOwnedChunkingFallback(t *testing.T) {
	card := parakeetPlanCard()
	card.ID = "provider-chunker"
	card.Provider = "remote"
	card.Chunking = &asrcontract.ChunkingCapabilities{
		SupportsEngineChunking:   false,
		SupportsProviderChunking: true,
		PreferredMode:            string(ChunkingModeProvider),
		SupportsBatching:         false,
	}
	card.RecommendedDefaults = map[string]any{
		asrcontract.CommonParameterChunkingMode: string(ChunkingModeFixed),
	}

	plan, err := buildExecutionPlan(context.Background(), planRequest{
		Params: models.ASRParams{},
		Steps: []resolvedASRStep{{
			Kind:       models.ASRStepTranscription,
			ProviderID: "remote",
			Model:      "provider-chunker",
		}},
		Models: []asrcontract.ModelCard{card},
		Limits: defaultPlanLimits(),
	})

	require.NoError(t, err)
	require.Equal(t, ChunkingModeProvider, plan.Steps[0].Chunking.Mode)
	require.True(t, plan.Steps[0].Chunking.ProviderOwned)
}

func TestBuildExecutionPlanRejectsUnsupportedExplicitChunking(t *testing.T) {
	card := parakeetPlanCard()
	card.ID = "provider-chunker"
	card.Provider = "remote"
	card.Chunking = &asrcontract.ChunkingCapabilities{
		SupportsEngineChunking:   false,
		SupportsProviderChunking: true,
		PreferredMode:            string(ChunkingModeProvider),
	}

	_, err := buildExecutionPlan(context.Background(), planRequest{
		Params: models.ASRParams{
			Pipeline: []models.ASRStep{{
				Kind:    models.ASRStepTranscription,
				Model:   "provider-chunker",
				Options: map[string]any{asrcontract.CommonParameterChunkingMode: string(ChunkingModeFixed)},
			}},
		},
		Steps: []resolvedASRStep{{
			Kind:       models.ASRStepTranscription,
			ProviderID: "remote",
			Model:      "provider-chunker",
		}},
		Models: []asrcontract.ModelCard{card},
		Limits: defaultPlanLimits(),
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "chunking mode")
}

func TestExecutionPlanSummaryIsPathFreeAndDeterministic(t *testing.T) {
	plan, err := buildExecutionPlan(context.Background(), planRequest{
		Params: models.ASRParams{},
		Steps: []resolvedASRStep{{
			Kind:       models.ASRStepTranscription,
			ProviderID: "local",
			Model:      "parakeet-v3",
		}},
		Models: []asrcontract.ModelCard{parakeetPlanCard()},
		Limits: defaultPlanLimits(),
	})
	require.NoError(t, err)

	data, err := json.Marshal(plan.Summary())
	require.NoError(t, err)
	text := string(data)
	require.Contains(t, text, `"chunking_mode":"fixed"`)
	require.Contains(t, text, `"batch_size":1`)
	require.NotContains(t, text, "/")
	require.False(t, strings.Contains(strings.ToLower(text), "token"))
}

func TestExecutionPlanBoundaryHookChecksCancellation(t *testing.T) {
	plan := ExecutionPlan{Steps: []PlannedStep{{
		Operation:  models.ASRStepTranscription,
		ProviderID: "local",
		Model:      "parakeet-v3",
	}}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := plan.ReportBoundary(ctx, models.ASRStepTranscription, &boundaryRecorder{})

	require.ErrorIs(t, err, context.Canceled)
}

func TestExecutionPlanBoundaryHookReportsProgress(t *testing.T) {
	plan := ExecutionPlan{Steps: []PlannedStep{{
		Operation:  models.ASRStepDiarization,
		ProviderID: "remote",
		Model:      "diarization-default",
	}}}
	recorder := &boundaryRecorder{}

	err := plan.ReportBoundary(context.Background(), models.ASRStepDiarization, recorder)

	require.NoError(t, err)
	require.Equal(t, models.ASRStepDiarization, recorder.boundary.Operation)
	require.Equal(t, "remote", recorder.boundary.ProviderID)
	require.Equal(t, 0.70, recorder.boundary.Progress)
}

type boundaryRecorder struct {
	boundary PlanBoundary
	err      error
}

func (r *boundaryRecorder) ReportPlanBoundary(ctx context.Context, boundary PlanBoundary) error {
	if r == nil {
		return nil
	}
	if r.err != nil {
		return r.err
	}
	r.boundary = boundary
	return nil
}

func parakeetPlanCard() asrcontract.ModelCard {
	return asrcontract.ModelCard{
		ID:       "parakeet-v3",
		Provider: "local",
		Family:   "nemo_transducer",
		Capabilities: asrcontract.Capabilities{
			Transcription: true,
		},
		Chunking: &asrcontract.ChunkingCapabilities{
			SupportsEngineChunking:   true,
			SupportsProviderChunking: false,
			PreferredMode:            string(ChunkingModeFixed),
			RecommendedChunkSeconds:  floatPtr(30),
			MaxChunkSeconds:          floatPtr(60),
			SupportsBatching:         true,
			RecommendedBatchSize:     intPtr(1),
			MaxBatchSize:             intPtr(2),
		},
		RecommendedDefaults: map[string]any{
			asrcontract.CommonParameterRuntimeNumThreads:    float64(4),
			asrcontract.CommonParameterChunkingMode:         string(ChunkingModeFixed),
			asrcontract.CommonParameterChunkingChunkSeconds: float64(30),
			asrcontract.CommonParameterBatchingBatchSize:    float64(1),
		},
	}
}

func floatPtr(value float64) *float64 { return &value }

func intPtr(value int) *int { return &value }
