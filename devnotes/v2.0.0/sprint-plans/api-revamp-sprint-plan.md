# Scriberr API Revamp Sprint Plan

## Scope Guardrails

The API revamp is limited to API-facing code and API tests only. Do not modify frontend code, transcription engine internals, database migrations, model schemas, queue internals, LLM modules, notes/chat/summaries implementations, documentation site code, or unrelated services unless an API boundary cannot compile without a narrow adapter.

The new canonical API is `/api/v1` as defined in `devnotes/new-api-spec.md`. Legacy API routes and their tests should be removed instead of extended. Deferred modules such as summaries, chat, notes, speaker editing, webhooks, multi-track, and team administration should not be rebuilt during the first clean pass.

Transcription execution, YouTube import execution, video extraction, model discovery, queue internals, SSE backend, and logs backend may use explicit placeholders or `501 Not Implemented` responses where the spec allows it. Route shape, authentication, validation, response contracts, and security behavior should still be real.

## Engineering Standards

- Follow test-driven development: write or update the smallest meaningful tests first, implement the API behavior, then refactor.
- Keep tests focused on hot paths and failure modes. Avoid chasing 100% coverage.
- Security tests should be broader than feature tests and cover authentication, API key handling, path leakage, range abuse, malformed input, and authorization boundaries.
- Keep handlers thin: authentication, request parsing, validation, response formatting, and service calls only.
- Put resource and persistence logic behind API service interfaces so the HTTP layer can be tested independently.
- Use stable response envelopes: collections return `{ "items": [], "next_cursor": null }`; errors return `{ "error": { "code", "message", "field", "request_id" } }`.
- Never expose local filesystem paths in API responses, errors, logs returned through API endpoints, or tests.
- Use Zap for structured logging. Default log level should be low-noise for normal users, while development can enable debug-level request, validation, auth, and service traces through configuration.
- Treat the revamped database schema as the source of truth. API response IDs should be opaque and prefixed where new public resources require it, even if the current database uses UUIDs or numeric IDs internally.

## Sprint 0: API Inventory and Removal Plan

Goal: identify exactly what old API code and tests are removed before adding the new API surface.

Tasks:

- Inventory all current API handlers, middleware, route registrations, API docs artifacts, and API-specific tests.
- Classify existing routes as canonical replacement, temporary compatibility requirement, or delete.
- Identify compile dependencies from `cmd/server`, CLI auth/install endpoints, and static web serving before deleting files.
- Decide which existing tests are API-specific and should be removed with the old API surface.
- Capture the minimal internal interfaces required for the new API foundation.

Acceptance criteria:

- A route removal matrix exists in implementation notes or commit notes.
- No non-API package is selected for modification except through a narrow interface already exposed to API code.
- The first implementation branch can delete old API code without touching frontend or transcription internals.

Testing focus:

- No new product tests required in this sprint.
- Use compile checks to expose unexpected coupling after removal.

## Sprint 1: API Foundation

Goal: establish the clean `/api/v1` HTTP foundation and remove old API behavior.

Tasks:

- Remove legacy API handlers, route groups, generated old API docs, and corresponding API tests.
- Create a new API router with:
  - `GET /health`
  - `GET /api/v1/health`
  - `GET /api/v1/ready`
  - request ID middleware
  - panic recovery
  - structured error writer
  - JSON response helpers
  - CORS behavior scoped to current config
  - auth middleware hooks for JWT and API keys
- Replace the current logger package usage in API paths with Zap-backed structured logging.
- Add configurable log level, defaulting to a balanced production-friendly level and allowing debug logging for development.
- Add route skeletons for files, transcriptions, profiles, settings, events, models, and admin queue.
- Return `501 Not Implemented` for allowed placeholders while preserving error shape and request IDs.

Acceptance criteria:

- Old `/api/v1/transcription/*` routes are not registered as canonical routes.
- Health/readiness endpoints return the exact response shape from the spec.
- All API error responses include `error.code`, `error.message`, and `error.request_id`.
- `X-Request-ID` is accepted from clients and echoed in responses; missing IDs are generated.
- API logs include request ID, method, route, status, duration, and auth type when available.

