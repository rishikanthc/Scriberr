# Backend Architecture Refactor Sprint 07 Bootstrap Extraction

Date: 2026-05-03

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-architecture-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-architecture-refactor-sprint-tracker.md`

## Goal

Move backend composition out of `cmd/server/main.go` so startup wiring can be tested without binding an HTTP listener.

## Completed

- Added `internal/app` as the backend composition root.
- Moved database initialization, repositories, providers, services, API handler wiring, route setup, worker startup, and bounded shutdown behind explicit app lifecycle methods:
  - `app.Build`
  - `App.Start`
  - `App.Server`
  - `App.Shutdown`
- Kept `cmd/server/main.go` focused on:
  - version flag handling
  - logging and configuration loading
  - application lifecycle calls
  - HTTP listener startup
  - signal handling
  - process exit status
- Added tests that build the backend graph with temporary dependencies and call `/api/v1/ready` through the router without starting a listener.
- Added a command package guard that prevents backend service composition imports from creeping back into `cmd/server/main.go`.

## Notes

`api.NewHandler` still receives explicit dependencies from the composition root. Worker startup remains after database recovery and route construction. Shutdown still runs in bounded order after HTTP server shutdown: transcription workers, summary workers, recording finalizers, local provider close, and database close.

## Verification

Passed:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./cmd/server ./internal/app
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./cmd/server ./internal/app ./internal/api -run 'TestEvent|TestSSE|Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'
git diff --check
```

Blocked by sandbox:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./cmd/server ./internal/app ./internal/api
```

The blocked test is `TestLLMProviderSettingsSaveTestsConnectionAndMasksKey`; it uses `httptest.NewServer`, and the sandbox denies loopback bind.

## Commit

```txt
backend: extract app bootstrap
```
