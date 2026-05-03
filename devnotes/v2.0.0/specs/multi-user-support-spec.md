# Multi-User Support Spec

Date: 2026-05-03

## Purpose

Add secure multi-user support to Scriberr while preserving the modular monolith architecture and the shared durable transcription queue.

The feature must make user data separation non-optional. Users share worker capacity, model cache, and queue infrastructure, but they do not share files, transcriptions, transcripts, summaries, chat sessions, annotations, tags, recordings, API keys, settings, or events.

## Current State From Code Review

- The backend is now composed through `internal/app`; production API code no longer imports `internal/database`.
- Most user-owned tables already have `user_id` and many repositories expose `ByUser`/`ForUser` methods.
- First-user registration exists and closes once any user exists. `models.User.Role` defaults to `admin`, but role authorization is not yet enforced on `/api/v1/admin/*`.
- `/api/v1/admin/queue` currently returns queue stats for the current user, not a real admin global view.
- The durable worker queue claims jobs globally from SQLite using `priority DESC, queued_at ASC`; scheduler policy is hard-coded in `ClaimNextTranscription`.
- Several legacy or system methods still bypass user scoping, including generic `FindByID`, `FindByStatus`, `ListWithParams`, `UpdateStatus`, and some background-service lookups. Some are acceptable system paths, but they must be quarantined before public multi-user launch.
- SQLite foreign keys are enabled in `internal/database/open.go`; schema work should lean on real relational constraints and indexes rather than application-only conventions.

## Goals

- First registered account becomes admin.
- Admins can create, list, disable, enable, and manage users.
- Normal users can manage only their own data and settings.
- All users share one transcription queue and worker pool.
- Admins can choose queue scheduler policy: FIFO, priority, weighted by priority and audio duration, fair share, or future intelligent schedulers.
- Queue scheduling is secure, deterministic, testable, and isolated from HTTP handlers and ASR providers.
- User-owned data remains cleanly separated at API, service, repository, database, event, and storage boundaries.
- Database design uses relational modeling, constraints, and indexes for core product state.

## Non-Goals

- Organizations, teams, project sharing, or ACL-based collaboration.
- Per-user billing.
- Multiple physical queues.
- Cross-user transcript sharing.
- Replacing SQLite in this feature. The design should remain portable to Postgres later.

## Roles And Principal Model

Use a simple role model now:

```txt
admin
user
```

Recommended principal struct at the API boundary:

```go
type Principal struct {
    UserID   uint
    Username string
    Role     string
    AuthType string
    APIKeyID *uint
}
```

JWT claims should include only stable identity and authorization fields needed to avoid a DB read on every ordinary request:

```txt
sub/user_id
username
role
iat
exp
```

For high-risk admin operations, the admin service should reload the user and verify `role = admin` and `status = active` before mutation. API keys should authenticate a user but should not automatically authorize admin operations unless an explicit future API-key scope model is added.

## Authentication And User Lifecycle

### Registration

- `GET /api/v1/auth/registration-status` remains public.
- `POST /api/v1/auth/register` is allowed only while user count is zero.
- The first registered user is created with `role = admin` and `status = active`.
- After the first user exists, public registration returns `409 Conflict`.

### Admin User Management

Admin routes require JWT auth from an active admin user:

```http
GET    /api/v1/admin/users
POST   /api/v1/admin/users
GET    /api/v1/admin/users/{user_id}
PATCH  /api/v1/admin/users/{user_id}
POST   /api/v1/admin/users/{user_id}:reset-password
POST   /api/v1/admin/users/{user_id}:disable
POST   /api/v1/admin/users/{user_id}:enable
```

Initial scope:

- Admin creates users with username, optional email/display name, role, and temporary password.
- Admin can disable users. Disabled users cannot login, refresh tokens, use API keys, connect to events, or enqueue work.
- Disabling a user revokes refresh tokens and API keys for that user.
- Admin cannot disable or demote the last active admin.
- Password reset invalidates existing refresh tokens for the target user.

## Per-User Settings

Move durable core settings toward relational columns rather than expanding `users.settings_json`.

Target table:

```txt
user_settings
  user_id PK/FK users(id) ON DELETE CASCADE
  default_profile_id FK transcription_profiles(id) ON DELETE SET NULL
  auto_transcription_enabled BOOLEAN NOT NULL DEFAULT true
  auto_rename_enabled BOOLEAN NOT NULL DEFAULT true
  summary_default_model TEXT NULL
  created_at
  updated_at
```

Rules:

- User settings are read and updated only for the authenticated user unless an admin endpoint explicitly manages another user.
- `default_profile_id` must point to a profile owned by the same user. SQLite cannot express that with a simple FK against the current single-column profile primary key, so the repository/service must enforce it. A later schema can add composite ownership keys if needed.
- Auto-transcription requires a valid default profile for that user.
- Auto-rename requires that user's active LLM provider config to have a small model.

## Database Design

### Relational Principles