Testing focus:

- Router smoke tests for health, readiness, unknown route, panic recovery, and request ID propagation.
- Error contract tests for malformed JSON, validation failure, unauthorized access, and placeholder routes.
- Logging configuration test for default level and debug override.

## Sprint 2: Authentication and API Keys

Goal: implement the security foundation for single-user JWT auth and script-friendly API keys.

Tasks:

- Implement the new auth response contracts:
  - registration status
  - register
  - login
  - refresh
  - logout
  - current user
  - change password
  - change username
- Ensure refresh tokens are stored hashed and rotated/revoked correctly.
- Implement API key list/create/delete using hashed storage.
- Return raw API keys only once at creation.
- Return public key previews, never hashes or full stored key material.
- Restrict account-management and API-key-management endpoints to JWT auth.
- Allow resource API endpoints to accept either `Authorization: Bearer` or `X-API-Key`.
- Normalize auth failures into the structured error shape.

Acceptance criteria:

- API keys and refresh tokens are never returned except for one-time API key creation and refresh-token response contracts.
- Invalid, revoked, expired, malformed, and missing credentials produce consistent `401` responses.
- JWT-only endpoints reject API keys with `403` or `401` according to the final middleware contract.
- Successful auth context resolves to the single current user without exposing multi-user concepts.

Testing focus:

- Registration/login/refresh/logout hot paths.
- Password and username validation failures.
- API key create/list/delete behavior.
- Security tests for raw key non-persistence, hash-only storage, revoked keys, expired refresh tokens, token reuse after logout, malformed bearer headers, missing credentials, and auth method restrictions.

## Sprint 3: Files and Range Streaming

Goal: implement file resources and secure audio streaming against the revamped database/storage model.

Tasks:

- Implement:
  - `POST /api/v1/files`
  - `POST /api/v1/files:import-youtube`
  - `GET /api/v1/files`
  - `GET /api/v1/files/{id}`
  - `PATCH /api/v1/files/{id}`
  - `DELETE /api/v1/files/{id}`
  - `GET /api/v1/files/{id}/audio`
- Map current persisted transcription/file data into the public File model without leaking `source_file_path`.
- Use opaque public IDs and stable response fields from the spec.
- Implement upload limits, media type validation, safe filenames, and storage path isolation.
- Implement HTTP range streaming without reading entire files into memory.
- Return `202` and a processing placeholder for YouTube import.

Acceptance criteria:

- File responses contain no absolute local paths.
- Full and partial audio streaming return correct `Content-Type`, `Accept-Ranges`, `Content-Length`, and `Content-Range` headers.
- Invalid ranges return `416` with `Content-Range: bytes */size`.
- Missing files, deleted files, unsupported media, oversized uploads, and invalid IDs return structured errors.

Testing focus:

- Upload success and validation failures.
- List/get/update/delete hot paths.
- Full audio stream, bounded range, suffix range, open-ended range, invalid range, and range beyond EOF.
- Security tests for path traversal filenames, path disclosure in errors, unauthenticated streaming, unsupported content type, oversized body, and memory-safe streaming behavior where practical.

## Sprint 4: Transcriptions API Skeleton and Transcript Reads

Goal: provide the canonical transcription resource API while keeping execution internals placeholder-safe.

Tasks:

- Implement:
  - `POST /api/v1/transcriptions`
  - `POST /api/v1/transcriptions:submit`
  - `GET /api/v1/transcriptions`
  - `GET /api/v1/transcriptions/{id}`
  - `PATCH /api/v1/transcriptions/{id}`
  - `DELETE /api/v1/transcriptions/{id}`
  - `POST /api/v1/transcriptions/{id}:cancel`
  - `POST /api/v1/transcriptions/{id}:retry`
  - `GET /api/v1/transcriptions/{id}/transcript`
  - `GET /api/v1/transcriptions/{id}/audio`
  - `GET /api/v1/transcriptions/{id}/events`
  - `GET /api/v1/transcriptions/{id}/logs`
  - `GET /api/v1/transcriptions/{id}/executions`
