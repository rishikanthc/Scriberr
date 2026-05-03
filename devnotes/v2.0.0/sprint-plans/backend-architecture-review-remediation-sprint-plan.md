# Backend Architecture Review Remediation Sprint Plan

This plan remediates the findings documented in:

- `devnotes/v2.0.0/status-updates/backend-architecture-code-review-2026-05-03.md`

Primary standards:

- `devnotes/v2.0.0/specs/architecture-design.md`
- `devnotes/v2.0.0/rules/backend-rules.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-architecture-review-remediation-sprint-tracker.md`

## Product Goal

Bring the backend closer to strict compliance with the v2.0.0 architecture rules before more multi-user, scheduler, remote-provider, or automation work is built on top of it.

This series initially addressed eight review findings:

1. Global SSE is not user-scoped.
2. LLM API keys are stored raw.
3. Queue terminal updates are not claim-owned.
4. Recovery requeues every processing job.
5. Chat generation runs inside the HTTP handler.
6. External LLM provider adapter lives in `internal/api`.
7. Admin route has no admin authorization.
8. Generic/global repository methods remain exposed.

A follow-up backend review after Sprint 8 added six more multi-user and architecture findings:

9. Admin queue stats are still user-scoped.
10. Scheduler policy boundary is still missing.
11. User status and disabled-user enforcement are absent.
12. Admin user-management API is not implemented.
13. Settings remain in `users.settings_json` instead of relational settings tables.
14. API response mapping still touches local file paths.

## Refactor Goal

Preserve existing user-visible behavior while hardening the boundaries that matter for future maintainability:

```txt
HTTP/API -> services -> repositories/providers/storage -> database/files/engines
```

The end state for this remediation series:

- Events are filtered by authorized user or explicit admin audience.
- Admin routes have explicit role authorization.
- Provider credentials are not stored raw.
- LLM provider probing is outside the API adapter.
- Queue recovery and terminal transitions respect leases and worker ownership.
- Chat generation orchestration is owned by `internal/chat`, not handlers.
- Legacy/global repository methods are removed or quarantined behind system-only interfaces.
- Architecture tests prevent regressions in the reviewed areas.

## Operating Rules

Each sprint should be reviewable as one focused change set.

Required TDD workflow for every sprint:

1. Start from a clean or intentionally documented worktree.
2. Write the smallest failing test, regression test, or architecture guard that proves the finding.
3. Run that focused test and record the expected failure in the status note or tracker.
4. Implement the smallest behavior change that makes the test pass.
5. Refactor only within the sprint scope after tests pass.
6. Run focused package tests.
7. Run broader backend tests when practical.
8. Run `git diff --check`.
9. Update the tracker and write a short status note.
10. Commit one coherent sprint with a descriptive message.

Commit format:

```txt
backend: <remediation scope>
```

Do not mix frontend changes, generated assets, dependency churn, or unrelated refactors into these commits.

## Code Quality Bar

Every sprint must preserve these quality expectations:

- Keep handlers thin: authenticate, bind/validate, call one service method, map response/events.
- Keep service ports narrow and fakeable.
- Keep queue state transitions repository-owned and atomic.
- Keep user-owned operations explicitly user-scoped.
- Wrap errors with useful context at package boundaries, but do not leak secrets or local paths to public responses.
- Prefer deterministic tests with fakes over sleeping or model/network-dependent tests.
- Add concurrency tests for queue, lease, cancellation, and event fanout behavior.
- Avoid broad abstractions until the sprint has a concrete duplication or boundary problem to solve.
- Do not leave TODOs in production code unless the tracker captures the deferred work.

## Default Verification Set

Use narrower tests while developing. Before completing a sprint, run the focused verification listed under that sprint plus `git diff --check`.

