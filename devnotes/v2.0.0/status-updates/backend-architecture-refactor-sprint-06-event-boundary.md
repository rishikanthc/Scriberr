# Backend Architecture Refactor Sprint 06 Event Boundary

Date: 2026-05-02

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-architecture-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-architecture-refactor-sprint-tracker.md`

## Goal

Harden SSE/event payloads so they stay small, public-ID based, and free of internal file/provider paths.

## Completed

- Added adapter-level event payload regression coverage for:
  - recording events
  - summary status events
  - worker/transcription status events
  - externally published file events
- Verified payloads use public IDs:
  - `rec_*`
  - `file_*`
  - `tr_*`
- Added shallow sanitization for externally supplied file/transcription event maps.
- Dropped internal path keys from externally published event payloads:

```txt
path
audio_path
source_file_path
output_json_path
output_srt_path
output_vtt_path
logs_path
```

- Preserved existing event names and stream behavior.

## Notes

Business services still publish small domain events or status structs. The API adapter remains responsible for mapping those into SSE payloads. This keeps public event shape out of domain services while still allowing services to notify the UI.

## Verification

Passed:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestEvent|TestSSE|Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/files ./internal/recording ./internal/summarization ./internal/transcription/...
git diff --check
```

## Commit

```txt
backend: harden event boundary
```
