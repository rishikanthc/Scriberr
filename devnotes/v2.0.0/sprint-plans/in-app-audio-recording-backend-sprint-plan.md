# In-App Audio Recording Backend Sprint Plan

This plan adds the backend foundation for recording audio inside Scriberr. It is backend-only and intentionally stops before frontend implementation.

Related rules:

- `devnotes/v2.0.0/rules/backend-architecture-rules.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/in-app-audio-recording-backend-sprint-tracker.md`

## Product Goal

Users can start a browser microphone or tab recording, stream small audio chunks to Scriberr while recording, stop or cancel the session, and have the backend finalize the chunks into a normal Scriberr audio file that can be transcribed through the existing transcription queue.

Required backend behavior:

- Create a durable recording session before accepting chunks.
- Accept browser `MediaRecorder` chunks progressively without holding full recordings in memory.
- Store chunks immediately under controlled server storage.
- Make chunk upload idempotent so clients can retry safely.
- Finalize chunks into a playable audio artifact after stop.
- Treat chunk files as temporary ingest artifacts, not durable user-facing media.
- Remove temporary chunks after final audio creation is safely persisted.
- Create a normal Scriberr file row after finalization so existing file list, audio streaming, tagging, and transcription creation keep working.
- Optionally create and enqueue a transcription after finalization when requested.
- Publish small recording/file/transcription events for UI cache invalidation and progress.
- Recover or expire interrupted sessions after process restart.
- Keep handlers thin; services own state transitions, storage owns paths, repositories own persistence.

Out of scope for the backend sprint:

- Frontend recording controls.
- Live transcription during recording.
- WebSocket ingestion as the first transport.
- Offline browser queueing.
- Multi-track recording.
- Cloud object storage.

## Current Codebase Notes

Scriberr currently uses `transcriptions` rows for two related concepts:

- File/source rows have `source_file_hash IS NULL`.
- Transcription rows reference a source file through `source_file_hash`.

The existing transcription worker is durable and already handles queue claiming, lease renewal, cancellation, completion, and events. Recording should reuse that downstream path instead of adding a parallel ASR workflow.

Important existing boundaries:

- `internal/api` owns route registration, auth, request binding, and response mapping.
- `internal/repository.JobRepository` owns transcription/file persistence methods.
- `internal/transcription/worker.Service` owns queued transcription execution.
- `internal/mediaimport` is a useful precedent for import orchestration, but recording needs a stronger durable state machine because chunk upload spans many requests.
- File upload and video extraction still contain some handler-owned persistence/filesystem work. Recording should follow the target architecture rules instead of extending that pattern.

## Architecture Decision

Use stream-to-backend with an idempotent HTTP chunk API as the first implementation.

Preferred flow:

```txt
POST /api/v1/recordings
  -> recording session persisted

PUT /api/v1/recordings/{id}/chunks/{index}
  -> validate auth/session/status/content type/checksum
  -> write chunk to recording storage
  -> persist chunk metadata
  -> publish small progress event

POST /api/v1/recordings/{id}:stop
  -> mark session stopping
  -> enqueue durable finalization

recording finalizer worker
  -> claim stopping session
  -> reconstruct raw browser stream from chunks
  -> ffmpeg validate/transcode to final audio
  -> create normal Scriberr file row
  -> optionally create/enqueue transcription row
  -> remove temporary chunks after durable handoff succeeds
  -> mark recording ready or failed
  -> publish recording/file/transcription events
```

Why HTTP first:

- `PUT` by chunk index makes retry behavior simple and testable.
- Out-of-order chunk arrival is easy to support.
- Existing auth, middleware, request limits, route contract tests, and error envelopes apply directly.
- The domain service can expose the same `AppendChunk` command for a later WebSocket adapter.

WebSocket can be added later for lower overhead or live transcription. It should call the same recording service methods and must not introduce a second persistence model.

## Target Backend Model

Add first-class recording records instead of overloading a partially uploaded file row.

Recommended tables:

```txt
recording_sessions
recording_chunks
```

`recording_sessions` columns:

```txt
id                       string primary key
user_id                  uint not null indexed
title                    text null
status                   varchar not null indexed
source_kind              varchar not null default 'microphone'
mime_type                varchar not null
codec                    varchar null
chunk_duration_ms        integer null
expected_final_index     integer null
received_chunks          integer not null default 0
received_bytes           integer not null default 0
duration_ms              integer null
file_id                  string null indexed
transcription_id         string null indexed
auto_transcribe          boolean not null default false
profile_id               string null
transcription_options_json json not null default {}
started_at               timestamp not null
stopped_at               timestamp null
finalize_queued_at       timestamp null
finalize_started_at      timestamp null
completed_at             timestamp null
failed_at                timestamp null
expires_at               timestamp null indexed
last_error               text null
progress                 real not null default 0
progress_stage           varchar null
claimed_by               varchar null
claim_expires_at         timestamp null indexed
metadata_json            json not null default {}
created_at               timestamp
updated_at               timestamp
deleted_at               soft delete
```

