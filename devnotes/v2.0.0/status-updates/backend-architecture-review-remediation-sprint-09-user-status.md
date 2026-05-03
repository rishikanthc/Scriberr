# Backend Architecture Review Remediation Sprint 09: User Status

Date: 2026-05-03

## Goal

Add durable user lifecycle state and enforce disabled-user behavior across authentication and protected API entry points.

## Changes

- Added user lifecycle fields to `models.User`:
  - `status`
  - `last_login_at`
  - `password_changed_at`
- Defaulted new and migrated users to `status = active`.
- Added account service checks so disabled users cannot:
  - log in;
  - refresh access tokens;
  - authenticate with API keys.
- Added JWT middleware active-user validation so existing bearer tokens stop working after a user is disabled.
- Updated password changes to set `password_changed_at` and revoke active refresh tokens for the user.
- Added repository support for revoking refresh tokens by user.
- Added regression tests for disabled login, refresh, API-key auth, event stream access, and transcription enqueue.
- Added schema assertions for the new user lifecycle columns.

## TDD Notes

Initial focused regression:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run TestDisabledUserCannotAuthenticateOrUseExistingCredentials
```

Expected failure before implementation:

```txt
expected: 401
actual  : 200
```

The test passed after account service and middleware enforcement were added.

## Verification

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestAuthRegisterLoginRefreshMeLogout|TestAuthValidationAndPasswordChanges|TestDisabledUserCannotAuthenticateOrUseExistingCredentials'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database -run TestFreshSchemaInitialization
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/account ./internal/auth ./internal/api -run 'TestAuth|TestSecurity|TestAPIKey|TestEvent|TestTranscription'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/database ./internal/repository
git diff --check -- internal/models/auth.go internal/account/service.go internal/api/middleware.go internal/repository/implementations.go internal/api/auth_test.go internal/database/database_test.go
```

All commands passed.

## Commits

- `ed824fe backend: enforce user account status`

## Follow-Up

- Sprint 10 should move cross-user lifecycle actions into `internal/admin` and add disable/enable/reset-password routes.
- Sprint 10 should revoke API keys on admin disable. Sprint 9 rejects disabled users during API-key authentication, but does not add admin lifecycle routes yet.