- Enqueue or stub transcription work according to the spec.
- Validate `file_id`, `profile_id`, language, diarization, pagination, sorting, and status filters.
- Map the revamped `transcriptions` and `transcription_executions` schema into public response models.
- Sanitize logs and execution errors before returning them.
- Keep multi-track behavior out of scope.

Acceptance criteria:

- Create and submit return `202 Accepted` and queued transcription resources.
- Transcript responses match `{ transcription_id, text, segments, words }`, with placeholders allowed if execution is not implemented.
- Cancel/retry have deterministic behavior for unsupported or invalid states.
- Transcription audio is a secure alias to source-file audio behavior.
- Events/logs/executions route shapes exist, with `501` where backend support is deferred.

Testing focus:

- Create/list/get/update/delete hot paths.
- Validation tests for missing file, invalid profile, invalid status filter, invalid sort, invalid language, malformed options JSON, and state conflicts.
- Security tests for unauthenticated access, cross-resource ID probing, path leakage through transcript/log errors, and invalid range through transcription audio alias.

## Sprint 5: Profiles, Settings, Capabilities, and Admin

Goal: complete the foundation endpoints that make the API usable without exposing deferred product modules.

Tasks:

- Implement profiles:
  - list/create/get/update/delete
  - set default
- Implement settings:
  - get settings
  - update settings
- Implement events:
  - `GET /api/v1/events` as real SSE if the existing broadcaster can be adapted only through API boundaries, otherwise `501`.
- Implement models/capabilities:
  - `GET /api/v1/models/transcription` with static/local placeholder data if discovery is deferred.
- Implement admin:
  - `GET /api/v1/admin/queue` with queue stats if available through existing API-safe interfaces, otherwise a clear placeholder.

Acceptance criteria:

- Profiles preserve single default profile semantics.
- Settings expose only single-user application settings from the spec.
- Capabilities do not imply silent cloud fallback.
- Admin queue endpoint does not expose worker internals or filesystem paths.

Testing focus:

- Profile CRUD and set-default behavior.
- Settings patch validation and partial update behavior.
- Security tests for auth-required endpoints and local-only/cloud-provider privacy expectations.

## Sprint 6: OpenAPI, Compatibility Decision, and Cleanup

Goal: finalize the API surface and make it maintainable.

Tasks:

- Generate or hand-maintain API documentation for only the new canonical API.
- Remove stale generated docs for deleted routes.
- Decide whether any temporary legacy aliases are required for the current frontend or CLI. If needed, implement aliases as thin adapters to canonical services and mark them temporary.
- Audit all API responses for stable shapes, path leakage, timestamp formatting, and status code consistency.
- Audit API package boundaries for handler bloat and misplaced business logic.
- Run full API/security test suite and a project compile check.

Acceptance criteria:

- API docs match implemented route shapes and status codes.
- No deleted legacy route appears in API docs.
- Any legacy alias has an explicit removal note and is not used as a design anchor.
- API package is organized around middleware, transport models, handlers, service interfaces, and tests.

Testing focus:

- Contract smoke tests against route registration.
- Golden-ish response shape tests for representative resources and errors.
- Security regression suite for auth, path leakage, request smuggling-style malformed input where applicable, oversized payloads, range handling, and sanitized logs.

## Suggested Test Budget

Keep the first pass lean:

- Foundation/router: 8-12 tests.
- Auth/API keys: 15-25 tests, with most being security-oriented.
- Files/range streaming: 12-18 tests.
- Transcriptions: 10-16 tests.
- Profiles/settings/admin/capabilities: 8-12 tests.
- Cross-cutting security regression suite: 20-30 table-driven cases.

Prefer table-driven tests and shared test fixtures over many near-duplicate tests.

## Initial File Boundary

Expected API-owned areas:

- `internal/api/**`
- `pkg/middleware/**` only where API middleware is being replaced or adapted.
- `pkg/logger/**` only for Zap-backed API logging changes, unless a new API-local logger package is cleaner.
- API-specific tests under `tests/*api*`, `tests/security_test.go`, `tests/cli_handlers_test.go` if legacy API behavior is removed, and any new `internal/api/**/*_test.go`.
- `api-docs/**` only when old API docs are removed or new API docs are generated.

Avoid modifying:

- `web/**`
- `internal/web/**`
- `internal/transcription/**` except through existing interfaces or placeholders.
- `internal/database/**` and `internal/models/**` unless the user explicitly approves API-required schema/model changes.
- `internal/llm/**`, `internal/queue/**`, `internal/sse/**`, `internal/processing/**`, `internal/audio/**`, and CLI internals unless a compile-only adapter is unavoidable.

---

# Follow-on API Completion Sprint Plan

## Follow-on Scope Guardrails

This sprint set continues from Sprint 6 and is meant to complete the API-facing spec without rebuilding the transcription engine itself.

Included:

- Split `internal/api/router.go` into maintainable API modules and service boundaries.
- Rich list filtering, pagination, cursor behavior, sorting, and query validation.
- Idempotency handling for mutating API endpoints.
- Real SSE events for API-visible file/transcription/profile/settings changes.
- Real YouTube import/download behavior through a narrow API-safe adapter.
- API documentation and contract tests for the completed API surface.

Explicitly deferred outside this sprint set:

- Actual transcription execution.
- Transcription logs backend.
- Transcription executions backend.

Deferred endpoints must stay explicit and honest. `GET /api/v1/transcriptions/{id}/logs` and `GET /api/v1/transcriptions/{id}/executions` should keep stable route shapes, authentication, structured error responses, and sanitized placeholders until their backends are intentionally implemented in a later engine/backend sprint.

## Follow-on Engineering Standards

- Continue TDD. Write the smallest meaningful failing tests first.
- Keep commits small and grouped by behavior: refactor-only, feature, docs, and test-support changes should be separate where practical.
- Do not stage or commit `devnotes/**`.
- Prefer real local tests with temporary SQLite databases, temporary upload directories, fake media files, and local test HTTP servers.
- Use mocks only for hard external boundaries such as YouTube/network execution and long-running streams.
- Keep API handlers thin. Handlers should perform auth, parsing, validation, response mapping, and service delegation only.
- Keep response IDs opaque and prefixed.
- Preserve stable error shape and request ID propagation.
- Avoid path leakage in responses, errors, SSE payloads, and docs.
- Maintain a clear `go test -v` endpoint contract table so API coverage is visible in test output.

## Sprint 7: API Modularization and Service Boundaries

Goal: split the large API router into maintainable modules without changing behavior.

Tasks:

- Split `internal/api/router.go` into focused files:
  - `router.go` for route registration and shared handler construction.
  - `middleware.go` for request ID, recovery, CORS, and auth middleware.
  - `errors.go` for structured errors and request IDs.
  - `auth_handlers.go`
  - `api_key_handlers.go`
  - `file_handlers.go`
  - `transcription_handlers.go`
  - `profile_handlers.go`
  - `settings_handlers.go`
  - `events_handlers.go`
  - `admin_handlers.go`
  - `response_models.go`
- Introduce narrow service interfaces where the handler currently mixes transport and persistence heavily:
  - file service
  - transcription service
  - profile service
  - settings service
  - events publisher
- Keep the first refactor mostly mechanical. Avoid changing behavior unless tests expose a defect.
- Preserve the public `NewHandler` and `SetupRoutes` entry points unless there is a clear compile-safe replacement.

Acceptance criteria:

- Route registration and all existing behavior remain unchanged.
- `internal/api/router.go` becomes small enough to read as route composition rather than business logic.
- Existing tests pass without weakening assertions.
- No non-API packages are modified unless needed for compile-safe API interfaces.

Testing focus:

- Existing API suite must pass before and after each major file move.
- Route contract test must prove no route changed accidentally.
- Add no broad new feature tests unless a refactor exposes an actual bug.

