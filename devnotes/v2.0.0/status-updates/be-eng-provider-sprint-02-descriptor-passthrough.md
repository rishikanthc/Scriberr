# BE-ENG-PROVIDER Sprint 2: Descriptor Passthrough For Local Provider

Date: 2026-05-05

Status: complete.

## Scope

This sprint removes backend-owned local model schema/default synthesis. Local provider model cards now come from engine descriptors instead of backend family heuristics.

## Changes

- `scriberr-engine` model cards now include the model descriptor used by the engine planner.
- The backend local provider maps descriptor fields into `asrcontract.ModelCard` mechanically:
  - tasks
  - language support
  - runtime/resource backends
  - chunking and batching limits
  - parameter schema
  - recommended defaults
  - artifact metadata
- Deleted active backend synthesis of local sherpa defaults and schemas:
  - `chunkingCapabilitiesForModel`
  - `parameterSchemaForModel`
  - `recommendedDefaultsForModel`
  - Parakeet/Whisper family-specific default helpers

## Cleanup Decisions

- The local provider keeps a reduced fallback mapper only for tests or older engine cards with no descriptor. It does not synthesize schemas/defaults in that fallback.
- Artifact requirements are preserved under model card extensions as descriptor metadata, not converted into backend-invented path parameters.
- Backend tests now assert descriptor passthrough values from engine defaults, including Parakeet fixed 30s chunking, threads 4, and batch 1.

## Validation

```sh
GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider
GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...
GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api
GOCACHE=/tmp/scriberr-engine-go-cache go test ./...
```

Notes:

- The transcription and API package runs need local `httptest` listeners, so they were run with test execution allowed outside the sandbox.
