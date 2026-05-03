# Backend Architecture Code Review - 2026-05-03

Review scope:

- Codebase: Scriberr Go backend.
- Primary standards:
  - `devnotes/v2.0.0/specs/architecture-design.md`
  - `devnotes/v2.0.0/rules/backend-rules.md`
- Review mode: architecture, security, queue, database, provider extensibility, file/audio handling, testing, and operations.
- Verification run: `go test ./internal/api -run 'TestProduction|TestBackendDependencyDirection'` passed with `GOCACHE=/private/tmp/scriberr-go-cache`.

This document records the review before further implementation. It is intentionally strict because the target architecture is a clean modular monolith with future multi-user support, a shared durable queue, and extensible local/remote ASR providers.

## Overall Assessment

The backend has made substantial progress toward the intended architecture. The composition root is mostly centralized in `internal/app`, production API code no longer imports `internal/database`, most product areas have service and repository layers, durable queue state is SQL-backed, user IDs are present on major owned tables, and the local ASR engine is isolated behind `internal/transcription/engineprovider`.

The remaining risks are concentrated in boundary leaks and state ownership:

- API still owns non-trivial business workflows for chat streaming and LLM provider probing.
- Event delivery is not user-scoped, which is incompatible with the multi-user architecture constraint.
- Queue terminal transitions are not lease-owner aware, which can let stale workers overwrite newer state.
- LLM provider API keys are persisted as plaintext JSON.
- Generic/global repository methods and legacy queue paths still exist and can bypass user-scoped domain operations.

These are not cosmetic issues. They are the places most likely to cause security leaks, duplicate ASR work, difficult provider extensions, and future agent confusion.

## Top 5 Highest-Priority Fixes

1. Add user/audience scoping to SSE subscriptions and published events.
2. Encrypt or externalize LLM provider credentials instead of persisting raw API keys.
3. Make transcription terminal queue transitions claim-aware.
4. Recover only expired processing jobs, not every processing job.
5. Move chat streaming and LLM provider probing out of `internal/api`.

## Findings

### 1. Global SSE is not user-scoped

- Severity: High
- Location: `internal/api/events_handlers.go`, `eventBroker.subscribe`, `eventBroker.publish`, `streamEvents`
- Related rules:
  - Backend rule 8: every user-owned operation is scoped by `user_id`.
  - Backend rule 11: events are small notifications, filtered by authorized audience.
  - Architecture design: every request has exactly one authenticated principal.

Problem:

`streamEvents` subscribes with an empty transcription ID, and `eventBroker.publish` delivers any event with an empty subscriber filter to that subscriber. The subscription does not record `user_id`, admin role, API key scope, or any other authorization audience. Any authenticated client connected to `/api/v1/events` can receive events emitted by other users.

Why it matters:

Even if payloads are intentionally small, events reveal activity and identifiers: file IDs, transcription IDs, recording IDs, summary status, settings changes, tag updates, progress states, and timing. Once more than one user exists, this is cross-user information disclosure.

Recommended fix:

- Add an audience field to `apiEvent`, at minimum `UserID uint` and optionally `AdminOnly bool`.
- Add `userID` and role/API-key scope to `eventSubscriber`.
- In `publish`, deliver an event only when:
  - subscriber is the same user;
  - subscriber is authorized admin for admin/system events;
  - and transcription-specific subscriptions also match the requested transcription.
- Ensure all event publisher interfaces carry user identity from services/workers.
- Add regression tests with two users connected to `/api/v1/events`.

Example shape:

```go
type apiEvent struct {
    Name            string
    Data            gin.H
    UserID          uint
    TranscriptionID string
    AdminOnly       bool
}

type eventSubscriber struct {
    id              string
    userID          uint
    isAdmin         bool
    transcriptionID string
    ch              chan apiEvent
}
```

### 2. LLM provider API keys are stored raw

- Severity: High
- Location: `internal/models/transcription.go`, `LLMConfig.BeforeSave` and `LLMConfig.AfterFind`
- Related rules:
  - Backend rule 14: provider credentials are encrypted or otherwise protected before persistence.
  - Architecture design: store secret material only as hashes or encrypted values.

Problem:

`LLMConfig.BeforeSave` serializes `APIKey` directly into `llm_profiles.config_json`, and `AfterFind` restores it as plaintext. The API response masks it, but the database stores the provider secret in recoverable plaintext JSON.

Why it matters:

A database leak exposes provider credentials. It also creates a precedent that provider-specific config bags can carry secrets without a secret boundary.

