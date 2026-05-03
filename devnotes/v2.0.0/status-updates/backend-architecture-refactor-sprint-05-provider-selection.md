# Backend Architecture Refactor Sprint 05 Provider Selection

Date: 2026-05-02

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-architecture-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-architecture-refactor-sprint-tracker.md`

## Goal

Move ASR provider choice behind an explicit selector so future providers can be added without changing API handlers, repositories, queue code, or the transcription worker.

## Completed

- Added `engineprovider.SelectionRequest`.
- Added `Registry.Select(ctx, req)`.
- Implemented deterministic provider selection in `StaticRegistry`:
  - explicit provider ID returns that provider
  - explicit provider plus model/capability validates against provider capabilities
  - model/capability fallback scans providers by sorted provider ID
  - no request falls back to the default provider
- Added selector tests for:
  - explicit provider and model
  - deterministic capability fallback
  - missing provider
  - missing model
  - missing capability
- Updated `orchestrator.Processor` to resolve providers through `Registry.Select`.
- Added processor coverage proving explicit `EngineID` routes execution to the selected provider.

## Notes

The orchestrator currently passes explicit provider ID only. Model and capability fallback is now available in the registry, but job/profile request modeling for automatic capability requirements is deferred until a feature needs it.

## Verification

Passed:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/transcription/... ./cmd/server
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'
git diff --check
```

## Commit

```txt
backend: add provider capability selection
```
