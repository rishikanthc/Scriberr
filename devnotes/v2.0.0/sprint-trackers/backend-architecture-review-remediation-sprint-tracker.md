# Sprint Tracker: Backend Architecture Review Remediation

This tracker belongs to `devnotes/v2.0.0/sprint-plans/backend-architecture-review-remediation-sprint-plan.md`.

Status: Sprint 0 complete. Runtime remediation starts with Sprint 1.

## Run Rules

- Follow test-driven development for every sprint.
- Write or update a failing test/architecture guard before implementation.
- Keep each sprint scoped to one coherent finding group.
- Run focused tests before broader backend tests.
- Run `git diff --check` before marking a sprint complete.
- Update this tracker and write the sprint status note in the same change set as implementation.
- Leave unrelated dirty worktree changes untouched and document them when relevant.

## Review Finding Coverage

| Finding | Severity | Sprint | Status |
| --- | --- | --- | --- |
| Global SSE is not user-scoped | High | Sprint 2 | pending |
| LLM API keys are stored raw | High | Sprint 4 | pending |
| Queue terminal updates are not claim-owned | High | Sprint 5 | pending |
| Recovery requeues every processing job | High | Sprint 5 | pending |
| Chat generation runs inside HTTP handler | High | Sprint 6 | pending |
| External LLM provider adapter lives in API | Medium | Sprint 3 | pending |
| Admin route has no admin authorization | Medium | Sprint 1 | pending |
| Generic/global repository methods remain exposed | Medium | Sprint 7 | pending |

## Sprint 0: Baseline, Guard Plan, And Review Anchors

Status: complete

TDD and scope checks:

- [x] Record current worktree state and unrelated changes.
- [x] Inventory existing tests covering events, auth/admin, LLM provider settings, queue transitions, chat streaming, and architecture guards.
- [x] Add or update architecture guard tests in failing/inventory mode where feasible.
- [x] Run focused guard tests and record baseline result.
- [x] Write status note `backend-architecture-review-remediation-sprint-00-baseline.md`.

Acceptance checks:

- [x] Tracker maps every review finding to a sprint.
- [x] Baseline status note exists.
- [x] No runtime behavior changes.
- [x] Known test blockers are documented.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestProduction|TestBackendDependencyDirection|TestCanonicalRouteRegistration|TestEndpointContractSmoke'`
- [x] `git diff --check`

Artifacts:

- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-00-baseline.md`

Commit:

- [x] `backend: plan architecture review remediation`

## Sprint 1: Auth Principal And Admin Authorization

Status: pending

Addresses:

- Finding 7: Admin route has no admin authorization.

TDD and scope checks:

- [ ] Add failing tests for non-admin access to `/api/v1/admin/queue`.
- [ ] Add failing tests for API-key access policy to admin routes.
- [ ] Add principal/role helper tests.
- [ ] Implement principal extraction and `adminRequired()` middleware.
- [ ] Apply admin middleware to admin routes.
- [ ] Keep non-admin route auth behavior unchanged.
- [ ] Write status note `backend-architecture-review-remediation-sprint-01-admin-auth.md`.

Acceptance checks:

- [ ] Anonymous admin route access returns 401.
- [ ] Non-admin JWT admin route access returns 403.
- [ ] API key admin route access follows the documented policy.
- [ ] Admin JWT access succeeds.
- [ ] Admin authorization is centralized and reusable.

Verification:

- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestSecurity|TestAuth|TestAPIKey|TestCanonicalRouteRegistration|TestEndpointContractSmoke'`
- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/account ./internal/auth ./cmd/server`
- [ ] `git diff --check`

Artifacts:

- `internal/api/middleware.go`
- `internal/api/router.go`
- `internal/api/security_regression_test.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-01-admin-auth.md`

Commit:

- [ ] `backend: enforce admin authorization`

## Sprint 2: User-Scoped Event Delivery

Status: pending

Addresses:

- Finding 1: Global SSE is not user-scoped.

TDD and scope checks:

- [ ] Add failing two-user SSE isolation tests.
- [ ] Add failing test that global event stream filters another user's file event.
- [ ] Add failing test that transcription event stream filters another user's progress.
- [ ] Extend event types and subscriber state with user/audience metadata.
- [ ] Update all event publisher adapters to include user ID.
- [ ] Preserve path sanitization and public IDs.
- [ ] Write status note `backend-architecture-review-remediation-sprint-02-user-scoped-events.md`.

Acceptance checks:

- [ ] User A does not receive user B global events.
- [ ] User A does not receive user B transcription-specific events.
- [ ] User-specific events still deliver correctly.
- [ ] Admin event visibility, if any, is explicitly tested and documented.
- [ ] Event payloads remain small and path-free.

Verification:

- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestEvent|TestSSE|TestSecurity|TestProduction'`
- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/files ./internal/recording ./internal/summarization ./internal/tags ./internal/annotations ./internal/transcription/worker`
- [ ] `git diff --check`

Artifacts:

- `internal/api/events_handlers.go`
- `internal/api/events_test.go`
- Event publisher type updates in service/worker packages.
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-02-user-scoped-events.md`

