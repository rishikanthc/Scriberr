# Sprint Tracker: Backend Architecture Review Remediation

This tracker belongs to `devnotes/v2.0.0/sprint-plans/backend-architecture-review-remediation-sprint-plan.md`.

Status: Sprint 14 complete. Second backend architecture review remediation complete.

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
| Global SSE is not user-scoped | High | Sprint 2 | complete |
| LLM API keys are stored raw | High | Sprint 4 | complete |
| Queue terminal updates are not claim-owned | High | Sprint 5 | complete |
| Recovery requeues every processing job | High | Sprint 5 | complete |
| Chat generation runs inside HTTP handler | High | Sprint 6 | complete |
| External LLM provider adapter lives in API | Medium | Sprint 3 | complete |
| Admin route has no admin authorization | Medium | Sprint 1 | complete |
| Generic/global repository methods remain exposed | Medium | Sprint 7 | complete |
| Admin queue stats are still user-scoped | High | Sprint 13 | complete |
| Scheduler policy boundary is still missing | High | Sprint 12, Sprint 13 | complete |
| User status and disabled-user enforcement are absent | High | Sprint 9 | complete |
| Admin user-management API is not implemented | High | Sprint 10 | complete |
| Settings remain in `users.settings_json` instead of relational settings tables | Medium | Sprint 11, Sprint 12 | complete |
| API response mapping still touches local file paths | Medium | Sprint 14 | complete |

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

Status: complete

Addresses:

- Finding 7: Admin route has no admin authorization.

TDD and scope checks:

- [x] Add failing tests for non-admin access to `/api/v1/admin/queue`.
- [x] Add failing tests for API-key access policy to admin routes.
- [x] Add principal/role helper tests.
- [x] Implement principal extraction and `adminRequired()` middleware.
- [x] Apply admin middleware to admin routes.
- [x] Keep non-admin route auth behavior unchanged.
- [x] Write status note `backend-architecture-review-remediation-sprint-01-admin-auth.md`.

Acceptance checks:

- [x] Anonymous admin route access returns 401.
- [x] Non-admin JWT admin route access returns 403.
- [x] API key admin route access follows the documented policy.
- [x] Admin JWT access succeeds.
- [x] Admin authorization is centralized and reusable.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestSecurity|TestAuth|TestAPIKey|TestCurrentPrincipal|TestCanonicalRouteRegistration|TestEndpointContractSmoke'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/account ./internal/auth ./cmd/server`
- [x] `git diff --check`

Artifacts:

- `internal/api/middleware.go`
- `internal/api/router.go`
- `internal/api/security_regression_test.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-01-admin-auth.md`

Commit:

- [x] `backend: enforce admin authorization`

## Sprint 2: User-Scoped Event Delivery

Status: complete

Addresses:

- Finding 1: Global SSE is not user-scoped.

TDD and scope checks:

- [x] Add failing two-user SSE isolation tests.
- [x] Add failing test that global event stream filters another user's file event.
- [x] Add failing test that transcription event stream filters another user's progress.
- [x] Extend event types and subscriber state with user/audience metadata.
- [x] Update all event publisher adapters to include user ID.
- [x] Preserve path sanitization and public IDs.
- [x] Write status note `backend-architecture-review-remediation-sprint-02-user-scoped-events.md`.

Acceptance checks:

- [x] User A does not receive user B global events.
- [x] User A does not receive user B transcription-specific events.
- [x] User-specific events still deliver correctly.
- [x] Admin event visibility, if any, is explicitly tested and documented.
- [x] Event payloads remain small and path-free.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestEvent|TestSSE|TestSecurity|TestProduction'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/files ./internal/recording ./internal/summarization ./internal/tags ./internal/annotations ./internal/transcription/worker`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/mediaimport ./internal/automation`
- [x] `git diff --check`

Artifacts:

- `internal/api/events_handlers.go`
- `internal/api/events_test.go`
- Event publisher type updates in service/worker packages.
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-02-user-scoped-events.md`

Commit:

- [x] `backend: scope sse events by user`

## Sprint 3: Move LLM Provider Probing Out Of API

Status: complete

Addresses:

- Finding 6: External LLM provider adapter lives in `internal/api`.

TDD and scope checks:

- [x] Add failing architecture guard or inventory test for API-owned LLM provider adapters.
- [x] Add or move LLM provider probing tests into `internal/llmprovider`.
- [x] Move concrete HTTP tester out of `internal/api`.
- [x] Wire concrete tester from `internal/app`.
- [x] Keep API handlers limited to request/response mapping.
- [x] Write status note `backend-architecture-review-remediation-sprint-03-llm-provider-boundary.md`.

Acceptance checks:

- [x] `internal/app` no longer wires `llmprovider.Service` using an `api` concrete tester.
- [x] Provider probing behavior is covered outside API tests.
- [x] LLM provider settings response shape remains stable.
- [x] Architecture guard protects the new boundary.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/llmprovider ./internal/api -run 'TestHTTPConnectionTester|TestLLMProvider|TestProduction|TestBackendDependencyDirection'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/app ./cmd/server`
- [x] `git diff --check`

Artifacts:

- `internal/llmprovider/service.go`
- `internal/api/llm_provider_handlers.go`
- `internal/app/app.go`
- `internal/api/architecture_test.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-03-llm-provider-boundary.md`

Commit:

- [x] `backend: move llm provider probing out of api`

## Sprint 4: Protect LLM Provider Credentials

Status: complete

Addresses:

- Finding 2: LLM API keys are stored raw.

TDD and scope checks:

- [x] Add failing test proving saved LLM config does not contain the raw API key.
- [x] Add failing API test proving raw API key is never returned.
- [x] Add backward-compatibility test for existing plaintext config, if migration is deferred.
- [x] Implement minimal credential protection boundary.
- [x] Inject credential protection through config/app/service wiring.
- [x] Avoid logging or returning raw keys.
- [x] Write status note `backend-architecture-review-remediation-sprint-04-llm-secrets.md`.

Acceptance checks:

- [x] New LLM provider API keys are not stored raw.
- [x] Existing provider settings behavior remains usable.
- [x] Public DTOs expose only safe credential metadata.
- [x] Credential behavior is tested without network/model dependencies.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/config ./internal/database ./internal/models ./internal/repository ./internal/llmprovider`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestLLMProvider|TestSecurity'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/app ./cmd/server`
- [x] `git diff --check`

Artifacts:

- Credential protection package or service.
- `internal/models/transcription.go`
- `internal/llmprovider/service.go`
- `internal/repository/implementations.go`
- `internal/config/config.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-04-llm-secrets.md`

Commit:

- [x] `backend: protect llm provider credentials`

## Sprint 5: Queue Lease Ownership And Safe Recovery

Status: complete

Addresses:

- Finding 3: Queue terminal updates are not claim-owned.
- Finding 4: Recovery requeues every processing job.

TDD and scope checks:

- [x] Add failing test: active processing job with future lease is not recovered.
- [x] Add failing test: expired processing job is recovered.
- [x] Add failing test: stale worker cannot complete after recovery/reclaim.
- [x] Add failing test: stale worker cannot fail after cancellation.
- [x] Add failing test: terminal transition updates only latest/current execution.
- [x] Implement lease-aware recovery.
- [x] Implement claim-owned terminal transitions.
- [x] Update worker flow to pass worker/execution ownership.
- [x] Write status note `backend-architecture-review-remediation-sprint-05-queue-leases.md`.

Acceptance checks:

- [x] Recovery only targets expired or missing leases.
- [x] Complete/fail/cancel terminal writes require current claim owner.
- [x] Stale workers cannot overwrite newer terminal state.
- [x] Existing FIFO/priority behavior remains stable.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository -run 'TestJobRepository'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/worker ./internal/transcription/orchestrator ./internal/api -run 'TestCapabilitiesQueue|TestEngineWorker'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/worker ./internal/transcription/orchestrator`
- [x] `git diff --check`

Artifacts:

- `internal/repository/implementations.go`
- `internal/repository/job_queue_test.go`
- `internal/transcription/worker/service.go`
- Worker tests.
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-05-queue-leases.md`

Commit:

- [x] `backend: enforce queue lease ownership`

## Sprint 6: Move Chat Generation Workflow Into Chat Service

Status: complete

Addresses:

- Finding 5: Chat generation runs inside the HTTP handler.

TDD and scope checks:

- [x] Add failing service-level tests for chat streaming workflow with fake LLM.
- [x] Add or update handler test proving handler delegates to service and writes SSE.
- [x] Move model selection, context build, LLM calls, persistence, and terminal run handling into `internal/chat`.
- [x] Move `chatClientForConfig` out of `internal/api`.
- [x] Add architecture guard: production API must not import `internal/llm`.
- [x] Preserve existing SSE event names and payload shape unless deliberately changed.
- [x] Write status note `backend-architecture-review-remediation-sprint-06-chat-service.md`.