## Sprint 8: List Filters, Pagination, Sorting, and Query Validation

Goal: implement the richer collection query behavior from the spec for files and transcriptions, with foundations reusable by profiles where useful.

Tasks:

- Implement shared query parsing helpers for:
  - `limit`
  - `cursor`
  - `q`
  - `status`
  - `kind`
  - `updated_after`
  - `sort`
- Implement stable cursor pagination for:
  - `GET /api/v1/files`
  - `GET /api/v1/transcriptions`
- Implement filtering:
  - files: `kind`, `q`, `updated_after`
  - transcriptions: `status`, `q`, `updated_after`
- Implement sorting:
  - `created_at`
  - `-created_at`
  - `updated_at`
  - `-updated_at`
  - `title`
  - `-title`
- Return structured `422` errors for invalid query params.
- Keep cursor values opaque enough for public API use.

Acceptance criteria:

- Collection responses keep `{ "items": [], "next_cursor": null|string }`.
- Invalid limit, cursor, status, kind, timestamp, or sort values fail consistently with field-specific errors.
- Pagination order is deterministic, including records with identical timestamps.
- File list continues to exclude transcription rows.
- Transcription list continues to exclude uploaded source-file rows.

Testing focus:

- Table-driven query validation tests.
- Pagination hot path with more records than the page limit.
- Sort/filter interaction tests for representative cases.
- Security test for malformed cursor payloads and path-leak-free errors.

## Sprint 9: Idempotency for Mutating API Endpoints

Goal: implement safe retry behavior for client-visible create/update command endpoints.

Tasks:

- Define idempotency semantics for `Idempotency-Key`.
- Add middleware/service support for idempotency keys on:
  - `POST /api/v1/files`
  - `POST /api/v1/files:import-youtube`
  - `POST /api/v1/transcriptions`
  - `POST /api/v1/transcriptions:submit`
  - `POST /api/v1/transcriptions/{id}:cancel`
  - `POST /api/v1/transcriptions/{id}:retry`
  - `POST /api/v1/profiles`
  - `POST /api/v1/profiles/{id}:set-default`
  - `POST /api/v1/api-keys`
- Store idempotency state using an API-owned persistence approach compatible with the existing database model constraints.
- If schema changes are required, pause and ask for approval before modifying models/migrations.
- Bind idempotency keys to user, method, path, and request fingerprint.
- Return the original response for exact retries.
- Return `409 Conflict` or `422 Validation Error` for key reuse with a different request fingerprint.
- Expire or safely bound stored idempotency records if persistence support exists.

Acceptance criteria:

- Retrying the same create request with the same idempotency key does not create duplicate resources.
- Reusing the same key with a different request body is rejected.
- Idempotency records never expose raw request bodies, filesystem paths, API keys, or tokens.
- Endpoints still work normally without `Idempotency-Key`.

Testing focus:

- Duplicate upload/transcription/profile/API-key create retries.
- Key reuse with changed body.
- User isolation for same key.
- Malformed or overlong idempotency keys.
- Concurrent duplicate request behavior where practical.

## Sprint 10: Real SSE Event Stream

Goal: implement `GET /api/v1/events` and transcription event routes as real authenticated SSE streams for API-visible state changes.

Tasks:

- Add an API-local event broker or adapt an existing broadcaster only through a narrow API-safe interface.
- Implement:
  - `GET /api/v1/events`
  - `GET /api/v1/transcriptions/{id}/events`
- Publish events for API-visible state changes:
  - `file.ready`
  - `file.processing`
  - `file.deleted`
  - `transcription.created`
  - `transcription.updated`
  - `transcription.canceled`
  - `transcription.completed` when status is set by API/test fixtures
  - `profile.updated`
  - `settings.updated`
- Include request-safe payloads with public IDs only.
- Support disconnect cleanup and avoid goroutine leaks.
- Send heartbeat/comment events often enough for clients/proxies to keep the stream alive.
- Keep event history optional; if replay is not supported, document that `Last-Event-ID` is ignored for now.

