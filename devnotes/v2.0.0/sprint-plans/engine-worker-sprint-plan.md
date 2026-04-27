# Sprint Run: Engine Worker Integration

Run ID: `EWI`

Status: planning only. Do not implement code from this document until the user explicitly starts an implementation sprint.

## Scope

This sprint run implements `devnotes/engine-worker-integration-spec.md` end to end: local Go engine integration, durable transcription workers, canonical transcript JSON, executions/logs/models APIs, removal of legacy Python adapter startup paths, docs, and verification with fixture audio.

The work should stay backend-first. Frontend changes are out of scope unless an API contract change makes the current UI unable to compile or use the canonical endpoints.

## Engineering Rules

- Follow test-driven development: write the narrow failing tests first, implement, then refactor.
- Keep commits small and intentional. Each sprint should usually produce 2-5 commits, grouped by behavior:
  - tests that define the target behavior,
  - implementation,
  - cleanup/docs,
  - verification fixes.
- Do not mix unrelated cleanup into implementation commits.
- Keep the API layer thin. Handlers validate, map requests/responses, and call service interfaces.
- Keep SQLite as the source of truth. In-memory state may wake workers and cancel local jobs only.
- Do not leak local file paths, model cache paths, tokens, raw command output, or env-specific internals through API responses, logs endpoints, or SSE events.
- Preserve future multi-user scheduling hooks by carrying `user_id` through claim, execution, cancellation, and stats code.
- Prefer fake providers/processors for fast tests. Real engine tests must be opt-in and skipped cleanly when runtime/model dependencies are unavailable.
- Use `test-audio/jfk.wav` for fast real-path smoke tests. Use longer audio only for opt-in local performance checks.

## Validation Baseline

Run these checks before and after each implementation sprint when possible:

```sh
GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware
GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware
git diff --check
```

Real engine validation is opt-in:

```sh
SCRIBERR_ENGINE_ITEST=1 SPEECH_ENGINE_AUTO_DOWNLOAD=true GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/... -run 'Test.*RealEngine|Test.*JFK'
```

Performance-oriented manual validation should record:

- audio file used,
- selected provider and resolved provider,
- model download time if any,
- transcription wall time,
- CPU/GPU mode,
- resulting transcript word/segment counts.

## EWI-Sprint 0: Integration Inventory and Commit Plan

Goal: lock down the exact files, dependencies, and deletion targets before changing runtime behavior.

Tasks:

- Inventory current transcription API handlers, queue package usage, `internal/transcription` packages, server startup wiring, config fields, schema fields, docs, and Docker env examples.
- Identify all imports of legacy adapter/registry/pipeline code and decide whether each file is removed, replaced, or kept as a compatibility wrapper.
- Confirm `references/engine` compiles as a local replacement module and document any platform/runtime prerequisites.
- Write a route/API impact matrix for create/submit/retry/cancel/models/logs/executions/events.
- Create a commit checklist for the remaining sprints.

Acceptance criteria:

- Deletion list for legacy Python adapter stack is explicit.
- No implementation sprint starts with unknown server startup dependencies.
- The intended commit grouping is documented.

Testing focus:

- Compile-only discovery if Go dependencies are available.
- No product behavior tests required.

## EWI-Sprint 1: Config and Engine Module Wiring

Goal: add engine/worker configuration and module wiring without starting workers or downloading models.

Tasks:

- Add `require scriberr-engine v0.0.0` and `replace scriberr-engine => ./references/engine`.
- Add `EngineConfig` and `WorkerConfig` to `internal/config`.
- Parse and validate:
  - `SPEECH_ENGINE_CACHE_DIR`
  - `SPEECH_ENGINE_PROVIDER`
  - `SPEECH_ENGINE_THREADS`
  - `SPEECH_ENGINE_MAX_LOADED`
  - `SPEECH_ENGINE_AUTO_DOWNLOAD`
  - `TRANSCRIPTION_WORKERS`
  - `TRANSCRIPTION_QUEUE_POLL_INTERVAL`
  - `TRANSCRIPTION_LEASE_TIMEOUT`
- Make invalid numeric, duration, boolean, and provider values fail startup with clear config errors.
- Keep startup free of model downloads.
- Start de-emphasizing `WHISPERX_ENV` in config without breaking existing callers until legacy startup code is removed.

Acceptance criteria:

- Defaults match the spec exactly.
- `auto`, `cpu`, and `cuda` provider values parse; other values fail.
- Config errors are actionable and do not panic.

Testing focus:

- Defaults.
- Invalid provider.
- Invalid integer/duration/boolean.
- Auto-download default.
- Worker default remains one worker.

Commit guidance:

- Commit config tests first.
- Commit config implementation and module wiring second.

## EWI-Sprint 2: Engine Provider Abstraction

Goal: create a provider boundary that hides `scriberr-engine` from API, repository, and worker callers.

Tasks:

- Add `internal/transcription/engineprovider`.
- Define provider, registry, capability, request, result, transcript word/segment, and diarization segment types.
- Implement an in-memory registry with default provider lookup.
- Implement a fake provider for tests.
- Implement local provider wrapping `scriberr-engine/speech/engine`.
- Map Scriberr requests to engine requests and map engine results back to internal provider results.
- Implement provider capabilities and installed state via engine model metadata and `IsModelInstalled`.
- Sanitize provider errors before they can be returned to API clients.
- Log detailed errors internally without public path leakage.

Acceptance criteria:

- Only `engineprovider` imports `scriberr-engine`.
- Local provider `ID()` is `local`.
- Defaults are `whisper-base` and `diarization-default`.
- Missing words return `Words: []`, never nil-dependent API behavior.
- Provider registry supports future non-local providers.

Testing focus:

- Request mapping.
- Result mapping with words.
- Empty words.
- Diarization result mapping.
- Capability listing and installed flags.
- Error sanitization.

Commit guidance:

- Commit interface/fake-provider tests first.
- Commit local provider implementation separately.

## EWI-Sprint 3: Queue Schema and Repository Methods

Goal: make transcription job state durable enough for workers, leases, progress, and executions.

Tasks:

- Add schema fields to `models.TranscriptionJob`:
  - `queued_at`
  - `started_at`
  - `failed_at`
  - `progress`
  - `progress_stage`
  - `claimed_by`
  - `claim_expires_at`
  - `engine_id`
- Ensure `TranscriptionJobExecution` stores provider, model name/family, started/completed/failed timestamps, sanitized error, output JSON path, and request/config JSON.
- Add indexes:
  - `idx_transcriptions_queue_claim(status, queued_at)`
  - `idx_transcriptions_claim_expires_at(claim_expires_at)`
- Add repository methods for enqueue, claim, renew, progress, complete, fail, cancel, execution listing, and startup recovery.
- Use transactions for claim and terminal-state updates.
- Keep claim policy isolated behind a FIFO scheduler policy.

Acceptance criteria:

- Claim returns the oldest queued job by `queued_at`, `created_at`, and `id`.
- Concurrent claims do not return the same job.
- Lease renewal updates only the owning worker.
- Startup recovery requeues processing rows regardless of stale process-local owner state.
- Terminal updates keep job and latest execution consistent.

Testing focus:

- Migration/schema fields and indexes.
- Enqueue state transition.
- FIFO claim.
- Concurrent claim race.
- Lease renewal owner mismatch.
- Startup recovery.
- Complete/fail/cancel terminal transactions.

Commit guidance:

- Commit schema/repository tests first.
- Commit schema/model updates and repository implementation second.

## EWI-Sprint 4: Durable Worker Service

Goal: implement the worker loop, wake signals, cancellation, stats, leases, and shutdown behavior using fake processors.

Tasks:

- Add `internal/transcription/worker`.
- Define `QueueService` as specified.
- Implement enqueue as durable DB update plus non-blocking wake signal.
- Implement workers that poll, claim one job, renew lease, process with cancellable context, and write terminal state through repository/orchestrator results.
- Track cancel funcs for currently running process-local jobs.
- Implement cancel behavior for queued, local running, and orphaned processing jobs.
- Implement queue stats by user.
- Implement clean `Start` and `Stop`.

Acceptance criteria:

- `Enqueue` moves jobs to queued and wakes workers.
- Workers process jobs without duplicate claims.
- Running jobs renew leases until terminal state.
- `Stop` cancels local running jobs and waits within a bounded timeout.
- Cancel returns conflict for completed/failed/canceled jobs.

Testing focus:

- Enqueue/wake hot path.
- Fake processor completes a job.
- Fake processor observes cancellation.
- Cancel queued.
- Cancel running.
- Claim lease renewal.
- Worker shutdown.
- Queue stats.

