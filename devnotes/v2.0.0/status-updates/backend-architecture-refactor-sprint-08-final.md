# Backend Architecture Refactor Sprint 08 Final Gates

Date: 2026-05-03

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-architecture-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-architecture-refactor-sprint-tracker.md`

## Goal

Close the backend architecture refactor with matching docs, stricter dependency guards, and cleanup of dead legacy packages.

## Completed

- Removed unreferenced legacy packages:
  - `internal/dropzone`
  - `internal/interfaces`
  - `internal/service`
- Updated architecture docs to reflect `internal/app` as the composition root and `cmd/server` as process lifecycle only.
- Added hard architecture inventories for production imports:
  - `internal/database` is only imported by `internal/app` outside the database package.
  - `internal/api` is only imported by `internal/app` outside the API package.
- Kept existing dependency direction guards for models, repositories, engine providers, workers, and production API database access.

## Residual Debt

- The legacy Python transcription adapter tree has since been removed from the active repository; it is not wired into `cmd/server`.
- The older `internal/queue` package remains because tests still exercise it. The production server uses `internal/transcription/worker`.
- The full `internal/api` package test run is still blocked in the sandbox by `httptest.NewServer` loopback bind restrictions.

## Verification

Passed:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestEvent|TestSSE|Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestProductionInternalDatabaseImportInventory|TestProductionInternalAPIImportInventory|TestBackendDependencyDirection'
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./internal/files ./internal/recording ./internal/summarization ./internal/chat ./internal/account ./internal/profile ./internal/llmprovider ./internal/automation ./cmd/server ./internal/app
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go list ./...
git diff --check
```

Blocked by sandbox:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./internal/files ./internal/recording ./internal/summarization ./internal/chat ./internal/account ./internal/profile ./internal/llmprovider ./internal/automation ./cmd/server ./internal/app
```

The blocked test is `TestLLMProviderSettingsSaveTestsConnectionAndMasksKey`; it uses `httptest.NewServer`, and the sandbox denies loopback bind.

## Commit

```txt
backend: finalize architecture refactor gates
```
