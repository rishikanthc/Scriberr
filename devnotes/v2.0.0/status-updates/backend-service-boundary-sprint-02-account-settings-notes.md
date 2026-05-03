# Backend Service Boundary Sprint 02 Account and Settings Notes

Date: 2026-05-02

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-service-boundary-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-service-boundary-refactor-sprint-tracker.md`

## Goal

Move account/auth, API-key, API-key middleware, and settings persistence out of HTTP handlers.

## Changes

- Added `internal/account.Service`.
- Moved these workflows behind the account service:
  - registration status
  - register
  - login
  - refresh-token rotation
  - logout refresh-token revocation
  - current user lookup
  - password change
  - username change
  - API key list/create/delete
  - API-key authentication and last-used update
  - settings read/update persistence
- Added `auto_rename_enabled` to user settings JSON.
- Added settings validation needed by the General tab:
  - enabling `auto_transcription_enabled` requires a valid default profile
  - enabling `auto_rename_enabled` requires an active LLM provider with a configured small model
- Updated auth, API key, settings, and API-key middleware handlers to call the account service instead of `database.DB`.
- Wired account service construction in `cmd/server/main.go` and API test setup.
- Updated the architecture guard inventory to remove:
  - `auth_handlers.go`
  - `api_key_handlers.go`
  - `middleware.go`
  - `settings_handlers.go`
- Added regression coverage for invalid automation setting enablement.

## Architecture Result

The account and settings API paths no longer perform direct database access.

Remaining production API database access is still tracked in:

```txt
admin_handlers.go
chat_handlers.go
file_handlers.go
llm_provider_handlers.go
profile_handlers.go
recording_handlers.go
summary_handlers.go
summary_widget_handlers.go
transcription_handlers.go
```

## Verification

Passed:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestAPIKey|TestIdempotency|TestSettings|TestAuth|TestProductionAPIDatabaseAccessInventory|TestSecurity'
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/database ./internal/repository ./internal/account
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./cmd/server
git diff --check
```

Full `go test ./internal/api` remains blocked in this sandbox by LLM provider tests that use `httptest.NewServer`.

## Next Sprint

Sprint 3 should move profile and LLM provider settings behavior behind services. It should remove database access from:

- `profile_handlers.go`
- `llm_provider_handlers.go`
- recording profile validation in `recording_handlers.go` if profile service lookup is shared there
