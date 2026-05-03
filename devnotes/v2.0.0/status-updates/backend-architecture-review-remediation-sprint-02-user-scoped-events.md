# Backend Architecture Review Remediation Sprint 02 User Scoped Events

Date: 2026-05-03

## Scope

Sprint 2 addresses Finding 1: global SSE delivery was not scoped by user, so authenticated users could receive another user's events.

## Changes

- Added user audience metadata to API events and SSE subscribers.
- Global and transcription-specific SSE subscribers now bind to the authenticated principal.
- Event broker delivery now requires matching `UserID` when an event has a user audience.
- Added hidden `user_id` metadata to map-style file/transcription event payloads and stripped it from public SSE payloads.
- Threaded domain event `UserID` through:
  - transcription progress/status events;
  - file upload, delete, media import, and recording finalizer file events;
  - recording events;
  - summary status and summary-driven file updates;
  - annotation and tag events;
  - settings, profile, LLM provider, and summary widget events;
  - automation-created transcription events.

## TDD Evidence

The new two-user file isolation test failed before implementation:

```txt
TestGlobalSSEFiltersFileEventsByUser: first user's /api/v1/events stream contained second user's file.ready event.
```

After implementation, the focused isolation tests passed.

## Verification

Passed:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestGlobalSSEFiltersFileEventsByUser|TestTranscriptionSSEFiltersProgressByUser'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestEvent|TestSSE|TestSecurity|TestProduction'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/files ./internal/recording ./internal/summarization ./internal/tags ./internal/annotations ./internal/transcription/worker
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/mediaimport ./internal/automation
git diff --check
```

## Notes

Admin users do not receive all user events by default. Admin visibility should be modeled as an explicit event audience if a future admin event stream needs cross-user observability.
