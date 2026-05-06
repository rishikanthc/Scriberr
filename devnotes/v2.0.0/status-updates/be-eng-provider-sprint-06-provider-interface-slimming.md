# BE-ENG-PROVIDER Sprint 6: Provider Interface Slimming

Status: complete.

## Changes

- Split task execution into small provider interfaces:
  - `TranscriptionProvider`
  - `DiarizationProvider`
  - `SpeakerIdentificationProvider`
- Removed provider-level `Capabilities()`; the registry now derives selectable capabilities from provider model cards.
- Removed provider-level `Prepare()` from the execution path.
- Kept the local sherpa provider as a direct Go adapter over `scriberr-engine`.
- Kept remote provider code isolated under the existing remote client package for future REST-style providers.

## Verification

- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider ./internal/transcription/engineprovider/remote ./internal/transcription/engineprovider/contracttest`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/orchestrator`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api ./internal/app`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/...`
- `GOCACHE=/private/tmp/scriberr-go-cache go vet ./internal/transcription/... ./internal/profile ./internal/api`

## Notes

- `Registry.Capabilities()` remains as a backend/frontend projection API, but it is now derived data rather than a provider contract requirement.
- Orchestrator step resolution still selects by advertised model capability, then verifies that the chosen provider implements the required task interface before execution.
