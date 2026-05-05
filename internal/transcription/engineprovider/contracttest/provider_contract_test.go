package contracttest

import (
	"context"
	"strings"
	"testing"

	"scriberr/internal/transcription/asrcontract"
	"scriberr/internal/transcription/engineprovider"
)

type contractStubProvider struct {
	models []asrcontract.ModelCard
}

func (p contractStubProvider) ID() string { return "contract-stub" }
func (p contractStubProvider) Inspect(context.Context) (*asrcontract.ProviderInfo, error) {
	return &asrcontract.ProviderInfo{
		ContractVersion: asrcontract.ContractVersionV1,
		Provider:        asrcontract.ProviderIdentity{ID: "contract-stub", Name: "Contract Stub"},
		AudioInput:      asrcontract.AudioInputSpec{RequiredSampleRate: 16000, RequiredChannels: 1},
	}, nil
}
func (p contractStubProvider) Models(context.Context) ([]asrcontract.ModelCard, error) {
	return p.models, nil
}
func (p contractStubProvider) Status(context.Context) (*asrcontract.ProviderStatus, error) {
	return &asrcontract.ProviderStatus{
		State:    asrcontract.ProviderStateIdle,
		Capacity: asrcontract.ProviderCapacity{MaxConcurrentJobs: 1, AvailableSlots: 1},
	}, nil
}
func (p contractStubProvider) LoadModel(context.Context, asrcontract.LoadModelRequest) error {
	return nil
}
func (p contractStubProvider) UnloadModel(context.Context, asrcontract.UnloadModelRequest) error {
	return nil
}
func (p contractStubProvider) LoadedModels(context.Context) ([]asrcontract.LoadedModel, error) {
	return nil, nil
}
func (p contractStubProvider) Capabilities(context.Context) ([]engineprovider.ModelCapability, error) {
	return []engineprovider.ModelCapability{{ID: "bad-model", Provider: "contract-stub", Capabilities: []string{"transcription"}}}, nil
}
func (p contractStubProvider) Prepare(context.Context) error { return nil }
func (p contractStubProvider) Transcribe(context.Context, engineprovider.TranscriptionRequest) (*engineprovider.TranscriptionResult, error) {
	return nil, nil
}
func (p contractStubProvider) Diarize(context.Context, engineprovider.DiarizationRequest) (*engineprovider.DiarizationResult, error) {
	return nil, nil
}
func (p contractStubProvider) IdentifySpeakers(context.Context, asrcontract.SpeakerIDRequest) (*asrcontract.SpeakerIDResult, error) {
	return nil, nil
}
func (p contractStubProvider) Close() error { return nil }

func TestRunProviderContractFailsInvalidParameterSchema(t *testing.T) {
	var failed bool
	RunProviderContract(stubTB{TB: t, failed: &failed}, contractStubProvider{models: []asrcontract.ModelCard{{
		ID:       "bad-model",
		Provider: "contract-stub",
		Capabilities: asrcontract.Capabilities{
			Transcription: true,
		},
		ParameterSchema: asrcontract.ParameterSchema{{
			Key:   "tail_paddings",
			Type:  asrcontract.ParameterTypeInteger,
			Scope: asrcontract.ParameterScopeDecoding,
		}},
	}}}, Options{})
	if !failed {
		t.Fatal("RunProviderContract did not fail invalid parameter schema")
	}
}

type stubTB struct {
	testing.TB
	failed *bool
}

func (tb stubTB) Fatal(args ...any) {
	*tb.failed = true
}

func (tb stubTB) Fatalf(format string, args ...any) {
	*tb.failed = true
	if !strings.Contains(format, "contract is invalid") {
		tb.TB.Fatalf("unexpected contract failure: "+format, args...)
	}
}