Default broad backend check:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api ./internal/account ./internal/auth ./internal/chat ./internal/config ./internal/database ./internal/files ./internal/llm ./internal/llmprovider ./internal/recording ./internal/repository ./internal/summarization ./internal/transcription/... ./cmd/server
git diff --check
```

If network, loopback, ASR model, or sandbox restrictions block a test, record the blocker in the tracker and run the nearest focused non-blocked tests.

## Non-Goals

- Do not redesign the public REST API unless a sprint explicitly calls out a route contract change.
- Do not add remote ASR providers in this series.
- Do not replace GORM or SQLite.
- Do not redesign the whole storage layer.
- Do not build a full metrics/tracing system.
- Do not split the backend into services or processes.

## Review Finding Coverage Matrix

| Finding | Sprint(s) |
| --- | --- |
| Global SSE is not user-scoped | Sprint 2 |
| LLM API keys are stored raw | Sprint 4 |
| Queue terminal updates are not claim-owned | Sprint 5 |
| Recovery requeues every processing job | Sprint 5 |
| Chat generation runs inside HTTP handler | Sprint 6 |
| External LLM provider adapter lives in API | Sprint 3 |
| Admin route has no admin authorization | Sprint 1 |
| Generic/global repository methods remain exposed | Sprint 7 |
| Admin queue stats are still user-scoped | Sprint 13 |
| Scheduler policy boundary is still missing | Sprint 12, Sprint 13 |
| User status and disabled-user enforcement are absent | Sprint 9 |
| Admin user-management API is not implemented | Sprint 10 |
| Settings remain in `users.settings_json` instead of relational settings tables | Sprint 11, Sprint 12 |
| API response mapping still touches local file paths | Sprint 14 |

## Sprint 0: Baseline, Guard Plan, And Review Anchors

Goal: establish a clean remediation baseline before changing runtime behavior.

Tasks:

- Record current worktree state and known unrelated changes.
- Add a status note linking the code review findings to this sprint plan and tracker.
- Inventory tests that currently cover:
  - event payloads and SSE;
  - auth/admin route access;
  - LLM provider settings;
  - queue claim/recover/terminal transitions;
  - chat streaming;
  - repository architecture guards.
- Add or update architecture guard placeholders where feasible without changing behavior:
  - API must not import concrete LLM provider/client packages unless temporarily allowed by inventory.
  - production code should not import legacy `internal/queue` once migration is complete.
  - admin route tests should be able to assert explicit admin middleware after Sprint 1.

Acceptance criteria:

- Tracker is initialized with every review finding mapped to a sprint.
- Status note records current baseline and known test blockers.
- No runtime behavior changes.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestProduction|TestBackendDependencyDirection|TestCanonicalRouteRegistration|TestEndpointContractSmoke'
git diff --check
```

Commit:

```txt
backend: plan architecture review remediation
```

## Sprint 1: Auth Principal And Admin Authorization

Addresses:

- Finding 7: Admin route has no admin authorization.

Goal: introduce explicit principal/role authorization and protect admin routes before adding future admin scheduler controls.

Tasks:

- Define an API principal helper that exposes:
  - `UserID`
  - `Username`
  - `Role`
  - `AuthType`
  - optional `APIKeyID`
- Ensure JWT claims include enough role data, or load role safely through the account/auth boundary where needed.
- Decide whether API keys can access admin routes. Default should be no unless a scoped API-key model exists.
- Add `adminRequired()` middleware.
- Apply `adminRequired()` to `/api/v1/admin/queue`.
- If `/api/v1/admin/queue` remains user-scoped, either:
  - keep it admin-only because of route namespace, or
  - move user stats to a non-admin route in a separate API sprint.
- Add route/security regression tests:
  - anonymous user gets 401;
  - non-admin authenticated user gets 403;
  - API key gets 403 unless explicitly allowed;
  - admin JWT succeeds.

Acceptance criteria:

- Admin routes require explicit admin role authorization.
- Role checks are centralized and reusable.
- Existing non-admin routes continue to accept the same supported auth mechanisms.
- Route contract tests document the intended admin behavior.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestSecurity|TestAuth|TestAPIKey|TestCanonicalRouteRegistration|TestEndpointContractSmoke'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/account ./internal/auth ./cmd/server
git diff --check
```

Commit:

```txt
backend: enforce admin authorization
```

## Sprint 2: User-Scoped Event Delivery

Addresses:

- Finding 1: Global SSE is not user-scoped.

Goal: make SSE safe for multi-user operation by filtering events by authenticated audience.

Tasks:

- Extend event publisher event types to carry `UserID` where missing:
  - file events;
  - transcription worker/orchestrator events;
  - recording events;
  - summary events;
  - annotation/tag events;
  - settings/profile events.
- Extend `apiEvent` and `eventSubscriber` with audience metadata.
- Update global `/api/v1/events` subscriptions to record the current principal.
- Update transcription-specific subscriptions to require both:
  - user authorization for the transcription; and
  - matching transcription ID.
- Filter event delivery by user ID or explicit admin audience.
- Preserve public ID formatting and path sanitization.
- Add two-user regression tests:
  - user A does not receive user B file events;
  - user A does not receive user B transcription progress;
  - user-specific transcription SSE still receives its own events;
  - admin behavior is explicitly tested if admin global visibility is supported.

Acceptance criteria:

- No authenticated user receives another user's non-admin events.
- Existing single-user SSE behavior remains stable.
- Event payloads remain small and path-free.
- Missed events remain recoverable through REST reads.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestEvent|TestSSE|TestSecurity|TestProduction'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/files ./internal/recording ./internal/summarization ./internal/tags ./internal/annotations ./internal/transcription/worker
git diff --check
```

Commit:

```txt
backend: scope sse events by user
```

## Sprint 3: Move LLM Provider Probing Out Of API

Addresses:

- Finding 6: External LLM provider adapter lives in `internal/api`.

Goal: make `internal/api` a pure HTTP adapter for LLM provider settings.

Tasks:

- Move `LLMProviderConnectionTester` and HTTP probing helpers out of `internal/api`.
- Preferred target: `internal/llmprovider`.
- Keep `llmprovider.Service` depending on a narrow `ConnectionTester` interface.
- Wire the concrete tester from `internal/app`.
- Leave handler responsibilities as:
  - authenticate;
  - bind/validate request syntax;
  - call `llmprovider.Service`;
  - map service errors and DTOs.
- Update tests to use service fakes or package-local fake testers.
- Add/extend architecture guard:
  - production `internal/api` must not define concrete LLM provider adapters;
  - production `internal/api` should not import `internal/llm` unless still temporarily needed for chat before Sprint 6.

Acceptance criteria:

- App wiring no longer constructs `llmprovider.Service` with an `api` concrete tester.
- Provider probing can be tested without importing API.
- LLM provider handlers remain behavior-compatible.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/llmprovider ./internal/api -run 'TestLLMProvider|TestProduction|TestBackendDependencyDirection'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/app ./cmd/server
git diff --check
```

Commit:

```txt
backend: move llm provider probing out of api
```

## Sprint 4: Protect LLM Provider Credentials

Addresses:

- Finding 2: LLM API keys are stored raw.

Goal: stop persisting LLM provider API keys as plaintext.

Tasks:

- Design a minimal credential protection boundary.
- Choose a startup-provided encryption secret/key source:
  - explicit env/config value in production;
  - persisted development key similar to JWT secret only if acceptable;
  - clear validation errors when required production config is missing.
- Encrypt LLM provider API keys before persistence.
- Decrypt only inside service/provider client construction paths.
- Keep public DTOs limited to `has_api_key`, `key_preview`, and model/config metadata.
- Add migration/backward compatibility for existing plaintext values:
  - detect legacy plaintext config JSON;
  - encrypt on next save or during migration;
  - avoid logging raw values.
- Add tests:
  - saved config JSON does not contain raw key;
  - service can still test/list models using decrypted key;
  - public API never returns raw key;
  - old plaintext config can be read and re-saved safely if migration is deferred.

Acceptance criteria:

- No new LLM provider credentials are stored raw.
- Existing behavior for configuring providers is preserved.
- Credential protection is injected/configured, not hidden in handlers.
- Tests prove API responses and logs do not expose raw keys.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/config ./internal/database ./internal/models ./internal/repository ./internal/llmprovider ./internal/api -run 'TestLLMProvider|TestSecurity'
git diff --check
```

Commit:

```txt
backend: protect llm provider credentials
```

## Sprint 5: Queue Lease Ownership And Safe Recovery

Addresses:

- Finding 3: Queue terminal updates are not claim-owned.
- Finding 4: Recovery requeues every processing job.

Goal: make queue recovery and terminal transitions safe under concurrent workers and future multi-process workers.

Tasks:

- Change recovery to target only expired or missing leases:
  - `status = processing`;
  - `claim_expires_at IS NULL OR claim_expires_at <= now`.
- Change terminal transition repository methods to require claim ownership:
  - worker ID;
  - `status = processing`;
  - `claimed_by = workerID`;
  - ideally `latest_execution_id = executionID`.
- Return a clear domain error for lost lease/stale worker conflicts.
- Update worker processor flow to carry worker ID and execution ID into terminal methods.
- Ensure cancellation of running jobs cannot be overwritten by stale completion.
- Add tests:
  - active lease is not recovered;
  - expired lease is recovered;
  - stale worker cannot complete after recovery;
  - stale worker cannot fail after cancellation;
  - terminal update changes only the latest execution.
- Update logs to include job ID, worker ID, user ID, execution ID, terminal status, and duration where practical.

Acceptance criteria:

- Recovery does not disturb actively leased jobs.
- Terminal writes require current lease ownership.
- Stale workers produce conflict/lost-lease behavior, not state overwrite.
- Existing single-process queue behavior remains stable.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository -run 'TestJobRepository'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/worker ./internal/transcription/orchestrator ./internal/api -run 'TestCapabilitiesQueue|TestEngineWorker'
git diff --check
```

Commit:

```txt
backend: enforce queue lease ownership
```

## Sprint 6: Move Chat Generation Workflow Into Chat Service

Addresses:

- Finding 5: Chat generation runs inside the HTTP handler.

Goal: make chat generation a service-owned workflow while preserving existing streaming API behavior.

Tasks:

- Define a service-level chat streaming command:
  - user ID;
  - session ID;
  - content;
  - model override;
  - temperature;
  - event sink or returned event channel.
- Move model selection, model availability checks, context building, LLM client calls, message persistence, run persistence, usage persistence, and terminal state handling into `internal/chat.Service`.
- Keep handler responsibilities to:
  - authenticate;
  - parse/validate JSON syntax;
  - call one service method;
  - write service stream events as SSE frames.
- Move `chatClientForConfig` out of `internal/api`.
- Add fake LLM streaming tests at the service layer.
- Preserve route response/event names unless intentionally changed.
- Add architecture guard to ensure production `internal/api` no longer imports `internal/llm`.

Acceptance criteria:

- `streamChatMessage` no longer owns generation orchestration.
- Chat generation is testable without Gin.
- Handler remains thin and focused on HTTP/SSE.
- Existing chat route tests continue to pass.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/chat ./internal/api -run 'TestChat|TestProduction|TestBackendDependencyDirection'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/llm ./internal/llmprovider ./cmd/server
git diff --check
```

Commit:

```txt
backend: move chat generation into service
```

## Sprint 7: Quarantine Legacy And Global Repository Methods

Addresses:

- Finding 8: Generic/global repository methods remain exposed.

Goal: prevent future code from bypassing user scoping and queue state-machine methods.

Tasks:

- Inventory production references to:
  - `internal/queue`;
  - `Repository[T]` generic methods;
  - `JobRepository.FindByStatus`;
  - `JobRepository.CountByStatus`;
  - `JobRepository.UpdateStatus`;
  - `JobRepository.UpdateError`;
  - `JobRepository.ListWithParams`;
  - unscoped `FindByID` for user-owned records.
- Remove dead legacy queue package if no production code requires it.
- If removal is too large, mark it legacy and add import guards forbidding new production imports.
- Split system-only queue methods into a narrow worker/system interface.
- Keep user-facing repository methods explicitly named `ForUser` or `ByUser`.
- Add architecture tests that fail on new unsafe repository usage where possible.
- Update services to use scoped methods.

Acceptance criteria:

- Product services do not use generic unscoped repository methods for user-owned data.
- Legacy queue methods are removed or quarantined.
- New unsafe imports/usages are blocked by tests or documented guards.
- Existing repository tests still cover concrete GORM behavior.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository ./internal/transcription/... ./internal/files ./internal/account ./internal/profile ./internal/automation ./internal/api -run 'TestProduction|TestBackendDependencyDirection'
git diff --check
```

Commit:

```txt
backend: quarantine legacy repository paths
```

## Sprint 8: Final Enforcement, Documentation, And Broad Verification

Goal: close the remediation series with durable guardrails and updated documentation.

Tasks:

- Update the code review remediation tracker with final status, commits, and verification.
- Write one final status note summarizing:
  - completed findings;
  - remaining deferred risks;
  - test commands and blockers;
  - architecture guard coverage.
- Tighten architecture tests from inventory mode to hard enforcement where feasible:
  - no LLM concrete adapters in API;
  - no production API import of concrete LLM clients;
  - no production import of legacy `internal/queue`;
  - admin routes require admin middleware;
  - user-owned event publishers carry user ID.
- Update `architecture-design.md` or `backend-rules.md` only if implementation made a deliberate architecture decision that changes the documented target.
- Run broad backend verification.

Acceptance criteria:

- Every review finding has either a completed remediation or an explicit deferred item with rationale.
- Architecture tests protect the new boundaries.
- Final status note is written.
- Broad backend tests pass or blockers are documented with focused substitute tests.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api ./internal/account ./internal/auth ./internal/chat ./internal/config ./internal/database ./internal/files ./internal/llm ./internal/llmprovider ./internal/recording ./internal/repository ./internal/summarization ./internal/transcription/... ./cmd/server
git diff --check
```

Commit:

```txt
backend: finalize architecture review remediation
```

## Sprint 9: User Status And Auth Enforcement

Addresses:

- Follow-up Finding 11: User status and disabled-user enforcement are absent.

Goal: add durable user lifecycle state and enforce disabled-user behavior consistently across authentication and product entry points.

Tasks:

- Add typed user lifecycle fields:
  - `status` with `active` and `disabled`;
  - `last_login_at`;
  - `password_changed_at`.
- Add schema migration/backfill that preserves existing installs as active users.
- Update account/auth service flows:
  - login rejects disabled users;
  - refresh rejects disabled users;
  - API-key authentication rejects disabled users;
  - successful login updates `last_login_at`;
  - password changes update `password_changed_at` and revoke existing refresh tokens when required.
- Update request auth middleware or account service ports so event streams and enqueue-capable routes cannot proceed for disabled users.
- Keep JWT validation pure; perform active-user checks through account/auth boundaries where persisted state matters.
- Add tests:
  - first registration creates an active admin;
  - disabled user cannot login;
  - disabled user cannot refresh;
  - disabled user cannot authenticate with API key;
  - disabled user cannot open `/api/v1/events`;
  - disabled user cannot enqueue transcription work.

Acceptance criteria:

- Disabled-user restrictions match the multi-user spec.
- Existing users migrate to `status = active`.
- Auth and API-key behavior remains stable for active users.
- No public DTO leaks password or token state.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/account ./internal/auth ./internal/api -run 'TestAuth|TestSecurity|TestAPIKey|TestEvent|TestTranscription'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database ./internal/repository
git diff --check
```

Commit:

```txt
backend: enforce user account status
```

## Sprint 10: Admin User Management Service And Routes

Addresses:

- Follow-up Finding 12: Admin user-management API is not implemented.

Goal: implement explicit admin-only user lifecycle workflows behind an `internal/admin` service.

Tasks:

- Add `internal/admin.Service` for cross-user workflows.
- Add repository methods for admin user management:
  - create user by admin;
  - list users for admin;
  - get user by admin;
  - update role/status/display/email fields;
  - reset password;
  - disable user;
  - enable user;
  - count active admins.
- Add admin routes:
  - `GET /api/v1/admin/users`
  - `POST /api/v1/admin/users`
  - `GET /api/v1/admin/users/:user_id`
  - `PATCH /api/v1/admin/users/:user_id`
  - `POST /api/v1/admin/users/:user_id:reset-password`
  - `POST /api/v1/admin/users/:user_id:disable`
  - `POST /api/v1/admin/users/:user_id:enable`
- Enforce last-active-admin invariant for disable and demotion.
- Revoke target refresh tokens and API keys on disable.
- Revoke target refresh tokens on password reset.
- Keep API keys disallowed for admin operations until explicit scopes exist.
- Add route contract and security tests for anonymous, non-admin, API-key, and admin JWT access.

Acceptance criteria:

- Admin user routes exist and require active admin JWT auth.
- Normal users cannot manage other users.
- Admin cannot disable or demote the last active admin.
- Disable and password-reset side effects are persisted and tested.
- Admin DTOs may include user IDs/status/role; normal user DTOs remain self-scoped.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/admin ./internal/account ./internal/api -run 'TestAdmin|TestSecurity|TestAuth|TestCanonicalRouteRegistration|TestEndpointContractSmoke'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository ./internal/database
git diff --check
```

