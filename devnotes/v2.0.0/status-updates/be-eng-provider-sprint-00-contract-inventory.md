# BE-ENG-PROVIDER Sprint 0: Contract Duplication Inventory

Date: 2026-05-05

Status: complete.

## Scope

This sprint freezes the backend/engine duplication map before removing legacy provider contract fields. It makes no runtime behavior changes.

## Current Duplication Map

Backend provider boundary:

- `internal/transcription/asrcontract/types.go` defines public provider DTOs, model cards, capabilities, chunking descriptors, parameter schemas, progress events, and remote job/result shapes.
- `internal/transcription/engineprovider/types.go` defines an internal provider interface plus another set of request/result structs.
- `internal/transcription/engineprovider/local_provider.go` adapts Scriberr backend calls to `scriberr-engine`, but still tries to populate legacy typed engine request fields that no longer exist.
- `internal/transcription/engineprovider/remote/client.go` adapts the same internal provider request to REST provider DTOs and still depends on the legacy typed fields.
- `internal/transcription/orchestrator/planner.go` plans local runtime threads, fixed chunking, provider chunking, and chunk seconds even though the engine now owns local execution planning.

Engine boundary:

- `references/engine/speech/engine/requests.go` is now the canonical in-process engine request shape for local transcription and diarization.
- `references/engine/speech/engine/contract.go` exposes provider identity/status/model cards and progress stages.
- `references/engine/speech/providers/descriptor.go` owns provider-neutral model descriptors and parameter metadata.
- `references/engine/speech/results/results.go` owns canonical engine result words, segments, and metrics-oriented result fields.

## Redundant Or Conflicting Surfaces

Request fields duplicated in the backend and engine:

- `Language`
- `Task`
- `Threads` / `NumThreads`
- `TailPaddings`
- `EnableTokenTimestamps`
- `EnableSegmentTimestamps`
- `DecodingMethod`
- `Chunking`
- `ChunkDurationSec`
- `BatchSize`
- Diarization threshold and duration tuning fields

Schema/default duplication:

- `local_provider.go` synthesizes model schemas and defaults with `chunkingCapabilitiesForModel`, `parameterSchemaForModel`, and `recommendedDefaultsForModel`.
- The engine already has descriptor-backed defaults and should be the source of truth for local sherpa models.

Execution planning duplication:

- The backend orchestrator currently chooses local chunking and runtime thread settings.
- The engine already owns fixed-window/VAD/batch planning and must remain the only local planner.

Metrics duplication:

- The backend local provider recomputes decode duration, estimated audio duration, RTF, chunk count, and hypothesis word count.
- The engine should return plan summaries and metrics that the backend stores without local inference.

## Cleanup Decisions

- Collapse backend transcription and diarization requests to `Parameters map[string]any`.
- Keep `JobID`, `UserID`, `AudioPath`, `ModelID`, `Progress`, and `Parameters` at the backend provider request boundary.
- Delete backend-local sherpa schema/default synthesis after the engine exposes descriptor-equivalent model metadata.
- Delete backend local fixed/VAD/batch planning after profile options flow directly to the provider.
- Preserve REST provider support as a future external provider path, separate from the local in-process sherpa adapter.
- Keep `scriberr-engine` imports isolated to `internal/transcription/engineprovider/local_provider.go`.

## Existing Guardrails

The backend already has architecture tests in `internal/api/architecture_test.go` that enforce:

- only the local ASR provider adapter imports `scriberr-engine`
- profile services do not import `scriberr-engine`
- provider packages do not depend on API or repository packages
- `asrcontract` does not depend on backend runtime packages

These guardrails should be extended in later sprints after legacy fields are removed, because strict field-level checks would fail on the current duplicated implementation.

## Validation Notes

Current compile baseline confirms the next sprint target:

```text
go test ./internal/transcription/engineprovider
```

fails because `local_provider.go` still populates removed fields on `speechengine.TranscriptionRequest`:

- `Language`
- `Task`
- `TailPaddings`
- `EnableTokenTimestamps`
- `EnableSegmentTimestamps`
- `DecodingMethod`
- `Chunking`
- `ChunkDurationSec`
- `NumThreads`
- `Provider`

Sprint 1 must remove this mismatch by forwarding `Parameters` into the current engine API.
