package profile

import (
	"context"
	"strings"

	"scriberr/internal/transcription/asrcontract"
)

type ProviderModelRegistry interface {
	Models(ctx context.Context) ([]asrcontract.ModelCard, error)
}

type ProviderModelCatalog struct {
	registry ProviderModelRegistry
}

func NewProviderModelCatalog(registry ProviderModelRegistry) *ProviderModelCatalog {
	return &ProviderModelCatalog{registry: registry}
}

func (c *ProviderModelCatalog) ResolveTranscriptionModel(ctx context.Context, provider string, model string) (ModelInfo, error) {
	return c.ResolveModel(ctx, provider, model, asrcontract.CapabilityTranscription)
}

func (c *ProviderModelCatalog) ResolveModel(ctx context.Context, provider string, model string, capability asrcontract.Capability) (ModelInfo, error) {
	if c == nil {
		return ModelInfo{}, ErrInvalidModel
	}
	if c.registry == nil {
		return ModelInfo{}, ErrInvalidModel
	}
	provider = strings.TrimSpace(provider)
	model = strings.TrimSpace(model)
	models, err := c.registry.Models(ctx)
	if err != nil {
		return ModelInfo{}, err
	}
	for _, card := range models {
		if provider != "" && card.Provider != provider {
			continue
		}
		if model != "" && card.ID != model {
			continue
		}
		if model == "" && !card.Default {
			continue
		}
		if !card.Supports(capability) {
			continue
		}
		return ModelInfo{
			ID:              card.ID,
			ModelType:       card.ModelType,
			Capabilities:    card.Capabilities,
			Default:         card.Default,
			ParameterSchema: card.ParameterSchema,
		}, nil
	}
	return ModelInfo{}, ErrInvalidModel
}
