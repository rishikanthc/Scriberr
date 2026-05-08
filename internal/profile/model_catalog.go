package profile

import (
	"context"
	"strings"

	"scriberr/internal/transcription/asrcontract"
)

type ProviderModelRegistry interface {
	SelectModel(ctx context.Context, providerID string, modelID string, required ...asrcontract.Capability) (asrcontract.ModelCard, error)
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
	card, err := c.registry.SelectModel(ctx, provider, model, capability)
	if err != nil {
		return ModelInfo{}, ErrInvalidModel
	}
	return ModelInfo{
		ID:              card.ID,
		ModelType:       card.ModelType,
		Capabilities:    card.Capabilities,
		Default:         card.Default,
		ParameterSchema: card.ParameterSchema,
	}, nil
}
