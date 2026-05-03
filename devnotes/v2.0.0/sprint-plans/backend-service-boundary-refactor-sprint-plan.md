# Backend Service Boundary Refactor Sprint Plan

This plan refactors the core backend framework so new settings automation can be implemented cleanly under `devnotes/v2.0.0/rules/backend-architecture-rules.md`.

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-service-boundary-refactor-sprint-tracker.md`

## Product Goal

Prepare the backend for General settings features:

- user password change from the General settings tab
- global auto-transcribe setting that queues newly ready audio through the default transcription profile
- global auto-rename setting that uses the configured small LLM model when available

Before those features are extended, the backend must have a clean architecture where handlers are HTTP adapters only and domain workflows are owned by services and repositories.

## Refactor Goal

Remove direct database access from `internal/api` and route new feature work through injected service interfaces.

Target flow:

```txt
HTTP handler
  -> typed request/auth boundary
  -> injected service command/query
  -> domain repository/storage/queue/LLM boundary
  -> response mapper/event publisher
```

Hard rule for this workstream:

```txt
internal/api must not import scriberr/internal/database
internal/api must not call database.DB
```

The only allowed database access after this refactor is inside repositories, migrations, tests, and composition/bootstrap code that wires repositories into services.

## Current Codebase Notes

The backend already has several strong foundations:

- Durable transcription queue and repository state transitions.
- Recording service and finalizer with domain-specific repositories.
- Summarization service with LLM provider lookup and graceful no-op behavior for missing LLM configuration.
- LLM provider settings endpoints and persistence.
- Profile persistence and default-profile metadata.
- Auth helpers for password hashing and token issuance.

The main architectural problem is inconsistent layering:

- `internal/api` still imports `internal/database` and performs GORM queries directly.
- Settings, auth, profiles, files, transcriptions, and LLM provider handlers still own persistence decisions.
- File upload and media-import completion paths do not share a post-file-ready automation service.
- Default profile resolution differs by workflow: some paths use `user.DefaultProfileID`, while recording finalization uses `profiles.is_default`.
- Existing user service code is legacy-shaped and not wired into the canonical API.

This refactor should preserve behavior while moving decisions to services in small, testable slices.

## Architecture Decisions

### 1. Services Own Workflow Decisions

Add or modernize focused services for:

- account/auth commands
- user settings commands
- profile commands and default-profile resolution
- file upload/import and file metadata commands
- transcription creation/cancel/retry commands
- LLM provider settings commands
- post-file-ready automation

Handlers should bind requests, call one service method, and map results.

### 2. Repositories Own Persistence Shape

Repositories should expose domain methods, not generic handler query helpers. Examples:

```txt
FindUserForAuth
UpdatePasswordHash
GetSettingsForUser
UpdateSettingsForUser
SetDefaultProfileForUser
FindDefaultProfileForUser
CreateUploadedFile
MarkFileReady
CreateTranscriptionFromFile
FindActiveLLMConfigForUser
```

Repositories should not publish events, select providers, or decide whether automation should run.

### 3. File-Ready Automation Is a Domain Service

Introduce one backend service that receives a file-ready event from all file creation paths:

```txt
direct upload
video audio extraction completion
YouTube import completion
recording finalizer handoff
future imports
```

It should make durable decisions:

- if auto-transcribe is enabled and a default profile exists, create and enqueue a transcription
- if auto-transcribe is enabled but no default profile exists, no-op gracefully and report a settings validation issue when enabling
- if auto-rename is enabled and a small LLM is configured, enqueue or invoke the approved title-generation workflow
- if auto-rename is enabled but LLM config is missing, no-op gracefully and report a settings validation issue when enabling

No handler should directly decide post-file automation.

### 4. Settings Validation Happens Before Persistence

The settings service should reject enabling invalid settings:

- `auto_transcription_enabled=true` requires a valid default transcription profile.
- `auto_rename_enabled=true` requires an active LLM provider and a configured small model.

Runtime workflows should still no-op gracefully if configuration is later removed or becomes invalid.

### 5. Composition Root Wires Concrete Dependencies

`cmd/server/main.go` should create repositories and services, then pass service interfaces into `api.NewHandler`.

The API package should not construct repositories as a fallback. Missing required services should fail clearly in tests/startup rather than silently reaching into globals.

## Non-Goals

- Do not redesign the database schema except for settings fields needed by later sprints.
- Do not change public API routes unless route contract tests are updated deliberately.
- Do not rewrite transcription engine internals.
- Do not add a second queue system.
- Do not implement frontend settings UI in this backend refactor.
- Do not implement the final General settings feature until the service boundary is in place.

## Sprint 0: Inventory, Dependency Map, and Stop-The-Line Guard

Goal: make the current direct database access and service gaps explicit before moving code.

Tasks:

- Inventory every `internal/api` import of `internal/database`.
- Inventory every `database.DB` call in handlers and classify it by domain.
- Identify existing repository methods that can be reused without new abstractions.
- Identify missing domain repository methods.
- Add a lightweight architecture guard test or script that fails when `internal/api` imports `internal/database`.
- Document the intended service interfaces for auth, settings, profiles, files, transcriptions, LLM provider config, and automation.

Acceptance criteria:

- The direct DB usage inventory exists in status notes.
- The first guard prevents adding new `database.DB` access to `internal/api`.
- No behavior changes are made except the guard.

Testing focus:

- Architecture guard.
- Existing API route contract still passes.

## Sprint 1: Composition Root and Handler Dependency Injection

Goal: make the API depend on service interfaces instead of global database access.

Tasks:

- Define API-facing service interfaces in or near `internal/api` only where they are consumed.
- Update `Handler` to receive required services explicitly.
- Remove API fallback repository construction from `NewHandler`.
- Wire concrete services in `cmd/server/main.go`.
- Update API test server setup to inject fake or real services intentionally.
- Keep event publishing interfaces narrow and explicit.

Acceptance criteria:

- `api.NewHandler` no longer constructs repositories from `database.DB`.
- Test setup controls all handler dependencies.
- No direct feature behavior is changed.

Testing focus:

- Route contract.
- API auth smoke tests.
- Server construction tests.

## Sprint 2: Account and Settings Services

Goal: move auth/account and settings persistence out of handlers.

Tasks:

- Replace handler-level user lookup, registration, login, refresh-token, logout, password, and username persistence with an account service.
- Replace settings handler persistence with a settings service.
- Add user repository methods for password hash updates, refresh-token commands, and settings updates.
- Add settings validation hooks for default profile and active LLM readiness.
- Add `auto_rename_enabled` to persisted user settings, but keep runtime feature disabled until the automation sprint.
- Keep API response shapes unchanged except for the new settings field when introduced.

Acceptance criteria:

- Auth and settings handlers perform no database queries.
- Password change remains available for the General tab.
- Enabling auto-transcription without a default profile returns a validation error.
- Enabling auto-rename without a configured active LLM provider and small model returns a validation error.
- Runtime settings reads return enough capability metadata for the frontend to disable invalid toggles.

Testing focus:

- Password change behavior.
- Refresh-token rotation/revocation.
- Settings partial update.
- Validation failures for missing default profile and missing LLM small model.
- Migration/backfill for new settings field.

## Sprint 3: Profile and LLM Provider Services

Goal: make profile/default-profile and LLM provider configuration reusable by settings and automation.

Tasks:

- Move profile list/create/get/update/delete/set-default logic into a profile service.
- Ensure one canonical default-profile source is used by all workflows.
- Move LLM provider get/update/test/save logic into an LLM provider service.
- Add repository methods for active LLM config lookup and readiness checks.
- Keep provider connection tests and model validation inside the service, not the handler.

Acceptance criteria:

- Profile and LLM provider handlers perform no database queries.
- Default profile resolution is consistent across transcriptions, recordings, and settings validation.
- LLM provider readiness can be checked without duplicating model/provider logic.

Testing focus:

- Profile default uniqueness.
- Default profile clearing on delete.
- LLM provider save with model validation.
- Missing provider and missing small-model readiness results.

## Sprint 4: File and Media Import Service Boundary

Goal: centralize file creation, file readiness, and storage policy.

Tasks:

- Add a file service for direct uploads, metadata list/get/update/delete, and audio streaming lookup.
- Move upload storage path construction and DB persistence out of handlers.
- Move video extraction completion persistence behind service/repository methods.
- Adapt YouTube import completion to report file readiness through the same service or event boundary.
- Ensure all ready-file paths call the post-file-ready automation service hook.
- Keep filesystem paths out of public responses.

Acceptance criteria:

- File handlers perform no database queries and no storage path policy decisions beyond streaming an approved file handle/result.
- Direct upload, video extraction, YouTube import, and recording finalizer all converge on one file-ready handoff.
- Existing file list/detail/audio contracts remain stable.

Testing focus:

- Upload success and cleanup on persistence failure.
- Video extraction success/failure.
- YouTube import completion.
- File-ready event publishing.
- Path traversal and path disclosure regressions.

## Sprint 5: Transcription Command Service Boundary

Goal: centralize transcription creation, profile resolution, queue commands, and state conflicts.

Tasks:

- Move transcription create/list/get/update/delete/cancel/retry logic into a transcription service.
- Move default profile resolution out of handlers.
- Ensure create-from-file and auto-create-from-file use the same domain command.
- Keep queue operations behind the transcription service or an injected queue command boundary.
- Preserve repository-owned state transitions for enqueue/cancel/retry.

Acceptance criteria:

- Transcription handlers perform no database queries.
- Manual transcription creation and automated transcription creation share profile resolution and validation.
- Queue state conflicts are returned as explicit domain errors and mapped by the handler.

Testing focus:

- Manual transcription creation with explicit and default profile.
- Missing default profile behavior.
- Queue stopped behavior.
- Cancel/retry state conflicts.
- Event payloads remain small and path-free.

## Sprint 6: Post-File Automation Service

Goal: implement the backend framework needed by the General settings features without wiring the frontend yet.

Tasks:

- Add a post-file automation service with a single entry point, for example `HandleFileReady(ctx, fileID, userID, source)`.
- Implement auto-transcribe decision logic using settings service and transcription command service.
- Implement auto-rename decision logic behind an LLM/title generation boundary.
- Decide whether auto-rename runs immediately from file metadata, after transcription completion, or after summary completion; document the chosen trigger before coding it.
- Ensure missing default profile or missing LLM small model is logged/no-op at runtime without failing file creation.
- Publish `file.updated` or transcription events only after durable persistence.

Acceptance criteria:

- A newly ready file can trigger automatic transcription through the default profile.
- Runtime missing configuration is a graceful no-op.
- Automation decisions are not made in HTTP handlers or import/finalizer workers.
- Automation can be tested with fake settings, transcription, and LLM/title services.

Testing focus:

- Auto-transcribe enabled with default profile.
- Auto-transcribe enabled but default profile missing.
- Auto-transcribe disabled.
- Auto-rename enabled with small LLM configured.
- Auto-rename enabled but LLM provider missing.
- Event publishing after durable update only.

## Sprint 7: Remove Remaining API Database Access and Harden Contracts

Goal: finish the architecture migration and make regressions hard.

Tasks:

- Remove all remaining `internal/database` imports from `internal/api`.
- Remove all remaining direct `database.DB` calls from handlers.
- Expand architecture guard coverage for API package import boundaries.
- Update route contract tests if any response shape deliberately changed.
- Add service-level tests for state-machine and validation behavior moved out of handlers.
- Run focused backend package tests and `git diff --check`.

Acceptance criteria:

- `rg 'internal/database|database\\.DB' internal/api` returns no production handler usage.
- All API handlers call services instead of repositories/database globals.
- Public API behavior remains stable except documented settings additions.
- The codebase is ready for the frontend General settings tab implementation.

Testing focus:

- Full `internal/api`, `internal/repository`, relevant service packages, recording, summarization, worker, and `cmd/server` tests.
- Architecture guard.
- Security regression tests for auth, path leakage, malformed JSON, and cross-resource access.

## Verification Commands

Use focused commands during each sprint:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/repository ./internal/recording ./internal/summarization ./internal/transcription/worker
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./cmd/server
git diff --check
rg 'internal/database|database\.DB' internal/api
```

The final `rg` command should return no production API database access. Tests may keep direct DB setup where appropriate, but production handlers must not.
