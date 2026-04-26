# EWI-Sprint 0 Inventory: Engine Worker Integration

Status: completed

Purpose: document the implementation surface for `devnotes/engine-worker-integration-spec.md` before changing runtime code.

## Current Runtime Path

Server startup currently follows the legacy transcription path:

- `cmd/server/main.go` initializes logging and config.
- `cmd/server/main.go` calls `registerAdapters(cfg)` before database setup.
- `registerAdapters` registers Python/OpenAI adapters through `internal/transcription/registry`.
- `cmd/server/main.go` builds `transcription.NewUnifiedJobProcessor`.
- `cmd/server/main.go` calls `unifiedProcessor.InitEmbeddedPythonEnv()`.
- `cmd/server/main.go` builds `transcription.NewQuickTranscriptionService`.
- `cmd/server/main.go` starts `queue.NewTaskQueue(2, unifiedProcessor, jobRepo)`.
- API handler construction receives legacy queue and processor values through variadic arguments, but current `internal/api.NewHandler` ignores them.

This must be replaced by the new path:

- config load and validation,
- DB/migrations,
- repositories,
- local engine provider registry,
- durable worker recovery,
- worker start,
- API service injection,
- graceful shutdown of HTTP, workers, providers, and DB.

## Current Code Inventory

### Startup and Lifecycle

- `cmd/server/main.go`
  - imports legacy `internal/queue`, `internal/transcription`, `internal/transcription/adapters`, and `internal/transcription/registry`.
  - owns adapter registration and Python env bootstrap.
  - starts old queue before API route setup.
  - shuts down SSE and HTTP, but worker/provider shutdown ordering is not aligned with the spec.

### Config

- `internal/config/config.go`
  - has `WhisperXEnv`.
  - lacks `EngineConfig`.
  - lacks `WorkerConfig`.
  - `Load()` currently returns `*Config` and does not expose validation errors.

Sprint 1 will need a compatibility path while introducing startup-failing config errors. Recommended shape:

- add `LoadWithError() (*Config, error)`,
- keep `Load() *Config` as a compatibility wrapper for tests/callers until all call sites migrate,
- make `cmd/server/main.go` use `LoadWithError()` and log fatal startup config errors.

### Models and Schema

- `internal/models/transcription.go`
  - `TranscriptionJob` already has:
    - `Status`
    - `AudioPath`
    - `Transcript`
    - `OutputJSONPath`
    - `LatestExecutionID`
    - `ErrorMessage`
    - `CompletedAt`
    - metadata compatibility fields.
  - missing queue/lease/progress fields:
    - `QueuedAt`
    - `StartedAt`
    - `FailedAt`
    - `Progress`
    - `ProgressStage`
    - `ClaimedBy`
    - `ClaimExpiresAt`
    - `EngineID`
  - `TranscriptionJobExecution` already has:
    - `Provider`
    - `ModelName`
    - `ModelFamily`
    - `StartedAt`
    - `CompletedAt`
    - `FailedAt`
    - `ErrorMessage`
    - `OutputJSONPath`
    - `RequestJSON`
    - `ConfigJSON`
    - `LogsPath`
  - execution fields are close to the spec and should be reused rather than replaced.

- `internal/database/schema.go`
  - current latest schema version is `2`.
  - current indexes include `idx_transcriptions_status_created_at`.
  - missing queue indexes:
    - `idx_transcriptions_queue_claim(status, queued_at)`
    - `idx_transcriptions_claim_expires_at(claim_expires_at)`

### Repository

- `internal/repository/implementations.go`
  - `JobRepository` has CRUD, listing, execution creation/update, status/error updates, and status counts.
  - missing durable worker methods:
    - `EnqueueTranscription`
    - `ClaimNextTranscription`
    - `RenewClaim`
    - `RecoverOrphanedProcessing`
    - `UpdateProgress`
    - `CompleteTranscription`
    - `FailTranscription`
    - `CancelTranscription`
    - `ListExecutions`
  - `CreateExecution` already allocates execution numbers in a transaction and updates `latest_execution_id`; keep and extend this pattern.

### Current Queue

- `internal/queue/queue.go`
  - in-memory channel queue with auto-scaling.
  - recovers `pending` rows into memory on startup.
  - manually kills process trees for legacy subprocess-based adapters.
  - uses `EnqueueJob`, `KillJob`, `GetQueueStats`.
  - duplicates some status persistence and recovery behavior that belongs in the new durable worker package.

Replacement target:

