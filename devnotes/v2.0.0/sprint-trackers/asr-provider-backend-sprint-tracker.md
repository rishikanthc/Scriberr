# Sprint Run Tracker: ASR Provider Backend Architecture

Run ID: `ASRP`

Status: completed through ASRP-Sprint 13.

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
- [x] Documented legacy provider-specific ASR parameter usage and removal target.
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

Status: completed

Completed tasks:

- [x] Replaced provider-specific ASR parameter structs with provider-neutral ASR profile/job option types.
- [x] Removed old provider-specific naming from services, handlers, repository payload helpers, tests, and docs.
- [x] Replaced single-model profile assumptions with explicit one-step pipeline data.
- [x] Updated create/update/list/get profile API tests to use the new pipeline-bearing contract.
- [x] Updated transcription creation and recording finalizer paths to pass ASR profile options.
- [x] Added architecture guards that fail if production code references old provider-specific ASR identifiers.
- [x] Removed dead option fields from the shared ASR parameter surface.

Acceptance checks:

- [x] No production code references old provider-specific ASR parameter identifiers.
- [x] New ASR option types are provider-neutral and pipeline-oriented.
- [x] Existing backend features compile against the new types.
- [x] Old compatibility helpers are deleted, not left as wrappers.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/models ./internal/profile ./internal/transcription/... ./internal/recording ./internal/database`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestASREngineImportInventory|TestProductionCodeDoesNotUseOldASRParameterIdentifiers|TestProfileServiceDoesNotImportSherpaEngine|TestBackendDependencyDirection|TestProfile|TestListTranscriptionModels|TestTranscriptionExecutions'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api -run 'TestService|TestProfile|TestSettingsDefaultProfile|TestASREngineImportInventory|TestProductionCodeDoesNotUseOldASRParameterIdentifiers|TestProfileServiceDoesNotImportSherpaEngine|TestBackendDependencyDirection|TestListTranscriptionModels|TestTranscriptionExecutions'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/models ./internal/profile ./internal/transcription/... ./internal/recording ./internal/database ./internal/api`
- [x] `git diff --check`
- [x] Broad `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ...` was blocked by the existing sandbox loopback restriction in `TestLLMProviderSettingsSaveTestsConnectionAndMasksKey`.

Artifacts:

- `internal/models/transcription.go`
- `internal/profile/service.go`
- `internal/api/profile_handlers.go`
- `internal/api/response_models.go`
- `internal/api/architecture_test.go`
- `internal/transcription/service.go`
- `internal/recording/finalizer.go`
- `internal/database/legacy.go`
- `devnotes/v2.0.0/sprint-plans/asr-provider-backend-sprint-plan.md`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 7: Remote Provider REST Client

Status: completed

Completed tasks:

- [x] Added `internal/transcription/engineprovider/remote`.
- [x] Implemented control-plane REST endpoints.
- [x] Implemented ephemeral job REST endpoints.
- [x] Enforced request timeouts and response size limits.
- [x] Poll remote job status and replay progress/events.
- [x] Mapped typed remote errors with bounded sanitized details.
- [x] Propagated cancellation through `DELETE /v1/jobs/{job_id}`.

Acceptance checks:

- [x] Remote client implements the internal provider interface.
- [x] Fake HTTP provider tests cover success, progress, busy, unsupported operation, provider error, malformed JSON, timeout, and cancellation.
- [x] Remote provider URLs come only from injected config.
- [x] No URL/path/token leaks through public surfaces.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider/remote`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestASREngineImportInventory|TestProductionCodeDoesNotUseOldASRParameterIdentifiers|TestASRProvidersDoNotDependOnAPIOrRepositories|TestBackendDependencyDirection|TestListTranscriptionModels|TestTranscriptionExecutions'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/transcription/...`
- [x] `git diff --check`

Artifacts:

- `internal/transcription/engineprovider/remote/client.go`
- `internal/transcription/engineprovider/remote/client_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 8: Remote Provider Configuration And App Wiring

Status: completed

Completed tasks:

- [x] Extended `internal/config` with ASR provider configuration.
- [x] Validated remote provider specs, duplicates, defaults, durations, and mount values.
- [x] Kept the local provider directly in-process.
- [x] Built remote provider clients from config.
- [x] Registered providers before worker startup.
- [x] Kept local-only deployment behavior intentional after legacy ASR removal.

Acceptance checks:

- [x] Local-only deployments use the new provider registry path.
- [x] Invalid config fails startup with actionable errors.
- [x] Remote provider wiring is testable without starting the HTTP listener.
- [x] `internal/app` remains the only composition root.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/config ./internal/app`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/config ./internal/app ./internal/transcription/...`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestASREngineImportInventory|TestProductionCodeDoesNotUseOldASRParameterIdentifiers|TestASRProvidersDoNotDependOnAPIOrRepositories|TestBackendDependencyDirection|TestListTranscriptionModels|TestTranscriptionExecutions'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/config ./internal/app ./internal/transcription/...`
- [x] `git diff --check`

