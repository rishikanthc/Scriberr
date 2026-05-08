# Sprint Run Tracker: ASR Legacy Profile Cleanup

Run ID: `ASR-LEGACY-CLEANUP`

Status: not started.

This tracker belongs to `devnotes/v2.0.0/sprint-plans/asr-legacy-profile-cleanup-sprint-plan.md`.

## Run Rules

- Delete legacy ASR profile compatibility instead of preserving it.
- Keep ASR profiles pipeline-only.
- Keep cleanup scoped.
- Update this tracker in the same change set as each completed sprint.
- Run `git diff --check` before closing every sprint.

## Validation Checklist

- [ ] Focused tests for touched frontend/backend/database packages.
- [ ] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api ./internal/database ./internal/repository ./internal/models`.
- [ ] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/profile ./internal/api ./internal/database ./internal/repository ./internal/models`.
- [ ] `npm --prefix web/frontend run build` for frontend changes.
- [ ] `git diff --check`.
- [ ] Final `rg` check for removed legacy flat ASR fields.

## ASR-LEGACY-CLEANUP-Sprint 0: Inventory And Deletion Map

Status: pending

Planned tasks:

- [ ] Inventory frontend legacy flat profile fields and normalization.
- [ ] Inventory backend request structs and validation guards.
- [ ] Inventory database legacy profile migration code and tests.
- [ ] Decide which denormalized columns remain.
- [ ] Produce deletion map in this tracker.

Acceptance checks:

- [ ] Every legacy flat ASR profile path has a delete/keep decision.
- [ ] No runtime behavior changes.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-LEGACY-CLEANUP-Sprint 1: Frontend Type And Payload Cleanup

Status: pending

Planned tasks:

- [ ] Change frontend profile options to pipeline-only.
- [ ] Delete flat-field default/normalization helpers.
- [ ] Remove save payload mapping for legacy fields.
- [ ] Update settings code to compile against pipeline-only types.
- [ ] Leave full dialog revamp to `ASR-PROFILE-FE`.

Acceptance checks:

- [ ] Frontend cannot emit legacy flat ASR profile fields.
- [ ] Active TypeScript types are pipeline-only.
- [ ] Build passes or compile blockers are explicitly handed to the frontend sprint.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-LEGACY-CLEANUP-Sprint 2: Backend/API Cleanup

Status: pending

Planned tasks:

- [ ] Remove legacy flat fields from profile request DTOs.
- [ ] Remove `legacyProfileOptionField` if no longer needed.
- [ ] Keep validation focused on `options.pipeline`.
- [ ] Update API tests.
- [ ] Ensure profile service tests cover invalid/missing pipeline.

Acceptance checks:

- [ ] API profile create/update structs contain no legacy flat ASR fields.
- [ ] Legacy field rejection does not require active legacy DTO fields.
- [ ] Pipeline validation remains descriptor-backed.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-LEGACY-CLEANUP-Sprint 3: Database And Migration Cleanup

Status: pending

Planned tasks:

- [ ] Delete obsolete legacy ASR profile migration structs/code.
- [ ] Update database tests away from legacy flat ASR profile expectations.
- [ ] Keep current target schema creation intact.
- [ ] Keep non-ASR migration behavior intact.
- [ ] Document any retained denormalized columns as pipeline-derived.

Acceptance checks:

- [ ] Database migration code no longer preserves removed flat ASR profile fields.
- [ ] Tests reflect pipeline-only ASR profiles.
- [ ] No unrelated schema behavior changes.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-LEGACY-CLEANUP-Sprint 4: Guardrails And Final Search

Status: pending

Planned tasks:

- [ ] Add or update guardrails for removed field names where practical.
- [ ] Run `rg` for removed legacy fields in active code.
- [ ] Allow mentions only in devnotes, absence/rejection tests, or changelog-style docs.
- [ ] Run validation baseline.

Acceptance checks:

- [ ] Active code has no legacy flat ASR profile compatibility.
- [ ] Guardrails catch obvious reintroduction.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.
