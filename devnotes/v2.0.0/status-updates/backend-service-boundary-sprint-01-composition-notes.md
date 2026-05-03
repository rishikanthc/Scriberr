# Backend Service Boundary Sprint 01 Composition Notes

Date: 2026-05-02

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-service-boundary-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-service-boundary-refactor-sprint-tracker.md`

## Goal

Start moving the API package away from global database access by making handler dependencies explicit at construction time.

## Changes

- Added `api.HandlerDependencies` as the explicit API composition input.
- Changed `api.NewHandler` to accept `HandlerDependencies` instead of untyped variadic services.
- Removed fallback repository/service construction from `api.NewHandler`.
- Removed the `scriberr/internal/database` import from `internal/api/router.go`.
- Moved readiness injection to the composition root through `HandlerDependencies.ReadinessCheck`.
- Wired the real API dependencies from `cmd/server/main.go`:
  - transcription queue service
  - engine model registry
  - annotation service
  - tag service
  - recording service
  - recording finalizer wake hook
  - database readiness check
- Added explicit tag service construction in `cmd/server/main.go`; previously tags were created implicitly by the API handler fallback.
- Updated API test server setup to inject annotation, tag, and recording services explicitly.
- Updated the architecture guard allowlist to remove `router.go`.

## Architecture Result

`internal/api/router.go` no longer imports `scriberr/internal/database` and no longer creates repositories from `database.DB`.

Remaining production API database access is still present in domain handlers and middleware. Those entries remain tracked by `internal/api/architecture_test.go` and will be removed in later sprints.

## Verification

Passed:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestProductionAPIDatabaseAccessInventory|TestCanonicalRouteRegistration|TestEndpointContractSmoke'
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./cmd/server
```

Blocked by sandbox loopback restrictions:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api
```

The full API package reached `TestLLMProviderSettingsSaveTestsConnectionAndMasksKey`, which uses `httptest.NewServer`. In this sandbox it failed with:

```txt
listen tcp6 [::1]:0: bind: operation not permitted
```

This is the same class of loopback binding limitation noted in earlier backend trackers.

## Next Sprint

Sprint 2 should move account/auth and settings persistence behind services. It should remove database access from:

- `auth_handlers.go`
- `api_key_handlers.go`
- `settings_handlers.go`
- the API-key lookup section of `middleware.go`