Commit guidance:

- Commit queue-service tests first.
- Commit worker implementation separately.

## EWI-Sprint 5: Orchestrator, Transcript Mapping, and Speaker Merge

Goal: convert claimed jobs into completed canonical transcripts through provider calls.

Tasks:

- Add `internal/transcription/orchestrator`.
- Implement `Processor` with job repository, provider registry, event publisher, and job logger dependencies.
- Resolve audio path, provider, transcription model, diarization model, language, task, and diarization options.
- Create execution rows at processing start.
- Publish progress stages:
  - queued,
  - preparing,
  - transcribing,
  - diarizing,
  - merging,
  - saving,
  - completed/failed/canceled.
- Persist canonical transcript JSON in `transcriptions.transcript_text`.
- Write the same JSON to `data/transcripts/{jobID}/transcript.json` and store internal output path.
- Generate fallback segments when needed.
- Preserve `words: []` when words are absent.
- Merge diarization speakers into words and segments using overlap.
- Sanitize failures and distinguish user cancellation from engine failure.

Acceptance criteria:

- Fake provider can complete a transcription job end to end.
- Canonical transcript JSON matches the spec.
- No diarization leaves speaker fields absent.
- Diarization assigns stable `SPEAKER_00` style labels.
- Failures update job and execution consistently.
- Context cancellation marks canceled, not failed.

Testing focus:

- Transcript mapper with words.
- Transcript mapper without words.
- Plain-text legacy fallback.
- Older JSON without `words`.
- Segment fallback.
- Word and segment speaker overlap assignment.
- Provider failure sanitization.
- Cancellation path.

Commit guidance:

- Commit mapper/merge tests first.
- Commit orchestrator tests and implementation second.

## EWI-Sprint 6: API Wiring for Real Queue Execution

Goal: replace transcription placeholders with queue-backed behavior while preserving canonical API contracts.

Tasks:

- Inject queue service, provider registry/model service, execution service, and log reader into API handler construction.
- Update create/submit to create queued rows and call `QueueService.Enqueue`.
- Update retry to reset eligible terminal jobs and enqueue a new attempt.
- Update cancel to call `QueueService.Cancel`.
- Add progress fields to transcription get/list responses.
- Implement transcript endpoint through canonical transcript parser.
- Implement executions endpoint with sanitized metadata.
- Implement logs endpoint as authenticated plain text with sanitization.
- Implement models endpoint from provider capabilities.
- Keep SSE payloads path-safe and progress-shaped.

Acceptance criteria:

- Create/submit return `202` queued resources.
- Queue shutdown produces `503` without losing durable job state.
- Fake engine worker can complete a job, and transcript endpoint returns text, segments, and words.
- Executions endpoint returns execution metadata.
- Logs endpoint returns sanitized text.
- Models endpoint returns local capabilities and installed/default flags.
- No API response/event leaks upload path, temp path, model cache path, or raw internal error details.

Testing focus:

- Create enqueues.
- Submit upload creates file, job, and enqueue.
- Retry valid and conflict states.
- Cancel queued/running/conflict states.
- Transcript canonical response.
- Fake worker completion through API-visible state.
- Events progress and completed payloads.
- Executions/logs/models endpoints.
- Path-leak regression tests.

Commit guidance:

- Commit API tests first.
- Commit dependency injection and service wiring separately.
- Commit endpoint behavior updates last.

## EWI-Sprint 7: Server Startup, Shutdown, and Legacy Adapter Removal

Goal: make the real engine worker the default runtime path and remove obsolete Python adapter bootstrapping.

Tasks:

- Update `cmd/server/main.go` startup order to match the spec.
- Initialize local engine provider registry after DB/repositories and before worker service.
- Run DB recovery before starting workers.
- Start workers before HTTP serving.
- On shutdown, stop workers, cancel running jobs, close engine providers, then close DB.
- Remove server startup dependencies on Python, `WHISPERX_ENV`, and legacy adapter registration.
- Delete or stop compiling:
  - `internal/transcription/adapters/**`
  - legacy registry/pipeline abstractions that only serve Python adapters
  - obsolete adapter tests
- Keep compatibility wrappers only when needed to compile API or service boundaries.

Acceptance criteria:

- Fresh server startup does not require Python env setup.
- Startup logs include cache dir, requested provider, resolved provider when available, threads, max-loaded, and auto-download.
- Shutdown releases worker and provider resources.
- Legacy Python adapter code no longer participates in server runtime.

Testing focus:

- Server construction with fake provider/worker where practical.
- Startup recovery is invoked before worker start.
- Shutdown order test or focused unit test around lifecycle coordinator.
- Compile regression proving deleted adapter code is no longer referenced.

Commit guidance:

- Commit lifecycle tests first.
- Commit startup wiring.
- Commit legacy deletion as its own reviewable commit.

## EWI-Sprint 8: Real Engine Integration Tests and Performance Smoke

Goal: prove the local engine path works with fixture audio without making CI slow or flaky.

Tasks:

- Add opt-in real engine integration tests gated by `SCRIBERR_ENGINE_ITEST=1`.
- Use `test-audio/jfk.wav` for fast transcription verification.
- Skip cleanly with clear messages when ffmpeg, engine runtime, model downloads, CUDA runtime, or network access are unavailable.
- Validate that first-run auto-download works when enabled.
- Validate `SPEECH_ENGINE_AUTO_DOWNLOAD=false` fails jobs cleanly when models are missing.
- Add a small benchmark or timed smoke test that reports wall time without setting brittle pass/fail thresholds.
- Document manual performance commands for `jfk.wav`, `sample.wav`, and optional longer fixtures.

Acceptance criteria:

- Default CI remains fake-provider only.
- Opt-in real test can produce a completed transcript for `test-audio/jfk.wav`.
- Real test asserts non-empty text and path-safe public outputs.
- Auto-download disabled path fails with a sanitized model-unavailable error.

Testing focus:

- Real local provider transcription with `jfk.wav`.
- Optional diarization only if model/runtime availability makes it practical.
- Auto-download enabled/disabled behavior.
- No path leakage from real failures.

Commit guidance:

- Commit gated integration tests separately from runtime implementation fixes.

## EWI-Sprint 9: Hardening, Cleanup

Goal: audit the full implementation for correctness, performance, privacy, and maintainability.

Tasks:

- Run the full backend test/vet baseline.
- Run focused race/concurrency tests for queue claiming if practical.
- Run opt-in `jfk.wav` real engine smoke.
- Audit public API responses and events for path/token leakage.
- Audit handler packages for business logic that should move behind services.
- Audit repository terminal updates for transaction boundaries.
- Audit worker shutdown and cancellation behavior.
- Remove dead code, obsolete TODOs, and stale docs from earlier sprints.
- Update `devnotes/engine-worker-sprint-tracker.md` with final verification results.

Acceptance criteria:

- Full fake-provider test suite passes.
- Real `jfk.wav` smoke passes or has a documented external dependency blocker.
- No known path leaks in API, events, logs endpoint, or tests.
- Queue restart recovery is covered.
- Commit history is organized by sprint and behavior.

Testing focus:

- Full package tests and vet.
- API regression tests.
- Queue concurrency/recovery tests.
- Transcript compatibility tests.
- Real engine smoke.

Commit guidance:

- Commit hardening fixes in narrow patches.
- Final commit should be docs/tracker updates only.

## Minimum Test Coverage Set

The target is not 100% coverage. The minimum set must cover the highest-risk paths:

- Config parse defaults and invalid values.
- Provider request/result mapping and sanitized errors.
- Queue enqueue, claim, lease renew, recovery, cancel queued, cancel running, and no duplicate claim.
- Orchestrator success, provider failure, cancellation, canonical JSON, words absent, and diarization merge.
- API create/submit/retry/cancel/transcript/events/executions/logs/models hot paths.
- Security/path-leak regression for API responses, events, logs, and provider errors.
- Gated real engine transcription using `test-audio/jfk.wav`.

## Completion Definition

This sprint run is complete when:

- Fresh install starts without Python environment setup.
- Missing models download automatically on first job when enabled.
- A transcription moves from queued to processing to completed or failed.
- Completed transcripts include text, segments, and word timestamps when available.
- Missing word timestamps are represented as `words: []`.
- Diarization assigns public-safe speaker labels when requested and available.
- Events report progress and terminal states.
- Executions and logs endpoints are implemented and sanitized.
- Queue state survives restart and recovers orphaned processing jobs.
- Legacy Python adapter bootstrap is gone from server startup.
- Real `jfk.wav` smoke has been run or blocked by a documented external dependency.
