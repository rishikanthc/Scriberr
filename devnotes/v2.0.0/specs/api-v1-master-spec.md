````md
# Scriberr API v1 Master Spec

## Design Goal

Scriberr’s API should be simple, predictable, and pleasant to build against.

For now, Scriberr is a **single-user local audio intelligence app**. The database has foundations for multi-user, but the API should not expose multi-user concepts yet.

Core lifecycle:

```txt
Upload/import file
→ Create transcription
→ Poll status or subscribe to events
→ Read transcript
→ Stream audio with Range support
````

Out of scope for this API version:

```txt
multi-track audio
multi-user administration
teams / orgs / permissions
complex worker orchestration APIs
```

---

# API Principles

## 1. Resource-first design

Use nouns for resources:

```http
/files
/transcriptions
/api-keys
/settings
```

Use commands only when an operation is not clean CRUD:

```http
POST /transcriptions/{id}:cancel
POST /transcriptions/{id}:retry
POST /files:import-youtube
```

## 2. Thin HTTP layer

Handlers should only handle:

```txt
auth
validation
request parsing
response formatting
calling services
```

They should not contain transcription/runtime logic.

## 3. Stable response shapes

Collections:

```json
{
  "items": [],
  "next_cursor": null
}
```

Errors:

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "language is invalid",
    "field": "options.language",
    "request_id": "req_123"
  }
}
```

## 4. No filesystem paths in public API

Never expose local absolute paths such as:

```txt
/Users/...
/tmp/...
/app/data/...
```

Expose stable IDs and artifact URLs instead.

## 5. Privacy-first

No silent cloud fallback. Any cloud/provider behavior must be explicit in settings or request options.

## 6. Range streaming is required

Audio streaming must support:

```http
Range: bytes=start-end
206 Partial Content
416 Range Not Satisfiable
Accept-Ranges: bytes
Content-Range
```

---

# Base URL

```http
/api/v1
```

Health endpoints may also exist outside versioning:

```http
GET /health
```

---

# Authentication

Single-user mode still supports auth because Scriberr may run on a LAN or server.

Supported auth methods:

```http
Authorization: Bearer <jwt>
X-API-Key: <api_key>
```

JWT is intended for the first-party UI.

API keys are intended for scripts, CLI tools, and integrations.

Raw API keys are only returned once during creation.

Refresh tokens and API keys must be stored hashed.

---

# Common Headers

## Request ID

Clients may provide:

```http
X-Request-ID: req_custom
```

Server must return:

```http
X-Request-ID: req_custom
```

If missing, server generates one.

## Idempotency

Create-like operations may support:

```http
Idempotency-Key: unique-client-key
```

Recommended for:

```txt
file uploads
transcription creation
imports
```

---

# Status Codes

Use consistently:

```txt
200 OK                  successful read/update
201 Created             synchronous resource creation
202 Accepted            async job accepted
204 No Content          successful delete
400 Bad Request         malformed input
401 Unauthorized        missing/invalid auth
403 Forbidden           auth valid but action not allowed
404 Not Found           resource missing
409 Conflict            duplicate/conflicting state
413 Payload Too Large   upload too large
415 Unsupported Media   unsupported file/content type
416 Range Not Satisfiable invalid audio range
422 Unprocessable Entity semantically invalid request
429 Too Many Requests   rate limited
500 Internal Error      sanitized unexpected error
501 Not Implemented     route exists but feature deferred
```

---

# Resource IDs

Use opaque IDs.

Recommended prefixes:

```txt
file_...
tr_...
key_...
```

Do not expose database integer IDs unless already unavoidable internally.

---

# Core Data Models

## File

A file is an uploaded or imported media source.

```json
{
  "id": "file_abc",
  "title": "Team sync",
  "kind": "audio",
  "status": "ready",
  "mime_type": "audio/mpeg",
  "size_bytes": 12345678,
  "duration_seconds": 3600.5,
  "created_at": "2026-04-25T18:00:00Z",
  "updated_at": "2026-04-25T18:00:00Z"
}
```

Allowed `kind`:

```txt
audio
video
youtube
```

Allowed `status`:

```txt
uploaded
processing
ready
failed
```

## Transcription

A transcription is an async job derived from a file.

```json
{
  "id": "tr_abc",
  "file_id": "file_abc",
  "title": "Team sync",
  "status": "queued",
  "language": "en",
  "diarization": true,
  "created_at": "2026-04-25T18:00:00Z",
  "updated_at": "2026-04-25T18:00:00Z",
  "started_at": null,
  "completed_at": null,
  "failed_at": null,
  "error": null
}
```

