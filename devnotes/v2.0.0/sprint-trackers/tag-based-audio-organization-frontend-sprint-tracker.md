# Sprint Tracker: Tag-Based Audio Organization Frontend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/tag-based-audio-organization-frontend-sprint-plan.md`.

Status: completed through Sprint 1. Sprint 2 tag API client and hooks is next.

## Sprint 1: Contract and Planning

Status: completed

Completed tasks:

- Added the frontend sprint plan.
- Added this sprint tracker.
- Identified that the requested Settings dialog field `when_to_use` needs a backend contract extension before frontend persistence.
- Extended backend tag persistence/API with `when_to_use`.
- Added focused model/database/service/API coverage for create, update, and response shape.

Artifacts:

- `internal/models/tag.go`
- `internal/database/schema.go`
- `internal/database/steps.go`
- `internal/database/database_test.go`
- `internal/repository/implementations.go`
- `internal/tags/service.go`
- `internal/tags/service_test.go`
- `internal/api/tag_handlers.go`
- `internal/api/tag_handlers_test.go`

Commits:

- `d5d7371` (`docs: plan tag organization frontend`)
- `ad09443` (`docs: plan tag detail page sprint`)
- `d4b875f` (`feat: add tag usage guidance field`)

Verification:

- `GOCACHE=/Users/zade/Code/asr/Scriberr/.tmp/go-build go test ./internal/tags ./internal/database ./internal/api -run 'TestTag|TestAudioTag|TestServiceCreateListUpdateDeleteTag'`

## Sprint 2: Tag API Client and Hooks

Status: pending

Remaining tasks:

- Add typed tag API functions.
- Add TanStack Query hooks and query keys.
- Add scoped invalidation for tag and transcription-tag mutations.

## Sprint 3: Sidebar Tags Navigation

Status: pending

Remaining tasks:

- Add the sidebar Tags accordion.
- Add loading/empty/error states.
- Update active-nav styling to accent text only.
- Add a dedicated `/tags/{tag_id}` page.
- Make sidebar tag clicks navigate to the dedicated tag page.
- Reuse the Home audio-list experience on the tag page with a tag filter.
- Show an empty state when the selected tag has no audio.

## Sprint 4: Settings Tags Management

Status: pending

Remaining tasks:

- Add Settings `Tags` tab.
- Add create/edit dialog.
- Add simple list with edit/delete actions.
- Add duplicate-name validation.

## Sprint 5: Audio Detail Tags

Status: pending

Remaining tasks:

- Add audio-detail tag display section.
- Add searchable multi-select tag picker.
- Add hover/focus remove affordance on assigned tags.

## Sprint 6: Verification

Status: pending

Remaining tasks:

- Run frontend type-check/build.
- Run focused backend tag tests.
- Inspect the key UI states in browser if a local dev server can run.
- Update this tracker with commits and verification.
