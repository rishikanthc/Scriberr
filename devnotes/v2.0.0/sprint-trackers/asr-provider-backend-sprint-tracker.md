# Sprint Run Tracker: ASR Provider Backend Architecture

Run ID: `ASRP`

Status: completed through ASRP-Sprint 5.

This tracker belongs to `devnotes/v2.0.0/sprint-plans/asr-provider-backend-sprint-plan.md` and the design spec in `devnotes/v2.0.0/specs/asr-provider-backend-architecture.md`.

## Run Rules

- Follow `devnotes/v2.0.0/rules/backend-rules.md`.
- Do not edit `references/engine` during this sprint run.
- Use fake providers, fake remote servers, and placeholders where needed.
- Keep local sherpa in-process through Go APIs.
- Keep external providers behind the REST adapter.
- Write tests or architecture guards before implementation.
- Update this tracker in the same change set as each completed sprint.
- Run `git diff --check` before closing every sprint.
- Document any skipped validation and the reason.
- Leave unrelated dirty worktree changes untouched and documented.

## Validation Checklist

Required before closing each implementation sprint when practical:

- [ ] Focused package tests for the sprint.
- [ ] Broad backend test baseline from the sprint plan, or documented blocker.
- [ ] `go vet` for touched backend packages, or documented blocker.
- [ ] `git diff --check`.
- [ ] Architecture boundary check for `scriberr-engine` imports.
- [ ] Path/secret leakage review for API responses, logs, events, and execution records touched by the sprint.

## ASRP-Sprint 0: Inventory, Guardrails, And Compatibility Map

Status: completed

Completed tasks:

- [x] Inventoried `scriberr-engine` imports and current ASR coupling.
- [x] Inventoried `ModelCapability`, profile validation, orchestrator selection, execution metadata, and audio path handling.
- [x] Added architecture tests for ASR dependency guardrails.
- [x] Documented legacy `WhisperXParams` usage and removal target.
- [x] Created route/API impact matrix for models, profiles, transcriptions, events, logs, and executions.

Acceptance checks:

- [x] Current ASR coupling is documented.
- [x] Guard tests fail on newly introduced forbidden imports.
- [x] No runtime behavior changes.
- [x] No runtime behavior changes.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestASREngineImportInventory|TestProfileServiceDoesNotImportSherpaEngine|TestASRProvidersDoNotDependOnAPIOrRepositories|TestBackendDependencyDirection'`
- [x] `git diff --check -- internal/api/architecture_test.go devnotes/v2.0.0/status-updates/asr-provider-backend-sprint-00-inventory.md devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Artifacts:

- `internal/api/architecture_test.go`
- `devnotes/v2.0.0/status-updates/asr-provider-backend-sprint-00-inventory.md`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 1: Pure ASR Contract Types

Status: completed

Completed tasks:

- [x] Added `internal/transcription/asrcontract`.
- [x] Defined provider info, model cards, capabilities, status, loaded models, progress, requests, results, and typed errors.
- [x] Added capability matching helpers.
- [x] Added provider error classification helpers.
- [x] Added JSON compatibility tests.

Acceptance checks:

- [x] Contract package imports only standard library packages.
- [x] Typed capabilities are available for new code paths.
- [x] Provider errors include code, sanitized message, retryable flag, and bounded details.
- [x] No runtime provider behavior changes.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/asrcontract`
- [x] `go list -f '{{join .Imports "\n"}}' ./internal/transcription/asrcontract`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/transcription/asrcontract`
- [x] `git diff --check -- internal/transcription/asrcontract devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Artifacts:

- `internal/transcription/asrcontract/types.go`
- `internal/transcription/asrcontract/types_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 2: Provider Interface And Registry V2

Status: completed

Completed tasks:

- [x] Updated `engineprovider` to expose `asrcontract` model cards and status types.
- [x] Added provider interface methods for inspect, models, status, model lifecycle, operations, and close.
- [x] Added `ProgressSink`.
- [x] Kept deterministic provider/model selection and added busy/unhealthy fallback exclusion.
- [x] Preserved current local provider behavior and legacy `Capabilities()` API.
- [x] Kept unsupported local speaker identification as a typed `UNSUPPORTED_OPERATION` response.

Acceptance checks:

- [x] Registry represents local and future remote providers through one interface.
- [x] Selection is deterministic and test-backed.
- [x] Existing default provider/model behavior remains stable.
- [x] A second fake provider requires no API, repository, or worker changes.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestListTranscriptionModels|TestCreateSubmitRetryUseQueueService|TestASREngineImportInventory|TestBackendDependencyDirection'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/transcription/... ./internal/api`
- [x] `git diff --check -- internal/transcription/engineprovider internal/transcription/orchestrator/processor_test.go internal/api/engine_worker_api_test.go devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Notes:

- A broader `go test ./internal/api -run 'Test.*Models|...'` was blocked by the existing `TestLLMProviderSettingsSavesSelectedModels` sandbox loopback failure from `httptest.NewServer`; the focused ASR/API subset passed.

Artifacts:

- `internal/transcription/engineprovider/types.go`
- `internal/transcription/engineprovider/registry.go`
- `internal/transcription/engineprovider/local_provider.go`
- `internal/transcription/engineprovider/local_provider_test.go`
- `internal/transcription/engineprovider/registry_test.go`
- `internal/transcription/orchestrator/processor_test.go`
- `internal/api/engine_worker_api_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 3: Model Catalog Service And Profile Validation

Status: completed

Completed tasks:

- [x] Added model catalog seam backed by provider registry model cards.
- [x] Moved profile model normalization and invalid model rejection into the profile service.
- [x] Removed `scriberr-engine` imports from profile handlers.
- [x] Kept existing profile JSON shape initially.
- [x] Kept `/api/v1/models/transcription` on the registry path while profile validation now uses the model catalog seam.

Acceptance checks:

- [x] API profile handlers do not import `scriberr-engine`.
- [x] Profile validation is service-owned and test-backed.
- [x] Existing profile create/update/list/get response shape remains stable.
- [x] Model listing remains registry-backed.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestProfileValidationAndAuth|TestASREngineImportInventory|TestProfileServiceDoesNotImportSherpaEngine|TestListTranscriptionModels'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestProfile|TestSettingsDefaultProfile|TestListTranscriptionModels|TestASREngineImportInventory|TestBackendDependencyDirection'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api -run 'TestService|TestProfile|TestListTranscriptionModels|TestASREngineImportInventory'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/app ./internal/profile ./internal/api -run 'TestProfile|TestListTranscriptionModels|TestASREngineImportInventory|TestBuild|TestService'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/profile ./internal/api ./internal/app`
- [x] `rg -n "scriberr-engine" internal/api internal/profile --glob '*.go'` returns only architecture test guard text.

Artifacts:

- `internal/profile/service.go`
- `internal/profile/model_catalog.go`
- `internal/profile/service_test.go`
- `internal/api/profile_handlers.go`
- `internal/api/architecture_test.go`
- `internal/app/app.go`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 4: Provider Progress And Execution Metadata

Status: completed

Completed tasks:

- [x] Added orchestrator `ProgressSink` implementation.
- [x] Mapped provider stages to durable progress and SSE events.
- [x] Stored provider step metadata in execution config JSON without audio paths.
- [x] Preserved provider error codes through typed provider error messages.
- [x] Preserved existing execution list response shape.

Acceptance checks:

- [x] Fake/local providers can emit progress events.
- [x] Progress updates persist through existing repository methods.
- [x] SSE payloads remain small and path-free.
- [x] Logs and executions expose sanitized provider details only.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/orchestrator`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestSSEReceivesTranscriptionEvents|TestEvents|TestTranscriptionExecutions|TestASREngineImportInventory|TestBackendDependencyDirection'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/transcription/... ./internal/api`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/... ./internal/api -run 'TestProcessor|TestLocalProvider|TestSSEReceivesTranscriptionEvents|TestTranscriptionExecutions|TestASREngineImportInventory|TestBackendDependencyDirection'`
- [x] `git diff --check -- internal/transcription/orchestrator internal/transcription/engineprovider/types.go devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Artifacts:

- `internal/transcription/engineprovider/types.go`
- `internal/transcription/orchestrator/processor.go`
- `internal/transcription/orchestrator/processor_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 5: Audio Preprocessing Boundary

