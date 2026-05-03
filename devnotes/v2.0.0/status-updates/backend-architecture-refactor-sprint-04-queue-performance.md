# Backend Architecture Refactor Sprint 04 Queue Performance

Date: 2026-05-02

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-architecture-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-architecture-refactor-sprint-tracker.md`

## Goal

Prepare durable transcription queue claiming for future fairness controls while preserving current single-user FIFO behavior for equal-priority jobs.

## Completed

- Added `TranscriptionJob.Priority` with default `0`.
- Updated the queue claim index to cover the claim order:

```txt
status, priority DESC, queued_at, created_at, id
```

- Updated `ClaimNextTranscription` ordering:

```txt
priority DESC
queued_at ASC
created_at ASC
id ASC
```

- Added repository coverage that:
  - the priority column exists
  - the queue claim index exists
  - equal-priority jobs still claim FIFO
  - higher-priority jobs claim before lower-priority jobs
  - FIFO still applies within the same priority
- Re-ran worker tests to verify queue service behavior remains stable.

## Notes

This sprint does not implement per-user concurrency limits yet. The queue now has a priority dimension and a single repository claim method where future per-user fairness can be introduced without changing API handlers, providers, or worker callers.

## Verification

Passed:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/database ./internal/repository ./internal/transcription/worker ./cmd/server
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'
git diff --check
```

## Commit

```txt
backend: prepare queue fairness and performance
```
