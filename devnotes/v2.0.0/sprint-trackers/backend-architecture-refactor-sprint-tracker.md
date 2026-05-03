# Sprint Tracker: Backend Architecture Refactor

This tracker belongs to `devnotes/v2.0.0/sprint-plans/backend-architecture-refactor-sprint-plan.md`.

Status: completed through Sprint 5.

## Run Rules

- One focused commit per completed sprint.
- Tests or architecture guards are written before implementation.
- Tracker and status note are updated in the same commit as the sprint work.
- `git diff --check` is required for every sprint.
- Dirty worktree entries unrelated to the sprint are documented and left untouched.

## Sprint 0: Baseline, Guard Inventory, And Commit Hygiene

Status: completed

Completed tasks:

- [x] Record current dirty worktree categories before implementation.
- [x] Add/update architecture tests for forbidden imports and dependency directions.
- [x] Add sprint commit hygiene checklist.
- [x] Create baseline status note with known backend coupling and test blockers.

Acceptance checks:

- [x] Architecture guards fail on newly introduced forbidden imports.
- [x] Baseline status note exists.
- [x] No runtime behavior changes.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'`
- [x] `git diff --check`

Artifacts:

- `internal/api/architecture_test.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-refactor-sprint-00-baseline.md`

Commit:

- [x] `backend: establish architecture refactor baseline`

## Sprint 1: API DTO Boundary

Status: completed

Completed tasks:

- [x] Inventory handlers returning persistence models directly.
- [x] Add DTO mapper tests for files, transcriptions, profiles, recordings, and summaries.
- [x] Introduce explicit DTOs/mappers while preserving JSON shape.
- [x] Add path omission and public ID formatting tests.

Acceptance checks:

- [x] Route/API contract tests remain stable.
- [x] Public responses do not expose raw local paths.
- [x] New API response work has DTO coverage.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'`
- [x] `git diff --check`

Notes:

- Full `go test ./internal/api` is blocked by sandbox loopback restrictions in `TestLLMProviderSettingsSaveTestsConnectionAndMasksKey`, which uses `httptest.NewServer`.

Artifacts:

- `internal/api/response_models.go`
- `internal/api/summary_handlers.go`
- `internal/api/response_models_test.go`
- Handler list/event call-site updates.
- `devnotes/v2.0.0/status-updates/backend-architecture-refactor-sprint-01-api-dto-boundary.md`

Commit:

- [x] `backend: harden api dto boundary`

## Sprint 2: Storage Boundary Consolidation

Status: completed

Completed tasks:

- [x] Define local artifact storage interface.
- [x] Move transcript JSON write policy out of the processor and behind the orchestrator artifact boundary.
- [x] Add path traversal and path-leak regression tests.
- [x] Preserve existing on-disk layout and API behavior.

Acceptance checks:

- [x] Handlers do not construct transcript artifact storage paths.
- [x] Processor uses storage/artifact dependency for transcript output.
- [x] Audio streaming and transcript retrieval remain stable in focused API and package tests.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/files ./internal/recording ./internal/transcription/... ./cmd/server`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'`
- [x] `git diff --check`

Artifacts:

- `internal/transcription/orchestrator/artifact_store.go`
- `internal/transcription/orchestrator/processor.go`
- `internal/transcription/orchestrator/processor_test.go`
- `cmd/server/main.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-refactor-sprint-02-storage-boundary.md`

Commit:

- [x] `backend: consolidate transcript artifact storage`

## Sprint 3: Repository Interface Narrowing

Status: completed

Completed tasks:

- [x] Inventory broad repository dependencies in services.
- [x] Split workflow-specific repository ports for automation and transcription.
- [x] Replace unsafe generic automation lookups with file/user-specific methods.
- [x] Add fake-backed service tests for narrowed ports.

Acceptance checks:

- [x] Services depend on smaller workflow-specific persistence ports where touched.
- [x] User-owned automation decisions load only ready file records before deriving the user scope.
- [x] Concrete repository tests still cover GORM implementation.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/account ./internal/profile ./internal/files ./internal/transcription ./internal/automation ./internal/repository ./cmd/server`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'`
- [x] `git diff --check`

Artifacts:

- `internal/automation/post_file_service.go`
- `internal/automation/post_file_service_test.go`
- `internal/transcription/service.go`
- `internal/repository/implementations.go`
- `internal/repository/job_queue_test.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-refactor-sprint-03-repository-ports.md`

Commit:

- [x] `backend: narrow repository service ports`

