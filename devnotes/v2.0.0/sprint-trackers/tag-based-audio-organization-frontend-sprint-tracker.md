# Sprint Tracker: Tag-Based Audio Organization Frontend

This tracker belongs to `devnotes/v2.0.0/sprint-plans/tag-based-audio-organization-frontend-sprint-plan.md`.

Status: completed through Sprint 4. Sprint 5 audio detail tags is next.

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

Status: completed

Completed tasks:

- Added typed tag API functions for list/create/update/delete tags.
- Added typed transcription-tag assignment API functions for list/replace/add/remove.
- Added tag query hooks and mutation hooks with scoped invalidation.
- Added tag-name normalization and duplicate-name helper.
- Extended transcription listing API/hook support for tag-filtered queries.

Artifacts:

- `web/frontend/src/features/tags/api/tagsApi.ts`
- `web/frontend/src/features/tags/hooks/useTags.ts`
- `web/frontend/src/features/transcription/api/transcriptionsApi.ts`
- `web/frontend/src/features/transcription/hooks/useTranscriptions.ts`

Commit:

- `93db494` (`feat: add frontend tag api hooks`)

Verification:

- `npm run type-check` from `web/frontend`

## Sprint 3: Sidebar Tags Navigation

Status: completed

Completed tasks:

- Added the sidebar Tags accordion.
- Added sidebar loading/empty/error states.
- Updated active-nav styling to accent text only.
- Added a dedicated `/tags/{tag_id}` page.
- Made sidebar tag clicks navigate to the dedicated tag page.
- Reused the Home audio-list experience on the tag page with a tag filter.
- Added an empty state when the selected tag has no audio.
- Displayed assigned tags below each audio description in the Home audio list.
- Used the same audio-card tag display on the dedicated tag page.

Artifacts:

- `web/frontend/src/App.tsx`
- `web/frontend/src/features/home/components/HomePage.tsx`
- `web/frontend/src/styles/design-system.css`

Commit:

- `8f5aa11` (`feat: add tag navigation audio list`)

Verification:

- `npm run type-check` from `web/frontend`

## Sprint 4: Settings Tags Management

Status: completed

Completed tasks:

- Added Settings `Tags` tab.
- Added create/edit tag dialog with name, description, and when-to-use fields.
- Added simple tag list with edit/delete icon actions.
- Added client-side duplicate-name validation.
- Added delete confirmation and surfaced mutation errors.
- Tightened the tag dialog width and spacing for a compact management workflow.

Artifacts:

- `web/frontend/src/features/settings/pages/SettingsPage.tsx`
- `web/frontend/src/features/tags/components/TagDialog.tsx`
- `web/frontend/src/features/tags/components/TagsSettingsPanel.tsx`
- `web/frontend/src/styles/design-system.css`

Commits:

- `8a75847` (`feat: add settings tag management`)
- `71e0ab0` (`style: compact tag dialog`)

Verification:

- `npm run type-check` from `web/frontend`

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
