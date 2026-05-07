package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"scriberr/internal/transcription/asrcontract"
	"scriberr/internal/transcription/engineprovider"

	"github.com/gin-gonic/gin"
)

const asrProviderAdminTimeout = 15 * time.Second

func (h *Handler) listASRProviders(c *gin.Context) {
	if h.modelRegistry == nil {
		c.JSON(http.StatusOK, gin.H{"items": []gin.H{}})
		return
	}
	providers := h.modelRegistry.Providers()
	items := make([]gin.H, 0, len(providers))
	for _, provider := range providers {
		summary, err := h.asrProviderSummary(c.Request.Context(), provider)
		if err != nil {
			writeASRProviderError(c, err)
			return
		}
		items = append(items, summary)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) getASRProvider(c *gin.Context) {
	provider, ok := h.asrProviderByID(c, c.Param("provider_id"))
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), asrProviderAdminTimeout)
	defer cancel()

	info, err := provider.Inspect(ctx)
	if err != nil {
		writeASRProviderError(c, err)
		return
	}
	status, err := provider.Status(ctx)
	if err != nil {
		writeASRProviderError(c, err)
		return
	}
	models, err := provider.Models(ctx)
	if err != nil {
		writeASRProviderError(c, err)
		return
	}
	loaded, err := provider.LoadedModels(ctx)
	if err != nil {
		writeASRProviderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":            provider.ID(),
		"info":          sanitizeProviderInfo(info),
		"status":        sanitizeProviderStatus(status),
		"models":        sanitizeModelCards(models),
		"loaded_models": sanitizeLoadedModels(loaded),
	})
}

func (h *Handler) loadASRProviderModel(c *gin.Context) {
	provider, ok := h.asrProviderByID(c, c.Param("provider_id"))
	if !ok {
		return
	}
	var req loadASRProviderModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body", nil)
		return
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "model is required", stringPtr("model"))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), asrProviderAdminTimeout)
	defer cancel()
	err := provider.LoadModel(ctx, asrcontract.LoadModelRequest{
		Model:      model,
		Operation:  asrcontract.Operation(strings.TrimSpace(req.Operation)),
		LoadPolicy: asrcontract.LoadPolicy(strings.TrimSpace(req.LoadPolicy)),
		Options:    req.Options,
	})
	if err != nil {
		writeASRProviderError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"provider": provider.ID(), "model": model, "status": "loading"})
}

func (h *Handler) unloadASRProviderModel(c *gin.Context) {
	provider, ok := h.asrProviderByID(c, c.Param("provider_id"))
	if !ok {
		return
	}
	var req unloadASRProviderModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body", nil)
		return
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		writeError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "model is required", stringPtr("model"))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), asrProviderAdminTimeout)
	defer cancel()
	err := provider.UnloadModel(ctx, asrcontract.UnloadModelRequest{
		Model:   model,
		Force:   req.Force,
		Options: req.Options,
	})
	if err != nil {
		writeASRProviderError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"provider": provider.ID(), "model": model, "status": "unloading"})
}

func (h *Handler) asrProviderByID(c *gin.Context, id string) (engineprovider.Provider, bool) {
	if h.modelRegistry == nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "ASR provider not found", nil)
		return nil, false
	}
	provider, ok := h.modelRegistry.Provider(strings.TrimSpace(id))
	if !ok {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "ASR provider not found", nil)
		return nil, false
	}
	return provider, true
}

func (h *Handler) asrProviderSummary(parent context.Context, provider engineprovider.Provider) (gin.H, error) {
	ctx, cancel := context.WithTimeout(parent, asrProviderAdminTimeout)
	defer cancel()
	info, err := provider.Inspect(ctx)
	if err != nil {
		return nil, err
	}
	status, err := provider.Status(ctx)
	if err != nil {
		return nil, err
	}
	loaded, err := provider.LoadedModels(ctx)
	if err != nil {
		return nil, err
	}
	return gin.H{
		"id":            provider.ID(),
		"info":          sanitizeProviderInfo(info),
		"status":        sanitizeProviderStatus(status),
		"loaded_models": sanitizeLoadedModels(loaded),
	}, nil
}

