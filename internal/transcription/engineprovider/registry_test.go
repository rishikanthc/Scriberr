package engineprovider

import (
	"context"
	"errors"
	"testing"

	"scriberr/internal/transcription/asrcontract"
)

type stubProvider struct {
	id           string
	capabilities []ModelCapability
	models       []asrcontract.ModelCard
	status       asrcontract.ProviderStatus
	err          error
}

func (p stubProvider) ID() string { return p.id }
func (p stubProvider) Inspect(context.Context) (*asrcontract.ProviderInfo, error) {
	return &asrcontract.ProviderInfo{ContractVersion: asrcontract.ContractVersionV1}, nil
}
func (p stubProvider) Models(ctx context.Context) ([]asrcontract.ModelCard, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.models == nil {
		return modelCardsFromCapabilities(p.capabilities), nil
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
func (p stubProvider) Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error) {
	return nil, nil
}
func (p stubProvider) Diarize(ctx context.Context, req DiarizationRequest) (*DiarizationResult, error) {
	return nil, nil
}
func (p stubProvider) IdentifySpeakers(context.Context, asrcontract.SpeakerIDRequest) (*asrcontract.SpeakerIDResult, error) {
	return nil, asrcontract.NewProviderError(asrcontract.CodeUnsupportedOperation, "speaker identification is not supported", false)
}
func (p stubProvider) Close() error { return nil }

func modelCardsFromCapabilities(capabilities []ModelCapability) []asrcontract.ModelCard {
	out := make([]asrcontract.ModelCard, 0, len(capabilities))
	for _, capability := range capabilities {
		out = append(out, asrcontract.ModelCard{
			ID:           capability.ID,
			DisplayName:  capability.Name,
			Provider:     capability.Provider,
			Installed:    capability.Installed,
			Default:      capability.Default,
			Capabilities: asrCapabilitiesFromStrings(capability.Capabilities),
		})
	}
	return out
}

func asrCapabilitiesFromStrings(names []string) asrcontract.Capabilities {
	out := asrcontract.Capabilities{Extensions: map[string]bool{}}
	for _, name := range names {
		switch name {
		case string(asrcontract.CapabilityTranscription):
			out.Transcription = true
		case string(asrcontract.CapabilityDiarization):
			out.Diarization = true
		case string(asrcontract.CapabilitySpeakerIdentification):
			out.SpeakerIdentification = true
		case string(asrcontract.CapabilityWordTimestamps):
			out.WordTimestamps = true
		case string(asrcontract.CapabilitySegmentTimestamps):
			out.SegmentTimestamps = true
		case string(asrcontract.CapabilityStreaming):
			out.Streaming = true
		default:
			out.Extensions[name] = true
		}
	}
	if len(out.Extensions) == 0 {
		out.Extensions = nil
	}
	return out
}

func TestRegistryReturnsDefaultProviderAndAggregatesCapabilities(t *testing.T) {
	local := stubProvider{
		id: "local",
		capabilities: []ModelCapability{
			{ID: "whisper-base", Provider: "local"},
		},
	}
	other := stubProvider{
		id: "other",
		capabilities: []ModelCapability{
			{ID: "remote-model", Provider: "other"},
		},
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

	capabilities, err := registry.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities returned error: %v", err)
	}
	if len(capabilities) != 2 {
		t.Fatalf("capabilities length = %d, want 2", len(capabilities))
	}
	if capabilities[0].Provider != "local" || capabilities[1].Provider != "other" {
		t.Fatalf("capabilities not sorted by provider id: %#v", capabilities)
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

func TestRegistryWrapsCapabilityErrors(t *testing.T) {
	registry, err := NewRegistry("local", stubProvider{id: "local", err: errors.New("boom")})
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}
	_, err = registry.Capabilities(context.Background())
	if err == nil {
		t.Fatalf("Capabilities returned nil error")
	}
}

func TestRegistrySelectsExplicitProviderAndModel(t *testing.T) {
	local := stubProvider{id: "local", capabilities: []ModelCapability{{ID: "whisper-base", Provider: "local", Capabilities: []string{"batch"}}}}
	remote := stubProvider{id: "remote", capabilities: []ModelCapability{{ID: "remote-large", Provider: "remote", Capabilities: []string{"batch", "word_timestamps"}}}}
	registry, err := NewRegistry("local", local, remote)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	selected, capability, err := registry.Select(context.Background(), SelectionRequest{
		ProviderID: "remote",
		ModelID:    "remote-large",
		Requires:   []string{"word_timestamps"},
	})

	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.ID() != "remote" {
		t.Fatalf("provider = %q, want remote", selected.ID())
	}
	if capability == nil || capability.ID != "remote-large" {
		t.Fatalf("capability = %#v, want remote-large", capability)
	}
}

func TestRegistrySelectsCapabilityFallbackDeterministically(t *testing.T) {
	local := stubProvider{id: "local", capabilities: []ModelCapability{{ID: "local-basic", Provider: "local", Capabilities: []string{"batch"}}}}
	remoteB := stubProvider{id: "remote-b", capabilities: []ModelCapability{{ID: "remote-b-model", Provider: "remote-b", Capabilities: []string{"batch", "diarization"}}}}
	remoteA := stubProvider{id: "remote-a", capabilities: []ModelCapability{{ID: "remote-a-model", Provider: "remote-a", Capabilities: []string{"batch", "diarization"}}}}
	registry, err := NewRegistry("local", remoteB, local, remoteA)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	selected, capability, err := registry.Select(context.Background(), SelectionRequest{Requires: []string{"diarization"}})

	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.ID() != "remote-a" {
		t.Fatalf("provider = %q, want deterministic remote-a", selected.ID())
	}
	if capability == nil || capability.ID != "remote-a-model" {
		t.Fatalf("capability = %#v, want remote-a-model", capability)
	}
}

func TestRegistrySelectReportsUnavailableProviderOrCapability(t *testing.T) {
	registry, err := NewRegistry("local", stubProvider{id: "local", capabilities: []ModelCapability{{ID: "whisper-base", Provider: "local", Capabilities: []string{"batch"}}}})
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	if _, _, err := registry.Select(context.Background(), SelectionRequest{ProviderID: "missing"}); err == nil {
		t.Fatalf("Select returned nil error for missing provider")
	}
	if _, _, err := registry.Select(context.Background(), SelectionRequest{ProviderID: "local", ModelID: "missing-model"}); err == nil {
		t.Fatalf("Select returned nil error for missing model")
	}
	if _, _, err := registry.Select(context.Background(), SelectionRequest{Requires: []string{"streaming"}}); err == nil {
		t.Fatalf("Select returned nil error for missing capability")
	}
}

func TestRegistrySkipsBusyProviderForFallbackSelection(t *testing.T) {
	busy := stubProvider{
		id:           "busy",
		capabilities: []ModelCapability{{ID: "busy-diarizer", Provider: "busy", Capabilities: []string{"diarization"}}},
		status:       asrcontract.ProviderStatus{State: asrcontract.ProviderStateBusy},
	}
	idle := stubProvider{
		id:           "idle",
		capabilities: []ModelCapability{{ID: "idle-diarizer", Provider: "idle", Capabilities: []string{"diarization"}}},
		status:       asrcontract.ProviderStatus{State: asrcontract.ProviderStateIdle},
	}
	registry, err := NewRegistry("idle", busy, idle)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	selected, capability, err := registry.Select(context.Background(), SelectionRequest{Requires: []string{"diarization"}})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.ID() != "idle" {
		t.Fatalf("provider = %q, want idle", selected.ID())
	}
	if capability == nil || capability.ID != "idle-diarizer" {
		t.Fatalf("capability = %#v, want idle-diarizer", capability)
	}
}
