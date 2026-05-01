# YouTube Import Sprint Plan

This plan adds first-class support for importing YouTube videos as audio recordings that can be transcribed through the existing Scriberr workflow.

Related rules:

- `devnotes/v2.0.0/rules/backend-architecture-rules.md`
- `devnotes/v2.0.0/rules/react-architecture-rules.md`
- `devnotes/v2.0.0/rules/design-system-philosophy.md`

Related tracker:

- `devnotes/v2.0.0/sprint-trackers/youtube-import-sprint-tracker.md`

## Product Goal

Users can paste a YouTube URL from the homepage import menu. Scriberr downloads only the video's audio using `yt-dlp`, stores the extracted audio as a normal recording, shows live import progress in the same progress shelf used for file uploads, and refreshes the homepage list when the recording is ready.

Required behavior:

- Homepage Import opens a compact dropdown with:
  - Upload files
  - Import from YouTube
- Upload files keeps the existing file picker behavior.
- Import from YouTube opens a focused dialog with one URL input.
- Submitting a valid YouTube URL creates a durable import record immediately and returns quickly.
- The homepage shows an import progress toast/shelf entry while `yt-dlp` downloads and extracts audio.
- When audio is available, the recording list updates reactively without a page refresh.
- Failed imports show a clear failure state without leaking local paths or raw command output.
- Imported YouTube audio behaves like other ready recordings for playback and transcription.

Out of scope:

- Downloading video tracks.
- Playlists or batch YouTube imports.
- Browser extension based downloads.
- Automatic transcription after import.
- YouTube authentication/cookies.
- Supporting non-YouTube URLs.

## Current Codebase Notes

There is partial YouTube scaffolding in `internal/api/file_handlers.go`, `internal/api/youtube_importer.go`, route tests, and OpenAPI docs. It is useful as prior work, but the sprint should normalize it into the target architecture instead of preserving handler-owned background work.

Key cleanup requirements:

- Register only the canonical route: `POST /api/v1/files:import-youtube`.
- Remove any dead legacy frontend YouTube helpers under transcription-specific hooks/components.
- Move durable import orchestration out of handlers.
- Keep `yt-dlp` behind a narrow provider/importer boundary.
- Use the existing file list/event model as the source of truth for homepage reactivity.

## Target Backend Model

Use the existing recording/transcription job record as the durable file entity, with a clean import workflow around it.

Preferred flow:

```txt
POST /api/v1/files:import-youtube
  -> typed request validation and auth
  -> media import service creates processing recording
  -> repository persists import state
  -> durable importer worker executes yt-dlp
  -> repository marks ready/failed
  -> event publisher emits file.processing/file.ready/file.failed
```

Backend boundaries:

- `internal/api`: request binding, auth, response mapping.
- `internal/imports` or focused media import service: create/import orchestration and state transitions.
- `internal/repository`: persistence methods for create processing import, complete import, fail import, list files.
- `internal/media` or importer provider: `yt-dlp` command execution and sanitized output.
- `internal/models`: persisted import/file fields only.

REST contract:

```http
POST /api/v1/files:import-youtube
Content-Type: application/json

{
  "url": "https://www.youtube.com/watch?v=...",
  "title": "optional user-visible title"
}
```

Response:

```http
202 Accepted
{
  "id": "file_<id>",
  "title": "YouTube import",
  "kind": "youtube",
  "status": "processing",
  "mime_type": "",
  "size_bytes": 0,
  "duration_seconds": null,
  "created_at": "...",
  "updated_at": "..."
}
```

Events:

- `file.processing`: import accepted or progress changed.
- `file.ready`: audio is downloaded and available.
- `file.failed`: import failed.

Progress payload should stay small:

```json
{
  "id": "file_<id>",
  "kind": "youtube",
  "status": "processing",
  "progress": 42
}
```

## Target Frontend Model

Keep homepage import UI under feature-owned files/home boundaries:

```txt
features/files/api/filesApi.ts
features/files/hooks/useFileImport.ts
features/files/components/UploadProgressShelf.tsx
features/home/components/HomePage.tsx
```

Frontend behavior:

- The topbar Import control becomes a dropdown menu, not a direct file input trigger.
- The hidden file input remains the implementation detail for Upload files.
- YouTube import dialog owns only URL text, submit state, validation error display, and dismissal.
- Server state remains in TanStack Query. Import progress entries are local workflow state keyed to returned `file_id`.
- SSE file events update the progress shelf and invalidate `filesQueryKey`.
- The recordings list remains derived from `useFiles`, upload/import workflow items, and transcription state.

Design:

- Keep the import dropdown compact and predictable.
- Use Lucide icons for upload and YouTube/video import.
- Use existing dialog/input/button primitives.
- Reuse the upload progress shelf visual language instead of adding a second notification system.
- Keep messages concise: "Importing from YouTube", "Preparing audio", "Ready", "Import failed".

## Sprint Runs

### Sprint 1: Backend Contract and Route Wiring

- Audit existing YouTube scaffolding and remove dead or unreachable paths.
- Register `POST /api/v1/files:import-youtube`.
- Keep request and response shapes typed at the HTTP boundary.
- Validate YouTube hosts and reject unsupported URLs with structured validation errors.
- Update route contract tests and OpenAPI docs for the canonical route.

### Sprint 2: Durable Import Service and Repository Boundary

- Add a media import service/use-case for YouTube imports.
- Add repository methods for creating a processing import, marking ready, marking failed, and progress updates.
- Ensure state changes publish `file.processing`, `file.ready`, and `file.failed`.
- Keep local paths and command details out of API responses and event payloads.
- Add tests for successful create, invalid state transitions, failure, and ownership/list behavior.

### Sprint 3: yt-dlp Provider and Progress Parsing

- Keep `yt-dlp` execution behind a narrow importer interface.
- Download only audio with `--extract-audio` and a deterministic stored output path.
- Parse useful progress from `yt-dlp` output when available and update the import record/event stream.
- Sanitize command errors before persistence or API exposure.
- Clean up partial files on failure.
- Add fake importer tests for progress, completion, cancellation, and failure.

### Sprint 4: Frontend API and Import Workflow Hook

- Add typed frontend `importYouTube` API helper.
- Extend `useFileImport` to support URL imports alongside file uploads.
- Represent YouTube import workflow items in the existing progress shelf.
- Update file event handling to reflect processing/progress/ready/failed states.
- Ensure files and audio-file query keys invalidate at the smallest useful scope.

### Sprint 5: Homepage Import Dropdown and Dialog

- Replace direct Import click with a dropdown menu.
- Add Upload files and Import from YouTube menu items.
- Add a YouTube URL dialog with one accessible text input.
- Submit the URL through the import hook and close the dialog after successful creation.
- Keep the hidden file input behavior unchanged for file uploads.
- Remove dead legacy frontend YouTube components/hooks.

### Sprint 6: UX Polish and End-to-End Verification

- Verify the progress shelf copy and transitions for upload, YouTube processing, ready, and failed states.
- Verify the recording list updates from SSE without refresh.
- Verify imported YouTube audio opens on the audio detail page and can be submitted for transcription.
- Verify desktop and mobile layout.
- Run backend route/import tests and frontend type-check/build.
- Update the sprint tracker with artifacts, verification, and any known follow-up work.
