package engineprovider

import (
	"context"
	"errors"
	"testing"
)

type stubProvider struct {
	id           string
	capabilities []ModelCapability
	err          error
}

func (p stubProvider) ID() string { return p.id }
func (p stubProvider) Capabilities(ctx context.Context) ([]ModelCapability, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.capabilities, nil
}
func (p stubProvider) Prepare(ctx context.Context) error { return nil }
func (p stubProvider) Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error) {
	return nil, nil
}
func (p stubProvider) Diarize(ctx context.Context, req DiarizationRequest) (*DiarizationResult, error) {
	return nil, nil
}
func (p stubProvider) Close() error { return nil }

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