- new package `internal/transcription/worker`,
- durable DB claim/lease model,
- in-memory wake/cancel only,
- no process-tree kill path for local Go engine.

### Current Transcription Stack

Legacy stack:

- `internal/transcription/adapters/**`
- `internal/transcription/registry/registry.go`
- `internal/transcription/pipeline/pipeline.go`
- `internal/transcription/unified_service.go`
- `internal/transcription/queue_integration.go`
- `internal/transcription/quick_transcription.go`
- `internal/transcription/interfaces/interfaces.go`

Deletion/replace decision:

- remove or stop compiling `internal/transcription/adapters/**` after the new provider path is wired.
- remove `registry` and `pipeline` when no longer imported.
- replace `unified_service.go` and `queue_integration.go` with orchestrator/worker behavior or delete them if no compatibility wrapper is needed.
- replace quick transcription with normal submit/create flow unless a current route still needs it; current canonical API does not expose a quick-transcription endpoint.
- remove obsolete tests tied only to adapter registration and Python execution.

### API Surface

Current canonical route state:

- `POST /api/v1/transcriptions`
  - creates a queued-looking DB row, but does not call a queue service.
- `POST /api/v1/transcriptions:submit`
  - uploads a file and creates a queued-looking DB row, but does not call a queue service.
- `POST /api/v1/transcriptions/{id}:cancel`
  - directly updates DB status to canceled.
- `POST /api/v1/transcriptions/{id}:retry`
  - creates a new queued-looking row, but does not enqueue.
- `GET /api/v1/transcriptions/{id}/transcript`
  - returns plain transcript text and empty arrays.
- `GET /api/v1/transcriptions/{id}/logs`
  - placeholder.
- `GET /api/v1/transcriptions/{id}/executions`
  - placeholder.
- `GET /api/v1/models/transcription`
  - static placeholder.
- `GET /api/v1/admin/queue`
  - counts statuses directly through `database.DB`.
- SSE events are real and API-local, but not connected to worker progress.

API migration decision:

- inject queue/model/execution/log services into `api.Handler`,
- keep handlers as request/response mappers,
- remove direct queue-state writes from handlers,
- keep direct DB access only where prior API sprint cleanup has not yet introduced service boundaries, then narrow it during Sprint 6.

### Docs and Docker

Current Python/WhisperX references:

- `README.md` documents `WHISPERX_ENV`.
- `Dockerfile`, `Dockerfile.cuda`, and `Dockerfile.cuda.12.9` set `WHISPERX_ENV`.
- `internal/transcription/README.md` describes legacy adapters and registry.
- adapter Python README and tests describe `data/whisperx-env`.

Docs migration decision:

- update README and Docker envs in EWI-Sprint 9.
- either remove or rewrite `internal/transcription/README.md` when the new provider architecture exists.
- keep Docker Python removal for the implementation sprint that removes adapter runtime dependencies, not Sprint 0.

### Test Fixtures

Available audio fixtures:

- `test-audio/jfk.wav`: primary fast real-engine smoke fixture.
- `test-audio/sample.wav`: optional broader local validation.
- `test-audio/linus.wav`: optional local validation.
- `test-audio/40min.wav`: opt-in performance/manual validation only.

## Route and Service Impact Matrix

| Area | Current behavior | Target service |
| --- | --- | --- |
| Create transcription | DB insert only | DB insert plus `QueueService.Enqueue` |
| Submit transcription | upload plus DB insert only | upload plus DB insert plus `QueueService.Enqueue` |
| Retry | creates replacement row only | reset/create retry attempt plus enqueue |
| Cancel | direct DB status update | `QueueService.Cancel` with queued/running/orphan semantics |
| Transcript | text plus empty arrays | canonical transcript parser |
| Events | API-local broker only | worker/orchestrator progress publisher |
| Logs | placeholder | sanitized job/execution log reader |
| Executions | placeholder | execution service/repository list |
| Models | static placeholder | provider capabilities |
| Admin queue | status counts | queue service stats |

## Logging Requirements for Implementation Sprints

Use structured logging through `pkg/logger` or package-level wrappers backed by it. Avoid `log.Printf` in new engine/worker/orchestrator code.

Minimum structured log events:

- config loaded:
  - cache dir,
  - requested provider,
  - threads,
  - max loaded,
  - auto-download,
  - worker count,
  - poll interval,
  - lease timeout.
- provider initialized:
  - provider id,
  - requested provider,
  - resolved provider when available,
  - cache dir,
  - max loaded.
