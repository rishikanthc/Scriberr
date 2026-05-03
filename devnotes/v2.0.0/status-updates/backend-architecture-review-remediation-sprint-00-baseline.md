# Backend Architecture Review Remediation Sprint 00 Baseline

Date: 2026-05-03

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-architecture-review-remediation-sprint-plan.md`
- `devnotes/v2.0.0/sprint-trackers/backend-architecture-review-remediation-sprint-tracker.md`
- `devnotes/v2.0.0/status-updates/backend-architecture-code-review-2026-05-03.md`

## Scope

Sprint 0 establishes the remediation baseline for the backend architecture review findings. It does not change runtime behavior.

## Worktree Baseline

Current branch at sprint start:

- `frontend-integration`, ahead of `origin/frontend-integration` by 2 commits.

Known untracked files at sprint start:

- `devnotes/v2.0.0/status-updates/backend-architecture-code-review-2026-05-03.md`
- `devnotes/v2.0.0/sprint-plans/backend-architecture-review-remediation-sprint-plan.md`
- `devnotes/v2.0.0/sprint-trackers/backend-architecture-review-remediation-sprint-tracker.md`
- `references/`
- `test-audio/`

The remediation docs are part of this sprint baseline. `references/` and `test-audio/` are treated as unrelated local artifacts and are left unstaged.

## Test And Guard Inventory

Existing coverage anchors:

- Events and SSE:
  - `internal/api/events_test.go`
  - `internal/api/recording_handlers_test.go`
  - `internal/sse/broadcaster_test.go`
  - service publisher tests in `internal/recording`, `internal/summarization`, `internal/annotations`, `internal/tags`, and `internal/automation`
- Auth, API keys, and admin route access:
  - `internal/api/auth_test.go`
  - `internal/api/security_regression_test.go`
  - `internal/api/route_contract_test.go`
  - `internal/api/profile_settings_test.go`
- LLM provider settings:
  - `internal/api/llm_provider_handlers_test.go`
  - `internal/config/config_test.go`
- Queue claim, recovery, and terminal transitions:
  - `internal/repository/job_queue_test.go`
  - `internal/transcription/worker/service_test.go`
  - legacy `internal/queue/queue_test.go`
- Chat streaming and context:
  - `internal/api/chat_handlers_test.go`
  - `internal/chat/context_builder_test.go`
  - `internal/chat/compactor_test.go`
- Architecture guards:
  - `internal/api/architecture_test.go`
  - `cmd/server/main_test.go`

Current architecture guard anchors:

- `internal/api` production code has an empty direct database import allowlist.
- Only `internal/app` may import `internal/database` outside the database package.
- Only `internal/app` may import `internal/api`.
- Models, repositories, engine providers, and workers have dependency direction guards.
- `cmd/server/main.go` is guarded against backend composition imports and legacy `internal/queue` startup references.

Deferred guard work:

- Sprint 1 will add explicit admin middleware/role tests for `/api/v1/admin/queue`.
- Sprint 2 will add two-user SSE isolation tests and event audience guards.
- Sprint 3 will move LLM provider HTTP probing out of `internal/api` and add a boundary guard for that concrete adapter.
- Sprint 5 will add lease-owned queue recovery and terminal transition tests.
- Sprint 7 will add or tighten repository exposure guards once the intended system-only interfaces are in place.

## Verification

Passed:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestProduction|TestBackendDependencyDirection|TestCanonicalRouteRegistration|TestEndpointContractSmoke'
```

Pending for sprint close:

```sh
git diff --check
```

## Notes

No network, ASR model, or loopback blocker was encountered in the Sprint 0 focused baseline command.
