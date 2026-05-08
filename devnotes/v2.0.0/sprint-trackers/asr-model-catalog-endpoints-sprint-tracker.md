# Sprint Run Tracker: ASR Model Catalog Endpoints

Run ID: `ASR-MODEL-CATALOG`

Status: completed through ASR-MODEL-CATALOG-Sprint 1.

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

Status: completed

Planned tasks:

- [x] Add route contract tests for the new model-card endpoint.
- [x] Test no filter returns all ASR model cards.
- [x] Test `capability=transcription` returns `parakeet-v2` and `parakeet-v3`.
- [x] Test `capability=diarization` returns `diarization-default`.
- [x] Test comma-separated filters.
- [x] Test invalid capability validation.

Acceptance checks:

- [x] API contract is explicit before implementation.
- [x] Route shape supports profile pipeline editing.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestASRModelCatalogEndpointFiltersCapabilities|TestRouteContract|TestTranscriptExecutionsLogsModelsAndStatsUseEngineServices'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/transcription/engineprovider ./internal/transcription/asrcontract`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/transcription/engineprovider ./internal/transcription/asrcontract`
- [x] `git diff --check -- internal/api/admin_handlers.go internal/api/router.go internal/api/route_contract_test.go internal/api/engine_worker_api_test.go devnotes/v2.0.0/sprint-trackers/asr-model-catalog-endpoints-sprint-tracker.md`

Artifacts:

- `internal/api/engine_worker_api_test.go`
- `internal/api/route_contract_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-model-catalog-endpoints-sprint-tracker.md`

Commit:

- Pending.

## ASR-MODEL-CATALOG-Sprint 1: Endpoint Implementation

Status: completed

Planned tasks:

- [x] Register the new route under `/api/v1/models`.
- [x] Add capability query parsing.
- [x] Reuse registry model-card retrieval.
- [x] Reuse `sanitizeModelCards`.
- [x] Keep `/api/v1/models/transcription` until frontend callers migrate to `/api/v1/models?capability=transcription`.

Acceptance checks:

- [x] Transcription and diarization model cards are available to the frontend.
- [x] `diarization-default` includes `parameter_schema`.
- [x] Parakeet TDT v2/v3 remain visible through the new endpoint.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestASRModelCatalogEndpointFiltersCapabilities|TestRouteContract|TestTranscriptExecutionsLogsModelsAndStatsUseEngineServices'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/transcription/engineprovider ./internal/transcription/asrcontract`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/transcription/engineprovider ./internal/transcription/asrcontract`
- [x] `git diff --check -- internal/api/admin_handlers.go internal/api/router.go internal/api/route_contract_test.go internal/api/engine_worker_api_test.go devnotes/v2.0.0/sprint-trackers/asr-model-catalog-endpoints-sprint-tracker.md`

Artifacts:

- `internal/api/admin_handlers.go`
- `internal/api/router.go`
- `internal/api/engine_worker_api_test.go`
- `internal/api/route_contract_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-model-catalog-endpoints-sprint-tracker.md`

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