Commit:

```txt
backend: add admin user management
```

## Sprint 11: Relational User Settings

Addresses:

- Follow-up Finding 13: Settings remain in `users.settings_json` instead of relational settings tables.

Goal: move core per-user settings into a typed `user_settings` table while preserving the current settings API shape.

Tasks:

- Add `models.UserSettings`.
- Add `repository.UserSettingsRepository`.
- Add migration/backfill from `users.settings_json`.
- Keep compatibility reads from legacy JSON only as an explicit migration fallback.
- Move account settings reads/writes to `user_settings`:
  - `default_profile_id`;
  - `auto_transcription_enabled`;
  - `auto_rename_enabled`;
  - `summary_default_model`.
- Enforce that `default_profile_id` belongs to the same user.
- Preserve current `/api/v1/settings` response shape.
- Add tests:
  - settings row is created/backfilled for existing users;
  - partial updates do not overwrite unrelated fields;
  - default profile ownership is enforced;
  - auto-transcription and auto-rename validation still works.

Acceptance criteria:

- Durable core user settings no longer depend on expanding `users.settings_json`.
- Settings API behavior remains backward compatible.
- Per-user settings have relational indexes and foreign-key behavior where SQLite can enforce it.
- Legacy settings JSON is not used by new write paths.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/account ./internal/api -run 'TestSettings|TestProfile'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database ./internal/repository ./internal/automation ./internal/summarization
git diff --check
```

Commit:

```txt
backend: move user settings to relational table
```

## Sprint 12: System Settings And Scheduler Policy Boundary

Addresses:

- Follow-up Finding 10: Scheduler policy boundary is still missing.
- Follow-up Finding 13: Settings remain in `users.settings_json` instead of relational settings tables.

Goal: introduce durable system settings and a scheduler policy package without changing queue behavior yet.

Tasks:

- Add `models.SystemSetting` and `repository.SystemSettingsRepository`.
- Add migration/backfill with initial key `queue.scheduler = {"policy":"priority"}`.
- Add `internal/transcription/scheduler` with:
  - policy constants;
  - config struct;
  - strict validation;
  - default config.
- Add admin service methods for queue scheduler config:
  - get scheduler config;
  - update scheduler config.
- Add admin routes:
  - `GET /api/v1/admin/queue/scheduler`
  - `PUT /api/v1/admin/queue/scheduler`
- Keep API handlers thin; validation belongs in admin/scheduler service code.
- Add tests for strict scheduler config validation and admin route access.

Acceptance criteria:

- Scheduler config is durable and admin-managed.
- Invalid scheduler config is rejected before persistence.
- Priority remains the default behavior after migration.
- No worker claim SQL changes are made until Sprint 13.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/scheduler ./internal/admin ./internal/api -run 'TestAdmin|TestScheduler|TestSecurity'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database ./internal/repository ./internal/app
git diff --check
```

Commit:

```txt
backend: add scheduler system settings
```

## Sprint 13: Configurable Queue Claims And Admin Queue Stats

Addresses:

- Follow-up Finding 9: Admin queue stats are still user-scoped.
- Follow-up Finding 10: Scheduler policy boundary is still missing.

Goal: make shared queue claiming policy configurable while giving admins a true global queue view.

Tasks:

- Change worker service to load scheduler config through a narrow settings/admin port.
- Change repository claim method to accept `scheduler.Config`.
- Implement claim policies:
  - priority default;
  - FIFO;
  - weighted duration;
  - fair share with `MaxConcurrentPerUser`.
- Keep claim atomicity inside repository transactions.
- Add repository indexes recommended by the multi-user spec where missing:
  - FIFO queue path;
  - priority queue path;
  - user/status queue path;
  - duration queue path;
  - user/status/updated list path.
