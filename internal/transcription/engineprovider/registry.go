package engineprovider

import (
	"context"
	"fmt"
	"sort"
	"strings"
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

func (r *StaticRegistry) Capabilities(ctx context.Context) ([]ModelCapability, error) {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var out []ModelCapability
	for _, id := range ids {
		capabilities, err := r.providers[id].Capabilities(ctx)
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
		capability, err := selectCapabilityForProvider(ctx, provider, modelID, requires)
		if err == nil {
			return provider, capability, nil
		}
	}
	return nil, nil, fmt.Errorf("no engine provider supports requested model or capabilities")
}

func selectCapabilityForProvider(ctx context.Context, provider Provider, modelID string, requires []string) (*ModelCapability, error) {
	capabilities, err := provider.Capabilities(ctx)
	if err != nil {
		return nil, fmt.Errorf("engine provider %q capabilities: %w", provider.ID(), err)
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
