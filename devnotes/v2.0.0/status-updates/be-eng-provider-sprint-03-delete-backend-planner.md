# BE-ENG-PROVIDER Sprint 3: Delete Backend Chunking And Batching Planner

Date: 2026-05-05

Status: complete.

## Scope

This sprint removes backend-owned local chunking, VAD, runtime, and batching planning from the transcription orchestrator.

## Changes

- Deleted the orchestrator execution planner and its tests.
- Removed backend plan boundary progress events.
- Kept orchestrator responsibility to pipeline sequencing, provider selection, preprocessing, transcript merging, and persistence.
- Provider pipeline step options are passed through unchanged; providers own validation and execution planning.
- Execution config now records pipeline steps and sanitized provider metadata after transcription succeeds.
- Engine/local provider metadata can persist its own plan summary under `provider_metadata`.

## Cleanup Decisions

- The backend no longer rejects chunking/batching combinations before provider execution.
- Backend progress no longer invents `transcribing`, `diarizing`, or `identifying_speakers` boundary stages. Those stages must come from provider progress events.
- Existing execution config still records provider/model pipeline ordering without source audio paths.

## Validation

```sh
GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/orchestrator
GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...
GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api
GOCACHE=/tmp/scriberr-engine-go-cache go test ./...
```

Notes:

- The transcription and API package runs need local `httptest` listeners, so they were run with test execution allowed outside the sandbox.
