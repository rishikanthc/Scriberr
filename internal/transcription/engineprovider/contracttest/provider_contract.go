package contracttest

import (
	"context"
	"strings"
	"testing"
	"time"

	"scriberr/internal/transcription/asrcontract"
	"scriberr/internal/transcription/engineprovider"
)

type Options struct {
	RequiredModel        string
	RequiredCapabilities []asrcontract.Capability
}

func RunProviderContract(t testing.TB, provider engineprovider.Provider, opts Options) {
	t.Helper()
	if provider == nil {
		t.Fatal("provider is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := provider.Inspect(ctx)
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if info == nil {
		t.Fatal("Inspect returned nil info")
	}
	if strings.TrimSpace(info.ContractVersion) == "" {
		t.Fatal("provider contract version is required")
	}
	if strings.TrimSpace(info.Provider.ID) == "" {
		t.Fatal("provider identity id is required")
	}
	if info.Provider.ID != provider.ID() {
		t.Fatalf("provider identity id = %q, want %q", info.Provider.ID, provider.ID())
	}
	if info.AudioInput.RequiredSampleRate != 0 && info.AudioInput.RequiredSampleRate != 16000 {
		t.Fatalf("audio sample rate = %d, want 16000 or unset", info.AudioInput.RequiredSampleRate)
	}
	if info.AudioInput.RequiredChannels != 0 && info.AudioInput.RequiredChannels != 1 {
		t.Fatalf("audio channels = %d, want mono or unset", info.AudioInput.RequiredChannels)
	}

	models, err := provider.Models(ctx)
	if err != nil {
		t.Fatalf("Models returned error: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("provider must expose at least one model card")
	}
	var matchedRequiredModel bool
	for _, model := range models {
		if strings.TrimSpace(model.ID) == "" {
			t.Fatal("model id is required")
		}
		if strings.TrimSpace(model.Provider) == "" {
			t.Fatalf("model %q provider is required", model.ID)
		}
		if !hasAnyCapability(model.Capabilities) {
			t.Fatalf("model %q must declare at least one capability", model.ID)
		}
		if opts.RequiredModel != "" && model.ID == opts.RequiredModel {
			matchedRequiredModel = true
			for _, capability := range opts.RequiredCapabilities {
				if !model.Supports(capability) {
					t.Fatalf("model %q does not support required capability %q", model.ID, capability)
				}
			}
		}
	}
	if opts.RequiredModel != "" && !matchedRequiredModel {
		t.Fatalf("required model %q was not advertised", opts.RequiredModel)
	}

	capabilities, err := provider.Capabilities(ctx)
	if err != nil {
		t.Fatalf("Capabilities returned error: %v", err)
	}
	if len(capabilities) == 0 {
		t.Fatal("provider must expose at least one selectable capability")
	}
	for _, capability := range capabilities {
		if strings.TrimSpace(capability.ID) == "" {
			t.Fatal("capability model id is required")
		}
		if strings.TrimSpace(capability.Provider) == "" {
			t.Fatalf("capability %q provider is required", capability.ID)
		}
		if len(capability.Capabilities) == 0 {
			t.Fatalf("capability %q must list supported features", capability.ID)
		}
	}

	status, err := provider.Status(ctx)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status == nil {
		t.Fatal("Status returned nil status")
	}
	if strings.TrimSpace(string(status.State)) == "" {
		t.Fatal("provider status state is required")
	}
	if status.Capacity.MaxConcurrentJobs < 0 || status.Capacity.AvailableSlots < 0 {
		t.Fatalf("provider capacity cannot be negative: %+v", status.Capacity)
	}

	if _, err := provider.LoadedModels(ctx); err != nil {
		t.Fatalf("LoadedModels returned error: %v", err)
	}
}

func hasAnyCapability(capabilities asrcontract.Capabilities) bool {
	return capabilities.Transcription ||
		capabilities.Diarization ||
		capabilities.SpeakerIdentification ||
		capabilities.Translation ||
		capabilities.WordTimestamps ||
		capabilities.SegmentTimestamps ||
		capabilities.TokenTimestamps ||
		capabilities.Streaming ||
		capabilities.CustomVocabulary ||
		capabilities.InitialPrompt ||
		capabilities.LanguageDetection ||
		capabilities.SpeakerEmbeddings ||
		len(capabilities.Extensions) > 0
}