`recording_chunks` columns:

```txt
id             string primary key
session_id     string not null indexed
user_id        uint not null indexed
chunk_index    integer not null
path           text not null
mime_type      varchar not null
sha256         varchar(64) null
size_bytes     integer not null
duration_ms    integer null
received_at    timestamp not null
created_at     timestamp
```

Indexes and constraints:

- Unique chunk per session/index: `(session_id, chunk_index)`.
- Claim path: `(status, finalize_queued_at)` and `claim_expires_at`.
- Session list/detail path: `(user_id, created_at DESC)`.
- Active cleanup path: `(status, expires_at)`.
- Foreign key from chunks to sessions with cascade delete.

Recording statuses:

```txt
recording   accepts chunks
stopping    stop accepted; finalization pending
finalizing  finalizer has claimed the session
ready       final file exists and file_id is set
failed      unrecoverable finalize/storage error
canceled    user canceled and chunks should be cleaned up
expired     abandoned session expired by recovery/cleanup
```

Valid transitions:

```txt
recording -> stopping
recording -> canceled
recording -> expired
stopping -> finalizing
stopping -> canceled
finalizing -> ready
finalizing -> failed
finalizing -> canceled
failed -> stopping      retry finalization only when chunks remain valid
```

## Storage Design

Add an explicit recording storage boundary, for example `internal/recording/storage.go`.

Default directories:

```txt
data/recordings/{session_id}/chunks/{000000.webm}
data/recordings/{session_id}/raw.webm
data/recordings/{session_id}/final.webm
```

Only `final.webm` or a later provider-normalized derivative is durable user-facing audio. Chunk files and `raw.webm` are temporary ingest/finalization artifacts and should be deleted after the final audio file is persisted and the file row is committed.

Storage rules:

- Generate all stored names from internal IDs and chunk indexes.
- Never trust client filenames for chunk paths.
- Write chunks through a temp file and atomic rename.
- Use restrictive permissions for chunk and finalized artifacts.
- Reject path traversal by construction, not by later string cleanup.
- Keep chunk paths internal; API responses return public IDs only.
- Clean temporary files on failed writes and canceled/expired sessions.
- Clean chunk files and raw reconstruction artifacts after successful finalization.
- Do not delete the finalized audio artifact while a file row points at it.

For browser `MediaRecorder` output, use `audio/webm;codecs=opus` as the preferred MIME type. The finalizer should reconstruct `raw.webm` by concatenating same-session chunks in index order, then run ffmpeg to validate and normalize. The first implementation can store `final.webm` for playback and existing transcription, with an optional later conversion step to `16kHz` mono WAV/FLAC if a provider requires it.

## API Contract

Canonical routes:

```http
POST   /api/v1/recordings
GET    /api/v1/recordings
GET    /api/v1/recordings/{recording_id}
PUT    /api/v1/recordings/{recording_id}/chunks/{chunk_index}
POST   /api/v1/recordings/{recording_id}:stop
POST   /api/v1/recordings/{recording_id}:cancel
POST   /api/v1/recordings/{recording_id}:retry-finalize
```

Create request:

```json
{
  "title": "Team sync",
  "source_kind": "microphone",
  "mime_type": "audio/webm;codecs=opus",
  "chunk_duration_ms": 3000,
  "auto_transcribe": false,
  "profile_id": "profile_optional",
  "options": {
    "language": "en",
    "diarization": true
  }
}
```

Create response:

```json
{
  "id": "rec_abc",
  "status": "recording",
  "title": "Team sync",
  "mime_type": "audio/webm;codecs=opus",
  "received_chunks": 0,
  "received_bytes": 0,
  "duration_seconds": null,
  "file_id": null,
  "transcription_id": null,
  "created_at": "...",
  "updated_at": "..."
}
```

Chunk upload:

```http
PUT /api/v1/recordings/rec_abc/chunks/0
Content-Type: audio/webm;codecs=opus
X-Chunk-SHA256: optional-hex-digest
X-Chunk-Duration-Ms: 3000

<raw chunk bytes>
```

Chunk response:

```json
{
  "recording_id": "rec_abc",
  "chunk_index": 0,
  "status": "stored",
  "received_chunks": 1,
  "received_bytes": 48213
}
```

Stop request:

```json
{
  "final_chunk_index": 127,
  "duration_ms": 384000,
  "auto_transcribe": true
}
```

Stop response:

```json
{
  "id": "rec_abc",
  "status": "stopping",
  "progress": 0.75,
  "progress_stage": "queued_for_finalization"
}
```

