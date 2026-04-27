# Sprint Tracker

## Sprint 0: API Inventory and Removal Plan

Status: completed

Completed tasks:

- Inventoried current API-owned files, API-adjacent middleware/logging, generated docs, and API tests.
- Classified current routes for canonical replacement, deletion, or possible temporary compatibility.
- Documented compile-sensitive dependencies from `cmd/server`, static web routing, and CLI upload behavior.
- Defined minimal Sprint 1 API interfaces.

Artifacts:

- `devnotes/sprints.md`
- `devnotes/api-sprint-0-inventory.md`

Verification:

- Compile/test verification was blocked because Go is not installed in this environment.

## Sprint 1: API Foundation

Status: completed

Completed tasks:

- Removed legacy API handlers, generated old API docs, and old API tests.
- Added canonical `/api/v1` route skeleton.
- Added health/readiness endpoints.
- Added request ID middleware and structured error contract.
- Added auth guard foundation for JWT and API keys.
- Added Zap-backed structured logging.
- Added API foundation tests before implementation.

Artifacts:

- `internal/api/router.go`
- `internal/api/router_test.go`
- `pkg/logger/logger.go`
- `devnotes/api-sprint-1-notes.md`

Verification:

- `git diff --check` passed.
- `go test` and `gofmt` were blocked because Go is not installed in this environment.

## Sprint 2: Authentication and API Keys

Status: completed

Completed tasks:

- Added focused Sprint 2 tests before implementation.
- Implemented auth registration status, register, login, refresh, logout, `/me`, change password, and change username.
- Implemented refresh-token hashing, rotation, and revocation behavior.
- Implemented API key create/list/delete.
- Implemented one-time raw API key return and redacted list responses.
- Enforced JWT-only access for account and API-key management.
- Allowed API keys on protected resource placeholders.

Verification:

- `git diff --check` passed.
- `go test` and `gofmt` were blocked because Go is not installed in this environment.

Artifacts:

- `internal/api/auth_test.go`
- `internal/api/router.go`
- `devnotes/api-sprint-2-notes.md`

## Sprint 3: Files and Range Streaming

Status: completed

Completed tasks:

- Added focused file API and range streaming tests before implementation.
- Implemented authenticated file upload, list, get, update, and delete routes.
- Implemented file response mapping with `file_` public IDs and no path leakage.
- Implemented safe filename handling and basic media type validation.
- Implemented authenticated audio streaming with full and ranged responses.
- Kept implementation inside the API layer and existing database model boundaries.

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `git diff --check` passed.

Artifacts:

- `internal/api/files_test.go`
- `internal/api/router.go`
- `devnotes/api-sprint-3-notes.md`

## Sprint 4: Transcriptions API Skeleton

Status: completed

Completed tasks:

- Added focused transcription API tests before implementation.
- Implemented authenticated transcription create, list, get, update, cancel, retry, and delete routes.
- Implemented transcript read endpoint for completed placeholder transcripts.
- Implemented transcription audio alias streaming through the existing uploaded audio file.
- Used `tr_` public IDs for transcriptions and `file_` public IDs for uploaded source files.
- Kept transcription execution, logs, and events as explicit placeholders for later sprints.

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `git diff --check` passed.

Artifacts:

- `internal/api/transcriptions_test.go`
- `internal/api/router.go`
- `devnotes/api-sprint-4-notes.md`

## Sprint 5: Profiles, Settings, Capabilities, and Admin

Status: completed

Completed tasks:

- Added focused Sprint 5 tests before implementation.
- Implemented authenticated profile list, create, get, update, delete, and set-default routes.
- Preserved single default profile semantics for each user.
- Implemented authenticated settings get and partial update routes.
- Validated default profile references and small transcription profile options.
- Implemented local transcription model capability response.
- Implemented queue stats from transcription job status counts.
- Kept the global events stream as an explicit placeholder.

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestProfile|TestSettings|TestCapabilities'` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `git diff --check` passed.

Artifacts:

- `internal/api/profile_settings_test.go`
- `internal/api/router.go`
- `devnotes/api-sprint-5-notes.md`

## Sprint 6: OpenAPI, Compatibility Decision, and Cleanup

Status: completed

Completed tasks:

- Added explicit route registration contract tests for the canonical `/api/v1` surface.
- Added endpoint smoke tests with method and path names visible in verbose test output.
- Added docs regression tests to keep deleted legacy transcription routes out of API docs.
- Replaced stale generated API docs with a concise canonical `docs/api/openapi.json`.
- Kept legacy singular transcription routes absent rather than adding temporary aliases.
- Suppressed expected GORM record-not-found noise in API tests.

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestAPIDocsContainOnlyCanonicalRoutes'` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `git diff --check` passed.

Artifacts:

