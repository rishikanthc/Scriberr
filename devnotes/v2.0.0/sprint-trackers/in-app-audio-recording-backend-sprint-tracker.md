# Sprint Tracker: In-App Audio Recording Backend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/in-app-audio-recording-backend-sprint-plan.md`.

Status: completed through Sprint 1. Recording schema, config, and storage boundary are in place; service/API/finalizer implementation has not started.

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

Status: pending

Planned tasks:

- [ ] Add `RecordingRepository` with domain-specific session/chunk/finalization methods.
- [ ] Add `recording.Service` commands for create, append chunk, stop, cancel, retry, get, and list.
- [ ] Enforce MIME type, chunk index, size, duration, ownership, and status validation.
- [ ] Implement idempotent chunk retry and checksum conflict behavior.
- [ ] Publish small recording events from service transitions.
- [ ] Add service and repository tests for transitions, conflicts, ownership, and events.

Verification:

- [ ] `go test ./internal/recording ./internal/repository`

## Sprint 3: HTTP Recording API

Status: pending

Planned tasks:

- [ ] Add request/response DTOs separate from persistence models.
- [ ] Register canonical `/api/v1/recordings` routes.
- [ ] Implement create/list/get handlers.
- [ ] Implement streaming raw chunk upload with request size limits.
- [ ] Implement stop/cancel/retry-finalize command handlers.
- [ ] Add `rec_...` public ID parsing and response mapping.
- [ ] Add route contract, handler, auth, validation, conflict, and size-limit tests.
- [ ] Update OpenAPI/docs after the route contract stabilizes.

Verification:

- [ ] `go test ./internal/api -run 'TestRecording|TestCanonicalRouteRegistration|TestEndpointContractSmoke'`

## Sprint 4: Finalizer Worker and Existing File/Transcription Handoff

Status: pending

Planned tasks:

- [ ] Add recording finalizer worker with claim, lease renewal, recovery, wake, and shutdown behavior.
- [ ] Add fakeable `MediaFinalizer` interface and ffmpeg implementation.
- [ ] Reconstruct raw browser audio from ordered chunks.
- [ ] Validate contiguous chunk indexes through the declared final chunk.
- [ ] Produce a final audio artifact and create a normal Scriberr file row.
- [ ] Remove temporary chunks and raw reconstruction artifacts after the final file row is committed.
- [ ] Optionally create and enqueue a transcription row after finalization.
- [ ] Publish recording/file/transcription events.
- [ ] Add finalizer tests for success, missing chunks, ffmpeg failure, cancellation, and auto-transcription.

Verification:

- [ ] `go test ./internal/recording ./internal/repository ./internal/api`

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
