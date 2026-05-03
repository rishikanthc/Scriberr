# Sprint Tracker: Backend Service Boundary Refactor

This tracker belongs to `devnotes/v2.0.0/sprint-plans/backend-service-boundary-refactor-sprint-plan.md`.

Status: in progress. Sprints 0 and 1 are complete; Sprint 2 is pending.

## Sprint 0: Inventory, Dependency Map, and Stop-The-Line Guard

Status: completed

Completed tasks:

- [x] Inventory every `internal/api` import of `internal/database`.
- [x] Inventory every direct `database.DB` use in production API handlers.
- [x] Classify direct DB usage by domain: auth, settings, profiles, files, transcriptions, LLM provider, events/admin.
- [x] Identify reusable repository methods and missing domain repository methods.
- [x] Add an architecture guard that blocks new `internal/api` imports of `internal/database` by freezing the current inventory.
- [x] Document API-facing service interfaces for the refactor.

Acceptance checks:

- [x] Direct database usage inventory exists in status notes.
- [x] Guard fails if new production API DB access is introduced.
- [x] Existing route contract remains stable.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory'`
- [x] `rg 'internal/database|database\.DB' internal/api`
- [x] `git diff --check`

Artifacts:

- `devnotes/v2.0.0/status-updates/backend-service-boundary-sprint-00-inventory.md`
- `internal/api/architecture_test.go`

## Sprint 1: Composition Root and Handler Dependency Injection

Status: completed

Completed tasks:

- [x] Define narrow API-facing service interfaces consumed by handlers.
- [x] Update `Handler` to receive required services explicitly.
- [x] Remove fallback repository construction from `api.NewHandler`.
- [x] Wire concrete services from `cmd/server/main.go`.
- [x] Update API test server setup to inject dependencies intentionally.
- [x] Keep event publisher interfaces narrow and explicit.

Acceptance checks:

- [x] `api.NewHandler` no longer constructs repositories from `database.DB`.
- [x] Test setup owns handler dependencies.
- [x] No public API behavior changes in focused route/API smoke coverage.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestProductionAPIDatabaseAccessInventory|TestCanonicalRouteRegistration|TestEndpointContractSmoke'`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./cmd/server`
- [x] `git diff --check`

Notes:

- Full `go test ./internal/api` was blocked by sandbox loopback restrictions in an LLM provider test that starts `httptest.NewServer`.

Artifacts:

- `internal/api/router.go`
- `cmd/server/main.go`
- API test setup files.
- `devnotes/v2.0.0/status-updates/backend-service-boundary-sprint-01-composition-notes.md`

## Sprint 2: Account and Settings Services

Status: pending

Planned tasks:

- [ ] Move registration/login/refresh/logout/account commands into an account service.
- [ ] Move password and username changes into the account service.
- [ ] Move settings get/update into a settings service.
- [ ] Add user repository methods for account and settings persistence.
- [ ] Add `auto_rename_enabled` to persisted user settings.
- [ ] Validate `auto_transcription_enabled=true` requires a valid default profile.
- [ ] Validate `auto_rename_enabled=true` requires an active LLM provider and configured small model.
- [ ] Return capability metadata needed to disable invalid settings toggles.

Acceptance checks:

- [ ] Auth handlers perform no database queries.
- [ ] Settings handlers perform no database queries.
- [ ] Password change behavior remains stable.
- [ ] Invalid auto-transcribe and auto-rename enablement fails with structured validation errors.

Verification:

- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestAuth|TestSettings|TestSecurity'`
- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/database ./internal/repository`
- [ ] `git diff --check`

Artifacts:

- Account service files.
- Settings service files.
- User repository changes.
- Migration/schema updates if needed for settings persistence.

## Sprint 3: Profile and LLM Provider Services

Status: pending

Planned tasks:

- [ ] Move profile CRUD and set-default logic into a profile service.
- [ ] Make default-profile resolution canonical for all workflows.
- [ ] Move LLM provider get/update/test/save logic into an LLM provider service.
- [ ] Add active LLM readiness methods for settings validation and automation.
- [ ] Remove provider/model validation decisions from handlers.

Acceptance checks:

- [ ] Profile handlers perform no database queries.
- [ ] LLM provider handlers perform no database queries.
- [ ] Default profile behavior is consistent across settings, transcriptions, and recordings.
- [ ] LLM small-model readiness is reusable outside the settings API.

Verification:

- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestProfile|TestLLMProvider|TestSettings'`
- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/repository`
- [ ] `git diff --check`

Artifacts:

- Profile service files.
- LLM provider service files.
- Repository method additions.

## Sprint 4: File and Media Import Service Boundary

Status: pending

Planned tasks:

- [ ] Add file service for upload/list/get/update/delete/audio lookup.
- [ ] Move upload storage path construction out of handlers.
- [ ] Move direct upload persistence into repository/service methods.
- [ ] Move video extraction completion persistence behind service/repository methods.
- [ ] Adapt YouTube import completion to report file readiness through the shared boundary.
- [ ] Ensure direct upload, video extraction, YouTube import, and recording finalizer call one file-ready handoff.

Acceptance checks:

- [ ] File handlers perform no database queries.
- [ ] File handlers do not construct durable storage paths.
- [ ] File-ready behavior is shared by all file creation paths.
- [ ] File responses remain path-free.

Verification:

- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestFile|TestRecording'`
- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/mediaimport ./internal/recording`
- [ ] `git diff --check`

Artifacts:

- File service files.
- File repository method additions.
- Media import adapter changes.
- Recording finalizer handoff changes.

## Sprint 5: Transcription Command Service Boundary

Status: pending

Planned tasks:

- [ ] Move transcription create/list/get/update/delete/cancel/retry into a transcription service.
- [ ] Move default profile resolution out of handlers.
- [ ] Make manual and automatic transcription creation share one command.
- [ ] Keep queue enqueue/cancel behavior behind service boundaries.
- [ ] Map domain state conflicts to API errors in handlers.

Acceptance checks:

- [ ] Transcription handlers perform no database queries.
- [ ] Manual and automatic transcription creation use the same validation and profile resolution path.
- [ ] Queue errors and state conflicts are explicit domain errors.

Verification:

- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestTranscription|TestCapabilitiesQueue'`
- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/transcription/worker ./internal/repository`
- [ ] `git diff --check`

Artifacts:

- Transcription service files.
- Repository method additions.
- Queue command adapter changes if needed.

## Sprint 6: Post-File Automation Service

Status: pending

Planned tasks:

- [ ] Add a post-file automation service entry point for newly ready files.
- [ ] Implement auto-transcribe decisions using settings and transcription services.
- [ ] Add graceful runtime no-op for missing default profile.
- [ ] Define and document the auto-rename trigger: immediate file-ready, transcription-completed, or summary-completed.
- [ ] Implement auto-rename decisions using LLM readiness and title generation boundary.
- [ ] Add fakeable dependencies for settings, transcription, and title generation.
- [ ] Publish events only after durable persistence.

Acceptance checks:

- [ ] Newly ready audio can trigger automatic transcription through the default profile.
- [ ] Runtime missing configuration does not fail file creation/import/finalization.
- [ ] Automation decisions do not live in handlers or workers.
- [ ] Auto-rename behavior is documented before implementation.

Verification:

- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api ./internal/recording ./internal/mediaimport`
- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/summarization`
- [ ] `git diff --check`

Artifacts:

- Post-file automation service files.
- Service tests with fakes.
- Status note documenting auto-rename trigger.

## Sprint 7: Remove Remaining API Database Access and Harden Contracts

Status: pending

Planned tasks:

- [ ] Remove all remaining `internal/database` imports from production `internal/api` files.
- [ ] Remove all remaining direct `database.DB` calls from production API handlers.
- [ ] Expand architecture guard coverage.
- [ ] Add service-level tests for migrated state-machine and validation behavior.
- [ ] Update route contract tests for documented settings additions.
- [ ] Run focused backend verification.

Acceptance checks:

- [ ] Production API package has no direct database access.
- [ ] Handlers call services, not repositories or database globals.
- [ ] General settings backend prerequisites are ready for frontend implementation.
- [ ] Public API behavior remains stable except documented settings additions.

Verification:

- [ ] `rg 'internal/database|database\.DB' internal/api`
- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api ./internal/repository ./internal/recording ./internal/summarization ./internal/transcription/worker ./cmd/server`
- [ ] `git diff --check`

Artifacts:

- Architecture guard test/script.
- Updated API service integrations.
- Final status note.
