# Sprint Run: ASR Legacy Profile Cleanup

Run ID: `ASR-LEGACY-CLEANUP`

Status: planning only. Do not implement code from this document until the user starts an implementation sprint.

Date: 2026-05-08

## Goal

Remove legacy flat ASR profile compatibility and migration code so the profile contract is simpler: ASR profile settings are stored, validated, returned, and saved only as `options.pipeline` with descriptor-keyed step options.

## Current State

- Frontend profile types still include flat fields such as `language`, `task`, `threads`, `tail_paddings`, `chunking_strategy`, and diarization tuning.
- Frontend save code still maps flat fields into an old options payload.
- Backend profile handlers reject legacy flat top-level options on create/update.
- Database and repository layers still contain legacy migration/compatibility code from older table shapes.
- The desired direction is no backward compatibility for legacy ASR profile fields.

## Target Direction

- Canonical profile request shape is:

```json
{
  "name": "Profile name",
  "description": "",
  "is_default": false,
  "options": {
    "pipeline": [
      {
        "kind": "transcription",
        "provider": "local",
        "model": "parakeet-v3",
        "options": {
          "chunking.mode": "fixed"
        }
      }
    ]
  }
}
```

- No legacy flat profile fields are accepted, normalized, migrated, or reconstructed.
- Existing unsupported legacy rows are not silently repaired.
- Tests should enforce rejection/removal rather than compatibility behavior.

## Engineering Rules

- Delete compatibility code instead of wrapping it.
- Keep `ASRParams` pipeline-only.
- Keep profile validation descriptor-backed.
- Do not add frontend migration UX for old flat fields.
- Do not preserve old database migration code unless it is needed for unrelated non-ASR schema upgrade paths.
- Keep cleanup scoped to ASR profile/provider compatibility; avoid broad database refactors.

## Validation Baseline

```sh
GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api ./internal/database ./internal/repository ./internal/models
GOCACHE=/tmp/scriberr-go-cache go vet ./internal/profile ./internal/api ./internal/database ./internal/repository ./internal/models
npm --prefix web/frontend run build
git diff --check
```

## ASR-LEGACY-CLEANUP-Sprint 0: Inventory And Deletion Map

Goal: identify every legacy flat ASR profile path before deleting code.

Tasks:

- Inventory frontend legacy flat profile fields and normalization.
- Inventory backend request structs and validation guards for legacy flat fields.
- Inventory database legacy profile migration code and tests.
- Decide which compatibility columns remain as denormalized indexes and which are obsolete.
- Produce a deletion map in this tracker before code cleanup.

Acceptance criteria:

- Every legacy flat ASR profile path has a delete/keep decision.
- No runtime behavior changes.

## ASR-LEGACY-CLEANUP-Sprint 1: Frontend Type And Payload Cleanup

Goal: remove frontend legacy ASR profile fields and payload mapping.

Tasks:

- Change frontend profile options to pipeline-only.
- Delete `defaultProfileParams`, `familyForModel`, and flat-field `normalizeParams` behavior if no longer needed.
- Remove save payload mapping for legacy fields.
- Update affected settings code to compile against pipeline-only types.
- Do not implement the full dynamic profile dialog in this cleanup sprint unless needed to keep compilation green; leave UI revamp to `ASR-PROFILE-FE`.

Acceptance criteria:

- Frontend cannot emit legacy flat ASR profile fields.
- Active TypeScript types are pipeline-only.
- Build passes or remaining compile blockers are explicitly handed to the frontend sprint.

## ASR-LEGACY-CLEANUP-Sprint 2: Backend/API Cleanup

Goal: remove backend request compatibility for legacy flat ASR profile fields.

Tasks:

- Remove legacy flat fields from profile request DTOs.
- Remove `legacyProfileOptionField` if no longer needed.
- Keep validation focused on `options.pipeline`.
- Update API tests to assert pipeline-only behavior.
- Ensure profile service tests cover invalid/missing pipeline.

Acceptance criteria:

- API profile create/update structs contain no legacy flat ASR fields.
- Legacy field rejection does not require carrying legacy field definitions in active DTOs.
- Pipeline validation remains descriptor-backed.

## ASR-LEGACY-CLEANUP-Sprint 3: Database And Migration Cleanup

Goal: remove obsolete legacy ASR profile migration code and tests.

Tasks:

- Delete obsolete legacy ASR profile migration structs/code that only exist to migrate removed flat ASR parameters.
- Update database tests away from legacy flat ASR profile expectations.
- Keep current target schema creation intact.
- Keep non-ASR migration behavior intact.
- If denormalized profile columns are still useful for listing/filtering, keep them and document that they are derived from pipeline, not compatibility inputs.

Acceptance criteria:

- Database migration code no longer preserves removed flat ASR profile fields.
- Tests reflect pipeline-only ASR profiles.
- No unrelated schema behavior changes.

## ASR-LEGACY-CLEANUP-Sprint 4: Guardrails And Final Search

Goal: prevent legacy flat ASR profile fields from returning.

Tasks:

- Add or update architecture/search tests for removed field names where practical.
- Run `rg` for removed legacy fields in active frontend/backend code.
- Allow mentions only in devnotes, tests that assert absence/rejection, or changelog-style docs.
- Run validation baseline.

Acceptance criteria:

- Active code has no legacy flat ASR profile compatibility.
- Guardrails catch obvious reintroduction.

## Commit Plan

1. Inventory/deletion map.
2. Frontend type/payload cleanup.
3. Backend/API cleanup.
4. Database migration cleanup.
5. Guardrails and final validation.