Artifacts:

- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 9: Pipeline Execution And Provider Chaining

Status: completed

Completed tasks:

- [x] Added internal pipeline resolution for transcription, diarization, and speaker identification steps.
- [x] Replaced old single-model execution with an internal provider pipeline path.
- [x] Resolved each step through the registry by provider, model, and required capability.
- [x] Executed steps serially.
- [x] Merged typed transcription and diarization artifacts into canonical transcript JSON.
- [x] Covered local/fake transcription plus remote/fake diarization.

Acceptance checks:

- [x] Jobs run through the new pipeline representation.
- [x] Diarization can run on a different provider than transcription.
- [x] Provider step failures stop the pipeline with sanitized error metadata.
- [x] Cancellation interrupts the active provider step.
- [x] Canonical transcript output remains stable for transcript API/UI consumers.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/orchestrator`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestASREngineImportInventory|TestProductionCodeDoesNotUseOldASRParameterIdentifiers|TestASRProvidersDoNotDependOnAPIOrRepositories|TestBackendDependencyDirection|TestListTranscriptionModels|TestTranscriptionExecutions'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/transcription/...`
- [x] `git diff --check`

Artifacts:

- `internal/transcription/orchestrator/processor.go`
- `internal/transcription/orchestrator/processor_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 10: Profile Pipeline Persistence

Status: completed

Completed tasks:

- [x] Store ordered ASR pipeline steps in profile JSON.
- [x] Require profile create/update requests to provide `options.pipeline`.
- [x] Resolve each profile step against provider model-card capabilities.
- [x] Bound and sanitize provider-specific option data.
- [x] Removed profile compatibility behavior that synthesized pipelines from old model/diarization fields.
- [x] Removed old ASR `diarize` and `diarize_model` parameter fields from `ASRParams`.
- [x] Extended architecture guard coverage for removed ASR parameter identifiers.
- [x] Updated job and profile metadata derivation to use pipeline steps.
- [x] Removed the stale `internal/queue` test file that imported deleted queue code.

Acceptance checks:

- [x] New profiles can persist multiple provider steps.
- [x] List/get profile responses expose the new pipeline contract.
- [x] Invalid pipeline shape is rejected before enqueue.
- [x] Legacy profile model/diarization inputs are not accepted as a fallback contract.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/transcription/... ./internal/recording`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'Test(Profile|Transcription|Recording|File)'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestASREngineImportInventory|TestProductionCodeDoesNotUseOldASRParameterIdentifiers|TestASRProvidersDoNotDependOnAPIOrRepositories|TestBackendDependencyDirection'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/profile ./internal/api ./internal/transcription/... ./internal/recording`
- [x] `git diff --check`
- [ ] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api` blocked in sandbox by `httptest.NewServer` bind denial in `TestLLMProviderSettingsSaveTestsConnectionAndMasksKey`.
- [ ] `GOCACHE=/tmp/scriberr-go-cache go test ./tests` blocked by pre-existing stale helpers referencing removed `models.Note`, old chat fields, and old API key repository APIs.

Artifacts:

- `internal/profile/service.go`
- `internal/profile/model_catalog.go`
- `internal/api/profile_handlers.go`
- `internal/api/response_models.go`
- `internal/api/types.go`
- `internal/api/architecture_test.go`
- `internal/models/transcription.go`
- `internal/transcription/orchestrator/processor.go`
- `internal/transcription/service.go`
- `internal/recording/finalizer.go`
- `tests/queue_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 11: Provider Admin/Diagnostics API

Status: completed

Completed tasks:

- [x] Added registry provider enumeration for diagnostics.
- [x] Added admin-only ASR provider list/detail endpoints.
- [x] Added admin-only provider model load/unload endpoints.
- [x] Kept `/api/v1/models/transcription` user-readable and unchanged.
- [x] Added bounded timeouts around provider diagnostics and model commands.
- [x] Sanitized provider diagnostics and provider error messages before API output.

Acceptance checks:

