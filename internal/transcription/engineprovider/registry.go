package engineprovider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"scriberr/internal/transcription/asrcontract"
)

type StaticRegistry struct {
	defaultID string
	providers map[string]Provider
}

func NewRegistry(defaultID string, providers ...Provider) (*StaticRegistry, error) {
	registry := &StaticRegistry{
		defaultID: strings.TrimSpace(defaultID),
		providers: make(map[string]Provider, len(providers)),
	}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		id := strings.TrimSpace(provider.ID())
		if id == "" {
			return nil, fmt.Errorf("engine provider id cannot be empty")
		}
		if _, exists := registry.providers[id]; exists {
			return nil, fmt.Errorf("duplicate engine provider id %q", id)
		}
		registry.providers[id] = provider
	}
	if registry.defaultID == "" {
		registry.defaultID = DefaultProviderID
	}
	if _, ok := registry.providers[registry.defaultID]; !ok {
		return nil, fmt.Errorf("default engine provider %q is not registered", registry.defaultID)
	}
	return registry, nil
}

func (r *StaticRegistry) DefaultProvider() Provider {
	return r.providers[r.defaultID]
}

func (r *StaticRegistry) Provider(id string) (Provider, bool) {
	provider, ok := r.providers[strings.TrimSpace(id)]
	return provider, ok
}

func (r *StaticRegistry) Providers() []Provider {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]Provider, 0, len(ids))
	for _, id := range ids {
		out = append(out, r.providers[id])
	}
	return out
}

func (r *StaticRegistry) Models(ctx context.Context) ([]asrcontract.ModelCard, error) {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var out []asrcontract.ModelCard
	for _, id := range ids {
		models, err := r.providers[id].Models(ctx)
		if err != nil {
			return nil, fmt.Errorf("engine provider %q models: %w", id, err)
		}
		out = append(out, models...)
	}
	return out, nil
}

func (r *StaticRegistry) Capabilities(ctx context.Context) ([]ModelCapability, error) {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var out []ModelCapability
	for _, id := range ids {
		capabilities, err := capabilitiesForProvider(ctx, r.providers[id])
		if err != nil {
			return nil, fmt.Errorf("engine provider %q capabilities: %w", id, err)
		}
		out = append(out, capabilities...)
	}
	return out, nil
}

func (r *StaticRegistry) Select(ctx context.Context, req SelectionRequest) (Provider, *ModelCapability, error) {
	providerID := strings.TrimSpace(req.ProviderID)
	modelID := strings.TrimSpace(req.ModelID)
	if providerID != "" {
		provider, ok := r.Provider(providerID)
		if !ok {
			return nil, nil, fmt.Errorf("engine provider %q is not available", providerID)
		}
		if modelID == "" && len(req.Requires) == 0 {
			return provider, nil, nil
		}
		capability, err := selectCapabilityForProvider(ctx, provider, modelID, req.Requires)
		if err != nil {
			return nil, nil, err
		}
		return provider, capability, nil
	}
	if modelID != "" || len(req.Requires) > 0 {
		return r.selectByCapability(ctx, modelID, req.Requires)
	}
	provider := r.DefaultProvider()
	return provider, nil, nil
}

func (r *StaticRegistry) selectByCapability(ctx context.Context, modelID string, requires []string) (Provider, *ModelCapability, error) {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		provider := r.providers[id]
		if !providerSelectable(ctx, provider) {
			continue
		}
		capability, err := selectCapabilityForProvider(ctx, provider, modelID, requires)
		if err == nil {
			return provider, capability, nil
		}
	}
	return nil, nil, fmt.Errorf("no engine provider supports requested model or capabilities")
}

func selectCapabilityForProvider(ctx context.Context, provider Provider, modelID string, requires []string) (*ModelCapability, error) {
	capabilities, err := capabilitiesForProvider(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("engine provider %q capabilities: %w", provider.ID(), err)
	}
	if modelID == "" {
		for i := range capabilities {
			capability := capabilities[i]
			if !capability.Default || !capabilitySupportsAll(capability, requires) {
				continue
			}
			return &capabilities[i], nil
		}
	}
	for i := range capabilities {
		capability := capabilities[i]
		if modelID != "" && capability.ID != modelID {
			continue
		}
		if !capabilitySupportsAll(capability, requires) {
			continue
		}
		return &capabilities[i], nil
	}
	if modelID != "" {
		return nil, fmt.Errorf("engine provider %q does not support model %q", provider.ID(), modelID)
	}
	return nil, fmt.Errorf("engine provider %q does not support requested capabilities", provider.ID())
}

func capabilitiesForProvider(ctx context.Context, provider Provider) ([]ModelCapability, error) {
	models, err := provider.Models(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ModelCapability, 0, len(models))
	for _, model := range models {
		out = append(out, ModelCapability{
			ID:           model.ID,
			Name:         model.DisplayName,
			Provider:     defaultString(model.Provider, provider.ID()),
			Installed:    model.Installed,
			Default:      model.Default,
			Capabilities: capabilityNames(model.Capabilities),
		})
	}
	return out, nil
}

func capabilityNames(capabilities asrcontract.Capabilities) []string {
	names := make([]string, 0, 8)
	if capabilities.Transcription {
		names = append(names, string(asrcontract.CapabilityTranscription))
	}
	if capabilities.Diarization {
		names = append(names, string(asrcontract.CapabilityDiarization))
	}
	if capabilities.SpeakerIdentification {
		names = append(names, string(asrcontract.CapabilitySpeakerIdentification))
	}
	if capabilities.Translation {
		names = append(names, string(asrcontract.CapabilityTranslation))
	}
	if capabilities.WordTimestamps {
		names = append(names, string(asrcontract.CapabilityWordTimestamps))
	}
	if capabilities.SegmentTimestamps {
		names = append(names, string(asrcontract.CapabilitySegmentTimestamps))
	}
	if capabilities.TokenTimestamps {
		names = append(names, string(asrcontract.CapabilityTokenTimestamps))
	}
	if capabilities.Streaming {
		names = append(names, string(asrcontract.CapabilityStreaming))
	}
	for key, enabled := range capabilities.Extensions {
		if enabled {
			names = append(names, strings.TrimSpace(key))
		}
	}
	return names
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

func providerSelectable(ctx context.Context, provider Provider) bool {
	status, err := provider.Status(ctx)
	if err != nil || status == nil {
		return true
	}
	switch status.State {
	case asrcontract.ProviderStateBusy, asrcontract.ProviderStateUnhealthy, asrcontract.ProviderStateStopping:
		return false
	default:
		return true
	}
}

func capabilitySupportsAll(capability ModelCapability, requires []string) bool {
	if len(requires) == 0 {
		return true
	}
	available := make(map[string]struct{}, len(capability.Capabilities))
	for _, item := range capability.Capabilities {
		available[strings.TrimSpace(item)] = struct{}{}
	}
	for _, required := range requires {
		if _, ok := available[strings.TrimSpace(required)]; !ok {
			return false
		}
	}
	return true
}