- Add global queue stats repository/service method for admins:
  - aggregate status counts across all users;
  - running counts;
  - per-user breakdown with username.
- Update `/api/v1/admin/queue` to return global aggregate stats and `by_user`.
- Keep normal user stats scoped to the current user.
- Add tests:
  - admin stats include jobs from multiple users;
  - normal stats do not include other users;
  - FIFO ordering is deterministic;
  - priority ordering remains default;
  - weighted-duration scoring is deterministic and ages jobs;
  - fair-share respects per-user concurrency.

Acceptance criteria:

- Queue scheduler policy is configurable by admin and used by workers.
- Admin queue stats are truly global.
- Normal queue stats and events remain user-isolated.
- Claim behavior remains atomic and deterministic.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository -run 'TestJobRepository|TestScheduler|TestQueue'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/worker ./internal/transcription/scheduler ./internal/admin ./internal/api -run 'TestAdmin|TestQueue|TestScheduler|TestSecurity'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database
git diff --check
```

Commit:

```txt
backend: configure shared queue scheduler
```

## Sprint 14: File Metadata And Storage Boundary Cleanup

Addresses:

- Follow-up Finding 14: API response mapping still touches local file paths.

Goal: remove filesystem path access from API response mapping and keep file metadata behind file/storage services.

Tasks:

- Remove `os.Stat` and path-derived filesystem metadata from `internal/api/response_models.go`.
- Add file service metadata methods or DTOs for:
  - size bytes;
  - MIME type;
  - kind;
  - duration;
  - safe display filename/title.
- Prefer persisted file metadata over filesystem probing for list/get responses.
- Keep audio streaming through `files.Service.OpenAudio` or an equivalent storage boundary.
- Ensure public responses never include local paths, object paths, or original unsafe filenames.
- Add/update architecture guard:
  - production API response mapping must not import `os` for file metadata;
  - production API must not construct local file paths.
- Add tests:
  - file list/get responses do not touch local path directly;
  - response still includes expected size/kind/mime metadata;
  - missing physical file does not leak a path in metadata responses;
  - audio streaming still performs ownership check before opening.

Acceptance criteria:

- API DTO mapping has no direct filesystem dependency for files.
- File metadata is owned by file/storage service boundaries.
- Existing response shape remains stable.
- Path redaction tests remain green.

Testing focus:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/files ./internal/api -run 'TestFile|TestResponse|TestProduction|TestBackendDependencyDirection'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/recording ./internal/mediaimport ./internal/transcription/orchestrator
git diff --check
```

Commit:

```txt
backend: keep file metadata behind service boundary
```

## Expected Status Notes

Each sprint should add one note under `devnotes/v2.0.0/status-updates/` using this naming pattern:

```txt
backend-architecture-review-remediation-sprint-00-baseline.md
backend-architecture-review-remediation-sprint-01-admin-auth.md
backend-architecture-review-remediation-sprint-02-user-scoped-events.md
backend-architecture-review-remediation-sprint-03-llm-provider-boundary.md
backend-architecture-review-remediation-sprint-04-llm-secrets.md
backend-architecture-review-remediation-sprint-05-queue-leases.md
backend-architecture-review-remediation-sprint-06-chat-service.md
backend-architecture-review-remediation-sprint-07-repository-legacy.md
backend-architecture-review-remediation-sprint-08-final.md
backend-architecture-review-remediation-sprint-09-user-status.md
backend-architecture-review-remediation-sprint-10-admin-users.md
backend-architecture-review-remediation-sprint-11-user-settings.md
backend-architecture-review-remediation-sprint-12-scheduler-settings.md
backend-architecture-review-remediation-sprint-13-queue-scheduler.md
backend-architecture-review-remediation-sprint-14-file-metadata-boundary.md
```

Each status note should include:

- sprint goal;
- files changed;
- behavior changes;
- tests run;
- blockers or skipped tests;
- follow-up debt.

## Stop Conditions

Stop and re-plan before continuing if any sprint requires:

- a public API breaking change not already covered by route contract tests;
- a database migration that cannot safely support existing deployments;
- a credential migration that could lose existing provider configuration;
- a queue change that can orphan running jobs without recovery;
- broad package renames unrelated to the finding being fixed.
