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

Status: not started

Planned artifacts:

- `internal/transcription/engineprovider`

Verification:

- Pending

## EWI-Sprint 3: Queue Schema and Repository Methods

Status: not started

Planned artifacts:

- `internal/models/transcription.go`
- `internal/database/schema.go`
- `internal/repository/*`
- focused repository/database tests

Verification:

- Pending

## EWI-Sprint 4: Durable Worker Service

Status: not started

Planned artifacts:

- `internal/transcription/worker`

Verification:

- Pending

## EWI-Sprint 5: Orchestrator, Transcript Mapping, and Speaker Merge

Status: not started

Planned artifacts:

- `internal/transcription/orchestrator`
- transcript mapper and speaker merge tests

Verification:

- Pending

## EWI-Sprint 6: API Wiring for Real Queue Execution

Status: not started

Planned artifacts:

- `internal/api/transcription_handlers.go`
- `internal/api/events_handlers.go`
- `internal/api/response_models.go`
- API tests for queue-backed behavior

Verification:

- Pending

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
