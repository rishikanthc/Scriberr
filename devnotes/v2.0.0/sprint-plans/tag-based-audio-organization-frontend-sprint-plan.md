# Tag-Based Audio Organization Frontend Sprint Plan

## Current Assessment

The backend tag foundation is complete through `devnotes/v2.0.0/sprint-plans/tag-based-audio-organization-backend-sprint-plan.md`.

Frontend state today:

- The source frontend lives in `web/frontend`; `internal/web/frontend` is the built bundle.
- The left app shell sidebar is currently implemented in `web/frontend/src/features/home/components/HomePage.tsx`.
- Settings currently has tabs for General, ASR, LLM Providers, and Summarization.
- Audio detail rendering is concentrated in `web/frontend/src/features/transcription/components/AudioDetailView.tsx`, with transcript and summary panels already split internally.
- Server state uses TanStack Query hooks for files, profiles, transcriptions, annotations, summaries, and events.

Required changes:

- Add a left sidebar `Tags` entry with expandable/collapsible tag list.
- Remove active-nav background highlight; active nav text should use the accent color only.
- Add a Settings `Tags` tab with add/edit/delete tag management.
- Add a tag dialog with tag name, optional description, and optional “When to use”.
- Prevent duplicate tag names in the UI and surface backend duplicate conflicts.
- Add audio-detail tag display styled like the supplied keyword area, replacing keyword copy with tag chips.
- Add an icon action button that opens a searchable multi-select dropdown for available tags.
- Allow tag removal from each assigned tag on hover/focus.

Backend contract gap:

- The current tag backend supports `name`, `color`, and `description`, but not `when_to_use`.
- Add `when_to_use` to the tag API before wiring the Settings dialog so user-entered guidance persists.

## Target Frontend Architecture

Feature placement:

```txt
features/tags/api
features/tags/hooks
features/tags/components
```

Use these boundaries:

- API functions define tag payload/response types and endpoint details.
- Hooks own TanStack Query keys and cache invalidation.
- Sidebar, settings, and audio-detail components consume hooks, not raw `fetch`.
- Shared visual primitives remain in `components/ui` or `shared/ui`; tag-specific workflow UI stays in `features/tags/components`.

Query keys:

```txt
tags
transcription-tags/{transcriptionId}
transcriptions
```

Invalidation:

- Tag create/update/delete invalidates `tags`.
- Tag delete also invalidates all transcription tag queries and transcriptions list filters where practical.
- Transcription tag add/remove/replace invalidates `transcription-tags/{id}` and `transcriptions`.

## Sprint 1: Contract and Planning

Goal: document the frontend work and close the `when_to_use` persistence gap.

Tasks:

- Add this sprint plan.
- Add a matching sprint tracker.
- Extend backend `AudioTag` persistence, service, API handlers, and tests with `when_to_use`.
- Keep this as a small backend contract commit before React work.

Acceptance criteria:

- `when_to_use` is returned in tag responses.
- Tag create/update supports `when_to_use`.
- Existing tag tests still pass.

## Sprint 2: Tag API Client and Hooks

Goal: provide typed frontend access to tag resources.

Tasks:

- Add `features/tags/api/tagsApi.ts`.
- Add `features/tags/hooks/useTags.ts`.
- Include typed APIs for:
  - list/create/update/delete tags,
  - list transcription tags,
  - replace transcription tags,
  - add/remove one transcription tag.
- Add duplicate-name helper for client-side validation.

Acceptance criteria:

- Components can read and mutate tags without inline endpoint construction.
- Hooks invalidate the smallest useful query keys.
- API types use backend snake_case fields explicitly.

## Sprint 3: Sidebar Tags Navigation

Goal: expose tags in the app shell without adding heavy navigation chrome.

Tasks:

- Add `Tags` sidebar section with an expand/collapse button.
- Show all tags when expanded.
- Show loading, empty, and error states compactly.
- Update active-nav styling to accent-colored text only; remove active background/border highlight.
- Add a dedicated tag detail route such as `/tags/{tag_id}`.
- Clicking a sidebar tag navigates to that tag detail route.
- The tag detail page should reuse the Home audio-list experience and show only audio whose latest/visible transcription has that tag.
- Show a graceful empty state when no audio exists for the selected tag.

Acceptance criteria:

- Sidebar has Home, Tags, and Settings.
- Active item uses accent text without background highlight.
- Empty tag list is graceful and does not shift layout unexpectedly.
- Clicking a tag opens a dedicated page for that tag.
- The tag page matches the Home audio list behavior while filtering to the selected tag.

## Sprint 4: Settings Tags Management

Goal: add CRUD tag management in Settings.

Tasks:

- Add a `Tags` tab in Settings.
- Add a simple tag list with edit/delete icon actions.
- Add `TagDialog` for create/edit with name, optional description, and optional when-to-use fields.
- Prevent duplicate tag names before submit.
- Confirm deletes to avoid accidental removal.
- Surface backend validation/conflict errors.

Acceptance criteria:

- Users can add, edit, and delete tags from Settings.
- Duplicate names are blocked client-side and by backend response handling.
- The list handles loading, empty, and error states.

## Sprint 5: Audio Detail Tags

Goal: display and manage assigned tags on the audio detail screen.

Tasks:

- Add an audio-detail tag section near the transcript/summary content, matching the supplied keyword-style layout but showing tags.
- Add icon action button with tooltip/label.
- Add searchable popover/dropdown with multi-select behavior and selected-state indicators.
- Allow adding multiple tags at a time.
- Allow removing assigned tags from tag chips on hover/focus.
- Keep tag mutations out of transcript hot-path render loops.

Acceptance criteria:

- Assigned tags are visible on audio detail.
- Users can search available tags and assign multiple tags.
- Users can remove assigned tags directly from each chip.
- Loading, empty, and mutation-pending states are explicit.

## Sprint 6: Verification

Goal: verify the integration and update the tracker.

Tasks:

- Run type-check/build for `web/frontend`.
- Run focused backend tests for the contract extension.
- Use the in-app browser or Playwright to inspect sidebar, settings tags, and audio-detail tag layout if a dev server can run.
- Update the sprint tracker with commits and verification results.

Acceptance criteria:

- Frontend type-check/build passes, or any blocker is documented.
- Backend tag tests pass.
- Tracker accurately reflects completed work and commits.