Idempotency rules:

- Re-uploading the same chunk index with the same checksum/size returns success.
- Re-uploading the same index with different bytes returns `409 CONFLICT`.
- Uploading a chunk after stop/cancel/ready returns `409 CONFLICT`.
- Stop is idempotent for an already stopping/finalizing/ready session when the final index matches.
- Cancel is idempotent for canceled sessions.

## Event Contract

Events should remain small status/cache notifications:

```txt
recording.created
recording.chunk.stored
recording.stopping
recording.finalizing
recording.ready
recording.failed
recording.canceled
file.ready
transcription.created
```

Payloads should use public IDs:

```json
{
  "id": "rec_abc",
  "status": "finalizing",
  "progress": 0.85,
  "stage": "transcoding",
  "file_id": null,
  "transcription_id": null
}
```

Do not include local paths, raw ffmpeg output, full chunk manifests, or transcript payloads in SSE events.

## Configuration

Add typed config fields and validation:

```txt
RECORDINGS_DIR=data/recordings
RECORDING_MAX_CHUNK_BYTES=26214400
RECORDING_MAX_DURATION=8h
RECORDING_SESSION_TTL=12h
RECORDING_FINALIZER_WORKERS=1
RECORDING_FINALIZER_POLL_INTERVAL=2s
RECORDING_FINALIZER_LEASE_TIMEOUT=10m
RECORDING_ALLOWED_MIME_TYPES=audio/webm;codecs=opus,audio/webm
```

Startup logging should include the recordings directory, max chunk size, max duration, finalizer workers, and allowed MIME types. Runtime code should receive this config through constructors and should not read environment variables directly.

## Backend Boundaries

Suggested package layout:

```txt
internal/models/recording.go
internal/recording/service.go
internal/recording/storage.go
internal/recording/finalizer.go
internal/recording/ffmpeg.go
internal/repository/recording_repository.go
internal/api/recording_handlers.go
```

Responsibilities:

- `internal/api`: parse public IDs, auth, bind requests, enforce HTTP size limits, call one service method, map response/error envelopes.
- `internal/recording.Service`: validate commands, enforce state transitions, coordinate repository/storage/finalizer queue, publish events.
- `internal/recording.Storage`: all chunk/final artifact paths and filesystem operations.
- `internal/recording.Finalizer`: durable worker that claims stopped sessions and runs finalization.
- `internal/recording.MediaFinalizer`: narrow ffmpeg interface, fakeable in tests.
- `internal/repository.RecordingRepository`: session/chunk persistence and atomic claim/state methods.
- `internal/repository.JobRepository`: create finalized file row and optional transcription row through domain-specific methods.

## Sprint 1: Schema, Config, and Storage Boundary

Goal: add durable recording persistence and centralized storage policy.

Tasks:

- Add `models.RecordingSession` and `models.RecordingChunk`.
- Register models in the target schema and bump the schema version.
- Add indexes and uniqueness constraints for session listing, chunk idempotency, finalizer claim, and cleanup.
- Add recording config fields, defaults, validation, and startup logging.
- Add recording storage abstraction for session dirs, chunk temp writes, atomic chunk commits, raw/final artifact paths, and cleanup.
- Add database tests for fresh schema, constraints, indexes, cascade delete, and migration.
- Add config tests for invalid sizes, durations, workers, and MIME lists.

Acceptance criteria:

- Fresh databases create both recording tables and indexes.
- Existing databases migrate forward deterministically.
- Duplicate chunk indexes are blocked at the database layer.
- Deleting a session deletes its chunk metadata.
- Storage tests prove generated paths stay under `RECORDINGS_DIR`.

## Sprint 2: Repository and Recording Service State Machine

Goal: put recording decisions behind domain methods.

Tasks:

- Add `RecordingRepository` methods:
  - `CreateSession`
  - `FindSessionForUser`
  - `ListSessionsForUser`
  - `AddChunk`
  - `FindChunk`
  - `ListChunks`
  - `MarkStopping`
  - `ClaimNextFinalization`
  - `RenewFinalizationClaim`
  - `MarkFinalizing`
  - `CompleteFinalization`
  - `FailFinalization`
  - `CancelSession`
  - `ExpireAbandonedSessions`
- Add `recording.Service` commands for create, append chunk, stop, cancel, retry finalization, get, and list.
- Enforce MIME type, chunk index, chunk size, duration, ownership, and status validation.
- Implement idempotent chunk behavior using checksum and size comparisons.
- Publish small recording events from service transitions.
- Add service tests for valid transitions, invalid transitions, idempotent retry, checksum conflicts, ownership, and event emission.

Acceptance criteria:

- No recording handler reads `database.DB` directly.
- All service and repository operations accept `context.Context`.
- State conflicts are explicit and mapped to `409`.
- Not found and unauthorized ownership checks do not leak whether another user's session exists.

