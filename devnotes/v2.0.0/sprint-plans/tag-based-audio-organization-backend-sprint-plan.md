# Tag-Based Audio Organization Backend Sprint Plan

## Current Assessment

Scriberr currently organizes audio primarily through uploaded/imported files and transcription records.

What exists now:

- Audio files and transcriptions are represented by `transcriptions` rows. Source file rows use `source_file_hash IS NULL`; transcription rows reference their source file through `source_file_hash`.
- The v1 API exposes file and transcription list/detail routes.
- Transcript annotations already demonstrate the target backend architecture: persistence models, repository methods, a domain service, thin REST handlers, public ID mapping, and event notifications.
- List endpoints use bounded pagination and stable response envelopes.

What is missing:

- No first-class tag table.
- No normalized many-to-many relationship between audio/transcription records and tags.
- No tag service or repository boundary.
- No REST routes for creating, editing, deleting, assigning, removing, or listing tags.
- No supported filter for fetching audio/transcription details by one or more tags.

## Target Backend Model

Use normalized relational tables rather than storing tag names in JSON or comma-separated columns.

Recommended tables:

```txt
audio_tags
audio_tag_assignments
```

`audio_tags` columns:

```txt
id              string primary key
user_id         uint not null indexed
name            string not null
normalized_name string not null
color           string null
description     text null
metadata_json   json not null default {}
created_at      timestamp
updated_at      timestamp
deleted_at      soft delete
```

`audio_tag_assignments` columns:

```txt
id               string primary key
user_id          uint not null indexed
tag_id           string not null indexed
transcription_id string not null indexed
created_at       timestamp
deleted_at       soft delete
```

Indexes and constraints:

- Unique active tag name per user: `(user_id, normalized_name) WHERE deleted_at IS NULL`.
- Unique active assignment per user/tag/transcription: `(user_id, tag_id, transcription_id) WHERE deleted_at IS NULL`.
- List tags by user/name: `(user_id, normalized_name)`.
- List assignments by transcription: `(user_id, transcription_id, created_at DESC)`.
- Fetch tagged audio by tag: `(user_id, tag_id, transcription_id)`.
- Foreign key from assignments to tags with cascade delete.
- Foreign key from assignments to transcriptions with cascade delete.

Design notes:

- Tags are user-owned resources even though Scriberr is currently single-user. All repository queries must include `user_id`.
- Assignments attach to transcription/audio records where `source_file_hash IS NOT NULL`, matching the user-facing audio/transcription list.
- Tag names are mutable; assignment identity is based on tag ID, not name.
- `normalized_name` is for uniqueness and lookup. The public API returns the display `name`.
- Public IDs should use `tag_...`; API responses must not expose raw database-only fields.

## Target REST API

Canonical routes:

```http
GET    /api/v1/tags
POST   /api/v1/tags
GET    /api/v1/tags/{tag_id}
PATCH  /api/v1/tags/{tag_id}
DELETE /api/v1/tags/{tag_id}

GET    /api/v1/transcriptions/{id}/tags
PUT    /api/v1/transcriptions/{id}/tags
POST   /api/v1/transcriptions/{id}/tags/{tag_id}
DELETE /api/v1/transcriptions/{id}/tags/{tag_id}
```

Transcription list filters:

```txt
tag=tag_abc
tag=meeting
tags=tag_abc,tag_def
tags=meeting,client
tag_match=any|all
```

Default matching should be `any`; clients can request `all` when every listed tag must be present.

Example create tag request:

```json
{
  "name": "Client Call",
  "color": "#E87539",
  "description": "Calls with customers and prospects"
}
```

Example tag response:

```json
{
  "id": "tag_abc",
  "name": "Client Call",
  "color": "#E87539",
  "description": "Calls with customers and prospects",
  "created_at": "2026-05-02T12:00:00Z",
  "updated_at": "2026-05-02T12:00:00Z"
}
```

Example replace tags request:

```json
{
  "tag_ids": ["tag_abc", "tag_def"]
}
```

Collection responses should use the standard envelope:

```json
{
  "items": [],
  "next_cursor": null
}
```

## Sprint 1: Schema and Migration

Goal: add a normalized tag schema with scalable relationship tables.

Tasks:

- Add `models.AudioTag` and `models.AudioTagAssignment` as persistence records only.
- Register both models in target schema migration.
- Add partial unique indexes for active tag names and active assignments.
- Add read-path indexes for tag listing and tagged transcription filtering.
- Add database tests for fresh schema, uniqueness, soft delete reuse, and cascade behavior.

Acceptance criteria:

- Fresh databases create `audio_tags` and `audio_tag_assignments`.
- A user cannot have two active tags with the same normalized name.
- A transcription cannot receive the same active tag twice.
- Soft-deleted tags and assignments do not block recreation.
- Deleting a transcription or tag removes associated assignments.

## Sprint 2: Repository and Service

Goal: put tag decisions behind domain methods and keep HTTP thin.

Tasks:

- Add a tag repository with domain methods:
  - `CreateTag`
  - `FindTagForUser`
  - `FindTagsForUserByPublicOrName`
  - `ListTagsForUser`
  - `UpdateTag`
  - `SoftDeleteTag`
  - `ListTagsForTranscription`
  - `ReplaceTagsForTranscription`
  - `AddTagToTranscription`
  - `RemoveTagFromTranscription`
  - `ListTranscriptionIDsByTags`
- Add a tag service that:
  - verifies transcription ownership,
  - parses public tag and transcription IDs,
  - normalizes tag names,
  - validates colors and request limits,
  - maps duplicate creates/assignments into explicit conflict or idempotent behavior,
  - emits small tag events.
- Add service tests for ownership, duplicate names, assignment idempotency, tag filtering, invalid IDs, and soft-delete behavior.

Acceptance criteria:

- No new tag handler reads `database.DB` directly.
- Every lookup is scoped by `user_id`.
- Service methods return explicit not-found, validation, and conflict errors.

## Sprint 3: REST API

Goal: expose tag management and tagged transcription fetching through v1 REST.

Tasks:

- Add request/response types separate from GORM models.
- Register `/api/v1/tags` and `/api/v1/transcriptions/{id}/tags` routes.
- Implement list/create/get/update/delete tag handlers.
- Implement list/replace/add/remove transcription tag handlers.
- Extend transcription list filtering by one or more tags with `any` and `all` matching.
- Add route contract and handler tests for success and error-envelope behavior.
- Update API docs/OpenAPI after the backend contract stabilizes.

Acceptance criteria:

- Authenticated users can add, edit, delete, and list their own tags.
- Authenticated users can assign multiple tags to one audio/transcription.
- Authenticated users can fetch transcriptions by one or more tags.
- Other users cannot access or assign tags they do not own.
- Responses use public IDs and never expose persistence-only fields.

## Sprint 4: Events and UI Readiness

Goal: make tags live-update friendly for future frontend work.

Tasks:

- Publish `tag.created`, `tag.updated`, and `tag.deleted` events.
- Publish `transcription.tags.updated` after assignment changes.
- Keep event payloads small: public tag IDs, public transcription IDs, and cache-invalidation fields only.
- Add event payload tests.
- Document frontend integration expectations in the matching frontend sprint.

Acceptance criteria:

- Tag mutations are observable over existing event infrastructure.
- Full page refresh can reconstruct tag state from persisted records.
- Events do not become a source of truth.