- worker lifecycle:
  - worker service start/stop,
  - worker id,
  - poll interval,
  - lease timeout,
  - recovery counts.
- queue operations:
  - enqueue,
  - claim,
  - lease renew success/failure,
  - cancel request,
  - shutdown cancellation.
- orchestration stages:
  - job id,
  - public-safe user id,
  - stage,
  - progress,
  - provider id,
  - model ids.
- terminal states:
  - completed duration,
  - failed sanitized error category,
  - canceled reason.

Public logs/API responses/SSE events must not include:

- absolute upload paths,
- model cache paths,
- temp paths,
- API keys or tokens,
- raw command output,
- full stack traces.

## Sprint-by-Sprint Commit Plan

EWI-Sprint 1:

- config tests,
- config implementation and module wiring.

EWI-Sprint 2:

- provider interfaces and fake provider tests,
- local provider implementation,
- provider capability and sanitization tests.

EWI-Sprint 3:

- schema/repository tests,
- model/schema migration updates,
- repository durable queue methods.

EWI-Sprint 4:

- worker service tests,
- worker service implementation,
- queue stats/cancel/recovery refinements.

EWI-Sprint 5:

- transcript mapper and speaker merge tests,
- orchestrator tests,
- orchestrator implementation.

EWI-Sprint 6:

- API queue-backed tests,
- handler dependency injection,
- create/submit/retry/cancel/models/logs/executions/transcript wiring,
- API path-leak regression fixes.

EWI-Sprint 7:

- lifecycle tests,
- server startup/shutdown wiring,
- legacy adapter deletion or compile exclusion,
- obsolete test deletion/update.

EWI-Sprint 8:

- gated real engine tests,
- `jfk.wav` smoke validation,
- performance smoke notes.

EWI-Sprint 9:

- README/Docker docs,
- troubleshooting updates,
- docs verification.

EWI-Sprint 10:

- full hardening pass,
- final validation,
- tracker updates.

## Risks and Open Decisions

- `references/engine/go.mod` declares Go `1.26`, while Scriberr declares Go `1.24.0` with toolchain `go1.24.4`. Sprint 1 must verify whether local toolchain auto-download is acceptable or whether the engine module needs a compatible Go version.
- `internal/config.Load()` currently cannot return validation errors. Sprint 1 should introduce an error-returning loader while preserving compatibility for older tests.
- `internal/api.NewHandler` currently accepts variadic dependencies and ignores legacy queue/processor args. Sprint 6 should replace this with explicit optional service dependencies without breaking existing tests unnecessarily.
- Current file rows and transcription rows share `models.TranscriptionJob`, distinguished by `source_file_hash`. Repository methods must keep this distinction intact.
- Existing API and tests rely on status string `queued` through `models.StatusPending`. New code should preserve public `queued`.
- Legacy `internal/dropzone` still depends on `EnqueueJob`. It is not wired in server startup today, but it must either receive an adapter over the new queue service or be updated in the cleanup sprint.

## Deletion Targets

Delete or stop compiling after replacement behavior is in place:

- `internal/transcription/adapters/base_adapter.go`
- `internal/transcription/adapters/whisperx_adapter.go`
- `internal/transcription/adapters/parakeet_adapter.go`
- `internal/transcription/adapters/canary_adapter.go`
- `internal/transcription/adapters/voxtral_adapter.go`
- `internal/transcription/adapters/openai_adapter.go`
- `internal/transcription/adapters/pyannote_adapter.go`
- `internal/transcription/adapters/sortformer_adapter.go`
- `internal/transcription/adapters/py/**`
- `internal/transcription/registry/registry.go`
- `internal/transcription/pipeline/pipeline.go`
- obsolete adapter tests:
  - `internal/transcription/adapters_test.go`
  - `tests/adapter_registration_test.go`
  - Python adapter tests under `internal/transcription/adapters/py/**`

Evaluate before deleting:

- `internal/transcription/unified_service.go`
- `internal/transcription/queue_integration.go`
- `internal/transcription/quick_transcription.go`
- `internal/transcription/interfaces/interfaces.go`
- `internal/queue/**`
- `internal/interfaces/queue.go`
- `internal/dropzone/dropzone.go`

## Sprint 0 Acceptance Check

- Legacy startup dependencies are identified.
- Deletion targets are explicit.
- API/service seams are documented.
- Logging requirements are documented.
- Commit plan for Sprints 1-10 is documented.
- Test fixtures for real validation are identified.