Acceptance checks:

- [x] Chat generation is testable without Gin.
- [x] Handler is thin and HTTP/SSE-focused.
- [x] Existing chat route behavior remains stable.
- [x] API no longer owns concrete LLM client construction.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/chat ./internal/api -run 'TestChat|TestProduction|TestBackendDependencyDirection'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/chat`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/llm ./internal/llmprovider ./cmd/server`
- [x] `git diff --check`

Artifacts:

- `internal/chat/service.go`
- `internal/chat/generation.go`
- `internal/chat/llm_client.go`
- `internal/chat/generation_service_test.go`
- `internal/api/chat_handlers.go`
- `internal/api/architecture_test.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-06-chat-service.md`

Commit:

- [x] `backend: move chat generation into service`

## Sprint 7: Quarantine Legacy And Global Repository Methods

Status: complete

Addresses:

- Finding 8: Generic/global repository methods remain exposed.

TDD and scope checks:

- [x] Add inventory/guard test for production imports of legacy `internal/queue`.
- [x] Add inventory/guard test for unsafe global repository methods where practical.
- [x] Inventory all production references to generic/unscoped methods.
- [x] Remove dead legacy queue code or quarantine it behind explicit legacy build/test boundaries.
- [x] Split worker/system repository interfaces from user-facing service interfaces.
- [x] Replace unscoped product-service lookups with `ForUser`/`ByUser` methods.
- [x] Write status note `backend-architecture-review-remediation-sprint-07-repository-legacy.md`.

Acceptance checks:

- [x] Product services do not use generic unscoped repository methods for user-owned data.
- [x] Legacy queue paths are removed or guarded.
- [x] New unsafe production imports/usages fail tests.
- [x] Repository implementation tests remain stable.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository ./internal/transcription/... ./internal/files ./internal/account ./internal/profile ./internal/automation ./internal/api -run 'TestProduction|TestBackendDependencyDirection'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/summarization ./internal/transcription/worker ./internal/repository`
- [x] `git diff --check`

Artifacts:

- `internal/repository/repository.go`
- `internal/repository/implementations.go`
- `internal/api/architecture_test.go`
- `internal/summarization/service.go`
- `internal/transcription/worker/service.go`
- Removed legacy `internal/queue`.
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-07-repository-legacy.md`

Commit:

- [x] `backend: quarantine legacy repository paths`

## Sprint 8: Final Enforcement, Documentation, And Broad Verification

Status: complete

TDD and scope checks:

- [x] Tighten architecture tests from inventory mode to hard enforcement where feasible.
- [x] Add final regression tests for any previously unguarded finding.
- [x] Update architecture docs only for deliberate decisions made during implementation.
- [x] Write final status note `backend-architecture-review-remediation-sprint-08-final.md`.
- [x] Run broad backend verification or document blockers with substitutes.

Acceptance checks:

- [x] Every review finding is completed or explicitly deferred with rationale.
- [x] Architecture guards protect new boundaries.
- [x] Final status note lists commits, verification, and residual debt.
- [x] Broad backend verification passes or blockers are documented.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestProduction|TestOnlyAppComposition|TestBackendDependencyDirection|TestSettingsPartialUpdateAndValidation'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api ./internal/account ./internal/auth ./internal/chat ./internal/config ./internal/database ./internal/files ./internal/llm ./internal/llmprovider ./internal/recording ./internal/repository ./internal/summarization ./internal/transcription/... ./cmd/server`
- [x] `git diff --check`

Artifacts:

- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-08-final.md`
- `internal/api/architecture_test.go`
- `internal/api/profile_settings_test.go`

Commit:

- [x] `backend: finalize architecture review remediation`

## Sprint 9: User Status And Auth Enforcement

Status: complete

Addresses:

- Follow-up Finding 11: User status and disabled-user enforcement are absent.

TDD and scope checks:

- [x] Add failing tests for disabled-user login, refresh, API-key auth, events, and transcription enqueue.
- [x] Add user status lifecycle fields and migration/backfill.
- [x] Enforce active-user checks in account/API-key auth paths.
- [x] Update login/password-change timestamp behavior.
- [x] Revoke or reject stale credentials as required by the multi-user spec.
- [x] Write status note `backend-architecture-review-remediation-sprint-09-user-status.md`.

Acceptance checks:

- [x] Existing users migrate to `active`.
- [x] First registration creates an active admin.
- [x] Disabled users cannot login.
- [x] Disabled users cannot refresh.
- [x] Disabled users cannot use API keys.
- [x] Disabled users cannot open event streams.
- [x] Disabled users cannot enqueue transcription work.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/account ./internal/auth ./internal/api -run 'TestAuth|TestSecurity|TestAPIKey|TestEvent|TestTranscription'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database ./internal/repository`
- [x] `git diff --check -- internal/models/auth.go internal/account/service.go internal/api/middleware.go internal/repository/implementations.go internal/api/auth_test.go internal/database/database_test.go`

