# BE-ENG-PROVIDER Sprint 1: Backend Request Contract Collapse

Date: 2026-05-05

Status: complete.

## Scope

This sprint collapses the backend provider execution request shape from model-specific fields to descriptor-keyed parameter maps.

## Changes

- `engineprovider.TranscriptionRequest` now carries only:
  - `JobID`
  - `UserID`
  - `AudioPath`
  - `Progress`
  - `ModelID`
  - `Parameters map[string]any`
- `engineprovider.DiarizationRequest` now uses the same shape with `Parameters map[string]any`.
- The local provider adapter now calls `speechengine.TranscriptionRequest{Parameters: ...}` and `speechengine.DiarizationRequest{Parameters: ...}` directly.
- The local provider no longer synthesizes Parakeet chunking, batch size, thread, task, timestamp, or provider defaults during execution.
- The remote provider client forwards `Parameters` through REST `options`; it only mirrors `task`, `language`, and timestamp feature booleans into the public remote request envelope where the remote contract still has explicit fields.
- The orchestrator now passes pipeline step `Options` directly into provider execution requests.
- Tests were updated so execution assertions check provider parameter maps instead of removed typed fields.

## Cleanup Decisions

- Legacy flat `ASRParams` fields remain in the model for existing storage/API cleanup work, but they are no longer copied into active provider execution requests.
- Backend-owned whisper decoding normalization was removed from active execution. Model-specific decoding behavior belongs in provider descriptors and provider-side validation.
- Backend-owned Parakeet execution defaults were removed from the local adapter. Local model defaults must come from the engine descriptor and execution planner.

## Validation

```sh
GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider
GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...
GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api
GOCACHE=/tmp/scriberr-engine-go-cache go test ./...
```

Notes:

- `./internal/transcription/...` and `./internal/profile ./internal/api` need local `httptest` listeners for existing REST/API tests. The sandboxed run cannot bind those ports, so the verification was rerun with test execution allowed outside the sandbox.