Allowed `status`:

```txt
queued
processing
completed
failed
canceled
```

## Transcript

```json
{
  "transcription_id": "tr_abc",
  "text": "Full transcript text...",
  "segments": [
    {
      "id": "seg_001",
      "start": 0.0,
      "end": 4.2,
      "speaker": "SPEAKER_00",
      "text": "Hello world."
    }
  ],
  "words": [
    {
      "start": 0.0,
      "end": 0.4,
      "word": "Hello",
      "speaker": "SPEAKER_00"
    }
  ]
}
```

---

# Health

## Get health

```http
GET /health
```

Response:

```json
{
  "status": "ok"
}
```

## Get API health

```http
GET /api/v1/health
```

Response:

```json
{
  "status": "ok"
}
```

## Get readiness

```http
GET /api/v1/ready
```

Response:

```json
{
  "status": "ready",
  "database": "ok"
}
```

---

# Auth API

## Get registration status

```http
GET /api/v1/auth/registration-status
```

Response:

```json
{
  "registration_enabled": true
}
```

## Register initial user

```http
POST /api/v1/auth/register
```

Request:

```json
{
  "username": "admin",
  "password": "password",
  "confirm_password": "password"
}
```

Response:

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "user": {
    "id": "user_self",
    "username": "admin"
  }
}
```

## Login

```http
POST /api/v1/auth/login
```

Request:

```json
{
  "username": "admin",
  "password": "password"
}
```

Response:

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "user": {
    "id": "user_self",
    "username": "admin"
  }
}
```

## Refresh token

```http
POST /api/v1/auth/refresh
```

Request:

```json
{
  "refresh_token": "..."
}
```

Response:

```json
{
  "access_token": "...",
  "refresh_token": "..."
}
```

## Logout

```http
POST /api/v1/auth/logout
```

Request:

```json
{
  "refresh_token": "..."
}
```

Response:

```json
{
  "ok": true
}
```

## Get current user

```http
GET /api/v1/auth/me
```

Response:

```json
{
  "id": "user_self",
  "username": "admin"
}
```

## Change password

```http
POST /api/v1/auth/change-password
```

Request:

```json
{
  "current_password": "old",
  "new_password": "new",
  "confirm_password": "new"
}
```

Response:

```json
{
  "ok": true
}
```

## Change username

```http
POST /api/v1/auth/change-username
```

Request:

```json
{
  "new_username": "newname",
  "password": "password"
}
```

Response:

```json
{
  "id": "user_self",
  "username": "newname"
}
```

---

# API Keys

## List API keys

```http
GET /api/v1/api-keys
```

Response:

```json
{
  "items": [
    {
      "id": "key_abc",
      "name": "CLI",
      "description": "Local scripts",
      "key_preview": "sk_...abcd",
      "is_active": true,
      "last_used_at": null,
      "created_at": "2026-04-25T18:00:00Z",
      "updated_at": "2026-04-25T18:00:00Z"
    }
  ],
  "next_cursor": null
}
```

## Create API key

```http
POST /api/v1/api-keys
```

Request:

```json
{
  "name": "CLI",
  "description": "Local scripts"
}
```

Response:

```json
{
  "id": "key_abc",
  "name": "CLI",
  "description": "Local scripts",
  "key": "sk_live_raw_key_returned_once",
  "key_preview": "sk_...abcd",
  "created_at": "2026-04-25T18:00:00Z"
}
```

## Delete API key

```http
DELETE /api/v1/api-keys/{id}
```

Response:

```http
204 No Content
```

---

# Files

## Upload file

```http
POST /api/v1/files
Content-Type: multipart/form-data
```

Fields:

```txt
file: audio/video file
title?: string
```

Response:

```http
201 Created
```

```json
{
  "id": "file_abc",
  "title": "Team sync",
  "kind": "audio",
  "status": "ready",
  "mime_type": "audio/mpeg",
  "size_bytes": 12345678,
  "duration_seconds": 3600.5,
  "created_at": "2026-04-25T18:00:00Z",
  "updated_at": "2026-04-25T18:00:00Z"
}
```

Notes:

* Audio files are stored directly.
* Video files may be accepted and converted/extracted internally.
* The API should not expose the storage path.

## Import YouTube audio

```http
POST /api/v1/files:import-youtube
```

Request:

```json
{
  "url": "https://www.youtube.com/watch?v=...",
  "title": "Optional title"
}
```

Response:

```http
202 Accepted
```

```json
{
  "id": "file_abc",
  "title": "Optional title",
  "kind": "youtube",
  "status": "processing"
}
```