Artifacts:

- `internal/models/auth.go`
- `internal/account/service.go`
- `internal/api/middleware.go`
- `internal/database/*`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-09-user-status.md`

Commit:

- [x] `backend: enforce user account status`

## Sprint 10: Admin User Management Service And Routes

Status: complete

Addresses:

- Follow-up Finding 12: Admin user-management API is not implemented.

TDD and scope checks:

- [x] Add failing admin route contract/security tests for user-management endpoints.
- [x] Add `internal/admin.Service`.
- [x] Add admin-scoped user repository methods.
- [x] Add admin user routes for list/create/get/update/reset-password/disable/enable.
- [x] Enforce active admin JWT only; keep API keys disallowed for admin operations.
- [x] Enforce last-active-admin invariant.
- [x] Revoke refresh tokens and API keys on disable.
- [x] Revoke refresh tokens on password reset.
- [x] Write status note `backend-architecture-review-remediation-sprint-10-admin-users.md`.

Acceptance checks:

- [x] Admin can create a normal user.
- [x] Admin can list and inspect users.
- [x] Admin can disable and enable users.
- [x] Admin can reset user passwords.
- [x] Admin cannot disable or demote the last active admin.
- [x] Normal users and API keys cannot access admin user management.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/admin ./internal/account ./internal/api -run 'TestAdmin|TestSecurity|TestAuth|TestCanonicalRouteRegistration|TestEndpointContractSmoke'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository ./internal/database ./internal/app`
- [x] `git diff --check -- internal/admin/service.go internal/repository/implementations.go internal/api/router.go internal/app/app.go internal/api/types.go internal/api/admin_handlers.go internal/api/auth_test.go internal/api/route_contract_test.go internal/api/admin_user_handlers_test.go internal/api/middleware.go`

Artifacts:

- `internal/admin/*`
- `internal/api/admin_handlers.go`
- `internal/api/router.go`
- `internal/repository/*`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-10-admin-users.md`

Commit:

- [x] `backend: add admin user management`

## Sprint 11: Relational User Settings

Status: complete

Addresses:

- Follow-up Finding 13: Settings remain in `users.settings_json` instead of relational settings tables.

TDD and scope checks:

- [x] Add failing migration/backfill tests for `user_settings`.
- [x] Add `models.UserSettings`.
- [x] Add `repository.UserSettingsRepository`.
- [x] Move account settings reads/writes to `user_settings`.
- [x] Preserve `/api/v1/settings` response shape.
- [x] Enforce same-user default profile ownership.
- [x] Keep legacy JSON only as explicit migration fallback.
- [x] Write status note `backend-architecture-review-remediation-sprint-11-user-settings.md`.

Acceptance checks:

- [x] Existing settings backfill into `user_settings`.
- [x] New writes use relational settings rows.
- [x] Partial settings updates preserve unrelated fields.
- [x] Auto-transcription/default-profile validation still works.
- [x] Auto-rename/small-model validation still works.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/account ./internal/api -run 'TestSettings|TestProfile'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database ./internal/repository ./internal/automation ./internal/summarization`
- [x] `git diff --check`

Artifacts:

- `internal/models/auth.go`
- `internal/database/*`
- `internal/repository/*`
- `internal/account/service.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-11-user-settings.md`

Commit:

- [x] `backend: move user settings to relational table`

## Sprint 12: System Settings And Scheduler Policy Boundary

Status: complete

Addresses:

- Follow-up Finding 10: Scheduler policy boundary is still missing.
- Follow-up Finding 13: Settings remain in `users.settings_json` instead of relational settings tables.

TDD and scope checks:

- [x] Add failing scheduler config validation tests.
- [x] Add `models.SystemSetting`.
- [x] Add `repository.SystemSettingsRepository`.
- [x] Add default `queue.scheduler` migration/backfill.
- [x] Add `internal/transcription/scheduler` policy/config package.
- [x] Add admin service methods for scheduler get/update.
- [x] Add `GET` and `PUT /api/v1/admin/queue/scheduler`.
- [x] Keep queue claim behavior unchanged until Sprint 13.
- [x] Write status note `backend-architecture-review-remediation-sprint-12-scheduler-settings.md`.

Acceptance checks:

- [x] Scheduler config is persisted in `system_settings`.
- [x] Invalid scheduler config is rejected before persistence.
- [x] Default policy is `priority`.
- [x] Admin scheduler routes require active admin JWT auth.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/scheduler ./internal/admin ./internal/api -run 'TestAdmin|TestScheduler|TestSecurity'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database ./internal/repository ./internal/app`
- [x] `git diff --check`

Artifacts:

- `internal/transcription/scheduler/*`
- `internal/admin/*`
- `internal/models/*`
- `internal/repository/*`
- `internal/api/admin_handlers.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-12-scheduler-settings.md`

Commit:

- [x] `backend: add scheduler system settings`

## Sprint 13: Configurable Queue Claims And Admin Queue Stats

Status: complete

Addresses:

- Follow-up Finding 9: Admin queue stats are still user-scoped.
- Follow-up Finding 10: Scheduler policy boundary is still missing.

TDD and scope checks:

- [x] Add failing tests proving admin queue stats include multiple users.
- [x] Add failing tests proving normal queue stats remain user-scoped.
- [x] Add failing scheduler claim tests for priority, FIFO, weighted duration, and fair share.
- [x] Change worker service to load scheduler config through a narrow port.
- [x] Change repository claim method to accept `scheduler.Config`.
- [x] Implement deterministic claim policies.
- [x] Add queue indexes required by policy/list paths.
- [x] Update `/api/v1/admin/queue` to return global aggregates and `by_user`.
- [x] Write status note `backend-architecture-review-remediation-sprint-13-queue-scheduler.md`.

Acceptance checks:

- [x] Admin queue stats are global and include per-user breakdown.
- [x] Normal queue stats are scoped to current user.
- [x] Priority remains default scheduler policy.
- [x] FIFO ordering is deterministic.
- [x] Weighted-duration policy is deterministic and includes aging.
- [x] Fair-share respects configured per-user concurrency.
- [x] Queue claim remains repository-owned and atomic.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository -run 'TestJobRepository|TestScheduler|TestQueue'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/worker ./internal/transcription/scheduler ./internal/admin ./internal/api -run 'TestAdmin|TestQueue|TestScheduler|TestSecurity'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database`
- [x] `git diff --check`

Artifacts:

- `internal/repository/implementations.go`
- `internal/transcription/worker/service.go`
- `internal/transcription/scheduler/*`
- `internal/api/admin_handlers.go`
- `internal/database/*`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-13-queue-scheduler.md`

Commit:

- [x] `backend: configure shared queue scheduler`

## Sprint 14: File Metadata And Storage Boundary Cleanup

Status: complete

Addresses:

- Follow-up Finding 14: API response mapping still touches local file paths.

TDD and scope checks:

- [x] Add failing architecture guard for API file response filesystem access.
- [x] Add failing tests for file metadata responses without direct path probing.
- [x] Move size/kind/MIME metadata lookup behind `internal/files`.
- [x] Prefer persisted metadata over filesystem probing for list/get responses.
- [x] Keep audio streaming ownership checks in the file service path.
- [x] Ensure public file DTOs remain path-free.
- [x] Write status note `backend-architecture-review-remediation-sprint-14-file-metadata-boundary.md`.

Acceptance checks:

- [x] `internal/api/response_models.go` no longer imports `os` for file metadata.
- [x] API code does not construct or inspect local file paths for DTO mapping.
- [x] File list/get response shape remains stable.
- [x] Missing physical files do not leak local paths.
- [x] Audio streaming still checks database ownership before opening storage.

Verification:

- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/files ./internal/api -run 'TestFile|TestResponse|TestProduction|TestBackendDependencyDirection'`
- [x] `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/recording ./internal/mediaimport ./internal/transcription/orchestrator`
- [x] `git diff --check`

Artifacts:

- `internal/files/service.go`
- `internal/api/file_handlers.go`
- `internal/api/response_models.go`
- `internal/api/architecture_test.go`
- `devnotes/v2.0.0/status-updates/backend-architecture-review-remediation-sprint-14-file-metadata-boundary.md`

Commit:

- [x] `backend: keep file metadata behind service boundary`