Recommended fix:

- Introduce a credential protection boundary in `internal/config` or a dedicated `internal/secrets` package.
- Encrypt provider API keys before storing them, using a configured key loaded at startup.
- Store `has_api_key` and `key_preview` separately or derive them without returning the raw key.
- Keep `models.LLMConfig` as persistence shape, but avoid public JSON tags on secret-bearing fields.
- Add migration support for existing raw keys if needed.

Example shape:

```go
type SecretProtector interface {
    Encrypt(ctx context.Context, plaintext string) (string, error)
    Decrypt(ctx context.Context, ciphertext string) (string, error)
}
```

### 3. Queue terminal transitions are not claim-owned

- Severity: High
- Location: `internal/repository/implementations.go`, `CompleteTranscription`, `FailTranscription`, `CancelTranscription`
- Related rules:
  - Backend rules 4 and 16: queue state transitions have one owner.
  - Architecture design: queue operations must be repository-owned and atomic.

Problem:

Terminal queue methods update by `jobID` only. They do not require `status = processing`, `claimed_by = workerID`, a non-expired lease, or the latest execution ID. A stale worker can complete or fail a job after the lease was recovered and claimed by another worker, or after the user canceled it.

Why it matters:

This can produce duplicate ASR runs and stale terminal writes. The current claim path prevents two workers from initially claiming the same pending job, but it does not protect completion from stale ownership after a lease expires or cancellation occurs.

Recommended fix:

- Change terminal repository methods to include worker ownership:
  - `CompleteClaimedTranscription(ctx, jobID, workerID, executionID, transcriptJSON, outputPath, completedAt)`
  - `FailClaimedTranscription(ctx, jobID, workerID, executionID, message, failedAt)`
  - `CancelClaimedTranscription(ctx, jobID, workerID, canceledAt)`
- Include `WHERE id = ? AND status = 'processing' AND claimed_by = ?`.
- Prefer also checking `latest_execution_id = ?`.
- Return a conflict/not-found error on zero rows.
- Update worker tests to cover stale worker completion after recovery.

Example:

```go
result := tx.Model(&models.TranscriptionJob{}).
    Where("id = ? AND status = ? AND claimed_by = ?", jobID, models.StatusProcessing, workerID).
    Updates(updates)
if result.RowsAffected == 0 {
    return ErrLeaseLost
}
```

### 4. Recovery requeues every processing job

- Severity: High
- Location: `internal/repository/implementations.go`, `RecoverOrphanedProcessing`
- Related rules:
  - Backend rules 4 and 16: recover is a queue state transition with one owner.
  - Architecture design: queue operations include renew lease and recover orphaned processing jobs.

Problem:

`RecoverOrphanedProcessing` updates all rows with `status = processing`. It ignores `claim_expires_at`. On worker startup, this can requeue jobs still actively running in another process.

Why it matters:

This is safe only for a strict single-process assumption. The architecture targets durable shared infrastructure, and the queue already has leases. Recovery should honor those leases.

Recommended fix:

- Recover only rows where:
  - `status = processing`
  - and `claim_expires_at IS NULL OR claim_expires_at <= now`
- Optionally limit batch size and log recovered job IDs.
- Add tests for active lease vs expired lease.

Example:

```go
Where("status = ? AND (claim_expires_at IS NULL OR claim_expires_at <= ?)", models.StatusProcessing, now)
```

### 5. Chat generation runs inside the HTTP handler

- Severity: High
- Location: `internal/api/chat_handlers.go`, `streamChatMessage`, `buildLLMMessages`, `chatClientForConfig`
- Related rules:
  - Backend rule 1: API is an HTTP adapter only.
  - Backend rule 3: long-running work never runs inside handlers.
  - Backend rule 6: LLMs stay behind narrow interfaces.

Problem:

`streamChatMessage` performs model availability checks, constructs the LLM client, persists messages/runs, builds context, calls the provider stream, handles token usage, mutates run state, and streams SSE. This is a full use case living in the API adapter.

Why it matters:

It is hard to test independently, hard to make durable/cancelable, and tightly couples provider behavior to Gin. It also makes future remote or queued chat generation harder.

Recommended fix:

- Move generation orchestration into `internal/chat.Service`.
- Define a narrow streaming return type or callback interface from the service.
- Keep handler responsibilities to:
  - authenticate;
  - parse/validate request syntax;
  - call one service method;
  - write SSE frames from service events.
