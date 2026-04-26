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
