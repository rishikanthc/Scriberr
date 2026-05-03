# Backend Architecture Refactor Sprint 02 Storage Boundary

Date: 2026-05-02

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-architecture-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-architecture-refactor-sprint-tracker.md`

## Goal

Move transcript artifact write policy out of the transcription processor and behind a small storage boundary.

This sprint keeps the existing transcript artifact layout:

```txt
<transcripts_dir>/<job_id>/transcript.json
```

## Completed

- Added `orchestrator.TranscriptStore`.
- Added `orchestrator.LocalTranscriptStore`.
- Moved transcript JSON directory creation, filename policy, permissions, and write behavior out of `Processor`.
- Kept `Processor.OutputDir` as a compatibility fallback for existing tests and construction paths.
- Wired production startup to inject `orchestrator.NewLocalTranscriptStore(cfg.TranscriptsDir)`.
- Added regression tests for:
  - preserving artifact path layout
  - writing transcript JSON with restrictive file permissions behavior
  - rejecting traversal-shaped job IDs
- Preserved API response path-leak checks from Sprint 1.

## Notes

Audio upload and recording storage still live in their existing packages. This sprint intentionally handled transcript artifacts first because that was the direct processor-owned storage policy. Later storage work can unify audio/import/recording object handling without changing the transcription worker boundary again.

## Verification

Passed:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/files ./internal/recording ./internal/transcription/... ./cmd/server
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'
git diff --check
```

## Commit

```txt
backend: consolidate transcript artifact storage
```