## List files

```http
GET /api/v1/files
```

Query params:

```txt
limit?: integer
cursor?: string
q?: string
kind?: audio|video|youtube
status?: uploaded|processing|ready|failed
sort?: created_at|-created_at|updated_at|-updated_at|title|-title
```

Response:

```json
{
  "items": [],
  "next_cursor": null
}
```

## Get file

```http
GET /api/v1/files/{id}
```

## Update file metadata

```http
PATCH /api/v1/files/{id}
```

Request:

```json
{
  "title": "New title"
}
```

## Delete file

```http
DELETE /api/v1/files/{id}
```

Response:

```http
204 No Content
```

---

# Audio Streaming

## Stream file audio

```http
GET /api/v1/files/{id}/audio
```

Supports full and ranged requests.

Full request:

```http
GET /api/v1/files/file_abc/audio
```

Partial request:

```http
GET /api/v1/files/file_abc/audio
Range: bytes=0-1048575
```

Successful partial response:

```http
206 Partial Content
Accept-Ranges: bytes
Content-Range: bytes 0-1048575/12345678
Content-Length: 1048576
Content-Type: audio/mpeg
```

Invalid range response:

```http
416 Range Not Satisfiable
Content-Range: bytes */12345678
```

Security:

* Must require auth.
* Must validate resource access.
* Must not reveal filesystem paths.
* Must not load entire file into memory.

---

# Transcriptions

## Create transcription

```http
POST /api/v1/transcriptions
```

Request:

```json
{
  "file_id": "file_abc",
  "title": "Team sync",
  "profile_id": "default",
  "options": {
    "language": "en",
    "diarization": true
  }
}
```

Response:

```http
202 Accepted
```

```json
{
  "id": "tr_abc",
  "file_id": "file_abc",
  "title": "Team sync",
  "status": "queued",
  "created_at": "2026-04-25T18:00:00Z"
}
```

Notes:

* Creating a transcription should enqueue work.
* It should not block until transcription completes.
* `profile_id` may be omitted to use default settings.
* `options` should be validated but kept small.

## Upload and create transcription

Convenience endpoint.

```http
POST /api/v1/transcriptions:submit
Content-Type: multipart/form-data
```

Fields:

```txt
file: audio/video file
title?: string
profile_id?: string
options?: JSON string
```

Response:

```http
202 Accepted
```

```json
{
  "id": "tr_abc",
  "file_id": "file_abc",
  "status": "queued"
}
```

## List transcriptions

```http
GET /api/v1/transcriptions
```

Query params:

```txt
limit?: integer
cursor?: string
q?: string
status?: queued|processing|completed|failed|canceled
updated_after?: RFC3339 timestamp
sort?: created_at|-created_at|updated_at|-updated_at|title|-title
```

Response:

```json
{
  "items": [
    {
      "id": "tr_abc",
      "file_id": "file_abc",
      "title": "Team sync",
      "status": "completed",
      "duration_seconds": 3600.5,
      "created_at": "2026-04-25T18:00:00Z",
      "updated_at": "2026-04-25T18:10:00Z"
    }
  ],
  "next_cursor": null
}
```

## Get transcription

```http
GET /api/v1/transcriptions/{id}
```

Response:

```json
{
  "id": "tr_abc",
  "file_id": "file_abc",
  "title": "Team sync",
  "status": "completed",
  "language": "en",
  "diarization": true,
  "created_at": "2026-04-25T18:00:00Z",
  "updated_at": "2026-04-25T18:10:00Z",
  "started_at": "2026-04-25T18:01:00Z",
  "completed_at": "2026-04-25T18:10:00Z",
  "failed_at": null,
  "error": null
}
```

## Update transcription metadata

```http
PATCH /api/v1/transcriptions/{id}
```

Request:

```json
{
  "title": "New title"
}
```

## Delete transcription

```http
DELETE /api/v1/transcriptions/{id}
```

Response:

```http
204 No Content
```

## Cancel transcription

```http
POST /api/v1/transcriptions/{id}:cancel
```

Response:

```json
{
  "id": "tr_abc",
  "status": "canceled"
}
```

## Retry transcription

```http
POST /api/v1/transcriptions/{id}:retry
```

Response:

```http
202 Accepted
```

```json
{
  "id": "tr_retry",
  "source_transcription_id": "tr_abc",
  "status": "queued"
}
```

## Get transcript

```http
GET /api/v1/transcriptions/{id}/transcript
```

Response:

```json
{
  "transcription_id": "tr_abc",
  "text": "Full transcript text...",
  "segments": [],
  "words": []
}
```

