# Backend Architecture Review Remediation: Sprint 12 Scheduler Settings

Status: complete

Commit: `aa8fd23 backend: add scheduler system settings`

## Scope

Sprint 12 introduced durable global system settings and the scheduler policy validation boundary. It intentionally did not change queue claim ordering; Sprint 13 owns applying the configured policy to worker claims.

## Changes

- Added `models.SystemSetting` and `repository.SystemSettingsRepository`.
- Added schema version 11 with `system_settings` and default `queue.scheduler = {"policy":"priority"}` backfill for fresh, upgraded, and legacy migrations.
- Added `internal/transcription/scheduler` with policy constants, default config, strict JSON parsing, and validation.
- Added admin service methods for reading/updating scheduler config.
- Added `GET /api/v1/admin/queue/scheduler` and `PUT /api/v1/admin/queue/scheduler`.
- Added tests proving invalid scheduler config is rejected before persistence and admin-only route access is enforced.

## Verification

- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/transcription/scheduler ./internal/admin ./internal/api -run 'TestAdmin|TestScheduler|TestSecurity'`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database ./internal/repository ./internal/app`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestCanonicalRouteRegistration|TestEndpointContractSmoke'`
- `git diff --check`

## Follow-Up

Sprint 13 should load this scheduler config through the worker queue path, update claim ordering for priority, FIFO, weighted-duration, and fair-share policies, and add global admin queue stats with per-user breakdowns.
