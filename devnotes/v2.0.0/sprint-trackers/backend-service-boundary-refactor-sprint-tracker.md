# Sprint Tracker: Backend Service Boundary Refactor

This tracker belongs to `devnotes/v2.0.0/sprint-plans/backend-service-boundary-refactor-sprint-plan.md`.

Status: in progress. Sprints 0 through 5 are complete; Sprint 6 is pending.

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

Status: completed

Completed tasks:

- [x] Move registration/login/refresh/logout/account commands into an account service.
- [x] Move password and username changes into the account service.
- [x] Move settings get/update into the account service.
- [x] Reuse existing user, refresh-token, API-key, profile, and LLM config repositories for account and settings persistence.
- [x] Add `auto_rename_enabled` to persisted user settings.
- [x] Validate `auto_transcription_enabled=true` requires a valid default profile.
- [x] Validate `auto_rename_enabled=true` requires an active LLM provider and configured small model.
- [x] Return the new setting field in settings responses.

Acceptance checks:

- [x] Auth handlers perform no database queries.
- [x] Settings handlers perform no database queries.
- [x] Password change behavior remains stable.
- [x] Invalid auto-transcribe and auto-rename enablement fails with structured validation errors.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestAPIKey|TestIdempotency|TestSettings|TestAuth|TestProductionAPIDatabaseAccessInventory|TestSecurity'`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/database ./internal/repository ./internal/account`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./cmd/server`
- [x] `git diff --check`

Artifacts:

- `internal/account/service.go`
- `internal/api/auth_handlers.go`
- `internal/api/api_key_handlers.go`
- `internal/api/settings_handlers.go`
- `internal/api/middleware.go`
- `internal/models/auth.go`
- `devnotes/v2.0.0/status-updates/backend-service-boundary-sprint-02-account-settings-notes.md`

## Sprint 3: Profile and LLM Provider Services

Status: completed

Completed tasks:

- [x] Move profile CRUD and set-default logic into a profile service.
- [x] Make default-profile mutation canonical through repository transaction methods.
- [x] Move LLM provider get/update/test/save logic into an LLM provider service.
- [x] Add active LLM readiness support for settings validation and automation through LLM config repository/service boundaries.
- [x] Remove provider/model persistence decisions from handlers.

Acceptance checks:

- [x] Profile handlers perform no database queries.
- [x] LLM provider handlers perform no database queries.
- [x] Default profile behavior is consistent across settings and recording validation; transcription default resolution remains scheduled for Sprint 5.
- [x] LLM small-model readiness is reusable outside the settings API.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestProfile|TestSettings|TestRecording|TestProductionAPIDatabaseAccessInventory'`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestLLMProviderSettingsEmptyAndAuth|TestProductionAPIDatabaseAccessInventory'`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/repository ./internal/profile ./internal/llmprovider ./cmd/server`
- [x] `git diff --check`

Notes:

- Full LLM provider API tests are blocked by sandbox loopback restrictions because provider connection tests use `httptest.NewServer`.

Artifacts:

- `internal/profile/service.go`
- `internal/llmprovider/service.go`
- `internal/api/profile_handlers.go`
- `internal/api/llm_provider_handlers.go`
- `internal/api/recording_handlers.go`
- `internal/repository/implementations.go`
- `devnotes/v2.0.0/status-updates/backend-service-boundary-sprint-03-profile-llm-notes.md`

## Sprint 4: File and Media Import Service Boundary

Status: completed

Planned tasks:

- [x] Add file service for upload/list/get/update/delete/audio lookup.
- [x] Move upload storage path construction out of handlers.
- [x] Move direct upload persistence into repository/service methods.
- [x] Move video extraction completion persistence behind service/repository methods.
- [x] Adapt YouTube import completion to report file readiness through the shared boundary.
- [x] Ensure direct upload, video extraction, YouTube import, and recording finalizer call one file-ready handoff.

Acceptance checks:

- [x] File handlers perform no database queries.
- [x] File handlers do not construct durable storage paths.
- [x] File-ready behavior is shared by all file creation paths.
- [x] File responses remain path-free.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestRecording|TestFile|TestProductionAPIDatabaseAccessInventory'`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/files ./internal/mediaimport ./internal/recording ./internal/repository ./cmd/server`
- [x] `git diff --check`

Artifacts:

- `internal/files/service.go`
- File repository method additions in `internal/repository/implementations.go`.
- Media import adapter changes.
- Recording finalizer handoff changes.
- `devnotes/v2.0.0/status-updates/backend-service-boundary-sprint-04-file-service-notes.md`

## Sprint 5: Transcription Command Service Boundary

Status: completed

Planned tasks:

- [x] Move transcription create/list/get/update/delete/cancel/retry into a transcription service.
- [x] Move default profile resolution out of handlers.
- [x] Make manual transcription creation and multipart submit share one command path ready for automation reuse.
- [x] Keep queue enqueue/cancel behavior behind service boundaries.
- [x] Map domain state conflicts to API errors in handlers.

Acceptance checks:

- [x] Transcription handlers perform no database queries.
- [x] Manual and future automatic transcription creation use the same validation and profile resolution path.
- [x] Queue errors and state conflicts are explicit domain errors.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestTranscription|TestCapabilitiesQueue|TestProductionAPIDatabaseAccessInventory'`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/transcription ./internal/transcription/worker ./internal/repository ./cmd/server`
- [x] `git diff --check`

Artifacts:

- `internal/transcription/service.go`
- Repository method additions in `internal/repository/implementations.go`.
- Handler injection and composition-root wiring.
- `devnotes/v2.0.0/status-updates/backend-service-boundary-sprint-05-transcription-notes.md`

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
