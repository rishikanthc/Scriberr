# Backend Architecture Refactor Sprint 00 Baseline

Date: 2026-05-02

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-architecture-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-architecture-refactor-sprint-tracker.md`

## Goal

Establish the baseline for the next backend architecture refactor series before behavior changes begin.

Sprint 0 adds guardrails and documentation only. It does not change runtime behavior.

## Worktree Baseline

Unrelated local/untracked workspace entries present before Sprint 0 implementation:

```txt
.playwright-mcp/
.tmp/
DM_Sans,Nunito.zip
DM_Sans,Nunito/
references/
test-audio/
```

Sprint-owned new documents:

```txt
devnotes/v2.0.0/architecture-design.md
devnotes/v2.0.0/backend-rules.md
devnotes/v2.0.0/sprint-plans/backend-architecture-refactor-sprint-plan.md
devnotes/v2.0.0/sprint-trackers/backend-architecture-refactor-sprint-tracker.md
devnotes/v2.0.0/status-updates/backend-architecture-refactor-sprint-00-baseline.md
```

Sprint-owned code guard:

```txt
internal/api/architecture_test.go
```

## Guards Added

Extended `internal/api/architecture_test.go` with backend dependency-direction checks:

- `internal/models` production code must not import `scriberr/internal/*`.
- `internal/repository` production code may import `internal/models`, but not other internal app/service/provider packages.
- `internal/transcription/engineprovider` production code must not import `internal/api` or `internal/repository`.
- `internal/transcription/worker` production code must not import `internal/api`.

The existing production API database guard remains strict:

```txt
internal/api production code must not import scriberr/internal/database
```

## Test Baseline

The default Go build cache under `~/Library/Caches/go-build` is not writable in this sandbox. Use the repo-local cache for backend verification:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'
```

Result: passed.

Also required for the sprint:

```sh
git diff --check
```

Result: passed.

## Commit Hygiene

Use one focused commit per sprint:

```txt
backend: <sprint scope>
```

For this sprint:

```txt
backend: establish architecture refactor baseline
```

Do not include unrelated local directories, test audio, reference modules, generated browser state, or frontend asset churn in backend architecture commits.

## Next Sprint

Sprint 1 should start with failing DTO/mapper tests for public responses before changing handlers.
