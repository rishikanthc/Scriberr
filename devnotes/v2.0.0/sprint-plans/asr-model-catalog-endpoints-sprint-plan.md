# Sprint Run: ASR Model Catalog Endpoints

Run ID: `ASR-MODEL-CATALOG`

Status: planning only. Do not implement code from this document until the user starts an implementation sprint.

Date: 2026-05-08

## Goal

Expose provider model cards through a clean frontend-accessible API that can return transcription, diarization, and future ASR task models by capability. This unblocks descriptor-driven pipeline profile editing without overloading `/api/v1/models/transcription`.

## Current State

- `GET /api/v1/models/transcription` filters model cards to `capabilities.transcription`.
- The local engine already has a `diarization-default` descriptor with a parameter schema.
- The frontend profile dialog needs diarization model cards to render diarization parameters without hard-coded fields.
- Returning diarization cards from `/api/v1/models/transcription` would make that endpoint misleading.

## Target Direction

- Add a general model-card endpoint that supports capability filtering.
- Prefer one canonical endpoint over compatibility aliases.
- Keep model cards sanitized through the same `sanitizeModelCards` path.
- Keep provider/model selection backed by `engineprovider.Registry`.

Proposed route:

```txt
GET /api/v1/models?capability=transcription
GET /api/v1/models?capability=diarization
GET /api/v1/models?capability=transcription,diarization
GET /api/v1/models
```

## Engineering Rules

- Do not import `scriberr-engine` from API handlers.
- Do not synthesize local model metadata in the backend.
- Do not add REST between Scriberr and the local sherpa engine.
- Keep handlers thin: parse query, call registry/service, map response.
- Validate capability filters against `asrcontract.Capability`.
- Return sanitized model cards only.

## Validation Baseline

```sh
GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/transcription/engineprovider ./internal/transcription/asrcontract
GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/transcription/engineprovider ./internal/transcription/asrcontract
git diff --check
```

## ASR-MODEL-CATALOG-Sprint 0: Route Contract And Capability Matrix

Goal: define and test the public model-card API shape.

Tasks:

- Add route contract tests for the new model-card endpoint.
- Test no filter returns all ASR model cards.
- Test `capability=transcription` returns transcription models including `parakeet-v2` and `parakeet-v3`.
- Test `capability=diarization` returns `diarization-default`.
- Test comma-separated filters return models supporting any requested capability.
- Test invalid capability returns a validation error.

Acceptance criteria:

- API contract is explicit before implementation.
- The route shape supports profile pipeline editing without task-specific endpoints.

## ASR-MODEL-CATALOG-Sprint 1: Endpoint Implementation

Goal: implement the capability-filtered model-card endpoint.

Tasks:

- Register the new route under `/api/v1/models`.
- Add query parsing for capability filters.
- Reuse registry model-card retrieval.
- Reuse `sanitizeModelCards`.
- Keep `/api/v1/models/transcription` only if current route contract tests still require it; otherwise migrate callers and remove it in cleanup.

Acceptance criteria:

- Transcription and diarization model cards are available to the frontend.
- `diarization-default` includes `parameter_schema`.
- Parakeet TDT v2/v3 remain visible through the new endpoint.

## ASR-MODEL-CATALOG-Sprint 2: Frontend API Hook Prep

Goal: prepare frontend API access for the later profile-dialog sprint without implementing the dialog.

Tasks:

- Add typed frontend API helper for the new model-card endpoint.
- Add TanStack Query hook keyed by capability filter.
- Keep this helper generic enough for transcription, diarization, and future ASR tasks.
- Do not rewrite `ASRProfileDialog` in this run.

Acceptance criteria:

- Frontend can fetch transcription and diarization model cards by capability.
- No legacy profile UI behavior changes in this sprint.

## Commit Plan

1. Route contract tests.
2. Backend endpoint implementation.
3. Frontend API helper and hook prep.