- Consider durable chat generation runs later, similar to transcription and summary workers.

Example service shape:

```go
type StreamRequest struct {
    UserID    uint
    SessionID string
    Content   string
    Model     string
    Temperature float64
}

type StreamEvent struct {
    Name string
    Data map[string]any
}

func (s *Service) StreamMessage(ctx context.Context, req StreamRequest) (<-chan StreamEvent, error)
```

### 6. LLM provider connection adapter lives in `internal/api`

- Severity: Medium
- Location:
  - `internal/api/llm_provider_handlers.go`, `LLMProviderConnectionTester`
  - `internal/app/app.go`, wiring of `llmprovider.NewService(..., api.LLMProviderConnectionTester{})`
- Related rules:
  - Backend rule 1: `internal/api` is an HTTP adapter only.
  - Backend rule 6: LLMs stay behind narrow interfaces.
  - Architecture design: providers/LLM/storage adapters own external libraries.

Problem:

The concrete HTTP adapter that probes OpenAI-compatible and Ollama endpoints is defined in `internal/api`, and `internal/app` imports that API type to construct the LLM provider service.

Why it matters:

This makes a product service depend on an API adapter for concrete integration behavior. It violates the intended layering even though the import direction technically goes through `internal/app`.

Recommended fix:

- Move provider probing to `internal/llmprovider` or `internal/llm`.
- Keep only request/response mapping in `internal/api/llm_provider_handlers.go`.
- Wire `llmprovider.NewHTTPConnectionTester(...)` from `internal/app`.
- Add an architecture test that production API does not define external integration adapters beyond DTO parsing/mapping.

### 7. Admin queue route is not admin-authorized

- Severity: Medium
- Location: `internal/api/router.go`, `GET /api/v1/admin/queue`
- Related rules:
  - Backend rule 9: cross-user operations are admin-only service use cases.
  - Backend rule 10: authorization is explicit.
  - Architecture design: admin routes must require role authorization.

Problem:

`/api/v1/admin/queue` uses `authRequired()`, not an admin role gate. The current implementation returns user-scoped stats, which reduces immediate blast radius, but the route name and future admin queue settings imply a cross-user/admin surface.

Why it matters:

Leaving admin-named routes under normal auth makes it easy to add global stats or scheduler controls later without adding role checks.

Recommended fix:

- Add role to auth context and JWT claims.
- Add `adminRequired()` middleware.
- Move admin queue operations to an admin service/use case.
- If the current endpoint is intended as user queue stats, rename it out of `/admin`.

### 8. Generic and global repository methods remain exposed

- Severity: Medium
- Location:
  - `internal/repository/repository.go`, generic `Repository[T]`
  - `internal/repository/implementations.go`, `JobRepository`
  - `internal/queue/queue.go`, legacy queue use of global methods
- Related rules:
  - Backend rule 5: repositories own persistence shape; services ask for domain operations.
  - Backend rule 8: user-owned operations are scoped by `user_id`.
  - Architecture design: repository methods should express Scriberr invariants.

Problem:

`JobRepository` embeds generic CRUD and still exposes global methods such as `FindByStatus`, `CountByStatus`, `UpdateStatus`, `UpdateError`, and `ListWithParams`. The legacy `internal/queue` package uses some of these methods.

Why it matters:

These methods make it easy for new services or future agents to bypass user scoping and lifecycle methods. They conflict with the documented approach of explicit methods like `FindFileByIDForUser`, `ClaimNextTranscription`, `RenewClaim`, `CompleteTranscription`, and `CancelTranscription`.

Recommended fix:

- Remove generic repository embedding from product repositories where possible.
- Split system/worker-only interfaces from user-facing repository interfaces.
- Delete or quarantine `internal/queue` if the durable transcription worker has replaced it.
- Add architecture tests that forbid new production imports of `internal/queue`.
- Add naming tests or inventory tests for user-owned reads lacking `ForUser`/`ByUser`.

### 9. Queue claim policy is embedded in repository SQL

- Severity: Medium
- Location: `internal/repository/implementations.go`, `ClaimNextTranscription`
- Related rules:
  - Backend rule 15: scheduler policy is configured by admin-only settings behind a scheduler boundary.
  - Architecture design: `internal/transcription/scheduler` should own claim policy.

Problem:

Claim order is fixed inside SQL as `priority DESC, queued_at ASC, created_at ASC, id ASC`. There is no scheduler boundary for FIFO, priority, fairness, per-user quotas, or provider-specific concurrency.

Why it matters:

