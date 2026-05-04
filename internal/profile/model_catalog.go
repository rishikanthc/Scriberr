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
	fallback ModelCatalog
}

func NewProviderModelCatalog(registry ProviderModelRegistry) *ProviderModelCatalog {
	return &ProviderModelCatalog{registry: registry, fallback: defaultModelCatalog()}
}

func (c *ProviderModelCatalog) ResolveTranscriptionModel(ctx context.Context, model string) (ModelInfo, error) {
	if c == nil {
		return defaultModelCatalog().ResolveTranscriptionModel(ctx, model)
	}
	if c.registry == nil {
		return c.fallback.ResolveTranscriptionModel(ctx, model)
	}
	model = strings.TrimSpace(model)
	models, err := c.registry.Models(ctx)
	if err != nil {
		return ModelInfo{}, err
	}
	for _, card := range models {
		if model != "" && card.ID != model {
			continue
		}
		if model == "" && !card.Default {
			continue
		}
		if !card.Supports(asrcontract.CapabilityTranscription) {
			continue
		}
		return ModelInfo{
			ID:           card.ID,
			Family:       card.Family,
			Capabilities: card.Capabilities,
			Default:      card.Default,
		}, nil
	}
	if model == "" {
		return c.fallback.ResolveTranscriptionModel(ctx, model)
	}
	return ModelInfo{}, ErrInvalidModel
}