- [x] Users can still list selectable transcription models.
- [x] Admin diagnostics show provider state, active operation, and loaded models without paths/secrets.
- [x] Load/unload failures return typed safe errors.
- [x] Route contract and security regression tests cover new endpoints.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider ./internal/api -run 'TestAdminASRProvider|TestEndpointContractSmoke|TestRouteContract|TestListTranscriptionModels|TestTranscriptExecutionsLogsModelsAndStatsUseEngineServices'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/... ./internal/profile ./internal/recording`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'Test(AdminASRProvider|EndpointContractSmoke|RouteContract|ListTranscriptionModels|TranscriptExecutionsLogsModelsAndStatsUseEngineServices|ASREngineImportInventory|ProductionCodeDoesNotUseOldASRParameterIdentifiers|ASRProvidersDoNotDependOnAPIOrRepositories|BackendDependencyDirection)'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api` with localhost binding allowed for existing `httptest` LLM provider tests.
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/transcription/... ./internal/profile ./internal/recording`

Artifacts:

- `internal/api/asr_provider_admin_handlers.go`
- `internal/api/router.go`
- `internal/api/types.go`
- `internal/api/engine_worker_api_test.go`
- `internal/api/route_contract_test.go`
- `internal/transcription/engineprovider/types.go`
- `internal/transcription/engineprovider/registry.go`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 12: Contract Tests, Example Provider, And Hardening

Status: completed

Completed tasks:

- [x] Added reusable provider contract test helper.
- [x] Added minimal REST example provider server test.
- [x] Added provider author guide.
- [x] Tightened architecture tests for ASR contract purity and provider docs coverage.
- [x] Reviewed performance hot paths and recorded residual follow-up.
- [x] Removed remaining stale ASR parameter test usage found by broad validation.
- [x] Ran broad validation baseline.

Acceptance checks:

- [x] Third-party provider implementers have a testable contract.
- [x] Architecture guards prevent API/model/provider coupling regressions.
- [x] Remote provider failures do not destabilize local-only transcription.
- [x] Existing frontend and backend features remain green in focused tests.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider/contracttest ./internal/transcription/engineprovider/remote -run 'TestExampleProviderServerSatisfiesContract|TestClient'` with localhost binding allowed for `httptest`.
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/... ./internal/profile ./internal/recording` with localhost binding allowed.
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestASRContractPackageDoesNotDependOnBackendRuntime|TestASRProviderAuthorGuideDocumentsRequiredContract|TestASREngineImportInventory|TestProductionCodeDoesNotUseOldASRParameterIdentifiers|TestASRProvidersDoNotDependOnAPIOrRepositories|TestBackendDependencyDirection|TestAdminASRProvider'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/...` with localhost binding allowed.
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./tests` with localhost binding allowed.
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/...`
- [x] `git diff --check`

Artifacts:

- `internal/transcription/engineprovider/contracttest/provider_contract.go`
- `internal/transcription/engineprovider/remote/example_provider_test.go`
- `internal/api/architecture_test.go`
- `internal/database/database_test.go`
- `devnotes/v2.0.0/specs/asr-provider-author-guide.md`
- `devnotes/v2.0.0/status-updates/asr-provider-backend-sprint-12-hardening.md`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 13: Provider Registry V2 And Dynamic Parameter Schemas

Status: completed

Completed tasks:

- [x] Extended model cards with provider-owned descriptor data for language support, chunking, parameter schemas, and recommended defaults.
- [x] Added typed parameter schema with scopes, defaults, bounds, enum options, and reload semantics.
- [x] Added provider-neutral common parameter keys.
- [x] Required provider-specific keys to be namespaced and bounded.
- [x] Updated local and fake/example provider descriptors.
- [x] Expanded provider contract tests for schema well-formedness.

Acceptance checks:

