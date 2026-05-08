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

func (r *StaticRegistry) Select(ctx context.Context, req SelectionRequest) (Provider, asrcontract.ModelCard, error) {
	providerID := strings.TrimSpace(req.ProviderID)
	modelID := strings.TrimSpace(req.ModelID)
	if providerID != "" {
		provider, ok := r.Provider(providerID)
		if !ok {
			return nil, asrcontract.ModelCard{}, fmt.Errorf("engine provider %q is not available", providerID)
		}
		if modelID == "" && len(req.Requires) == 0 {
			return provider, asrcontract.ModelCard{}, nil
		}
		model, err := selectModelForProvider(ctx, provider, modelID, req.Requires)
		if err != nil {
			return nil, asrcontract.ModelCard{}, err
		}
		return provider, model, nil
	}
	if modelID != "" || len(req.Requires) > 0 {
		return r.selectByCapability(ctx, modelID, req.Requires)
	}
	provider := r.DefaultProvider()
	return provider, asrcontract.ModelCard{}, nil
}

func (r *StaticRegistry) SelectModel(ctx context.Context, providerID string, modelID string, required ...asrcontract.Capability) (asrcontract.ModelCard, error) {
	provider, model, err := r.Select(ctx, SelectionRequest{
		ProviderID: providerID,
		ModelID:    modelID,
		Requires:   required,
	})
	if err != nil {
		return asrcontract.ModelCard{}, err
	}
	if provider == nil {
		return asrcontract.ModelCard{}, fmt.Errorf("selected engine provider is not available")
	}
	if strings.TrimSpace(model.ID) == "" {
		return asrcontract.ModelCard{}, fmt.Errorf("engine provider %q did not select a model", provider.ID())
	}
	return model, nil
}

func (r *StaticRegistry) selectByCapability(ctx context.Context, modelID string, requires []asrcontract.Capability) (Provider, asrcontract.ModelCard, error) {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		if id == r.defaultID {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	ordered := append([]string{r.defaultID}, ids...)
	for _, id := range ordered {
		provider := r.providers[id]
		if !providerSelectable(ctx, provider) {
			continue
		}
		model, err := selectModelForProvider(ctx, provider, modelID, requires)
		if err == nil {
			return provider, model, nil
		}
	}
	return nil, asrcontract.ModelCard{}, fmt.Errorf("no engine provider supports requested model or capabilities")
}

func selectModelForProvider(ctx context.Context, provider Provider, modelID string, requires []asrcontract.Capability) (asrcontract.ModelCard, error) {
	models, err := provider.Models(ctx)
	if err != nil {
		return asrcontract.ModelCard{}, fmt.Errorf("engine provider %q models: %w", provider.ID(), err)
	}
	if modelID == "" {
		for _, model := range models {
			if !model.Default || !model.Supports(requires...) {
				continue
			}
			return modelWithProvider(model, provider.ID()), nil
		}
	}
	for _, model := range models {
		if modelID != "" && model.ID != modelID {
			continue
		}
		if !model.Supports(requires...) {
			continue
		}
		return modelWithProvider(model, provider.ID()), nil
	}
	if modelID != "" {
		return asrcontract.ModelCard{}, fmt.Errorf("engine provider %q does not support model %q", provider.ID(), modelID)
	}
	return asrcontract.ModelCard{}, fmt.Errorf("engine provider %q does not support requested capabilities", provider.ID())
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

func modelWithProvider(model asrcontract.ModelCard, providerID string) asrcontract.ModelCard {
	if strings.TrimSpace(model.Provider) == "" {
		model.Provider = providerID
	}
	return model
}
