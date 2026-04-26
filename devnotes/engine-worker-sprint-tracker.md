# Sprint Run Tracker: Engine Worker Integration

Run ID: `EWI`

Status: planning only. No implementation has started.

This tracker belongs to `devnotes/engine-worker-sprints.md` and the implementation spec in `devnotes/engine-worker-integration-spec.md`.

## EWI-Sprint 0: Integration Inventory and Commit Plan

Status: completed

Completed tasks:

- Inventoried server startup, config, schema, repository, queue, transcription stack, API placeholders, docs, Docker, and test fixtures.
- Documented the legacy adapter deletion targets.
- Documented API/service seams for create, submit, retry, cancel, transcript, events, logs, executions, models, and queue stats.
- Added structured logging requirements for config, provider, worker, queue, orchestration, and terminal states.
- Added a sprint-by-sprint commit plan for EWI-Sprints 1-10.

Artifacts:

- `devnotes/engine-worker-sprint-0-inventory.md`

Verification:

- Inventory-only sprint. No runtime code changed.
- Focused repository inspection completed with `rg`, `find`, and targeted source reads.

## EWI-Sprint 1: Config and Engine Module Wiring

Status: completed

Completed tasks:

- Added local engine module wiring with `require scriberr-engine v0.0.0` and `replace scriberr-engine => ./references/engine`.
- Added `config.EngineConfig` and `config.WorkerConfig`.
- Added `config.LoadWithError()` for startup-failing validation while retaining `config.Load()` for compatibility.
- Parsed and validated all `SPEECH_ENGINE_*` and `TRANSCRIPTION_*` env vars from the spec.
- Updated server startup to fail clearly on invalid config.
- Added structured startup logging for engine and worker configuration.
- Added focused config tests before implementation.

Artifacts:

- `go.mod`
- `cmd/server/main.go`
- `internal/config/config.go`
- `internal/config/config_test.go`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/config` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `git diff --check` passed.

## EWI-Sprint 2: Engine Provider Abstraction

Status: completed

Completed tasks:

- Added `internal/transcription/engineprovider` provider and registry interfaces.
- Added internal provider request/result/capability types so `scriberr-engine` types do not leak outside the provider boundary.
- Added static provider registry with deterministic capability aggregation.
- Added local provider wrapper for `scriberr-engine/speech/engine`.
- Mapped Scriberr transcription and diarization requests to local engine requests.
- Forced token timestamps for local transcription requests.
- Mapped engine words and diarization segments to public-safe internal result structs.
- Added model capability discovery from the engine model specs with install state through `IsModelInstalled`.
- Added provider error sanitization for paths and token-like values.
- Added focused fake-engine tests for mapping, empty words, capabilities, diarization speakers, close behavior, and sanitized errors.
- Updated the main module to `go 1.26` because the local `scriberr-engine` module declares `go 1.26`.

Artifacts:

- `internal/transcription/engineprovider/types.go`
- `internal/transcription/engineprovider/registry.go`
- `internal/transcription/engineprovider/local_provider.go`
- `internal/transcription/engineprovider/sanitize.go`
- `internal/transcription/engineprovider/*_test.go`
- `go.mod`
- `go.sum`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider` passed.
- `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware` passed with escalation because an existing webhook integration test opens a local `httptest` listener.
- `git diff --check` passed.
- Verified no non-provider Go package imports `scriberr-engine`.

## EWI-Sprint 3: Queue Schema and Repository Methods

Status: completed

Completed tasks:

- Added durable queue/lease/progress fields to `models.TranscriptionJob`.
- Added queue claim and claim-expiry indexes to the target schema.
- Extended `JobRepository` with durable worker methods for enqueue, FIFO claim, lease renewal, startup recovery, progress, completion, failure, cancellation, and execution listing.
- Implemented transactional terminal updates that keep the job row and latest execution row consistent.
- Added focused repository tests for schema/indexes, enqueue, FIFO claim, concurrent claim deduplication, owner-only lease renewal, orphan recovery, progress updates, terminal transitions, and execution listing.
- Updated existing legacy transcription test mocks to satisfy the expanded repository interface until the legacy stack is removed in later sprints.

Artifacts:

- `internal/models/transcription.go`
- `internal/database/schema.go`
- `internal/repository/implementations.go`
- `internal/repository/job_queue_test.go`
- `internal/transcription/adapters_test.go`
- `tests/test_helpers.go`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/repository -run 'TestJobRepository'` passed.
- `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware` passed with escalation because an existing webhook integration test opens a local `httptest` listener.
- `git diff --check` passed.

## EWI-Sprint 4: Durable Worker Service

Status: completed

Completed tasks:

- Added `internal/transcription/worker` with the public queue service interface from the sprint plan.
- Implemented durable enqueue plus non-blocking worker wake signaling.
- Implemented worker startup recovery through `RecoverOrphanedProcessing`.
- Implemented polling/claim loop with configurable worker count, poll interval, lease timeout, renew interval, and stop timeout.
- Implemented lease renewal while processors are running.
- Implemented process-local cancel tracking for running jobs.
- Implemented cancel behavior for queued jobs, process-local running jobs, orphaned processing jobs, and terminal-state conflicts.
- Implemented user-scoped queue stats with process-local running counts.
- Added structured lifecycle, enqueue, worker, lease-renewal, cancellation, and shutdown logs.
- Added focused worker tests with fake processors for enqueue/wake/complete, cancel queued, cancel running, lease renewal, stop cancellation, stats, and cancel conflicts.
- Added repository status-count support needed by worker stats.

Artifacts:

- `internal/transcription/worker/service.go`
- `internal/transcription/worker/service_test.go`
- `internal/repository/implementations.go`
- `internal/transcription/adapters_test.go`
- `tests/test_helpers.go`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/worker` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./tests -run '^$'` passed.
- `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware` passed with escalation because an existing webhook integration test opens a local `httptest` listener.
- `git diff --check` passed.

## EWI-Sprint 5: Orchestrator, Transcript Mapping, and Speaker Merge

Status: completed

Completed tasks:

- Added `internal/transcription/orchestrator` with a worker-compatible processor.
- Added canonical transcript structs, JSON parsing, mapper, fallback segment generation, and legacy plain-text/older-JSON fallback parsing.
- Implemented overlap-based speaker assignment for words and segments with stable public `SPEAKER_00` labels.
- Implemented provider/model/language/task/diarization request resolution.
- Created execution rows at processor start with sanitized request/config metadata.
- Published progress stages for preparing, transcribing, diarizing, merging, saving, completed, failed, and canceled paths.
- Wrote canonical transcript JSON to the configured transcript output directory and returned the internal output path for worker completion.
- Preserved `words: []` when token timestamps are absent.
- Sanitized provider failures to redact paths and token-like values.
- Distinguished context cancellation from provider failure.

Artifacts:

- `internal/transcription/orchestrator/processor.go`
- `internal/transcription/orchestrator/transcript.go`
- `internal/transcription/orchestrator/processor_test.go`
- `internal/transcription/orchestrator/transcript_test.go`

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `git diff --check` passed.

## EWI-Sprint 6: API Wiring for Real Queue Execution

Status: completed

Completed tasks:

- Added API handler injection for durable queue service and engine provider registry.
- Wired create, submit, and retry to enqueue through the queue service.
- Mapped queue shutdown to `503 SERVICE_UNAVAILABLE` without deleting durable job rows.
- Wired cancel to queue service cancellation and mapped terminal-state conflicts to `409`.
- Added progress fields to transcription get/list responses.
- Implemented canonical transcript endpoint parsing for JSON, legacy text, and older JSON without `words`.
- Implemented executions endpoint with sanitized execution metadata and processing duration.
- Implemented logs endpoint as authenticated plain text derived from execution metadata/log files with path/token redaction.
- Implemented model listing from provider capabilities with installed/default flags.
- Updated queue stats to use queue service stats when injected, including canceled/running counts.
- Added an API event publisher adapter for orchestrator progress events with path-safe payloads.
- Added focused API tests for queue-backed create/retry/cancel, queue unavailable errors, transcript/execution/log/model/stats responses, and leak-safe errors.

Artifacts:

- `internal/api/router.go`
- `internal/api/transcription_handlers.go`
- `internal/api/admin_handlers.go`
- `internal/api/response_models.go`
- `internal/api/engine_worker_api_test.go`
- API test helper updates

Verification:

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...` passed.
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./cmd/server ./pkg/logger ./pkg/middleware` passed.
- `git diff --check` passed.

## EWI-Sprint 7: Server Startup, Shutdown, and Legacy Adapter Removal

Status: not started

Planned artifacts:

- `cmd/server/main.go`
- deleted or disabled legacy Python adapter stack
- lifecycle tests where practical

Verification:

- Pending

## EWI-Sprint 8: Real Engine Integration Tests and Performance Smoke

Status: not started

Planned artifacts:

- gated real engine integration tests
- `test-audio/jfk.wav` smoke notes

Verification:

- Pending

## EWI-Sprint 9: Docs, Docker, and Setup UX

Status: not started

Planned artifacts:

- `README.md`
- Docker compose files
- docs/troubleshooting updates

Verification:

- Pending

## EWI-Sprint 10: Hardening, Cleanup, and Release Candidate

Status: not started

Planned artifacts:

- final tracker updates
- hardening fixes
- cleanup commits

Verification:

- Pending