The current behavior is acceptable for a first queue, but it will be difficult to add per-user fairness or provider concurrency limits without changing repository semantics and worker code.

Recommended fix:

- Introduce a scheduler policy interface under `internal/transcription/scheduler`.
- Keep repository claim atomic, but let scheduler produce a claim plan or query options.
- Add admin-owned scheduler config later.
- Keep the default policy equivalent to current priority/FIFO ordering.

### 10. Completion observers run after terminal commit and can fail silently

- Severity: Medium
- Location: `internal/transcription/worker/service.go`, `claimAndProcess`, `enqueueCompletionWork`
- Related rules:
  - Backend rule 11: persist durable state first, publish after, clients can recover by re-fetching.
  - Architecture design: completion observers and file-ready handoffs should be explicit.

Problem:

The worker commits a completed transcription, then tries to enqueue completion work for summaries and annotations. If observer enqueue fails, it logs a warning and continues. That may be appropriate for best-effort observers, but automatic summaries/title generation are product workflows that may need durable retry.

Why it matters:

Users can get a completed transcription without expected follow-up work, and there is no durable record of the missed observer beyond logs.

Recommended fix:

- Classify completion observers as durable or best-effort.
- For durable observers, insert a job/outbox row in the same transaction as completion or have a recovery scanner.
- Add tests that summary enqueue failure can be recovered.

### 11. File storage is still local path based

- Severity: Medium
- Location:
  - `internal/files/service.go`, `Upload`, `OpenAudio`, video extraction
  - `internal/transcription/service.go`, `OpenAudio`, `Logs`
  - `internal/transcription/orchestrator/artifact_store.go`
  - `internal/recording/storage.go`
- Related rules:
  - Backend rule 7: file paths are internal.
  - Architecture design: target is one explicit storage boundary before S3/MinIO or user-scoped storage.

Problem:

The code correctly keeps paths mostly out of public responses, but product services still pass around and open local paths directly. There is no single storage abstraction for audio, transcript artifacts, logs, imports, and recordings.

Why it matters:

Remote storage or user-scoped object storage will require broad changes across files, transcription, recording, media import, and orchestrator packages.

Recommended fix:

- Introduce a storage interface after queue/security fixes.
- Store opaque object IDs on product records, not filesystem paths as the long-term durable reference.
- Keep local filesystem as the first adapter.
- Preserve separate original filename metadata.

### 12. Provider capability model is too coarse for future remote providers

- Severity: Medium
- Location: `internal/transcription/engineprovider/types.go`
- Related rules:
  - Backend rule 6: ASR stays behind narrow interfaces.
  - Architecture design: providers should advertise streaming, diarization, timestamps, alignment, models, and languages.

Problem:

`ModelCapability` has a string slice of capabilities, but no typed structure for streaming support, diarization support, word timestamps, alignment, languages, max duration, concurrency limits, transport type, or provider health.

Why it matters:

Remote providers over WebSocket/HTTP/gRPC will need richer capability and operational metadata without changing REST handlers or queue code.

Recommended fix:

- Keep the current provider interface, but add a typed capability descriptor.
- Include provider-level and model-level capability fields.
- Make provider selection validate required features before enqueue or before processing.

Example:

```go
type CapabilitySet struct {
    Streaming       bool
    Diarization     bool
    WordTimestamps  bool
    SegmentTimestamps bool
    Alignment       bool
    Languages       []string
    MaxConcurrency  int
    Transport       string
}
```

### 13. Repository order clauses depend on caller-provided columns

- Severity: Low
- Location:
  - `internal/repository/implementations.go`, `ListFilesByUser`, `ListTranscriptionsByUser`, cursor helpers
  - `internal/api/list_query.go`, `parseSort`
- Related rules:
  - Security: API inputs are validated.
  - Persistence boundary: repositories own SQL shape.

Problem:

The API validates sort keys against an allowlist before passing `SortColumn` to repositories, and current call sites appear safe. However, the repository methods accept raw column strings and concatenate them into SQL order/cursor clauses.

Why it matters:

The safety relies on every caller validating correctly. A future internal caller could pass untrusted strings and create SQL injection risk.

Recommended fix:

- Move sort-key validation into repository option constructors, or use an enum-like type.
- Reject unknown sort columns inside repository methods.
- Keep API allowlist as syntax validation, but do not rely on it as the only guard.

### 14. Static route setup has duplicate NoRoute ownership