## Stream transcription audio

```http
GET /api/v1/transcriptions/{id}/audio
```

Same range behavior as:

```http
GET /api/v1/files/{id}/audio
```

This is a convenience alias to the source file audio or normalized transcription artifact.

## Get transcription events

```http
GET /api/v1/transcriptions/{id}/events
Accept: text/event-stream
```

Streams job-specific progress events.

Example event:

```txt
event: transcription.progress
data: {"id":"tr_abc","status":"processing","progress":0.42}
```

## Get transcription logs

```http
GET /api/v1/transcriptions/{id}/logs
```

Response:

```http
Content-Type: text/plain
```

Logs must be sanitized.

## Get execution metadata

```http
GET /api/v1/transcriptions/{id}/executions
```

Response:

```json
{
  "items": [
    {
      "id": "exec_abc",
      "transcription_id": "tr_abc",
      "status": "completed",
      "started_at": "2026-04-25T18:01:00Z",
      "completed_at": "2026-04-25T18:10:00Z",
      "processing_duration_ms": 540000,
      "error": null
    }
  ],
  "next_cursor": null
}
```

---

# Profiles

Profiles are local presets for transcription options.

## List profiles

```http
GET /api/v1/profiles
```

## Create profile

```http
POST /api/v1/profiles
```

Request:

```json
{
  "name": "Fast local",
  "description": "Fast local transcription",
  "is_default": true,
  "options": {
    "model": "base",
    "language": "en",
    "diarization": false,
    "device": "auto"
  }
}
```

## Get profile

```http
GET /api/v1/profiles/{id}
```

## Update profile

```http
PATCH /api/v1/profiles/{id}
```

## Delete profile

```http
DELETE /api/v1/profiles/{id}
```

## Set default profile

```http
POST /api/v1/profiles/{id}:set-default
```

Response:

```json
{
  "id": "profile_abc",
  "is_default": true
}
```

---

# Settings

Settings are single-user application settings.

## Get settings

```http
GET /api/v1/settings
```

Response:

```json
{
  "auto_transcription_enabled": false,
  "default_profile_id": "profile_abc",
  "local_only": true,
  "max_upload_size_mb": 2048
}
```

## Update settings

```http
PATCH /api/v1/settings
```

Request:

```json
{
  "auto_transcription_enabled": true,
  "default_profile_id": "profile_abc"
}
```

---

# Events

Global event stream for UI updates.

```http
GET /api/v1/events
Accept: text/event-stream
```

Example events:

```txt
event: file.ready
data: {"id":"file_abc"}

event: transcription.updated
data: {"id":"tr_abc","status":"processing"}

event: transcription.completed
data: {"id":"tr_abc"}
```

---

# Models / Capabilities

## Get supported transcription models

```http
GET /api/v1/models/transcription
```

Response:

```json
{
  "items": [
    {
      "id": "base",
      "name": "Whisper base",
      "provider": "local",
      "capabilities": ["transcription"]
    }
  ]
}
```

---

# Admin / Diagnostics

Keep minimal for single-user mode.

## Queue stats

```http
GET /api/v1/admin/queue
```

Response:

```json
{
  "queued": 0,
  "processing": 0,
  "completed": 10,
  "failed": 1
}
```

---

# Deferred Modules

These are intentionally not part of the first clean API pass:

```txt
summaries
chat
notes
speaker editing
webhooks
multi-track
team/user administration
```

They should be added later as independent modules:

```http
/api/v1/summaries
/api/v1/chat/sessions
/api/v1/transcriptions/{id}/notes
/api/v1/transcriptions/{id}/speakers
/api/v1/webhooks
```

---

# Legacy Compatibility

Canonical API should use the clean routes above.

Legacy aliases may be added temporarily only if needed by the current frontend.

Do not design new code around old routes like:

```http
/api/v1/transcription/list
/api/v1/transcription/{id}/kill
/api/v1/transcription/upload-video
/api/v1/transcription/youtube
```

Map them to canonical services internally if compatibility is required.

---

# Sprint 1 Implementation Scope

Sprint 1 should implement the foundation, not the whole product.

Required:

```txt
router setup
route groups
middleware
request ID
panic recovery
structured errors
auth guard
API key auth foundation
JSON helpers
file route skeleton
transcription route skeleton
range streaming contract
minimal service interfaces
high-value tests
```

Allowed to stub with `501 Not Implemented`:

```txt
actual transcription execution
YouTube import execution
video extraction execution
model discovery
queue internals
SSE event backend
logs backend
```

But route shape and response contracts should be established.
