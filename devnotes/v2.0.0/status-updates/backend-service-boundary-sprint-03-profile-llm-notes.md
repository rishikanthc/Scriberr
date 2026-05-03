# Backend Service Boundary Sprint 03 Profile and LLM Provider Notes

Date: 2026-05-02

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-service-boundary-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-service-boundary-refactor-sprint-tracker.md`

## Goal

Move profile/default-profile and LLM provider settings behavior out of HTTP handlers, and make profile lookup reusable by recording validation.

## Changes

- Added `internal/profile.Service`.
- Added `internal/llmprovider.Service`.
- Added repository methods for profile workflows:
  - create profile with default-profile consistency
  - update profile with default-profile consistency
  - delete profile and clear matching user default
  - set default profile
- Added `LLMConfigRepository.ReplaceActiveByUser` so LLM provider replacement is repository-owned.
- Updated `profile_handlers.go` to use `profile.Service`.
- Updated `llm_provider_handlers.go` to use `llmprovider.Service`.
- Updated recording profile validation to use `profile.Service.Exists`.
- Wired profile and LLM provider services in `cmd/server/main.go` and API test setup.
- Removed production API database imports from:
  - `profile_handlers.go`
  - `llm_provider_handlers.go`
  - `recording_handlers.go`
- Updated the architecture guard inventory.

## Architecture Result

Profiles and LLM provider settings now have service boundaries. Default profile mutations are owned by repository transaction methods instead of handler transactions.

Remaining production API database access is still tracked in:

```txt
admin_handlers.go
chat_handlers.go
file_handlers.go
summary_handlers.go
summary_widget_handlers.go
transcription_handlers.go
```

## Verification

Passed:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestProfile|TestSettings|TestRecording|TestProductionAPIDatabaseAccessInventory'
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestLLMProviderSettingsEmptyAndAuth|TestProductionAPIDatabaseAccessInventory'
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/repository ./internal/profile ./internal/llmprovider ./cmd/server
git diff --check
```

Blocked by sandbox loopback restrictions:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestProfile|TestLLMProvider|TestSettings|TestRecording|TestProductionAPIDatabaseAccessInventory'
```

The LLM provider tests that exercise provider connection behavior use `httptest.NewServer`, which cannot bind loopback in this sandbox.

## Next Sprint

Sprint 4 should move file upload, file metadata, video extraction completion, and media import handoff behind a file service. It should remove database access from:

- `file_handlers.go`

It should also prepare one shared file-ready handoff point for later auto-transcribe and auto-rename automation.
