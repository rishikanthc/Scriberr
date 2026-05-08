package engineprovider

import (
	"context"
	"errors"
	"testing"

	"scriberr/internal/transcription/asrcontract"
)

type stubProvider struct {
	id     string
	models []asrcontract.ModelCard
	status asrcontract.ProviderStatus
	err    error
}

func (p stubProvider) ID() string { return p.id }
func (p stubProvider) Inspect(context.Context) (*asrcontract.ProviderInfo, error) {
	return &asrcontract.ProviderInfo{ContractVersion: asrcontract.ContractVersionV1}, nil
}
func (p stubProvider) Models(ctx context.Context) ([]asrcontract.ModelCard, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.models, nil
}
func (p stubProvider) Status(context.Context) (*asrcontract.ProviderStatus, error) {
	return &p.status, nil
}
func (p stubProvider) LoadModel(context.Context, asrcontract.LoadModelRequest) error     { return nil }
func (p stubProvider) UnloadModel(context.Context, asrcontract.UnloadModelRequest) error { return nil }
func (p stubProvider) LoadedModels(context.Context) ([]asrcontract.LoadedModel, error) {
	return p.status.LoadedModels, nil
}
func (p stubProvider) ExecuteTask(context.Context, TaskRequest) (*TaskResult, error) {
	return nil, nil
}
func (p stubProvider) Close() error { return nil }

func TestRegistryReturnsDefaultProvider(t *testing.T) {
	local := stubProvider{
		id:     "local",
		models: []asrcontract.ModelCard{testModelCard("local", "whisper-base", false, asrcontract.CapabilityTranscription)},
	}
	other := stubProvider{
		id:     "other",
		models: []asrcontract.ModelCard{testModelCard("other", "remote-model", false, asrcontract.CapabilityTranscription)},
	}

	registry, err := NewRegistry("local", other, local)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}
	if registry.DefaultProvider().ID() != "local" {
		t.Fatalf("default provider = %q, want local", registry.DefaultProvider().ID())
	}
	if _, ok := registry.Provider("other"); !ok {
		t.Fatalf("Provider(other) not found")
	}
}

func TestRegistryAggregatesModelCards(t *testing.T) {
	local := stubProvider{
		id: "local",
		models: []asrcontract.ModelCard{{
			ID:       "whisper-base",
			Provider: "local",
			Capabilities: asrcontract.Capabilities{
				Transcription: true,
			},
		}},
	}
	remote := stubProvider{
		id: "remote",
		models: []asrcontract.ModelCard{{
			ID:       "remote-diarizer",
			Provider: "remote",
			Capabilities: asrcontract.Capabilities{
				Diarization: true,
			},
		}},
	}
	registry, err := NewRegistry("local", remote, local)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	models, err := registry.Models(context.Background())
	if err != nil {
		t.Fatalf("Models returned error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("models length = %d, want 2", len(models))
	}
	if models[0].Provider != "local" || models[1].Provider != "remote" {
		t.Fatalf("models not sorted by provider id: %#v", models)
	}
}

func TestRegistryRejectsInvalidProviderSet(t *testing.T) {
	if _, err := NewRegistry("missing", stubProvider{id: "local"}); err == nil {
		t.Fatalf("NewRegistry returned nil error for missing default")
	}
	if _, err := NewRegistry("local", stubProvider{id: "local"}, stubProvider{id: "local"}); err == nil {
		t.Fatalf("NewRegistry returned nil error for duplicate providers")
	}
}

func TestRegistryWrapsModelErrors(t *testing.T) {
	registry, err := NewRegistry("local", stubProvider{id: "local", err: errors.New("boom")})
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}
	_, err = registry.Models(context.Background())
	if err == nil {
		t.Fatalf("Models returned nil error")
	}
}

func TestRegistrySelectsExplicitProviderAndModel(t *testing.T) {
	local := stubProvider{id: "local", models: []asrcontract.ModelCard{testModelCard("local", "whisper-base", false, asrcontract.CapabilityTranscription)}}
	remote := stubProvider{id: "remote", models: []asrcontract.ModelCard{testModelCard("remote", "remote-large", false, asrcontract.CapabilityTranscription, asrcontract.CapabilityWordTimestamps)}}
	registry, err := NewRegistry("local", local, remote)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	selected, model, err := registry.Select(context.Background(), SelectionRequest{
		ProviderID: "remote",
		ModelID:    "remote-large",
		Requires:   []asrcontract.Capability{asrcontract.CapabilityWordTimestamps},
	})

	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.ID() != "remote" {
		t.Fatalf("provider = %q, want remote", selected.ID())
	}
	if model.ID != "remote-large" {
		t.Fatalf("model = %#v, want remote-large", model)
	}
}

func TestRegistrySelectsCapabilityFallbackDeterministically(t *testing.T) {
	local := stubProvider{id: "local", models: []asrcontract.ModelCard{testModelCard("local", "local-basic", false, asrcontract.CapabilityTranscription)}}
	remoteB := stubProvider{id: "remote-b", models: []asrcontract.ModelCard{testModelCard("remote-b", "remote-b-model", false, asrcontract.CapabilityDiarization)}}
	remoteA := stubProvider{id: "remote-a", models: []asrcontract.ModelCard{testModelCard("remote-a", "remote-a-model", false, asrcontract.CapabilityDiarization)}}
	registry, err := NewRegistry("local", remoteB, local, remoteA)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	selected, model, err := registry.Select(context.Background(), SelectionRequest{Requires: []asrcontract.Capability{asrcontract.CapabilityDiarization}})

	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.ID() != "remote-a" {
		t.Fatalf("provider = %q, want deterministic remote-a", selected.ID())
	}
	if model.ID != "remote-a-model" {
		t.Fatalf("model = %#v, want remote-a-model", model)
	}
}