- Core authorization, ownership, queue, and settings fields must be typed columns with constraints and indexes.
- JSON columns are allowed for provider-specific parameters, prompt templates, metadata, and forward-compatible option bags only.
- New tables need explicit primary keys, foreign keys, timestamps, and lifecycle fields.
- Every user-owned table needs `user_id NOT NULL` and an index that supports the most common user-facing query.
- Per-user names and defaults use composite or partial unique indexes.
- Do not add public API behavior that depends on unindexed scans.

### Users

Extend `users`:

```txt
users
  id PK
  username UNIQUE NOT NULL
  email UNIQUE NULL
  display_name NULL
  password_hash NOT NULL
  role TEXT NOT NULL CHECK role IN ('admin', 'user')
  status TEXT NOT NULL DEFAULT 'active' CHECK status IN ('active', 'disabled')
  last_login_at NULL
  password_changed_at NULL
  created_at
  updated_at
  deleted_at
```

Indexes:

```txt
UNIQUE username
UNIQUE email WHERE email IS NOT NULL
INDEX role, status
```

### System Settings

Use one row per setting namespace instead of environment reads at runtime:

```txt
system_settings
  key TEXT PRIMARY KEY
  value_json TEXT NOT NULL
  updated_by_user_id FK users(id) ON DELETE SET NULL
  created_at
  updated_at
```

The initial key is `queue.scheduler`.

Storing scheduler config as JSON is acceptable because the row is a small global configuration document, not a high-cardinality queried dataset. Validate the JSON strictly in the admin service before saving.

### Queue Scheduler Metadata

Extend `transcriptions` only where fields are queryable:

```txt
priority INTEGER NOT NULL DEFAULT 0
queued_at DATETIME NULL
claim_expires_at DATETIME NULL
estimated_duration_ms INTEGER NULL
```

`source_duration_ms` exists today and should be populated for uploads/imports/recordings. Use it as the first duration estimate. Add a separate estimate column only if scheduler logic needs a stable normalized value independent of source metadata.

Recommended indexes:

```txt
idx_transcriptions_queue_fifo(status, queued_at, id)
idx_transcriptions_queue_priority(status, priority DESC, queued_at, id)
idx_transcriptions_queue_user(status, user_id, queued_at, id)
idx_transcriptions_queue_duration(status, source_duration_ms, queued_at, id)
idx_transcriptions_user_status_updated(user_id, status, updated_at DESC)
```

### Foreign-Key And Delete Policy

- `refresh_tokens.user_id`, `api_keys.user_id`, `user_settings.user_id`: `ON DELETE CASCADE`.
- User content rows should normally be soft-deleted with the user disabled first. Hard user delete can be deferred.
- Child content rows should cascade from their parent content row when the parent is deleted, as annotations, tags, chat context, and recording chunks already do in many models.
- Cross-table ownership must be enforced in repositories when SQLite cannot express composite ownership with the current primary keys.

## Scheduler Design

Add `internal/transcription/scheduler` as a policy boundary.

```go
type Policy string

const (
    PolicyFIFO             Policy = "fifo"
    PolicyPriority         Policy = "priority"
    PolicyWeightedDuration Policy = "weighted_duration"
    PolicyFairShare        Policy = "fair_share"
)

type Config struct {
    Policy                  Policy
    MaxConcurrentPerUser     int
    PriorityWeight           float64
    DurationWeight           float64
    AgingWeight              float64
    StarvationAfter          time.Duration
}
```

The worker service should ask the repository to claim according to a validated scheduler config:

```go
ClaimNextTranscription(ctx, workerID, leaseUntil, scheduler.Config)
```

This keeps claim atomicity in the repository while removing policy choice from hard-coded SQL.

### FIFO

Order:

```txt
queued_at ASC, id ASC
```

Use when predictability is more important than priority.

### Priority

Order:

```txt
priority DESC, queued_at ASC, id ASC
```

This matches current behavior and should remain the default migration-safe policy.

### Weighted Duration

Purpose: prefer high-priority jobs while accounting for audio duration so very long jobs do not always block short interactive jobs.

Suggested score:

```txt
score = priority_weight * priority
      + aging_weight * minutes_waiting
      - duration_weight * log1p(duration_minutes)
```

Notes:

- Use `source_duration_ms` where available; fall back to a conservative default when unknown.
- Always include aging to prevent starvation.
- Keep the SQL implementation deterministic. If SQLite expression complexity gets too high, select a bounded candidate window ordered by index-friendly fields, score in Go inside the repository transaction, then atomically update the chosen row.

### Fair Share

Purpose: prevent one user from monopolizing the shared queue.

Rules:

- Respect `MaxConcurrentPerUser` when greater than zero.
- Prefer users with no running jobs over users with active running jobs.
- Within each selected user, apply priority or FIFO order.
- Use deterministic tie-breakers.

### Admin Scheduler API

```http
GET /api/v1/admin/queue/scheduler
PUT /api/v1/admin/queue/scheduler
GET /api/v1/admin/queue
```

`GET /admin/queue` should return global aggregate stats for admins:

```json
{
  "scheduler": "priority",
  "queued": 2,
  "processing": 1,
  "completed": 10,
  "failed": 1,
  "canceled": 0,
  "running": 1,
  "by_user": [
    {
      "user_id": 1,
      "username": "admin",
      "queued": 1,
      "processing": 1,
      "running": 1
    }
  ]
}
```

