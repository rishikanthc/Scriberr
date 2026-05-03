# Backend Architecture Refactor Sprint 03 Repository Ports

Date: 2026-05-02

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-architecture-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-architecture-refactor-sprint-tracker.md`

## Goal

Reduce broad repository coupling in user-owned workflows and replace generic lookups where the service needs a domain-specific persistence invariant.

## Completed

- Narrowed `automation.Service` file lookup from generic `FindByID(ctx, interface{})` to `FindReadyFileByID(ctx, string)`.
- Narrowed `automation.Service` user lookup from generic `FindByID(ctx, interface{})` to `FindAutomationUserByID(ctx, uint)`.
- Added `repository.JobRepository.FindReadyFileByID`, constrained to uploaded source-file rows:

```txt
id = ?
source_file_hash IS NULL
status = uploaded
```

- Added `repository.UserRepository.FindAutomationUserByID` as an explicit user settings/readiness lookup for automation.
- Narrowed `transcription.Service` dependencies from full concrete repository interfaces to local workflow ports:
  - `transcription.JobStore`
  - `transcription.ProfileStore`
- Added fake-backed automation coverage for non-file events no-oping.
- Added concrete repository coverage that `FindReadyFileByID` excludes transcription rows and processing file rows.

## Notes

This sprint does not remove generic `FindByID` from the base repository. Existing code still uses it where it is not yet a user-owned invariant boundary. The important change is that new automation and transcription service dependencies now describe workflow-specific ports instead of full repository surfaces.

## Verification

Passed:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/account ./internal/profile ./internal/files ./internal/transcription ./internal/automation ./internal/repository ./cmd/server
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'
git diff --check
```

## Commit

```txt
backend: narrow repository service ports
```
