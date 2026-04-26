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
