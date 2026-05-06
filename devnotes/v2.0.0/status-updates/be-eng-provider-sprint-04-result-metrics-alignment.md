# BE-ENG-PROVIDER Sprint 4: Canonical Result And Metrics Alignment

Date: 2026-05-05

Status: complete.

## Scope

This sprint aligns provider result metadata with the backend canonical transcript and execution metadata path.

## Changes

- Local provider metadata now preserves engine `metrics` and `plan` as nested provider payloads.
- Removed local adapter RTF derivation from engine metric fields.
- Canonical transcript metadata sanitization now preserves nested safe maps and arrays.
- Execution config metadata uses the same sanitizer as canonical transcript metadata.
- Backend still owns transcript merging and speaker-label normalization, but does not infer local decode metrics.

## Cleanup Decisions

- Engine/provider metrics are stored as provider metadata; backend does not flatten or recompute them.
- Unsafe metadata keys and string values containing paths, tokens, or secrets are dropped recursively.
- Transcript JSON remains the canonical user-facing transcript artifact.

## Validation

```sh
GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider ./internal/transcription/orchestrator
GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...
GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api
GOCACHE=/tmp/scriberr-engine-go-cache go test ./...
```

Notes:

- The transcription and API package runs need local `httptest` listeners, so they were run with test execution allowed outside the sandbox.