Commit:

- [ ] `backend: scope sse events by user`

## Sprint 3: Move LLM Provider Probing Out Of API

Status: pending

Addresses:

- Finding 6: External LLM provider adapter lives in `internal/api`.

TDD and scope checks:

- [ ] Add failing architecture guard or inventory test for API-owned LLM provider adapters.
- [ ] Add or move LLM provider probing tests into `internal/llmprovider`.
- [ ] Move concrete HTTP tester out of `internal/api`.
- [ ] Wire concrete tester from `internal/app`.
- [ ] Keep API handlers limited to request/response mapping.
- [ ] Write status note `backend-architecture-review-remediation-sprint-03-llm-provider-boundary.md`.

Acceptance checks:

- [ ] `internal/app` no longer wires `llmprovider.Service` using an `api` concrete tester.
- [ ] Provider probing behavior is covered outside API tests.
- [ ] LLM provider settings response shape remains stable.
- [ ] Architecture guard protects the new boundary.

Verification:

- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/llmprovider ./internal/api -run 'TestLLMProvider|TestProduction|TestBackendDependencyDirection'`
- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/app ./cmd/server`
- [ ] `git diff --check`

Artifacts:

- `internal/llmprovider/service.go`
- `internal/api/llm_provider_handlers.go`
- `internal/app/app.go`
- `internal/api/architecture_test.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-03-llm-provider-boundary.md`

Commit:

- [ ] `backend: move llm provider probing out of api`

## Sprint 4: Protect LLM Provider Credentials

Status: pending

Addresses:

- Finding 2: LLM API keys are stored raw.

TDD and scope checks:

- [ ] Add failing test proving saved LLM config does not contain the raw API key.
- [ ] Add failing API test proving raw API key is never returned.
- [ ] Add backward-compatibility test for existing plaintext config, if migration is deferred.
- [ ] Implement minimal credential protection boundary.
- [ ] Inject credential protection through config/app/service wiring.
- [ ] Avoid logging or returning raw keys.
- [ ] Write status note `backend-architecture-review-remediation-sprint-04-llm-secrets.md`.

Acceptance checks:

- [ ] New LLM provider API keys are not stored raw.
- [ ] Existing provider settings behavior remains usable.
- [ ] Public DTOs expose only safe credential metadata.
- [ ] Credential behavior is tested without network/model dependencies.

Verification:

- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/config ./internal/database ./internal/models ./internal/repository ./internal/llmprovider ./internal/api -run 'TestLLMProvider|TestSecurity'`
- [ ] `git diff --check`

Artifacts:

- Credential protection package or service.
- `internal/models/transcription.go`
- `internal/llmprovider/service.go`
- `internal/repository/implementations.go`
- `internal/config/config.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-04-llm-secrets.md`

Commit:

- [ ] `backend: protect llm provider credentials`

## Sprint 5: Queue Lease Ownership And Safe Recovery

Status: pending

Addresses:

- Finding 3: Queue terminal updates are not claim-owned.
- Finding 4: Recovery requeues every processing job.

TDD and scope checks:

- [ ] Add failing test: active processing job with future lease is not recovered.
- [ ] Add failing test: expired processing job is recovered.
- [ ] Add failing test: stale worker cannot complete after recovery/reclaim.
- [ ] Add failing test: stale worker cannot fail after cancellation.
- [ ] Add failing test: terminal transition updates only latest/current execution.
- [ ] Implement lease-aware recovery.
- [ ] Implement claim-owned terminal transitions.
- [ ] Update worker flow to pass worker/execution ownership.
- [ ] Write status note `backend-architecture-review-remediation-sprint-05-queue-leases.md`.

Acceptance checks:

- [ ] Recovery only targets expired or missing leases.
- [ ] Complete/fail/cancel terminal writes require current claim owner.
- [ ] Stale workers cannot overwrite newer terminal state.
- [ ] Existing FIFO/priority behavior remains stable.

Verification:

- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository -run 'TestJobRepository'`
- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/worker ./internal/transcription/orchestrator ./internal/api -run 'TestCapabilitiesQueue|TestEngineWorker'`
- [ ] `git diff --check`

Artifacts:

- `internal/repository/implementations.go`
- `internal/repository/job_queue_test.go`
- `internal/transcription/worker/service.go`
- Worker tests.
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-05-queue-leases.md`