Normal user queue stats remain scoped to the current user.

## API Isolation Rules

- All public content endpoints continue to use current-user scoping.
- Public IDs remain opaque and do not encode user IDs.
- 404 is preferred over 403 for normal users attempting to access another user's resource by ID.
- Admin APIs may return 403 for insufficient role and 404 for missing target users/resources.
- DTOs must not include `user_id` in normal user responses unless a route is explicitly admin-only.
- Error messages must not reveal whether another user's resource exists.

## Event Isolation

The event broker may remain process-local, but subscriptions must filter:

- `/api/v1/events`: only events for the authenticated user.
- `/api/v1/transcriptions/{id}/events`: only if the transcription belongs to the authenticated user.
- Future `/api/v1/admin/events`: admin-only, explicitly global or filterable.

Every emitted event should carry internal `UserID` for filtering, but public payloads should omit `user_id` except admin routes.

## Storage Isolation

Local storage currently uses shared directories. Multi-user support should not rely on path secrecy alone.

Rules:

- Database ownership check happens before opening any audio, transcript, recording chunk, or generated artifact.
- Stored filenames remain opaque IDs, not original filenames.
- Future storage keys should be partitioned by user for operations:

```txt
users/{user_id}/uploads/{file_id}/source.ext
users/{user_id}/transcripts/{transcription_id}/transcript.json
users/{user_id}/recordings/{recording_id}/chunk-{index}.webm
```

The path partition is not the authorization model; it is an operational guardrail.

## Services And Repository Changes

Add:

```txt
internal/admin.Service
internal/transcription/scheduler
repository.SystemSettingsRepository
repository.UserSettingsRepository
```

Tighten:

- `account.Service`: registration, login, refresh, logout, current user, API keys, own settings.
- `admin.Service`: user CRUD, role/status changes, token/API-key revocation, global scheduler settings.
- `worker.Service`: load scheduler config, pass it into atomic claim, keep running counts by user.
- `repository.JobRepository`: separate public user-scoped methods from worker/system methods.

Quarantine or remove before launch:

```txt
ListWithParams
FindWithAssociations
FindByStatus
CountByStatus
UpdateStatus
UpdateError
DeleteExecutionsByJobID
generic FindByID on user-owned records in public service paths
```

If a method remains for background work, name it as system/worker scope and keep it out of API-facing services.

## Migration Plan

1. Add status fields and constraints-compatible validation to `users`.
2. Create `user_settings` and backfill from `users.settings_json`.
3. Create `system_settings` with `queue.scheduler = {"policy":"priority"}`.
4. Add or adjust queue/user indexes.
5. Backfill missing source durations where cheap; otherwise allow null duration estimates.
6. Preserve first-user admin semantics for existing single-user installs.
7. Remove new-code reliance on `primaryUserID` defaults.

## Implementation Slices

1. Principal and role enforcement:
   - Add principal helper.
   - Add admin role middleware.
   - Include role/status in auth flows.
   - Add tests for disabled users and non-admin admin-route access.

2. Admin user management:
   - Add admin service and user repository methods.
   - Add admin user endpoints.
   - Enforce last-active-admin invariant.

3. User settings table:
   - Add migration/backfill.
   - Move account settings reads/writes to typed relational columns.
   - Preserve response shape.

4. Scheduler config:
   - Add system settings repository.
   - Add admin scheduler endpoints.
   - Validate scheduler config strictly.

5. Scheduler claim policy:
   - Introduce scheduler package.
   - Extend queue claim repository method.
   - Add FIFO, priority, weighted-duration, and fair-share tests.

6. Isolation hardening:
   - Audit all user-owned repository calls.
   - Add cross-user API tests for files, transcriptions, summaries, chat, tags, annotations, recordings, API keys, events, logs, executions, and queue stats.
   - Add response tests that normal user DTOs do not leak `user_id`, local paths, or credentials.

## Test Requirements

- First registration creates an active admin.
- Public registration closes after first user.
- Admin can create a normal user.
- Normal user cannot access admin routes.
- Disabled user cannot login, refresh, use API keys, enqueue jobs, or open event streams.
- Users cannot list, read, stream, update, delete, cancel, retry, summarize, chat with, tag, annotate, or inspect logs/executions for another user's transcription.
- Shared queue claims jobs across users according to selected policy.
- Fair-share scheduler prevents one user from exceeding configured per-user concurrency.
- Weighted-duration scheduler prefers high-score jobs and ages waiting jobs.
- Admin global queue stats include all users; normal stats include only current user.
- Database migration creates required constraints and indexes.

## Acceptance Criteria

- No production handler imports `internal/database` or constructs repositories.
- All normal user endpoints are scoped by authenticated principal.
- All admin endpoints enforce active admin role.
- All user-owned new schema has `user_id`, constraints, and indexes.
- Queue scheduler policy is configurable by admin and test-covered.
- Shared queue execution cannot leak data through stats, logs, events, files, transcripts, or error messages.
- Existing single-user installs migrate with their current user as admin and keep working.
