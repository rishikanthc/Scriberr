# Backend Architecture Refactor Sprint 01 API DTO Boundary

Date: 2026-05-02

Related plan:

- `devnotes/v2.0.0/sprint-plans/backend-architecture-refactor-sprint-plan.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/backend-architecture-refactor-sprint-tracker.md`

## Goal

Start separating public API response contracts from persistence records.

This sprint keeps JSON shapes stable while moving high-risk response mappers to explicit DTO structs.

## Completed

- Added mapper tests for file, transcription, profile, recording, and summary responses.
- Added explicit DTO structs for:
  - user
  - recording session and chunk
  - file
  - transcription detail and list item
  - profile
  - settings
  - summary and summary widget run
- Updated list handlers to build typed DTO slices instead of `[]gin.H`.
- Updated event payload construction where handlers previously indexed response maps.
- Preserved public IDs and existing field names through JSON tags.
- Added path-leak regression checks for representative DTOs.

## Notes

The response mappers still accept persistence models as inputs. That is acceptable for this sprint; the boundary now has explicit output DTOs and tests. Later sprints can move service return types away from persistence records where useful.

## Verification

Passed:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api -run 'Test.*ResponseDTO|TestRepresentativeResponseShapes|TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestProductionAPIDatabaseAccessInventory|TestBackendDependencyDirection'
git diff --check
```

Blocked by sandbox loopback restrictions:

```sh
GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/api
```

The full API package reaches `TestLLMProviderSettingsSaveTestsConnectionAndMasksKey`, which starts `httptest.NewServer` and fails in this sandbox with:

```txt
listen tcp6 [::1]:0: bind: operation not permitted
```

This is the same known loopback limitation recorded in earlier backend notes.

## Commit

```txt
backend: harden api dto boundary
```