func writeASRProviderError(c *gin.Context, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		writeError(c, http.StatusGatewayTimeout, "PROVIDER_TIMEOUT", "ASR provider timed out", nil)
		return
	}
	var providerErr *asrcontract.ProviderError
	if errors.As(err, &providerErr) {
		status := http.StatusBadGateway
		switch providerErr.Code {
		case asrcontract.CodeInvalidRequest:
			status = http.StatusUnprocessableEntity
		case asrcontract.CodeUnsupportedModel, asrcontract.CodeModelNotInstalled:
			status = http.StatusNotFound
		case asrcontract.CodeProviderBusy:
			status = http.StatusConflict
		case asrcontract.CodeProviderUnhealthy, asrcontract.CodeInsufficientResources:
			status = http.StatusServiceUnavailable
		case asrcontract.CodeTimeout:
			status = http.StatusGatewayTimeout
		}
		message := sanitizePublicText(providerErr.Message)
		if strings.TrimSpace(message) == "" {
			message = "ASR provider request failed"
		}
		writeError(c, status, string(providerErr.Code), message, nil)
		return
	}
	writeError(c, http.StatusBadGateway, "PROVIDER_ERROR", "ASR provider request failed", nil)
}

func sanitizeProviderInfo(info *asrcontract.ProviderInfo) any {
	if info == nil {
		return nil
	}
	return gin.H{
		"contract_version": info.ContractVersion,
		"provider":         info.Provider,
		"runtime":          info.Runtime,
		"audio_input":      info.AudioInput,
	}
}

func sanitizeProviderStatus(status *asrcontract.ProviderStatus) any {
	if status == nil {
		return nil
	}
	return gin.H{
		"state":         status.State,
		"active_job":    sanitizeActiveJob(status.ActiveJob),
		"loaded_models": sanitizeLoadedModels(status.LoadedModels),
		"capacity":      status.Capacity,
	}
}

func sanitizeActiveJob(job *asrcontract.ActiveJob) any {
	if job == nil {
		return nil
	}
	return gin.H{
		"id":        sanitizePublicText(job.ID),
		"operation": job.Operation,
		"model":     sanitizePublicText(job.Model),
		"stage":     job.Stage,
		"progress":  job.Progress,
	}
}

func sanitizeLoadedModels(models []asrcontract.LoadedModel) []gin.H {
	out := make([]gin.H, 0, len(models))
	for _, model := range models {
		out = append(out, gin.H{
			"id":              sanitizePublicText(model.ID),
			"resource_kind":   sanitizePublicText(model.ResourceKind),
			"resource_role":   sanitizePublicText(model.ResourceRole),
			"runtime_backend": sanitizePublicText(model.RuntimeBackend),
			"threads":         model.Threads,
			"reload_key":      sanitizePublicText(model.ReloadKey),
			"loaded_at":       model.LoadedAt,
			"memory_mb":       model.MemoryMB,
		})
	}
	return out
}

func sanitizeModelCards(models []asrcontract.ModelCard) []gin.H {
	out := make([]gin.H, 0, len(models))
	for _, model := range models {
		out = append(out, gin.H{
			"id":                    sanitizePublicText(model.ID),
			"display_name":          sanitizePublicText(model.DisplayName),
			"provider":              sanitizePublicText(model.Provider),
			"family":                sanitizePublicText(model.Family),
			"version":               sanitizePublicText(model.Version),
			"installed":             model.Installed,
			"loaded":                model.Loaded,
			"default":               model.Default,
			"tasks":                 model.Tasks,
			"languages":             model.Languages,
			"capabilities":          model.Capabilities,
			"limits":                model.Limits,
			"resource_requirements": model.ResourceRequirements,
			"license":               sanitizePublicText(model.License),
		})
	}
	return out
}