## Sprint 3: HTTP Recording API

Goal: expose the recording workflow through v1 REST.

Tasks:

- Add request/response types separate from GORM models.
- Register canonical `/api/v1/recordings` routes.
- Implement create/list/get handlers.
- Implement raw chunk upload with `http.MaxBytesReader` and streaming to storage.
- Implement stop/cancel/retry-finalize command handlers.
- Add public ID helpers for `rec_...`.
- Add route contract tests and handler tests for success, validation errors, auth, conflicts, and request size limits.
- Add OpenAPI/docs updates after the route contract stabilizes.

Acceptance criteria:

- Chunk uploads do not buffer the full recording in memory.
- Responses use public IDs and never expose paths.
- Unsupported MIME types, missing content, invalid indexes, oversized chunks, and wrong owners return structured errors.
- Repeated chunk and stop requests are safe for client retries.

## Sprint 4: Finalizer Worker and Existing File/Transcription Handoff

Goal: convert stopped recordings into normal Scriberr files and optionally transcriptions.

Tasks:

- Add a recording finalizer worker with claim, lease renewal, recovery, shutdown, and wake behavior modeled after `transcription/worker`.
- Add fakeable `MediaFinalizer` interface and ffmpeg implementation.
- Reconstruct raw browser audio from chunks in order.
- Validate contiguous chunk indexes from `0` through `final_chunk_index`.
- Run ffmpeg to produce a final audio artifact.
- Add `JobRepository` methods for creating a file row from a finalized recording and creating/enqueuing a transcription row from that file when requested.
- Delete temporary chunk files and raw reconstruction artifacts only after the final file row transaction succeeds.
- Publish `recording.finalizing`, `recording.ready`, `recording.failed`, `file.ready`, and optional `transcription.created`.
- Add tests for successful finalization, missing chunks, ffmpeg failure, cancellation during finalization, and optional auto-transcription.

Acceptance criteria:

- Finalization survives HTTP request completion and process restart.
- A ready recording has `file_id` set and appears through existing file APIs.
- A ready recording has one durable final audio file; temporary chunks are removed.
- Auto-transcription uses the existing transcription queue and profile/options resolution policy.
- Failed finalization stores a sanitized error and does not expose ffmpeg output or paths.

## Sprint 5: Recovery, Cleanup, and Operational Hardening

Goal: make long recordings reliable and bounded.

Tasks:

- Recover expired finalizer claims on startup.
- Expire abandoned `recording` sessions past TTL.
- Clean chunk directories for canceled, expired, failed-after-retention, and safely completed sessions according to retention policy.
- Add session duration and byte accounting safeguards.
- Add progress calculations that avoid frequent large writes.
- Add startup/shutdown wiring in `cmd/server/main.go`.
- Add logs for finalizer start/stop, claim, completion, failure, expiration, and cleanup counts.
- Add tests for recovery, TTL expiration, cleanup, and graceful shutdown.

Acceptance criteria:

- Restarting the server does not strand stopped sessions forever.
- Abandoned sessions cannot accumulate unbounded storage.
- Shutdown cancels finalizers and leaves sessions recoverable.
- Progress/event updates stay small and bounded.

## Sprint 6: Contract, Security, and Performance Verification

Goal: lock down the backend contract before frontend work begins.

Tasks:

- Add route contract coverage for all recording endpoints.
- Add security regression tests for path traversal, cross-user access, MIME spoofing, oversized chunks, and unsafe error leakage.
- Add repository tests around claim atomicity and terminal state conflicts.
- Add storage tests for temp-file cleanup and idempotent duplicate uploads.
- Add API tests proving chunk upload streams from request body and honors cancellation.
- Run focused backend tests for database, repository, recording, API, and worker/finalizer packages.
- Update the sprint tracker with artifacts, verification commands, and residual risks.

Acceptance criteria:

- Backend contract is stable enough for frontend implementation.
- Hot paths are indexed: chunk insert, session lookup, finalizer claim, cleanup, and list.
- Every durable user-visible operation has persisted state, terminal states, cancellation behavior, and event observability.

## Residual Risks and Follow-Ups

- Browser `MediaRecorder` compatibility differs by engine. Backend should prefer `audio/webm;codecs=opus` and allow a small configurable MIME allowlist.
- Safari may need a later MIME expansion if frontend support targets it.
- Live transcription should be a later sprint using the same session/chunk model plus a separate partial transcript workflow.
- Object storage can be added later by replacing `recording.Storage`; API and service contracts should not expose filesystem assumptions.
- If final WebM artifacts are not ideal for a selected ASR provider, add a provider-aware normalization step to WAV/FLAC in the finalizer rather than changing browser upload format.