- [x] Profile parameters can be validated from registry data alone.
- [x] Existing model-list behavior remains compatible with additive descriptor fields.
- [x] Fake providers can expose different capabilities without API handler changes.
- [x] Invalid schemas fail contract tests.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/asrcontract`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider/contracttest`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestASRContractPackageDoesNotDependOnBackendRuntime|TestASRProviderAuthorGuideDocumentsRequiredContract|TestASREngineImportInventory|TestProductionCodeDoesNotUseOldASRParameterIdentifiers|TestASRProvidersDoNotDependOnAPIOrRepositories|TestBackendDependencyDirection|TestListTranscriptionModels|TestProfile'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider/remote -run TestExampleProviderServerSatisfiesContract` with localhost binding allowed.
- [x] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/profile ./internal/transcription/asrcontract ./internal/transcription/engineprovider ./internal/transcription/engineprovider/contracttest`
- [x] `git diff --check`
- [ ] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...` blocked in sandbox by `httptest.NewServer` bind denial in `TestExampleProviderServerSatisfiesContract`; the same test passed with localhost binding allowed.
- [ ] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api ./internal/transcription/asrcontract ./internal/transcription/engineprovider ./internal/transcription/engineprovider/contracttest` blocked in sandbox by existing `TestLLMProviderSettingsSaveTestsConnectionAndMasksKey` loopback bind denial; the focused ASR/API subset passed.

Artifacts:

- `internal/transcription/asrcontract/types.go`
- `internal/transcription/asrcontract/types_test.go`
- `internal/transcription/engineprovider/contracttest/provider_contract.go`
- `internal/transcription/engineprovider/contracttest/provider_contract_test.go`
- `internal/transcription/engineprovider/local_provider.go`
- `internal/transcription/engineprovider/remote/example_provider_test.go`
- `internal/profile/model_catalog.go`
- `internal/profile/service.go`
- `internal/profile/service_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-provider-backend-sprint-tracker.md`

Commit:

- Pending.

## ASRP-Sprint 14: Execution Planner And Engine-Owned Chunking Contract

Status: planned

Planned tasks:

- [ ] Add execution planner for profile params, model defaults, provider status, audio metadata, and global limits.
- [ ] Represent fixed, VAD, provider-owned, and no-chunking modes explicitly.
- [ ] Represent batching as an execution-plan concern.
- [ ] Add deterministic sanitized plan summaries.
- [ ] Add cancellation and progress hooks at chunk/batch boundaries.

Acceptance checks:

- [ ] Planner validates combinations before long-running provider work.
- [ ] Unsupported chunking/batching combinations fail before execution.
- [ ] Provider-owned chunking remains available for models that require it.
- [ ] Execution metadata remains path-free.

Verification:

- [ ] Not started.

Commit:

- Pending.

## ASRP-Sprint 15: Local Sherpa Model Registry And Runtime Defaults

Status: planned

Planned tasks:

- [ ] Add local sherpa descriptors for Whisper and Parakeet families.
- [ ] Expose common sherpa offline runtime/decoding parameters.
- [ ] Expose Whisper-specific language/task/timestamp parameters.
- [ ] Expose Parakeet/NeMo transducer artifact and model-type metadata.
- [ ] Mark construction-time parameters with `requires_reload`.
- [ ] Capture experiment-derived defaults as recommendations.

Acceptance checks:

- [ ] Dynamic profile UI has enough metadata for Whisper and Parakeet.
- [ ] Whisper and Parakeet expose different valid parameter sets.
- [ ] Parakeet CPU defaults prefer fixed 30s chunks, four threads, and batch size 1 for the measured local profile.
- [ ] Registry data does not leak cache paths or host internals.

Verification:

- [ ] Not started.

Commit:

- Pending.

## ASRP-Sprint 16: Engine Integration Of Experiment-Proven Parakeet Flow

Status: planned

Planned tasks:

- [ ] Fold fixed-window Parakeet decoding defaults into the core local engine path.
- [ ] Preserve VAD chunking as explicit optional behavior.
- [ ] Keep NeMo-style token-to-word aggregation and timestamp normalization covered by tests.
- [ ] Route batch decode through the execution planner.
- [ ] Produce canonical transcript text, words, segments, and metrics.

Acceptance checks:

- [ ] Core engine produces transcript text, word timings, segment timings, and metrics.
- [ ] Experiment path remains available for research.
- [ ] VAD behavior remains intact.
- [ ] Fixed 30s Parakeet behavior matches experiment-level accuracy within expected nondeterminism.

Verification:

- [ ] Not started.

Commit:

- Pending.

## ASRP-Sprint 17: Frontend-Facing Model/Profile Discovery API

Status: planned

Planned tasks:

- [ ] Expose task, language, capability, chunking, runtime, and parameter metadata for profile UI.
- [ ] Keep backend schema display metadata lightweight.
- [ ] Return validation errors keyed by stable parameter IDs.
- [ ] Add compatibility mapping for existing profiles.
- [ ] Prevent provider internals from leaking through discovery payloads.

Acceptance checks:

- [ ] Frontend can render ASR profile fields without hardcoding Whisper or Parakeet parameters.
- [ ] Invalid profile submissions fail with useful parameter-keyed errors.
- [ ] Existing profiles remain readable.
- [ ] Non-admin discovery payloads are bounded and path-free.

Verification:

- [ ] Not started.

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
