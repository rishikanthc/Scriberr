# Backend Architecture Review Remediation Sprint 07 Repository Legacy

Date: 2026-05-03

## Scope

Sprint 7 addresses:

- Finding 8: generic/global repository methods remained exposed to product-service paths.

## Changes

- Removed the dead legacy `internal/queue` package.
- Added architecture guards that fail when production code imports `internal/queue`.
- Added architecture guards for product-service usage of broad unscoped job lookups.
- Updated summarization job reads to use user-scoped transcription and source-file lookups.
- Narrowed the transcription worker dependency to a queue-specific repository interface instead of the full job repository interface.

## TDD Evidence

The new product-service guard failed before implementation because summarization used unscoped job lookups:

```txt
production product service uses unscoped job lookup "s.jobs.FindByID(" in:
../../internal/summarization/service.go
```

After implementation, the guard passed and summarization still passed its package tests.

## Verification

Passed:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestProductionDoesNotImportLegacyQueue|TestProductServicesDoNotUseUnscopedJobFindByID|TestBackendDependencyDirection'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/summarization ./internal/transcription/worker ./internal/repository
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository ./internal/transcription/... ./internal/files ./internal/account ./internal/profile ./internal/automation ./internal/api -run 'TestProduction|TestBackendDependencyDirection'
git diff --check
```

## Notes

A broader `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/... ./cmd/server` run still fails in `internal/api` at `TestSettingsPartialUpdateAndValidation`, where the test expects `false` and receives `true`. That failure reproduces with the single test and is outside this sprint's repository/legacy-queue scope.