Commit:

- [ ] `backend: enforce queue lease ownership`

## Sprint 6: Move Chat Generation Workflow Into Chat Service

Status: pending

Addresses:

- Finding 5: Chat generation runs inside the HTTP handler.

TDD and scope checks:

- [ ] Add failing service-level tests for chat streaming workflow with fake LLM.
- [ ] Add or update handler test proving handler delegates to service and writes SSE.
- [ ] Move model selection, context build, LLM calls, persistence, and terminal run handling into `internal/chat`.
- [ ] Move `chatClientForConfig` out of `internal/api`.
- [ ] Add architecture guard: production API must not import `internal/llm`.
- [ ] Preserve existing SSE event names and payload shape unless deliberately changed.
- [ ] Write status note `backend-architecture-review-remediation-sprint-06-chat-service.md`.

Acceptance checks:

- [ ] Chat generation is testable without Gin.
- [ ] Handler is thin and HTTP/SSE-focused.
- [ ] Existing chat route behavior remains stable.
- [ ] API no longer owns concrete LLM client construction.

Verification:

- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/chat ./internal/api -run 'TestChat|TestProduction|TestBackendDependencyDirection'`
- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/llm ./internal/llmprovider ./cmd/server`
- [ ] `git diff --check`

Artifacts:

- `internal/chat/service.go`
- `internal/chat/service_test.go`
- `internal/api/chat_handlers.go`
- `internal/api/architecture_test.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-06-chat-service.md`

Commit:

- [ ] `backend: move chat generation into service`

## Sprint 7: Quarantine Legacy And Global Repository Methods

Status: pending

Addresses:

- Finding 8: Generic/global repository methods remain exposed.

TDD and scope checks:

- [ ] Add inventory/guard test for production imports of legacy `internal/queue`.
- [ ] Add inventory/guard test for unsafe global repository methods where practical.
- [ ] Inventory all production references to generic/unscoped methods.
- [ ] Remove dead legacy queue code or quarantine it behind explicit legacy build/test boundaries.
- [ ] Split worker/system repository interfaces from user-facing service interfaces.
- [ ] Replace unscoped product-service lookups with `ForUser`/`ByUser` methods.
- [ ] Write status note `backend-architecture-review-remediation-sprint-07-repository-legacy.md`.

Acceptance checks:

- [ ] Product services do not use generic unscoped repository methods for user-owned data.
- [ ] Legacy queue paths are removed or guarded.
- [ ] New unsafe production imports/usages fail tests.
- [ ] Repository implementation tests remain stable.

Verification:

- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository ./internal/transcription/... ./internal/files ./internal/account ./internal/profile ./internal/automation ./internal/api -run 'TestProduction|TestBackendDependencyDirection'`
- [ ] `git diff --check`

Artifacts:

- `internal/repository/repository.go`
- `internal/repository/implementations.go`
- `internal/api/architecture_test.go`
- Service interfaces touched by the cleanup.
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-07-repository-legacy.md`

Commit:

- [ ] `backend: quarantine legacy repository paths`

## Sprint 8: Final Enforcement, Documentation, And Broad Verification

Status: pending

TDD and scope checks:

- [ ] Tighten architecture tests from inventory mode to hard enforcement where feasible.
- [ ] Add final regression tests for any previously unguarded finding.
- [ ] Update architecture docs only for deliberate decisions made during implementation.
- [ ] Write final status note `backend-architecture-review-remediation-sprint-08-final.md`.
- [ ] Run broad backend verification or document blockers with substitutes.

Acceptance checks:

- [ ] Every review finding is completed or explicitly deferred with rationale.
- [ ] Architecture guards protect new boundaries.
- [ ] Final status note lists commits, verification, and residual debt.
- [ ] Broad backend verification passes or blockers are documented.

Verification:

- [ ] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api ./internal/account ./internal/auth ./internal/chat ./internal/config ./internal/database ./internal/files ./internal/llm ./internal/llmprovider ./internal/recording ./internal/repository ./internal/summarization ./internal/transcription/... ./cmd/server`
- [ ] `git diff --check`

Artifacts:

- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-08-final.md`
- Architecture guard tests.
- Documentation updates if needed.

Commit:

- [ ] `backend: finalize architecture review remediation`
