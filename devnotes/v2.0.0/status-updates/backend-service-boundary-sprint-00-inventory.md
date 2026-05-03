# Backend Service Boundary Sprint 00 Inventory

Date: 2026-05-02

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-service-boundary-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-service-boundary-refactor-sprint-tracker.md`

## Goal

Freeze the current direct database access in `internal/api` before service-boundary refactoring starts.

Sprint 0 does not remove behavior. It records the current coupling and adds a guard so future work cannot add more production API database access while the refactor is underway.

## Architecture Rule

Target rule:

```txt
internal/api must not import scriberr/internal/database
internal/api must not call database.DB
```

Current code does not satisfy this yet. The Sprint 0 guard tracks the known production files that still import `scriberr/internal/database`. Later sprints must shrink the allowlist as each domain moves behind services.

## Current Production API Files Importing Database

The following production files currently import `scriberr/internal/database`:

```txt
admin_handlers.go
api_key_handlers.go
auth_handlers.go
chat_handlers.go
file_handlers.go
llm_provider_handlers.go
middleware.go
profile_handlers.go
recording_handlers.go
router.go
settings_handlers.go
summary_handlers.go
summary_widget_handlers.go
transcription_handlers.go
```

Test files are intentionally excluded from this inventory. Tests may continue to seed and inspect test databases directly when that is the most practical verification path.

## Domain Classification

| File | Domain | Direct DB responsibility to move |
| --- | --- | --- |
| `auth_handlers.go` | account/auth | user registration, login lookup, refresh token rotation, logout revocation, password and username updates, current user lookup |
| `api_key_handlers.go` | account/API keys | API key list/create/revoke |
| `middleware.go` | auth middleware | API key lookup and last-used update |
| `settings_handlers.go` | settings | user settings persistence |
| `profile_handlers.go` | profiles | profile CRUD, default profile updates, default profile/user setting consistency |
| `llm_provider_handlers.go` | LLM provider settings | active provider read/save, provider config transaction |
| `file_handlers.go` | files/media | direct upload persistence, video extraction persistence, file list/update/delete/get |
| `transcription_handlers.go` | transcriptions | manual transcription creation, default profile resolution, list/update/delete/retry/get |
| `recording_handlers.go` | recordings | profile existence validation |
| `admin_handlers.go` | admin/worker details | queue stats and transcription execution lookups |
| `summary_handlers.go` | summaries | summary and widget run lookups |
| `summary_widget_handlers.go` | summary widgets | widget CRUD through repository construction from global DB |
| `chat_handlers.go` | chat | chat repository construction, active LLM config lookup |
| `router.go` | composition fallback | fallback service construction from global DB |

## Reusable Existing Boundaries

These existing pieces should be reused instead of reimplemented:

- `repository.JobRepository` for transcription/file records, queue transitions, executions, summaries, and recording handoff.
- `repository.ProfileRepository` for profile lookup and default profile queries.
- `repository.RecordingRepository` for recording session/chunk/finalizer state.
- `repository.SummaryRepository` for summaries, summary widgets, and widget runs.
- `repository.ChatRepository` for chat sessions, messages, context, and run persistence.
- `repository.LLMConfigRepository` for active LLM provider lookup.
- `recording.Service` and `recording.FinalizerService` for recording workflows.
- `transcription/worker.Service` for durable queue enqueue, cancel, stats, and workers.
- `summarization.Service` for summary and existing LLM title/description generation behavior.
- `annotations.Service` and `tags.Service` for annotation and tag domain behavior.

## Missing Or Incomplete Boundaries

The following boundaries are needed for the later sprints:

- Account service wired into canonical API auth handlers.
- User settings service with validation against default profile and LLM readiness.
- API key service or account subservice for API key commands and middleware lookup.
- Profile service that owns default profile consistency.
- LLM provider settings service that owns provider testing and model validation.
- File service for upload, metadata, audio lookup, video extraction completion, and file-ready handoff.
- Transcription command service for manual and automated transcription creation.
- Post-file automation service for auto-transcribe and auto-rename decisions.
- API composition wiring from `cmd/server/main.go` instead of `api.NewHandler` fallback construction.

## Sprint 0 Guard

Added:

- `internal/api/architecture_test.go`

The guard scans production `.go` files in `internal/api`, excluding `_test.go`, and compares database imports against the known Sprint 0 inventory.

Expected behavior:

- Adding a new production API file that imports `scriberr/internal/database` fails the guard.
- Removing a production API database import fails until the expected inventory is reduced, forcing the tracker and inventory to stay current.
- The guard becomes stricter in later sprints as the allowlist shrinks toward zero.

## Next Sprint

Sprint 1 should start with dependency injection and composition-root cleanup:

- Define narrow API-facing service interfaces.
- Stop constructing fallback repositories/services in `api.NewHandler`.
- Wire concrete dependencies in `cmd/server/main.go`.
- Update API test setup to pass dependencies intentionally.