- `internal/api/route_contract_test.go`
- `internal/api/auth_test.go`
- `docs/api/openapi.json`
- `devnotes/api-sprint-6-notes.md`

## Sprint 7: API Modularization and Service Boundaries

Status: completed

Completed tasks:

- Split the large API router into focused API modules by domain.
- Kept `internal/api/router.go` focused on handler construction, route registration, and static fallback routing.
- Moved middleware, error helpers, response models, auth, API keys, files, transcriptions, profiles, settings, admin, and health handlers into separate files.
- Preserved public API entry points `NewHandler` and `SetupRoutes`.
- Preserved route registration and behavior through the existing route contract suite.
- Deferred non-empty service-interface extraction to Sprint 8 query/list work, where the abstraction will remove real duplication instead of adding unused scaffolding.

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run TestCanonicalRouteRegistration` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./pkg/logger ./pkg/middleware ./cmd/server` passed.
- `git diff --check` passed.

Artifacts:

- `internal/api/router.go`
- `internal/api/middleware.go`
- `internal/api/errors.go`
- `internal/api/types.go`
- `internal/api/response_models.go`
- `internal/api/auth_handlers.go`
- `internal/api/api_key_handlers.go`
- `internal/api/file_handlers.go`
- `internal/api/transcription_handlers.go`
- `internal/api/profile_handlers.go`
- `internal/api/settings_handlers.go`
- `internal/api/admin_handlers.go`
- `internal/api/health_handlers.go`

## Sprint 8: List Filters, Pagination, Sorting, and Query Validation

Status: completed

Completed tasks:

- Added focused list-query tests before implementation for files and transcriptions.
- Implemented shared list query parsing for `limit`, `cursor`, `q`, `updated_after`, and `sort`.
- Implemented opaque cursor pagination with deterministic sort and ID tie-break behavior.
- Implemented file filters for `kind`, `q`, and `updated_after`.
- Implemented transcription filters for `status`, `q`, and `updated_after`.
- Implemented sorting by `created_at`, `updated_at`, and `title`, including descending variants.
- Returned structured `422` errors for invalid limits, cursors, kinds, statuses, timestamps, and sort values.

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestFileListFiltersSortingPaginationAndValidation|TestTranscriptionListFiltersSortingPaginationAndValidation'` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./pkg/logger ./pkg/middleware ./cmd/server` passed.
- `git diff --check` passed.

Artifacts:

- `internal/api/list_query.go`
- `internal/api/file_handlers.go`
- `internal/api/transcription_handlers.go`
- `internal/api/files_test.go`
- `internal/api/transcriptions_test.go`

## Sprint 9: Idempotency for Mutating API Endpoints

Status: completed

Completed tasks:

- Added focused idempotency tests before implementation.
- Implemented API-local idempotency handling for key mutating routes.
- Cached successful responses for exact retries with the same `Idempotency-Key`.
- Rejected key reuse with a different request fingerprint.
- Validated idempotency keys and returned field-specific structured errors.
- Covered JSON create requests, multipart file upload, and transcription create hot paths.
- Applied idempotency handling to API key create, file upload/import, transcription create/submit/cancel/retry, profile create, and profile set-default routes.

Follow-up notes:

- Current idempotency is API-local process memory. Durable idempotency across server restarts or multiple server processes requires an approved persistence/schema change.

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestIdempotency'` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./pkg/logger ./pkg/middleware ./cmd/server` passed.
- `git diff --check` passed.

Artifacts:

- `internal/api/idempotency.go`
- `internal/api/idempotency_test.go`
- `internal/api/router.go`
- `internal/api/middleware.go`

## Sprint 10: Real SSE Event Stream

Status: completed

Completed tasks:

- Added focused SSE tests before implementation.
- Implemented an API-local authenticated event broker.
- Implemented `GET /api/v1/events` as a real SSE stream.
- Implemented `GET /api/v1/transcriptions/{id}/events` as a filtered per-transcription SSE stream.
- Published safe public-ID-only events for file, transcription, profile, and settings changes.
- Added disconnect cleanup behavior and subscriber-count assertions in tests.
- Updated previous placeholder tests to target remaining deferred logs routes instead of the now-real events endpoint.

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test -timeout 20s ./internal/api ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./pkg/logger ./pkg/middleware ./cmd/server` passed.
- `git diff --check` passed.

Artifacts:

- `internal/api/events_handlers.go`
- `internal/api/events_test.go`
- `internal/api/router.go`
- `internal/api/file_handlers.go`
- `internal/api/transcription_handlers.go`
- `internal/api/profile_handlers.go`
- `internal/api/settings_handlers.go`
- `internal/api/router_test.go`
- `internal/api/route_contract_test.go`
- `internal/api/profile_settings_test.go`
