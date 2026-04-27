# Sprint 2 Authentication and API Keys Notes

## Completed

- Added Sprint 2 tests before implementation in `internal/api/auth_test.go`.
- Implemented registration status, initial registration, login, refresh-token rotation, logout, current-user lookup, password change, and username change.
- Implemented JWT-only account-management routes.
- Implemented API key list/create/delete.
- Implemented one-time raw API key return on creation.
- Implemented API key list redaction so raw keys and hashes are not returned.
- Implemented hashed refresh-token storage and hashed API-key storage.
- Implemented revoked API-key rejection through the API auth guard.
- Kept resource endpoints as authenticated `501 Not Implemented` placeholders for later sprints.

## Security Coverage Added

- Registration disables after the first user exists.
- Invalid registration payloads return structured validation errors.
- Invalid login and wrong current password are rejected.
- Refresh tokens are rotated and the old token cannot be reused.
- Logout revokes the supplied refresh token.
- API keys can authenticate protected resource routes.
- API-key-management endpoints require JWT and reject API-key auth.
- API key list responses do not include raw key material or key hashes.
- Deleted API keys can no longer authenticate.

## Known Follow-Ups

- The current public API-key ID is `key_{database_id}` for Sprint 2 compatibility with the existing numeric database model. Later work can replace this with fully opaque generated IDs if the model changes.
- `go.sum` still needs to be updated for Zap after running `go mod tidy` or `go test` in an environment with Go installed.
- The API foundation currently uses direct database access inside handlers. Sprint 3+ should start extracting file/transcription/profile/settings service interfaces as those modules become real.

## Verification

- `git diff --check` passes.
- `go test ./internal/api` could not run because `go` is not installed in this environment.
- `gofmt` could not run because `gofmt` is not installed in this environment.
