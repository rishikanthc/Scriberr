# Backend Architecture Review Remediation: Sprint 13 Queue Scheduler

Status: complete

Commit: `f99f1e1 backend: configure shared queue scheduler`

## Scope

Sprint 13 applied the scheduler settings introduced in Sprint 12 to shared queue claims and changed admin queue stats from admin-user scoped counts to global queue visibility.

## Changes

- Changed repository queue claiming to accept `scheduler.Config` while preserving repository-owned atomic claim transitions.
- Implemented deterministic priority, FIFO, weighted-duration-with-aging, and fair-share policies.
- Added fair-share `max_concurrent_per_user` validation and API mapping.
- Wired worker claims through a narrow scheduler config provider backed by system settings.
- Added global queue status aggregation with per-user username breakdown.
- Updated `GET /api/v1/admin/queue` to return global totals and `by_user`.
- Kept normal user queue stats scoped to the requested user.
- Added queue indexes for FIFO, priority, user/status, duration, and user/status/updated paths.

## Verification

- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository -run 'TestJobRepository|TestScheduler|TestQueue'`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/worker ./internal/transcription/scheduler ./internal/admin ./internal/api -run 'TestAdmin|TestQueue|TestScheduler|TestSecurity'`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestAdminQueueStats|TestUserQueueStats|TestEndpointContractSmoke|TestCanonicalRouteRegistration'`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription ./internal/app`
- `git diff --check`

## Note

A broader `go test ./internal/api` run required unsandboxed localhost binding for `httptest.NewServer`; after escalation, it failed in `TestRecordingChunkRequestCancellationDoesNotPersist` with a deterministic 401 on a pre-canceled request. That failure is outside this sprint's queue/scheduler surface and is not part of the Sprint 13 verification target.

## Follow-Up

Sprint 14 remains: remove direct filesystem metadata probing from API response mapping and route file metadata through the file/storage boundary.