Acceptance criteria:

- SSE responses use `Content-Type: text/event-stream`.
- Unauthenticated streams return structured `401` errors.
- Events do not expose local paths, internal database IDs, tokens, or API key material.
- Per-transcription stream only emits events for the requested transcription.
- Disconnecting clients are removed from broker subscriber lists.

Testing focus:

- SSE headers and initial connection behavior.
- Publish-and-receive for file and transcription changes.
- Per-resource event filtering.
- Auth-required behavior.
- Disconnect cleanup with timeout-safe tests.
- Path leakage checks in event payloads.

## Sprint 11: Real YouTube Import API

Goal: replace the YouTube processing placeholder with a real API-visible import flow while keeping external execution isolated and testable.

Tasks:

- Introduce a `YouTubeImporter` interface owned by the API/service layer.
- Implement a production adapter using the project-approved downloader mechanism.
- Keep downloader invocation outside handlers and behind context-aware service methods.
- Validate YouTube URLs and reject unsupported schemes/hosts.
- Store imported media under the configured upload directory with safe generated filenames.
- Create a file resource with:
  - `kind: "youtube"`
  - `status: "processing"` while import is running
  - `status: "ready"` after successful download
  - `status: "failed"` on failed download if the existing model can represent it safely
- Publish SSE events for processing, ready, and failed states.
- Avoid logging full URLs if they may include sensitive query params; log sanitized host/video ID only.
- Do not implement actual transcription execution as part of this sprint.

Acceptance criteria:

- `POST /api/v1/files:import-youtube` creates a file import job and returns `202`.
- Successful import produces an audio/video streamable file resource.
- Failed import returns/surfaces a sanitized failure state without leaking command lines or local paths.
- The API can be tested with a fake importer and a local test HTTP server; no real YouTube network dependency is required in unit tests.

Testing focus:

- Valid import request creates processing file resource.
- Fake importer success transitions to ready and streamable.
- Fake importer failure transitions to failed with sanitized error.
- URL validation rejects unsupported domains/schemes.
- Duplicate/idempotent import behavior after Sprint 9.
- SSE event emission on state transitions.

## Sprint 12: Contract, Documentation, and Security Hardening

Goal: finalize this follow-on API pass and decide whether another sprint set is needed.

Tasks:

- Update `docs/api/openapi.json` for all implemented behavior:
  - query params
  - pagination
  - idempotency
  - SSE routes
  - YouTube import lifecycle
- Expand the route contract table so `go test -v` clearly shows major method/path coverage.
- Add a small cross-cutting security regression suite for:
  - auth-required routes
  - path leakage
  - malformed cursors
  - malformed idempotency keys
  - oversized/malformed JSON and multipart requests
  - SSE auth and disconnect behavior
- Audit API responses for spec consistency and stable field names.
- Audit the API package boundaries after modularization.
- Produce a final gap list for deferred work:
  - actual transcription execution
  - transcription logs backend
  - transcription executions backend

Acceptance criteria:

- API docs match implemented status codes and route behavior.
- No stale legacy route appears in docs or route registration.
- All completed sprint tests pass with clean test output.
- `go vet` passes for touched packages.
- Any remaining gaps are explicit and planned, not accidental placeholders.

Testing focus:

- Golden-ish representative response shape tests.
- Contract smoke tests for method/path/status visibility.
- Security table tests, focused on high-risk API behavior rather than broad coverage percentage.
- Full completed-sprints command:
  - `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./cmd/server ./pkg/logger ./pkg/middleware`
  - `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./pkg/logger ./pkg/middleware ./cmd/server`

## Deferred Engine/Backend Sprint Set

These are intentionally not part of the follow-on API completion sprint set above:

- Actual transcription execution.
- Transcription logs backend.
- Transcription executions backend.

When approved, plan them as a separate engine/backend sprint set because they likely touch queue internals, transcription processing, log storage, execution metadata, and possibly model/schema behavior outside the API-only boundary.
