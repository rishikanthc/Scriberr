# Sprint Run Tracker: ASR Legacy Profile Cleanup

Run ID: `ASR-LEGACY-CLEANUP`

Status: completed.

This tracker belongs to `devnotes/v2.0.0/sprint-plans/asr-legacy-profile-cleanup-sprint-plan.md`.

## Run Rules

- Delete legacy ASR profile compatibility instead of preserving it.
- Keep ASR profiles pipeline-only.
- Keep cleanup scoped.
- Update this tracker in the same change set as each completed sprint.
- Run `git diff --check` before closing every sprint.

## Validation Checklist

- [x] Focused tests for touched frontend/backend/database packages.
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api ./internal/database ./internal/repository ./internal/models`.
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/profile ./internal/api ./internal/database ./internal/repository ./internal/models`.
- [x] `npm --prefix web/frontend run build` for frontend changes.
- [x] `git diff --check`.
- [x] Final `rg` check for removed legacy flat ASR fields.

## ASR-LEGACY-CLEANUP-Sprint 0: Inventory And Deletion Map

Status: completed

Planned tasks:

- [x] Inventory frontend legacy flat profile fields and normalization.
- [x] Inventory backend request structs and validation guards.
- [x] Inventory database legacy profile migration code and tests.
- [x] Decide which denormalized columns remain.
- [x] Produce deletion map in this tracker.

Acceptance checks:

- [x] Every legacy flat ASR profile path has a delete/keep decision.
- [x] No runtime behavior changes.

Verification:

- [x] `rg 'language|task|threads|tail_paddings|decoding_method|chunking_strategy|num_speakers|diarization_threshold|min_duration_on|min_duration_off|diarization' web/frontend/src/features/settings internal/api internal/database internal/models internal/profile -n`

Artifacts:

- `devnotes/v2.0.0/sprint-trackers/asr-legacy-profile-cleanup-sprint-tracker.md`

Deletion map:

- Frontend settings profile code:
  - Delete flat `TranscriptionProfileOptions` fields in `web/frontend/src/features/settings/api/profilesApi.ts`.
  - Delete `defaultProfileParams`, `familyForModel`, and flat `normalizeParams`.
  - Delete legacy flat save payload construction.
  - Replace `ASRProfileDialog.tsx` and `SettingsPage.tsx` flat-field reads during the frontend profile revamp rather than preserving compatibility.
- Backend profile API:
  - Delete flat fields from `profileOptionsRequest`.
  - Delete `legacyProfileOptionField`.
  - Keep `options.pipeline` required.
  - Update tests that currently expect field-specific legacy rejection to expect generic pipeline-only validation or JSON unknown-field behavior.
- Database legacy migration:
  - Delete legacy ASR profile migration structs/code that only migrate removed embedded `ASRParams`.
  - Keep non-ASR migration behavior intact.
  - Keep denormalized `transcription_profiles` columns (`provider`, `model_name`, `model_family`, `diarization_enabled`) because current model hooks derive them from pipeline for listing/filtering. They are not compatibility inputs.

Commit:

- Pending.

## ASR-LEGACY-CLEANUP-Sprint 1: Frontend Type And Payload Cleanup

Status: completed

Planned tasks:

- [x] Change frontend profile options to pipeline-only.
- [x] Delete flat-field default/normalization helpers.
- [x] Remove save payload mapping for legacy fields.
- [x] Update settings code to compile against pipeline-only types.
- [x] Leave full dialog revamp to `ASR-PROFILE-FE`.

Acceptance checks:

- [x] Frontend cannot emit legacy flat ASR profile fields.
- [x] Active TypeScript types are pipeline-only.
- [x] Build passes or compile blockers are explicitly handed to the frontend sprint.

Verification:

- [x] `npm --prefix web/frontend run build`
- [x] `npm --prefix web/frontend run lint` (passes with existing warnings outside this sprint)
- [x] `rg -n "defaultProfileParams|normalizeParams|familyForModel|tail_paddings|decoding_method|chunking_strategy|num_speakers|diarization_threshold|min_duration_on|min_duration_off|profile\.options\.(model|language|task|diarize|chunking_strategy)" web/frontend/src/features -g '*.ts' -g '*.tsx'`
- [x] `git diff --check -- web/frontend/src/features/settings/api/profilesApi.ts web/frontend/src/features/settings/components/ASRProfileDialog.tsx web/frontend/src/features/settings/pages/SettingsPage.tsx web/frontend/src/features/home/components/HomePage.tsx devnotes/v2.0.0/sprint-trackers/asr-legacy-profile-cleanup-sprint-tracker.md`

Artifacts:

- `web/frontend/src/features/settings/api/profilesApi.ts`
- `web/frontend/src/features/settings/components/ASRProfileDialog.tsx`
- `web/frontend/src/features/settings/pages/SettingsPage.tsx`
- `web/frontend/src/features/home/components/HomePage.tsx`
- `devnotes/v2.0.0/sprint-trackers/asr-legacy-profile-cleanup-sprint-tracker.md`

Commit:

- Pending.

## ASR-LEGACY-CLEANUP-Sprint 2: Backend/API Cleanup

Status: completed

Planned tasks:

- [x] Remove legacy flat fields from profile request DTOs.
- [x] Remove `legacyProfileOptionField`.
- [x] Keep validation focused on `options.pipeline`.
- [x] Update API tests.
- [x] Ensure profile service tests cover invalid/missing pipeline.

Acceptance checks:

- [x] API profile create/update structs contain no legacy flat ASR fields.
- [x] Legacy field rejection does not require active legacy DTO fields.
- [x] Pipeline validation remains descriptor-backed.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestProfile'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/profile ./internal/api`
- [x] `git diff --check -- internal/api/types.go internal/api/profile_handlers.go internal/api/profile_settings_test.go devnotes/v2.0.0/sprint-trackers/asr-legacy-profile-cleanup-sprint-tracker.md`
- [x] `rg 'legacyProfileOptionField|tail_paddings|decoding_method|chunking_strategy|num_speakers|diarization_threshold|min_duration_on|min_duration_off' internal/api/types.go internal/api/profile_handlers.go internal/api/profile_settings_test.go -n`

Artifacts:

- `internal/api/types.go`
- `internal/api/profile_handlers.go`
- `internal/api/profile_settings_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-legacy-profile-cleanup-sprint-tracker.md`

Commit:

- Pending.

## ASR-LEGACY-CLEANUP-Sprint 3: Database And Migration Cleanup

Status: completed

Planned tasks:

- [x] Delete obsolete legacy ASR profile migration structs/code.
- [x] Update database tests away from legacy flat ASR profile expectations.
- [x] Keep current target schema creation intact.
- [x] Keep non-ASR migration behavior intact.
- [x] Document any retained denormalized columns as pipeline-derived.

Acceptance checks:

- [x] Database migration code no longer preserves removed flat ASR profile fields.
- [x] Tests reflect pipeline-only ASR profiles.
- [x] No unrelated schema behavior changes.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/database`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/database`
- [x] `rg -n "legacyTranscriptionProfile|migrateProfiles|legacy_transcription_profiles|DefaultProfileID:\s*legacyUser\.DefaultProfileID" internal/database -g '*.go'`
- [x] `git diff --check -- internal/database/legacy.go internal/database/database_test.go devnotes/v2.0.0/sprint-trackers/asr-legacy-profile-cleanup-sprint-tracker.md`

Artifacts:

- `internal/database/legacy.go`
- `internal/database/database_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-legacy-profile-cleanup-sprint-tracker.md`

Commit:

- Pending.

## ASR-LEGACY-CLEANUP-Sprint 4: Guardrails And Final Search

Status: completed

Planned tasks:

- [x] Add or update guardrails for removed field names where practical.
- [x] Run `rg` for removed legacy fields in active code.
- [x] Allow mentions only in devnotes, absence/rejection tests, or changelog-style docs.
- [x] Run validation baseline.

Acceptance checks:

- [x] Active code has no legacy flat ASR profile compatibility.
- [x] Guardrails catch obvious reintroduction.

Verification:

- [x] `rg -n "tail_paddings|decoding_method|chunking_strategy|num_speakers|diarization_threshold|min_duration_on|min_duration_off|defaultProfileParams|normalizeParams|familyForModel|legacyProfileOptionField|legacyTranscriptionProfile|migrateProfiles" internal web/frontend/src -g '*.go' -g '*.ts' -g '*.tsx'` (remaining matches are descriptor-backed parameter keys or rejection/contract tests, not flat profile compatibility)
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/profile ./internal/api ./internal/database ./internal/repository ./internal/models`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api ./internal/database ./internal/repository ./internal/models` (rerun with sandbox escalation because `httptest.NewServer` needs localhost bind)
- [x] `npm --prefix web/frontend run build`
- [x] `npm --prefix web/frontend run lint` (passes with existing warnings outside this sprint)
- [x] `git diff --check -- devnotes/v2.0.0/sprint-trackers/asr-legacy-profile-cleanup-sprint-tracker.md`

Artifacts:

- `devnotes/v2.0.0/sprint-trackers/asr-legacy-profile-cleanup-sprint-tracker.md`

Commit:

- Pending.