- Severity: Low
- Location:
  - `internal/web/static.go`, `SetupStaticRoutes`
  - `internal/api/router.go`, final `router.NoRoute`
- Related rules:
  - Architecture design: `internal/web` is static frontend serving adapter; API owns routes.

Problem:

`web.SetupStaticRoutes` installs a `NoRoute` handler, and then `api.SetupRoutes` installs another `NoRoute` handler after calling it. The latter wins, so the one inside `internal/web` is effectively dead/confusing.

Why it matters:

This is not a likely security issue because the final router handler does path cleaning. It is an ownership clarity issue and can confuse future static-serving changes.

Recommended fix:

- Make `internal/web` expose asset/index helpers or route-specific static handlers only.
- Keep SPA fallback/no-route policy in one place.

## Security Risks

Primary security risks:

- Cross-user SSE leakage through global event subscriptions.
- Plaintext LLM provider credentials in `llm_profiles.config_json`.
- Admin route naming without admin role enforcement.
- Future SQL injection risk if repository sort options are called without API validation.

Positive security notes:

- Production API code no longer imports `internal/database`.
- Uploads use `http.MaxBytesReader`.
- Filenames are sanitized before storage.
- Public event payload sanitizer removes obvious local path keys.
- Logs/transcription error responses have path/token redaction helpers.
- Refresh tokens and API keys use one-way hashes.

## Database and Job Queue Risks

Primary queue risks:

- Recovery ignores `claim_expires_at`.
- Terminal updates ignore `claimed_by`.
- Cancellation and completion can race with stale worker writes.
- Provider-specific concurrency and per-user fairness are not represented yet.
- Scheduler policy is hard-coded in repository SQL.

Positive queue notes:

- Jobs are durable in SQLite.
- Claims are atomic enough for basic concurrent workers.
- Lease renewal exists.
- Per-user stats are user-scoped.
- Tests cover basic FIFO, priority, concurrent claim, lease renewal ownership, and terminal updates.

Recommended queue test additions:

- Active processing job with future lease is not recovered.
- Expired processing job is recovered.
- Stale worker cannot complete after recovery/reclaim.
- Stale worker cannot fail after user cancellation.
- Terminal updates update only the latest execution.
- Provider concurrency limit can reject or delay claim without losing job state.

## Extensibility Risks

ASR provider extensibility is reasonably started but incomplete:

- Good: `engineprovider.Provider`, `Registry`, `TranscriptionRequest`, and `DiarizationRequest` isolate the local engine.
- Risk: provider selection is minimal and does not validate typed capability needs.
- Risk: queue claim does not understand provider-specific concurrency.
- Risk: transcription profiles still use legacy `WhisperXParams`, which is now broader than one provider but named after an old implementation.

LLM extensibility is weaker:

- API owns concrete LLM client creation and chat streaming.
- LLM provider probing lives in `internal/api`.
- Provider credentials are persistence details in `models`.

Recommended direction:

- Rename or wrap `WhisperXParams` behind a provider-neutral `TranscriptionOptions` type when practical.
- Add typed ASR capability metadata.
- Keep REST API independent from provider transport.
- Move LLM provider clients/probing into provider packages.

## File and Audio Handling Review

Positive notes:

- Upload size is bounded.
- Uploaded filenames are sanitized.
- Stored object names are generated from random/job IDs.
- Audio streaming uses `http.ServeContent`, which handles byte ranges safely.
- Public file/transcription responses do not intentionally expose local paths.
- Video extraction runs outside the handler in `internal/files`.

Risks:

- Services still open local paths directly.
- There is no single storage object abstraction.
- Uploaded media type validation is extension/header based, not content-sniffed.
- Video extraction creates async in-process work rather than a durable import queue.
- Cleanup policy is split across files/media import/recording/orchestrator.

Recommended direction:

- Add storage boundary after queue/security fixes.
- Store object IDs rather than paths in future schema.
- Introduce durable import/extraction jobs for video and remote imports.
- Add content sniffing or ffprobe validation for uploaded audio/video.

## Testing Review

What is covered well:

- API architecture import guardrails.
- Route contract and boundary-level API tests.
- Job queue concurrent claim and basic state transition tests.
- Recording storage/finalizer tests.
- Service-level tests for annotations, tags, summaries, chat context, and LLM provider behavior.
- Security regression tests for several API paths.

Important missing or insufficient coverage:

- Multi-user SSE isolation.
- Admin role authorization.
- Queue stale-worker terminal transition conflicts.
- Queue recovery lease expiry semantics.
- LLM credential encryption or secret handling.
- Repository sort-column validation independent of API.
- Durable recovery for post-transcription completion observers.
- Migration tests that inspect foreign keys and partial indexes for new owned tables.

## Observability and Operations

Positive notes:

- Startup logs include engine, worker, and recording configuration.
- Worker logs include worker IDs and job IDs for core transitions.
- Graceful shutdown stops HTTP server and application services.
- Workers use context cancellation and wait groups.

Risks:

- Queue logs should consistently include user ID, provider, model, execution ID, duration, and terminal reason.
- Completion observer failures are warnings only and may not be recoverable.
- Event delivery drops messages silently when subscriber channels are full. This is acceptable for ephemeral SSE, but clients must always recover by refetching REST state.
- No metrics/tracing boundary exists yet.

Recommended direction:

- Add structured event/log fields to all job terminal paths.
- Add simple metrics interfaces around queue depth, claim latency, processing duration, provider failures, and recovery counts.
- Keep metrics interfaces optional/no-op until a metrics backend is chosen.

## What Is Already Well-Designed

- `internal/app` is the main composition root and owns lifecycle wiring.
- `cmd/server` is mostly process/bootstrap code.
- Production `internal/api` does not import `internal/database`.
- Import-direction tests exist and passed.
- The ASR engine is behind `internal/transcription/engineprovider`.
- The transcription worker is SQL-backed and independent from HTTP request lifetime.
- Repository methods increasingly include `ForUser`/`ByUser` names.
- API DTO mapping avoids exposing obvious local path fields.
- Upload and recording chunk limits are enforced.
- Passwords, refresh tokens, and API keys use safer credential handling than provider API keys.
- Partial unique indexes exist for per-user default profiles/templates/LLM profiles.
- Tags and tag assignments use per-user uniqueness.

## Suggested Refactor Plan

### Phase 1: Security and authorization

1. Add authenticated principal struct with user ID, role, auth type, and optional API key ID.
2. Add `adminRequired` middleware.
3. Add user-scoped event subscriptions and event publisher user IDs.
4. Encrypt or externalize LLM provider API keys.
5. Add regression tests for two-user SSE isolation and admin route access.

### Phase 2: Queue state ownership

1. Change worker terminal repository methods to require `workerID`.
2. Include `claimed_by` and `status = processing` in terminal `WHERE` clauses.
3. Recover only expired claims.
4. Add stale worker and lease expiry tests.
5. Remove or quarantine legacy `internal/queue`.

### Phase 3: API boundary cleanup

1. Move chat streaming orchestration to `internal/chat.Service`.
2. Move LLM provider probing to `internal/llmprovider` or `internal/llm`.
3. Keep API handlers thin and focused on parse/validate/call/map.
4. Extend architecture tests to catch new integration adapters in `internal/api`.

### Phase 4: Scheduler and provider extensibility

1. Introduce `internal/transcription/scheduler`.
2. Preserve current priority/FIFO as the default policy.
3. Add typed ASR capability metadata.
4. Add provider-specific concurrency limits.
5. Add fairness/quota hooks without changing REST handlers.

### Phase 5: Storage abstraction

1. Add a local filesystem storage adapter behind a `Storage` interface.
2. Move file/audio/transcript/log object access behind object IDs.
3. Keep local paths internal to storage adapters.
4. Add cleanup ownership rules per object type.
5. Add tests for path traversal, object ownership, and streaming range behavior through storage.

## Enforcement Recommendations

Add or extend architecture tests for:

- `internal/api` must not import `internal/llm` concrete clients.
- `internal/api` must not define production types that implement provider/LLM/storage adapters.
- No production package except `internal/app`, `cmd/server`, and `internal/database` imports `internal/database`.
- No production package except `internal/app` imports `internal/api`.
- `internal/queue` is not imported by production code once legacy migration is complete.
- Repository methods that return user-owned records are either `ForUser`/`ByUser` or explicitly documented system/admin methods.
- Admin routes require `adminRequired`.

## Current Status

The backend is acceptable as a migration base, but it should not be considered strictly compliant with `architecture-design.md` and `backend-rules.md` yet.

Strict compliance blockers:

- User-scoped events are missing.
- Provider credentials are stored raw.
- Queue terminal transitions do not enforce lease ownership.
- Chat generation remains in API handlers.
- LLM provider probing lives in API.
- Admin route authorization is incomplete.

These should be addressed before building larger multi-user, scheduler, or remote-provider features on top of the current code.
