# Backend Architecture Review Remediation: Sprint 11 User Settings

Status: complete

Commit: `70f03d4 backend: move user settings to relational table`

## Scope

Sprint 11 addressed the user-scoped half of the settings-table finding. Core account settings now have a typed `user_settings` row instead of relying on `users.settings_json` as the write target.

## Changes

- Added `models.UserSettings` and schema version 10 migration coverage.
- Backfilled existing `users.settings_json` values into `user_settings`.
- Added `repository.UserSettingsRepository` and hydrated user settings through user repository reads.
- Moved `/api/v1/settings` updates to relational settings rows while preserving the public response shape.
- Kept legacy JSON as migration fallback only; new profile/default settings writes no longer store the profile ID in `users.settings_json`.

## Verification

- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/account ./internal/api -run 'TestSettings|TestProfile'`
- `GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database ./internal/repository ./internal/automation ./internal/summarization`
- `git diff --check`

## Follow-Up

Sprint 12 still needs the system/global side of the same finding: typed `system_settings` for scheduler policy and admin-managed scheduler configuration.