Status: completed

Completed tasks:

- [x] Added `internal/transcription/preprocess`.
- [x] Produce provider-ready 16 kHz mono WAV artifacts through `ffmpeg`.
- [x] Added config for normalized audio dir and provider mount root.
- [x] Cache normalized artifacts by source hash/job id where safe.
- [x] Updated orchestrator to use preprocessed audio.
- [x] Kept original audio API behavior unchanged.

Acceptance checks:

- [x] Providers receive provider-visible normalized paths.
- [x] Public APIs never expose normalized artifact paths.
- [x] Existing authorized audio streaming still serves original audio.
- [x] Preprocessing failures are sanitized.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/preprocess ./internal/config ./internal/transcription/orchestrator`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/... ./internal/config ./internal/app`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/transcription/... ./internal/config ./internal/app`
- [x] `git diff --check -- internal/transcription/preprocess internal/transcription/orchestrator internal/config internal/app devnotes/v2.0.0/sprint-plans/asr-provider-backend-sprint-plan.md devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Artifacts:

- `internal/transcription/preprocess/preprocess.go`
- `internal/transcription/preprocess/preprocess_test.go`
- `internal/transcription/orchestrator/processor.go`
- `internal/transcription/orchestrator/processor_test.go`
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/app/app.go`
- `devnotes/v2.0.0/sprint-plans/asr-provider-backend-sprint-plan.md`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 6: Legacy ASR Parameter Removal And Backend Streamlining

Status: pending

Planned tasks:

- [ ] Replace `models.WhisperXParams` with provider-neutral ASR profile/job option types.
- [ ] Remove old WhisperX naming from services, handlers, repository payload helpers, tests, and docs.
- [ ] Replace single-model profile assumptions with explicit one-step pipeline data.
- [ ] Update create/update/list/get profile API tests to use the new pipeline contract.
- [ ] Update transcription creation and recording finalizer paths to pass pipeline/profile options, not WhisperX params.
- [ ] Add architecture guards that fail if production code references `WhisperXParams` or `WhisperX`.
- [ ] Remove dead compatibility helpers created only for the earlier migration path.

Acceptance checks:

- [ ] No production code references `WhisperXParams`.
- [ ] New ASR option types are provider-neutral and pipeline-oriented.
- [ ] Existing backend features compile against the new types.
- [ ] Old compatibility helpers are deleted, not left as wrappers.

Verification:

- [ ] Not run yet.

Artifacts:

- To be filled during implementation.

Commit:

- Pending.

## ASRP-Sprint 7: Remote Provider REST Client

Status: pending

Planned tasks:

- [ ] Add `internal/transcription/engineprovider/remote`.
- [ ] Implement control-plane REST endpoints.
- [ ] Implement ephemeral job REST endpoints.
- [ ] Enforce timeouts and response size limits.
- [ ] Poll remote job status and replay progress.
- [ ] Map typed remote errors.
- [ ] Propagate cancellation through `DELETE /v1/jobs/{job_id}`.

Acceptance checks:

- [ ] Remote client implements the internal provider interface.
- [ ] Fake HTTP provider tests cover success, progress, busy, unsupported operation, provider error, malformed JSON, timeout, and cancellation.
- [ ] Remote provider URLs come only from injected config.
- [ ] No URL/path/token leaks through public surfaces.

Verification:

- [ ] Not run yet.

Artifacts:

- To be filled during implementation.

Commit:

- Pending.

## ASRP-Sprint 8: Remote Provider Configuration And App Wiring

Status: pending

Planned tasks:

- [ ] Extend `internal/config` with ASR provider configuration.
- [ ] Validate remote provider specs, duplicates, defaults, durations, and mount values.
- [ ] Build local provider directly in-process.
- [ ] Build remote provider clients from config.
- [ ] Register providers before worker startup.
- [ ] Keep local-only deployment behavior intentional after legacy ASR removal.

Acceptance checks:

- [ ] Local-only deployments use the new provider pipeline contract.
- [ ] Invalid config fails startup with actionable errors.
- [ ] Remote provider wiring is testable without starting the HTTP listener.
- [ ] `internal/app` remains the only composition root.

Verification:

- [ ] Not run yet.

Artifacts:

- To be filled during implementation.

Commit:

- Pending.

## ASRP-Sprint 9: Pipeline Execution And Provider Chaining

Status: pending

Planned tasks:

- [ ] Add internal pipeline representation.
- [ ] Replace old single-model execution with an internal provider pipeline.
- [ ] Resolve each step through the registry.
- [ ] Execute steps serially.
- [ ] Merge typed artifacts into canonical transcript JSON.
- [ ] Cover local/fake transcription plus remote/fake diarization.

Acceptance checks:

- [ ] Jobs run through the new pipeline representation.
- [ ] Diarization can run on a different provider than transcription.
- [ ] Provider step failures stop the pipeline with sanitized error metadata.
- [ ] Cancellation interrupts the active provider step.
- [ ] Canonical transcript output remains stable for transcript API/UI consumers.

Verification:

- [ ] Not run yet.

Artifacts:

- To be filled during implementation.

Commit:

- Pending.

## ASRP-Sprint 10: Profile Pipeline Persistence

Status: pending

Planned tasks:

- [ ] Store ordered pipeline steps in profile JSON.
- [ ] Validate provider-specific options against model-card schemas where supported.
- [ ] Bound and sanitize provider-specific option data.

Acceptance checks:

- [ ] New profiles can persist multiple provider steps.
- [ ] List/get profile responses expose the new pipeline contract.
- [ ] Invalid pipeline shape is rejected before enqueue.

Verification:

- [ ] Not run yet.

Artifacts:

- To be filled during implementation.

Commit:

- Pending.

## ASRP-Sprint 11: Provider Admin/Diagnostics API

Status: pending

Planned tasks:

- [ ] Add service methods for provider list/status/model load/unload.
- [ ] Add authenticated/admin-gated diagnostics endpoints as appropriate.
- [ ] Keep `/api/v1/models/transcription` user-readable.
- [ ] Use bounded timeouts for load/unload commands.
- [ ] Sanitize provider status messages.

Acceptance checks:

- [ ] Users can still list selectable transcription models.
- [ ] Admin diagnostics show provider state, active operation, and loaded models without paths/secrets.
- [ ] Load/unload failures return typed safe errors.
- [ ] Route contract and security regression tests cover new endpoints.

Verification:

- [ ] Not run yet.

Artifacts:

- To be filled during implementation.

Commit:

- Pending.

## ASRP-Sprint 12: Contract Tests, Example Provider, And Hardening

Status: pending

Planned tasks:

- [ ] Add provider contract test helpers.
- [ ] Add minimal fake/example provider server for development tests.
- [ ] Add provider author docs.
- [ ] Tighten architecture tests where feasible.
- [ ] Review performance hot paths.
- [ ] Run broad validation baseline.

Acceptance checks:

- [ ] Third-party provider implementers have a testable contract.
- [ ] Architecture guards prevent API/model/provider coupling regressions.
- [ ] Remote provider failures do not destabilize local-only transcription.
- [ ] Existing frontend and backend features remain green in focused tests.

Verification:

- [ ] Not run yet.

Artifacts:

- To be filled during implementation.

Commit:

- Pending.

## Deferred Work

- Rewriting `references/engine`.
- Moving local sherpa into a sidecar container.
- gRPC or WebSocket provider transports.
- Live streaming transcription.
- Multi-job concurrent execution inside a single provider.
- Automatic Docker service discovery.
- Cloud/object-storage audio handoff.
