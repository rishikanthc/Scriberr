# Sprint Run Tracker: ASR Model Catalog Endpoints

Run ID: `ASR-MODEL-CATALOG`

Status: not started.

This tracker belongs to `devnotes/v2.0.0/sprint-plans/asr-model-catalog-endpoints-sprint-plan.md`.

## Run Rules

- Keep model cards descriptor-backed.
- Do not synthesize local sherpa schemas in API code.
- Keep API responses sanitized.
- Update this tracker in the same change set as each completed sprint.
- Run `git diff --check` before closing every sprint.

## Validation Checklist

- [ ] Focused API tests for changed routes.
- [ ] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/transcription/engineprovider ./internal/transcription/asrcontract`.
- [ ] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/transcription/engineprovider ./internal/transcription/asrcontract`.
- [ ] `git diff --check`.

## ASR-MODEL-CATALOG-Sprint 0: Route Contract And Capability Matrix

Status: pending

Planned tasks:

- [ ] Add route contract tests for the new model-card endpoint.
- [ ] Test no filter returns all ASR model cards.
- [ ] Test `capability=transcription` returns `parakeet-v2` and `parakeet-v3`.
- [ ] Test `capability=diarization` returns `diarization-default`.
- [ ] Test comma-separated filters.
- [ ] Test invalid capability validation.

Acceptance checks:

- [ ] API contract is explicit before implementation.
- [ ] Route shape supports profile pipeline editing.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-MODEL-CATALOG-Sprint 1: Endpoint Implementation

Status: pending

Planned tasks:

- [ ] Register the new route under `/api/v1/models`.
- [ ] Add capability query parsing.
- [ ] Reuse registry model-card retrieval.
- [ ] Reuse `sanitizeModelCards`.
- [ ] Decide whether `/api/v1/models/transcription` remains or is removed.

Acceptance checks:

- [ ] Transcription and diarization model cards are available to the frontend.
- [ ] `diarization-default` includes `parameter_schema`.
- [ ] Parakeet TDT v2/v3 remain visible through the new endpoint.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-MODEL-CATALOG-Sprint 2: Frontend API Hook Prep

Status: pending

Planned tasks:

- [ ] Add typed frontend API helper for the new endpoint.
- [ ] Add TanStack Query hook keyed by capability filter.
- [ ] Keep helper generic for future ASR tasks.
- [ ] Avoid profile dialog rewrites in this run.

Acceptance checks:

- [ ] Frontend can fetch transcription and diarization model cards by capability.
- [ ] No legacy profile UI behavior changes in this sprint.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.
