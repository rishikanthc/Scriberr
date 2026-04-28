# Highlights and Notes Backend Sprint Plan

## Current Assessment

The backend does not currently have a clean notes/highlights implementation ready for use.

What exists now:

- Notes and highlights are listed as deferred modules in `devnotes/v2.0.0/specs/api-v1-master-spec.md`.
- The canonical route family is reserved under `/api/v1/transcriptions/{id}/notes`.
- The current transcript API returns transcript text, segments, and words from `GET /api/v1/transcriptions/{id}/transcript`.
- Transcription records already carry `user_id`, so future ownership checks have a clear parent boundary.

What was removed as legacy backend code:

- The old `models.Note` persistence record.
- The generic `NoteRepository`.
- Old note table schema registration and indexes.
- Legacy note migration/backfill code.
- Database tests that asserted the old note shape.

Why the old note schema was not a good foundation:

- It only modeled notes, not highlights as a first-class resource.
- It mixed compatibility fields into the persistence model.
- It had weak anchoring for transcript text changes and retranscription.
- It used internal transcription IDs directly without a clean public response contract.
- It did not give enough room for future multi-user behavior such as ownership, sharing, or audit-safe mutation.

## Target Backend Model

Use a new annotation-oriented schema rather than resurrecting the old notes table.

Recommended table: `transcript_annotations`

Core columns:

```txt
id                     string primary key
user_id                uint not null indexed
transcription_id       string not null indexed
kind                   string not null enum-like: highlight | note
content                text null
color                  string null
quote                  text not null
anchor_start_ms        integer not null
anchor_end_ms          integer not null
anchor_start_word      integer null
anchor_end_word        integer null
anchor_start_char      integer null
anchor_end_char        integer null
anchor_text_hash       string null
status                 string not null default active
metadata_json          json not null default {}
created_at             timestamp
updated_at             timestamp
deleted_at             soft delete
```

Indexes and constraints:

- `idx_transcript_annotations_user_transcription_created_at` on `(user_id, transcription_id, created_at DESC)`.
- `idx_transcript_annotations_user_kind_updated_at` on `(user_id, kind, updated_at DESC)`.
- `idx_transcript_annotations_transcription_time` on `(transcription_id, anchor_start_ms, anchor_end_ms)`.
- Foreign key from `transcription_id` to `transcriptions(id)` with cascade delete.
- Check constraint or service validation for `kind IN ('highlight', 'note')`.
- Service validation that `anchor_end_ms >= anchor_start_ms`.

Multi-user readiness:

- Every query must include `user_id`; never fetch by annotation ID alone.
- Authorization should be inherited from transcription ownership for now.
- Keep `user_id` on the annotation even though the transcription also has it. This supports future shared transcripts where annotation ownership may differ from transcript ownership.
- Public IDs should be opaque and prefixed, for example `ann_...`.
- API responses should never expose raw database IDs.

## Target REST API

Use thin handlers and route through a notes/highlights service.

Recommended canonical routes:

```http
GET    /api/v1/transcriptions/{id}/annotations
POST   /api/v1/transcriptions/{id}/annotations
GET    /api/v1/transcriptions/{id}/annotations/{annotation_id}
PATCH  /api/v1/transcriptions/{id}/annotations/{annotation_id}
DELETE /api/v1/transcriptions/{id}/annotations/{annotation_id}
```

Supported filters:

```txt
kind=highlight|note
updated_after=RFC3339 timestamp
cursor=...
limit=...
```

Convenience aliases may be added only if the frontend needs resource-specific URLs:

```http
GET  /api/v1/transcriptions/{id}/notes
GET  /api/v1/transcriptions/{id}/highlights
```

Those aliases should call the same annotation service with a fixed `kind` filter. Do not create separate persistence paths.

Example create request:

```json
{
  "kind": "note",
  "content": "Follow up on this decision",
  "color": "yellow",
  "quote": "We should ship the smaller model first",
  "anchor": {
    "start_ms": 12400,
    "end_ms": 18900,
    "start_word": 42,
    "end_word": 51,
    "start_char": 280,
    "end_char": 336,
    "text_hash": "sha256:..."
  }
}
```

Collection response:

```json
{
  "items": [],
  "next_cursor": null
}
```

## Sprint 1: Schema and Migration

Goal: add a clean annotations persistence model.

Tasks:

- Add `models.TranscriptAnnotation` with persistence fields only.
- Add schema migration for `transcript_annotations`.
- Add indexes for per-transcription list, per-kind list, and time-range lookup.
- Add database tests for fresh schema, foreign key behavior, soft delete, and user-scoped indexes.
- Keep legacy `notes` migration removed; old note data is intentionally not imported into the new model.

Acceptance criteria:

- Fresh databases create `transcript_annotations`.
- Deleting a transcription deletes its annotations.
- Invalid annotation kinds and invalid time ranges are rejected by service/model validation.

## Sprint 2: Repository and Service

Goal: keep HTTP thin and put annotation decisions behind a domain service.

Tasks:

- Add an annotation repository with domain methods:
  - `CreateAnnotation`
  - `FindAnnotationForUser`
  - `ListAnnotationsForTranscription`
  - `UpdateAnnotation`
  - `SoftDeleteAnnotation`
- Add an annotation service that:
  - verifies transcription ownership,
  - parses public IDs,
  - validates anchors,
  - normalizes content for highlights vs notes,
  - emits small annotation events.
- Add service tests for ownership, invalid IDs, kind validation, time-range validation, and soft-delete behavior.

Acceptance criteria:

- No new annotation handler reads `database.DB` directly.
- Annotation ID lookup is always scoped by `user_id` and `transcription_id`.
- Service methods return explicit not-found, validation, and conflict errors.

## Sprint 3: REST API

Goal: expose annotations through canonical v1 REST endpoints.

Tasks:

- Add request/response types separate from GORM models.
- Register routes under `/api/v1/transcriptions/{id}/annotations`.
- Implement list/create/get/update/delete handlers.
- Add pagination using the existing list-query helpers where appropriate.
- Add route contract tests and error-envelope tests.
- Add OpenAPI documentation for the new routes.

Acceptance criteria:

- Authenticated users can create, list, update, and delete their own transcript annotations.
- Other users cannot access annotations even if they know the public annotation ID.
- Responses use public IDs: `tr_...` and `ann_...`.
- API responses do not expose local paths or internal database-only fields.

## Sprint 4: Events and Transcript Integrity

Goal: make annotations live-update friendly and robust against transcript changes.

Tasks:

- Publish `annotation.created`, `annotation.updated`, and `annotation.deleted` SSE events.
- Include only public IDs and small cache-invalidation payloads.
- Add optional anchor verification against current transcript words/segments.
- Decide behavior on retranscription:
  - keep annotations but mark anchors as stale when text hash no longer matches, or
  - soft-delete annotations on destructive transcript replacement.
- Add tests for event payload shape and stale-anchor behavior.

Acceptance criteria:

- Annotation events are usable for cache invalidation.
- Full page refresh can reconstruct the correct state from persisted annotations.
- Retranscription behavior is explicit, tested, and documented.

## Sprint 5: Future Multi-User Hardening

Goal: avoid another schema rewrite when multi-user support arrives.

Tasks:

- Add tests that simulate two users with different transcriptions and annotations.
- Add service seams for future shared transcript permissions.
- Review whether annotation ownership should remain independent from transcription ownership.
- Add audit-friendly fields only if product requirements need them, such as `created_by_user_id` and `updated_by_user_id`.

Acceptance criteria:

- All annotation reads and writes are user-scoped.
- No repository method can accidentally return annotations across users.
- Future shared-transcript access can be added by changing authorization policy without changing the core annotation table.

