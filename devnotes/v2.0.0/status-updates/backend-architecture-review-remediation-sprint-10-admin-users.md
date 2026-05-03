# Backend Architecture Review Remediation Sprint 10: Admin Users

Date: 2026-05-03

## Goal

Implement explicit admin-only user lifecycle workflows behind an `internal/admin` service.

## Changes

- Added `internal/admin.Service` for cross-user user management.
- Added admin-scoped user repository methods:
  - list users;
  - get user by ID;
  - count active admins.
- Added credential revocation helpers:
  - refresh tokens by user;
  - API keys by user.
- Added admin user routes:
  - `GET /api/v1/admin/users`
  - `POST /api/v1/admin/users`
  - `GET /api/v1/admin/users/{user_id}`
  - `PATCH /api/v1/admin/users/{user_id}`
  - `POST /api/v1/admin/users/{user_id}:reset-password`
  - `POST /api/v1/admin/users/{user_id}:disable`
  - `POST /api/v1/admin/users/{user_id}:enable`
- Routed command-style admin endpoints through the existing `NoRoute` command parser because Gin cannot register `:command` suffixes on wildcard path segments.
- Enforced active admin JWT authorization in the admin service in addition to route middleware.
- Enforced last-active-admin protection for disable and demotion.
- Disabled users have refresh tokens and API keys revoked.
- Password resets revoke refresh tokens and update `password_changed_at`.
- Added API regression tests for:
  - full create/list/get/disable/enable/reset-password lifecycle;
  - anonymous, non-admin JWT, API-key, and admin JWT access policy;
  - last-active-admin disable/demotion conflicts.

## Verification

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/admin ./internal/api -run 'TestAdmin|TestCanonicalRouteRegistration|TestEndpointContractSmoke'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository ./internal/database
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/admin ./internal/account ./internal/api -run 'TestAdmin|TestSecurity|TestAuth|TestCanonicalRouteRegistration|TestEndpointContractSmoke'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/repository ./internal/database ./internal/app
git diff --check -- internal/admin/service.go internal/repository/implementations.go internal/api/router.go internal/app/app.go internal/api/types.go internal/api/admin_handlers.go internal/api/auth_test.go internal/api/route_contract_test.go internal/api/admin_user_handlers_test.go internal/api/middleware.go
```

All commands passed.

## Commits

- `2e5490a backend: add admin user management`

## Follow-Up

- Sprint 11 should move core user settings from `users.settings_json` into relational `user_settings`.
- Admin user list currently returns the first 100 users without cursor pagination. Add cursor pagination if the admin API needs large-instance ergonomics before launch.
