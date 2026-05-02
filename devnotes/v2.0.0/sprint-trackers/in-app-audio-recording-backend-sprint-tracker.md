# Sprint Tracker: In-App Audio Recording Backend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/in-app-audio-recording-backend-sprint-plan.md`.

Status: completed through Sprint 4. Recording schema, config, storage, repository, service state machine, HTTP API, and durable finalizer handoff are in place.

## Sprint 1: Schema, Config, and Storage Boundary

Status: completed

Completed tasks:

- [x] Add `models.RecordingSession` and `models.RecordingChunk`.
- [x] Register recording models in the target schema and bump schema version to 8.
- [x] Add recording indexes and uniqueness constraints for session listing, finalizer claims, cleanup, and chunk idempotency.
- [x] Add relational associations for recording owner, chunks, finalized file, optional transcription, and optional profile.
- [x] Add typed recording config defaults, validation, and startup logging.
- [x] Add recording storage abstraction for chunk/final artifact paths and cleanup.
- [x] Add database, migration, config, and storage tests.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/database ./internal/config ./internal/recording`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/database ./internal/config ./internal/recording ./cmd/server`

Artifacts:

- `internal/models/recording.go`
- `internal/database/schema.go`
- `internal/database/steps.go`
- `internal/database/database_test.go`
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/recording/storage.go`
- `internal/recording/storage_test.go`
- `cmd/server/main.go`

## Sprint 2: Repository and Recording Service State Machine

Status: completed

Completed tasks:

- [x] Add `RecordingRepository` with domain-specific session/chunk/finalization methods.
- [x] Add `recording.Service` commands for create, append chunk, stop, cancel, retry, get, and list.
- [x] Enforce MIME type, chunk index, size, duration, ownership, and status validation.
- [x] Implement idempotent chunk retry and checksum conflict behavior.
- [x] Keep storage commits non-overwriting so concurrent chunk retries cannot replace chunk bytes.
- [x] Publish small recording events from service transitions.
- [x] Add service and repository tests for transitions, conflicts, ownership, and events.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/recording ./internal/repository`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/database ./internal/config ./internal/repository ./internal/recording ./cmd/server`

Artifacts:

- `internal/repository/recording_repository.go`
- `internal/repository/recording_repository_test.go`
- `internal/recording/service.go`
- `internal/recording/service_test.go`
- `internal/recording/storage.go`

## Sprint 3: HTTP Recording API

Status: completed

Completed tasks:

- [x] Add request/response DTOs separate from persistence models.
- [x] Register canonical `/api/v1/recordings` routes.
- [x] Implement create/list/get handlers.
- [x] Implement streaming raw chunk upload with request size limits.
- [x] Implement stop/cancel/retry-finalize command handlers.
- [x] Add `rec_...` public ID parsing and response mapping.
- [x] Add route contract, handler, auth, validation, conflict, and size-limit tests.
- [x] Update OpenAPI docs after the route contract stabilized.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestRecording|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestAPIDocsContainOnlyCanonicalRoutes'`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/database ./internal/config ./internal/repository ./internal/recording ./internal/api ./cmd/server` outside sandbox because `httptest.NewServer` loopback binding is blocked inside the sandbox.

Artifacts:

- `internal/api/recording_handlers.go`
- `internal/api/recording_handlers_test.go`
- `internal/api/router.go`
- `internal/api/middleware.go`
- `internal/api/types.go`
- `internal/api/response_models.go`
- `internal/api/auth_test.go`
- `internal/api/route_contract_test.go`
- `docs/api/openapi.json`

## Sprint 4: Finalizer Worker and Existing File/Transcription Handoff

Status: completed

Completed tasks:

- [x] Add recording finalizer worker with claim, lease renewal, recovery, wake, and shutdown behavior.
- [x] Add fakeable `MediaFinalizer` interface and ffmpeg implementation.
- [x] Reconstruct raw browser audio from ordered chunks.
- [x] Validate contiguous chunk indexes through the declared final chunk.
- [x] Produce a final audio artifact and create a normal Scriberr file row.
- [x] Remove temporary chunks and raw reconstruction artifacts after the final file row is committed.
- [x] Optionally create and enqueue a transcription row after finalization.
- [x] Publish recording/file/transcription events.
- [x] Wake the finalizer after stop/retry commands.
- [x] Wire finalizer startup/shutdown in `cmd/server/main.go`.
- [x] Add finalizer tests for success, missing chunks, failure behavior, cleanup, and auto-transcription.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/recording ./internal/repository`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestRecording|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestAPIDocsContainOnlyCanonicalRoutes'`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/database ./internal/config ./internal/repository ./internal/recording ./internal/api ./cmd/server` outside sandbox because `httptest.NewServer` loopback binding is blocked inside the sandbox.

Artifacts:

- `internal/recording/finalizer.go`
- `internal/recording/finalizer_test.go`
- `internal/recording/storage.go`
- `internal/repository/recording_repository.go`
- `internal/repository/implementations.go`
- `internal/api/router.go`
- `internal/api/events_handlers.go`
- `internal/api/recording_handlers.go`
- `cmd/server/main.go`

## Sprint 5: Recovery, Cleanup, and Operational Hardening

Status: pending

Planned tasks:

- [ ] Recover expired finalizer claims on startup.
- [ ] Expire abandoned active sessions past TTL.
- [ ] Clean chunk directories for canceled, expired, failed-after-retention, and safely completed sessions.
- [ ] Add duration and byte accounting safeguards.
- [ ] Add bounded progress updates and small event payloads.
- [ ] Wire finalizer startup/shutdown in `cmd/server/main.go`.
- [ ] Add operational logs for lifecycle and cleanup counts.
- [ ] Add recovery, TTL, cleanup, and graceful shutdown tests.

Verification:

- [ ] `go test ./internal/recording ./cmd/server`

## Sprint 6: Contract, Security, and Performance Verification

Status: pending

Planned tasks:

- [ ] Add full route contract coverage for recording endpoints.
- [ ] Add security regression tests for path traversal, cross-user access, MIME spoofing, oversized chunks, and unsafe errors.
- [ ] Add repository tests for claim atomicity and terminal state conflicts.
- [ ] Add storage tests for temp-file cleanup and duplicate uploads.
- [ ] Add API tests for streaming behavior and request cancellation.
- [ ] Run focused backend test suite and update this tracker with artifacts and residual risks.

Verification:

- [ ] `go test ./internal/database ./internal/config ./internal/repository ./internal/recording ./internal/api ./cmd/server`
