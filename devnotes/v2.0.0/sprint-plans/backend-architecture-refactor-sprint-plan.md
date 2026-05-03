# Backend Architecture Refactor Sprint Plan

This plan follows the target design in:

- `devnotes/v2.0.0/architecture-design.md`
- `devnotes/v2.0.0/backend-rules.md`
- `devnotes/v2.0.0/rules/backend-architecture-rules.md`

It is the next backend hardening series after the completed service-boundary refactor. The goal is not a rewrite. The goal is to tighten existing seams, reduce model/path leakage, improve queue/storage/provider boundaries, and leave a clean, test-backed commit history.

## Operating Rules

Each sprint must be small enough to review as one focused change set.

Required workflow for every sprint:

1. Start from a clean or intentionally documented worktree.
2. Write or update failing tests/architecture guards first.
3. Implement the smallest change that satisfies the tests.
4. Run focused package tests, then broader backend tests where practical.
5. Run `git diff --check`.
6. Update the tracker and write a short status note.
7. Commit one coherent sprint with a descriptive message.

Commit format:

```txt
backend: <sprint scope>
```

Do not mix unrelated frontend, asset, formatting, dependency, or generated-output churn into backend refactor commits.

## Test And Quality Gates

Default verification set:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./internal/files ./internal/recording ./internal/summarization ./internal/chat ./internal/account ./internal/profile ./internal/llmprovider ./internal/automation ./cmd/server
git diff --check
```

Use narrower package tests while developing. If sandbox loopback restrictions block tests that use `httptest.NewServer`, record the blocker in the tracker and run the nearest focused non-loopback tests.

Performance-sensitive changes must include either:

- a repository/service test that protects query shape or state transitions, or
- a benchmark/micro-measurement note when the behavior is too integration-heavy for unit tests.

## Non-Goals

- Do not split the backend into microservices.
- Do not replace GORM during this series.
- Do not redesign the public API unless an API sprint explicitly owns the contract change.
- Do not add remote ASR providers until the provider-selection boundary is ready.
- Do not rename large package trees unless the sprint specifically includes migration tests and import guards.

## Sprint 0: Baseline, Guard Inventory, And Commit Hygiene

Goal: establish the guardrails before changing behavior.

Tasks:

- Record current dirty worktree categories before starting implementation.
- Add or update architecture tests for forbidden dependency directions:
  - production `internal/api` must not import `internal/database`
  - `internal/models` must not import services/API/providers
  - `internal/repository` must not import API/services/providers
  - provider packages must not import API/repository
- Add a lightweight checklist for sprint commit hygiene.
- Create a status note with the current backend coupling inventory and test baseline.

Acceptance criteria:

- Architecture guard tests fail on newly introduced forbidden imports.
- Tracker and status note define the baseline and known test blockers.
- No runtime behavior changes.

## Sprint 1: API DTO Boundary

Goal: stop new API contract work from depending on persistence structs.

Tasks:

- Inventory handler responses that expose `internal/models` directly.
- Add DTO mapper tests for the highest-risk public resources:
  - files
  - transcriptions
  - profiles
  - recordings
  - summaries
- Introduce explicit DTOs/mappers where handlers currently return persistence records.
- Preserve existing JSON response shape unless a route contract test is deliberately updated.
- Add a guard or review checklist item that new API responses use DTOs.

Acceptance criteria:

- Public response shape remains stable in route/API tests.
- New DTO mappers cover public ID formatting and path omission.
- No raw local paths appear in API JSON.

## Sprint 2: Storage Boundary Consolidation

Goal: centralize durable audio, import, recording, and transcript object access.

Tasks:

- Define a narrow storage/object interface for local durable artifacts.
- Move transcript JSON write/open behavior out of `orchestrator` into a storage adapter or transcript artifact service.
- Keep upload and recording storage path construction inside storage-owning packages.
- Add path traversal and path-leak regression tests.
- Preserve current on-disk layout and migration compatibility.

Acceptance criteria:

- Handlers do not construct or expose durable file paths.
- Transcription processor receives a storage/artifact dependency instead of directly owning output path policy.
- Existing audio streaming and transcript retrieval behavior is unchanged.

## Sprint 3: Repository Interface Narrowing

Goal: reduce broad repository coupling and remove generic persistence methods from service hot paths.

Tasks:

- Inventory services that depend on large concrete repository interfaces.
- Split service-owned ports by workflow where useful:
  - file metadata store
  - transcription command store
  - queue store
  - profile lookup store
  - LLM config lookup store
- Replace generic `FindByID` usage in services where ownership or user scope matters.
- Add fake-backed service tests for each narrowed port.

Acceptance criteria:

- Services depend on the smallest repository surface needed for their workflow.
- User-scoped operations do not use unscoped generic lookups.
- Repository tests still cover the concrete GORM implementation.

## Sprint 4: Queue Fairness And Performance Prep

Goal: prepare the durable queue for multi-user fairness without changing default single-user behavior.

Tasks:

- Add repository tests for queue claim ordering and indexed candidate selection.
- Add `priority` and per-user concurrency design if missing from schema, or document why it is deferred.
- Refactor claim logic so future per-user fairness can be added in one repository method.
- Verify queue stats, claim, renew, cancel, recover, complete, and fail remain atomic.
- Review indexes for queue polling and list endpoints.

Acceptance criteria:

- Current FIFO behavior remains stable unless the sprint explicitly changes it.
- Claim query behavior is covered by tests.
- Queue hot paths remain indexed and bounded.

## Sprint 5: Provider Capability Selection

Goal: move provider choice from default-only behavior toward capability-based selection.

Tasks:

- Extend provider capability types only as much as current UI/API needs.
- Add a provider selection service/function that accepts explicit provider/model first and capability requirements second.
- Keep `local` as the default provider.
- Add fake-provider tests for explicit selection, unavailable provider, missing capability, and deterministic fallback.
- Ensure orchestrator uses the selector instead of open-coded default provider resolution.

Acceptance criteria:

- Adding a second provider does not require handler, repository, or queue changes.
- Provider errors remain sanitized.
- Model capability listing stays deterministic.

## Sprint 6: Event Boundary Hardening

Goal: keep SSE/event publishing small, durable-state-backed, and testable.

Tasks:

- Inventory event payloads emitted by API, worker, files, recording, summarization, tags, and annotations.
- Add tests that event payloads contain public IDs and omit local paths/provider internals.
- Move event naming/payload mapping out of business services where it is currently too API-shaped.
- Ensure state is persisted before user-visible terminal events are emitted.

Acceptance criteria:

- Events are notification payloads only.
- Missed events can be recovered by REST reads.
- Event payload tests cover terminal transcription, file-ready, recording, and summary events.

## Sprint 7: Bootstrap Extraction

Goal: make startup wiring easier to test without changing runtime behavior.

Tasks:

- Extract server construction from `cmd/server/main.go` into `internal/app` if the composition root remains large.
- Keep `cmd/server/main.go` as flag parsing, logging startup, signal handling, and process exit.
- Add tests for app construction with fake or temporary dependencies.
- Preserve startup order: config, DB, repositories, providers, services, API, workers.

Acceptance criteria:

- Composition can be tested without starting the HTTP listener.
- `api.NewHandler` still receives explicit dependencies.
- Shutdown order remains explicit and bounded.

## Sprint 8: Cleanup, Documentation, And Final Architecture Gate

Goal: close the refactor series with durable docs and strict guards.

Tasks:

- Remove dead legacy interfaces/packages that are no longer referenced.
- Update `devnotes/v2.0.0/architecture-design.md` and `devnotes/v2.0.0/backend-rules.md` with any decisions made during implementation.
- Tighten architecture tests from warning/inventory mode to hard enforcement where feasible.
- Run the broad backend verification set.
- Write final status note with residual debt and deferred work.

Acceptance criteria:

- No stale architecture docs contradict the code.
- Final architecture guards protect the intended dependency direction.
- Tracker lists every completed sprint, verification command, and commit.