func TestRegistrySelectsDefaultProviderBeforeFallback(t *testing.T) {
	defaultProvider := stubProvider{id: "z-default", models: []asrcontract.ModelCard{testModelCard("z-default", "default-diarizer", true, asrcontract.CapabilityDiarization)}}
	earlierProvider := stubProvider{id: "a-earlier", models: []asrcontract.ModelCard{testModelCard("a-earlier", "earlier-diarizer", true, asrcontract.CapabilityDiarization)}}
	registry, err := NewRegistry("z-default", earlierProvider, defaultProvider)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	selected, model, err := registry.Select(context.Background(), SelectionRequest{Requires: []asrcontract.Capability{asrcontract.CapabilityDiarization}})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.ID() != "z-default" {
		t.Fatalf("provider = %q, want configured default", selected.ID())
	}
	if model.ID != "default-diarizer" {
		t.Fatalf("model = %#v, want default-diarizer", model)
	}
}

func TestRegistrySelectModelReturnsDescriptorCard(t *testing.T) {
	provider := stubProvider{id: "local", models: []asrcontract.ModelCard{{
		ID:          "parakeet",
		DisplayName: "Parakeet",
		Default:     true,
		Capabilities: asrcontract.Capabilities{
			Transcription: true,
		},
		ParameterSchema: asrcontract.ParameterSchema{{
			Key:   asrcontract.CommonParameterRuntimeNumThreads,
			Label: "Threads",
			Type:  asrcontract.ParameterTypeInteger,
			Scope: asrcontract.ParameterScopeRuntime,
		}},
	}}}
	registry, err := NewRegistry("local", provider)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	card, err := registry.SelectModel(context.Background(), "", "", asrcontract.CapabilityTranscription)
	if err != nil {
		t.Fatalf("SelectModel returned error: %v", err)
	}
	if card.ID != "parakeet" || card.Provider != "local" || len(card.ParameterSchema) != 1 {
		t.Fatalf("unexpected model card: %#v", card)
	}
}

func TestRegistrySelectReportsUnavailableProviderOrCapability(t *testing.T) {
	registry, err := NewRegistry("local", stubProvider{id: "local", models: []asrcontract.ModelCard{testModelCard("local", "whisper-base", false, asrcontract.CapabilityTranscription)}})
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	if _, _, err := registry.Select(context.Background(), SelectionRequest{ProviderID: "missing"}); err == nil {
		t.Fatalf("Select returned nil error for missing provider")
	}
	if _, _, err := registry.Select(context.Background(), SelectionRequest{ProviderID: "local", ModelID: "missing-model"}); err == nil {
		t.Fatalf("Select returned nil error for missing model")
	}
	if _, _, err := registry.Select(context.Background(), SelectionRequest{Requires: []asrcontract.Capability{asrcontract.CapabilityStreaming}}); err == nil {
		t.Fatalf("Select returned nil error for missing capability")
	}
}

func TestRegistrySkipsBusyProviderForFallbackSelection(t *testing.T) {
	busy := stubProvider{
		id:     "busy",
		models: []asrcontract.ModelCard{testModelCard("busy", "busy-diarizer", false, asrcontract.CapabilityDiarization)},
		status: asrcontract.ProviderStatus{State: asrcontract.ProviderStateBusy},
	}
	idle := stubProvider{
		id:     "idle",
		models: []asrcontract.ModelCard{testModelCard("idle", "idle-diarizer", false, asrcontract.CapabilityDiarization)},
		status: asrcontract.ProviderStatus{State: asrcontract.ProviderStateIdle},
	}
	registry, err := NewRegistry("idle", busy, idle)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	selected, model, err := registry.Select(context.Background(), SelectionRequest{Requires: []asrcontract.Capability{asrcontract.CapabilityDiarization}})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.ID() != "idle" {
		t.Fatalf("provider = %q, want idle", selected.ID())
	}
	if model.ID != "idle-diarizer" {
		t.Fatalf("model = %#v, want idle-diarizer", model)
	}
}

func testModelCard(provider, id string, isDefault bool, capabilities ...asrcontract.Capability) asrcontract.ModelCard {
	return asrcontract.ModelCard{
		ID:           id,
		DisplayName:  id,
		Provider:     provider,
		Installed:    true,
		Default:      isDefault,
		Capabilities: testCapabilities(capabilities...),
	}
}

func testCapabilities(capabilities ...asrcontract.Capability) asrcontract.Capabilities {
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
		case asrcontract.CapabilitySegmentTimestamps:
			out.SegmentTimestamps = true
		case asrcontract.CapabilityStreaming:
			out.Streaming = true
		default:
			out.Extensions[string(capability)] = true
		}
	}
	if len(out.Extensions) == 0 {
		out.Extensions = nil
	}
	return out
}