## Sprint 4: Queue Fairness And Performance Prep

Status: completed

Completed tasks:

- [x] Add claim ordering and index coverage tests.
- [x] Review queue polling and list endpoint indexes.
- [x] Refactor claim logic for future priority and per-user fairness.
- [x] Preserve current default single-user behavior for equal-priority jobs.

Acceptance checks:

- [x] Queue claim behavior is covered by tests.
- [x] Queue hot paths are indexed and bounded.
- [x] State transitions remain atomic.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/database ./internal/repository ./internal/transcription/worker ./cmd/server`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'`
- [x] `git diff --check`

Artifacts:

- `internal/models/transcription.go`
- `internal/database/schema.go`
- `internal/repository/implementations.go`
- `internal/repository/job_queue_test.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-refactor-sprint-04-queue-performance.md`

Commit:

- [x] `backend: prepare queue fairness and performance`

## Sprint 5: Provider Capability Selection

Status: completed

Completed tasks:

- [x] Add provider selector tests with fake providers.
- [x] Implement explicit provider/model selection with capability fallback.
- [x] Keep `local` default behavior stable.
- [x] Route orchestrator provider resolution through selector.

Acceptance checks:

- [x] Adding a second provider does not affect handlers, repositories, or queue code.
- [x] Provider list and selection behavior are deterministic.
- [x] Provider errors remain sanitized.

Verification:

- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/transcription/... ./cmd/server`
- [x] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'`
- [x] `git diff --check`

Artifacts:

- `internal/transcription/engineprovider/types.go`
- `internal/transcription/engineprovider/registry.go`
- `internal/transcription/engineprovider/registry_test.go`
- `internal/transcription/orchestrator/processor.go`
- `internal/transcription/orchestrator/processor_test.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-refactor-sprint-05-provider-selection.md`

Commit:

- [x] `backend: add provider capability selection`

## Sprint 6: Event Boundary Hardening

Status: planned

Planned tasks:

- [ ] Inventory event payloads.
- [ ] Add event payload tests for public IDs and path omission.
- [ ] Move API-shaped event mapping out of business services where needed.
- [ ] Verify terminal events publish after durable state changes.

Acceptance checks:

- [ ] Events stay small and path-free.
- [ ] REST reads can recover missed event state.
- [ ] Terminal event ordering is covered by tests.

Verification:

- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api ./internal/files ./internal/recording ./internal/summarization ./internal/transcription/...`
- [ ] `git diff --check`

Artifacts:

- Event mapper/tests.
- `devnotes/v2.0.0/status-updates/backend-architecture-refactor-sprint-06-event-boundary.md`

Commit:

- [ ] `backend: harden event boundary`

## Sprint 7: Bootstrap Extraction

Status: planned

Planned tasks:

- [ ] Extract app/server construction if composition remains too large.
- [ ] Keep `cmd/server/main.go` focused on process concerns.
- [ ] Add construction tests that do not bind an HTTP listener.
- [ ] Preserve startup and shutdown order.

Acceptance checks:

- [ ] Composition can be tested without starting the server.
- [ ] `api.NewHandler` dependency injection remains explicit.
- [ ] Shutdown remains bounded and ordered.

Verification:

- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./cmd/server ./internal/app ./internal/api`
- [ ] `git diff --check`

Artifacts:

- Optional `internal/app` package.
- `devnotes/v2.0.0/status-updates/backend-architecture-refactor-sprint-07-bootstrap.md`

Commit:

- [ ] `backend: extract app bootstrap`

## Sprint 8: Cleanup, Documentation, And Final Architecture Gate

Status: planned

Planned tasks:

- [ ] Remove dead legacy interfaces/packages that are no longer referenced.
- [ ] Update root architecture docs with final decisions.
- [ ] Tighten architecture guards to hard enforcement where feasible.
- [ ] Run broad backend verification.
- [ ] Write final residual-debt status note.

Acceptance checks:

- [ ] Architecture docs match code.
- [ ] Architecture guards protect final dependency direction.
- [ ] Tracker records all verification and commits.

Verification:

- [ ] `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./internal/files ./internal/recording ./internal/summarization ./internal/chat ./internal/account ./internal/profile ./internal/llmprovider ./internal/automation ./cmd/server`
- [ ] `git diff --check`

Artifacts:

- Updated architecture docs.
- Final architecture guard tests.
- `devnotes/v2.0.0/status-updates/backend-architecture-refactor-sprint-08-final.md`

Commit:

- [ ] `backend: finalize architecture refactor gates`
