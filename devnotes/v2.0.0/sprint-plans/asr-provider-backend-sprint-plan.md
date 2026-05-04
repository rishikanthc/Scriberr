# Sprint Run: ASR Provider Backend Architecture

Run ID: `ASRP`

Status: planning only. Do not implement code from this document until the user explicitly starts an implementation sprint.

## Scope

This sprint run implements the provider backend architecture in:

- `devnotes/v2.0.0/specs/asr-provider-backend-architecture.md`
- `devnotes/v2.0.0/specs/architecture-design.md`
- `devnotes/v2.0.0/rules/backend-rules.md`

The goal is to make Scriberr's ASR provider layer production-ready without breaking the existing transcription queue, profiles, API responses, summaries, chat, annotations, recordings, or file workflows.

The bundled sherpa-onnx provider remains in-process and is called through Go APIs. External providers use a REST adapter. Do not edit `references/engine` during this sprint run. If the local sherpa adapter needs behavior the current engine cannot provide yet, add a narrow placeholder, fake, or compatibility shim inside Scriberr's provider boundary and document the deferred engine work.

## Engineering Rules

- Follow `devnotes/v2.0.0/rules/backend-rules.md`.
- Write or update focused tests before implementation.
- Keep API handlers thin: authenticate, validate syntax, call one service, map DTOs.
- Keep queue state transitions in repository/worker/orchestrator services, never handlers.
- Keep provider contracts behind interfaces and fakeable in tests.
- Do not import `scriberr-engine` outside the local provider adapter package.
- Do not edit `references/engine`.
- Do not introduce runtime environment reads outside `internal/config`.
- Do not expose local paths, normalized audio paths, provider URLs, tokens, model cache paths, or stack traces through API responses, SSE events, logs endpoints, or execution rows.
- Prefer deterministic fake providers and fake remote servers for tests. Real engine and real provider integration tests must be opt-in.
- Keep behavior backwards compatible unless a sprint explicitly owns a migration. Existing single-model profiles must continue to run.
- Each sprint should be reviewable as one coherent change set. Avoid broad package renames and unrelated cleanup.

## Validation Baseline

Run focused tests during development. Run this broader backend set before closing each implementation sprint when practical:

```sh
GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./internal/files ./internal/recording ./internal/summarization ./internal/chat ./internal/account ./internal/profile ./internal/llmprovider ./internal/automation ./cmd/server ./pkg/logger ./pkg/middleware
GOCACHE=/tmp/scriberr-go-cache go vet ./internal/api ./internal/config ./internal/database ./internal/repository ./internal/transcription/... ./internal/files ./internal/recording ./internal/summarization ./internal/chat ./internal/account ./internal/profile ./internal/llmprovider ./internal/automation ./cmd/server ./pkg/logger ./pkg/middleware
git diff --check
```

If sandbox loopback restrictions block tests using `httptest.NewServer`, record the blocker in the tracker and run the nearest focused package tests.

Architecture checks required during this run:

```sh
rg 'scriberr-engine' internal --glob '*.go'
rg 'os\\.Getenv|getenv|LookupEnv' internal --glob '*.go'
rg 'source_file_path|AudioPath|output_json_path' internal/api --glob '*.go'
```

Expected outcome:

- `scriberr-engine` appears only in the local provider adapter and tests that explicitly verify the boundary.
- Runtime config reads are contained in `internal/config`.
- API code does not expose or construct local paths.

## ASRP-Sprint 0: Inventory, Guardrails, And Compatibility Map

Goal: lock down current ASR coupling and compatibility requirements before behavior changes.

Tasks:

- Inventory current imports and usages of:
  - `scriberr-engine`
  - `engineprovider.ModelCapability`
  - profile model validation
  - orchestrator provider selection
  - transcription execution metadata
  - file/audio path handling
- Add or update architecture tests for:
  - `internal/api` must not import `scriberr-engine`
  - profile handlers/services must not import `scriberr-engine`
  - only the local provider adapter may import `scriberr-engine`
  - provider packages must not import API or repository packages
- Document compatibility behavior for existing profiles using current `WhisperXParams`.
- Create a short route/API impact matrix for models, profiles, transcription creation, events, logs, and executions.

Acceptance criteria:

- Current ASR coupling is documented.
- Guard tests fail on newly introduced forbidden imports.
- Existing public API behavior is intentionally preserved.
- No runtime behavior changes.

Testing focus:

- Architecture tests.
- Compile-only package checks where useful.

Commit guidance:

- Commit guard tests and inventory artifacts together.

## ASRP-Sprint 1: Pure ASR Contract Types

Goal: introduce provider contract types that do not depend on sherpa, HTTP, GORM, or API DTOs.

Tasks:

- Add `internal/transcription/asrcontract`.
- Define pure types for:
  - `ProviderInfo`
  - `ModelCard`
  - typed capabilities
  - `ProviderStatus`
  - `LoadedModel`
  - progress events
  - load/unload requests
  - transcription, diarization, and speaker identification requests/results
  - typed provider errors
- Include JSON tags for future REST compatibility.
- Add helpers for capability checks and provider error classification.
- Keep provider-specific options as bounded `map[string]any` or `json.RawMessage` with explicit validation boundaries.

Acceptance criteria:

- Contract package imports only standard library packages.
- Typed capabilities replace free-form strings in new code paths.
- Provider errors carry code, sanitized message, retryable flag, and optional bounded details.
- No runtime provider behavior changes yet.

Testing focus:

- Capability matching.
- Error classification.
- JSON round-trip compatibility for model cards, progress, status, and results.

Commit guidance:

- Commit contract tests first, then contract implementation.

## ASRP-Sprint 2: Provider Interface And Registry V2

Goal: evolve the internal provider boundary and registry while preserving the current local provider behavior.

Tasks:

- Update `internal/transcription/engineprovider` to use `asrcontract` model cards and status types.
- Add the internal provider interface with:
  - inspect
  - models
  - status
  - load/unload
  - loaded models
  - transcribe
  - diarize
  - identify speakers
  - close
- Add `ProgressSink`.
- Implement deterministic provider/model selection by provider ID, model ID, capability requirements, health, and busy state.
- Preserve compatibility adapters for existing callers where needed.
- Keep the local provider in-process. Do not route local sherpa through REST.
- Do not edit `references/engine`; unsupported local operations may return `UNSUPPORTED_OPERATION` or placeholder status.

Acceptance criteria:

- Registry can represent local and remote providers through the same interface.
- Selection is deterministic and test-backed.
- Existing default provider/model behavior remains stable.
- Adding a second fake provider requires no API, repository, or worker changes.

Testing focus:

- Explicit provider/model selection.
- Model-only selection.
- Capability-only selection.
- Busy/unhealthy provider exclusion.
- Unsupported operation behavior.
- Backwards-compatible capability listing.

Commit guidance:

- Commit fake-provider registry tests first.
- Commit interface/registry implementation second.
- Commit compatibility cleanup separately if needed.

## ASRP-Sprint 3: Model Catalog Service And Profile Validation

Goal: move model/profile validation out of API handlers and away from `scriberr-engine` metadata.

Tasks:

- Add a narrow model catalog interface/service backed by the provider registry.
- Update profile service validation to use model cards and capability checks.
- Keep API handlers as request/response adapters only.
- Preserve existing profile JSON shape initially.
- Normalize existing single-model profile inputs into a legacy-compatible internal selection command.
- Add validation for unsupported provider/model/feature combinations.
- Keep old profiles loadable and runnable.

Acceptance criteria:

- `internal/api/profile_handlers.go` no longer imports `scriberr-engine`.
- Profile validation is service-owned and test-backed.
- Existing profile create/update/list/get API response shape remains stable.
- `/api/v1/models/transcription` is backed by model cards from the registry.

Testing focus:

- Profile create/update valid current local model.
- Invalid model.
- Unsupported capability.
- Explicit provider unavailable.
- Existing profile compatibility.
- API route/response shape tests.

Commit guidance:

- Commit profile service tests before handler changes.

## ASRP-Sprint 4: Provider Progress And Execution Metadata

Goal: route provider progress through Scriberr's existing durable progress/event system without making events the source of truth.

Tasks:

- Add `ProgressSink` implementation in the orchestrator.
- Map provider stages to job progress and small SSE events.
- Store provider step metadata in execution config/request JSON without leaking paths.
- Add provider error code and operation kind to execution metadata where schema compatibility allows.
- Preserve existing execution list API response shape, adding fields only if route contract tests are updated deliberately.
- Ensure terminal state is persisted before terminal events are published.

Acceptance criteria:

- Local/fake providers can emit progress events.
- Progress updates are persisted through existing job progress methods.
- SSE payloads are small and path-free.
- Logs/executions expose sanitized provider details only.

Testing focus:

- Provider progress maps to repository progress.
- Terminal provider event after durable complete/fail.
- Canceled context maps to stopped/canceled behavior.
- Path/token redaction in events, logs, and executions.

Commit guidance:

- Commit orchestrator progress tests before implementation.

## ASRP-Sprint 5: Audio Preprocessing Boundary

Goal: make Scriberr responsible for provider-ready normalized audio artifacts without changing public file behavior.

Tasks:

- Add `internal/transcription/preprocess` with a narrow interface for provider-ready audio.
- Produce 16 kHz mono WAV artifacts under configured internal storage.
- Add config for normalized audio directory and provider-visible mount root.
- Cache normalized artifacts by source hash/job id where safe.
- Keep all normalized paths internal and sanitized.
- Update orchestrator to use preprocessed audio for provider requests.
- Keep existing source file streaming and file metadata behavior unchanged.

Acceptance criteria:

- Providers receive provider-visible normalized paths, not public paths.
- API responses never expose normalized artifact paths.
- Existing audio streaming endpoint still serves the original authorized audio.
- Preprocessing failures fail the job with sanitized errors.

Testing focus:

- Preprocess request path mapping.
- Cache hit/miss behavior.
- Path traversal rejection.
- Sanitized failure.
- Orchestrator passes normalized audio to fake provider.

Commit guidance:

- Commit pure preprocess unit tests first.
- Commit orchestrator integration tests second.

## ASRP-Sprint 6: Remote Provider REST Client

Goal: add REST support for external provider containers using fake HTTP providers only.

Tasks:

- Add `internal/transcription/engineprovider/remote`.
- Implement REST client for:
  - `GET /v1/health`
  - `GET /v1/provider`
  - `GET /v1/models`
  - `GET /v1/status`
  - `GET /v1/models/loaded`
  - `POST /v1/models/{model_id}:load`
  - `POST /v1/models/{model_id}:unload`
  - `POST /v1/jobs`
  - `GET /v1/jobs/{job_id}`
  - `GET /v1/jobs/{job_id}/events`
  - `DELETE /v1/jobs/{job_id}`
- Enforce request/response size limits and timeouts.
- Poll remote job status for sporadic progress.
- Map remote typed errors into `asrcontract.ProviderError`.
- Propagate cancellation with `DELETE /v1/jobs/{job_id}`.
- Do not add real external provider containers in this sprint.

Acceptance criteria:

- Remote client implements the same internal provider interface.
- Fake HTTP provider tests cover success, progress, busy, unsupported operation, provider error, malformed JSON, timeout, and cancellation.
- Remote provider URLs come only from injected config.
- No provider URL/path/token leaks through public surfaces.

Testing focus:

- HTTP method/path/body contracts.
- Model card mapping.
- Status mapping.
- Progress polling and event replay.
- Error sanitization.
- Context cancellation cleanup.

Commit guidance:

- Commit fake server tests and contract fixtures first.

## ASRP-Sprint 7: Remote Provider Configuration And App Wiring

Goal: wire local and remote providers in `internal/app` without changing the durable queue contract.

Tasks:

- Extend `internal/config` with:
  - local sherpa enabled flag
  - default ASR provider
  - remote provider endpoint list
  - normalized audio dir
  - provider audio mount root
  - remote provider timeout/poll interval
- Validate duplicate provider IDs, invalid URLs, missing defaults, invalid durations, and unsafe mount values.
- Build local provider directly in-process.
- Build remote provider clients from config.
- Register all providers in the registry before worker startup.
- Keep startup tolerant of unavailable remote providers unless configured as required.
- Preserve existing default behavior when no remote providers are configured.

Acceptance criteria:

- Existing deployments with only local sherpa keep working.
- Invalid config fails startup with actionable errors.
- Remote provider wiring is testable without starting the HTTP listener.
- `internal/app` remains the only composition root.

Testing focus:

- Config defaults.
- Invalid remote provider specs.
- Duplicate provider IDs.
- Default provider not registered.
- App build with fake/disabled remote provider configuration.

Commit guidance:

- Commit config tests first.
- Commit app wiring after provider client tests pass.

## ASRP-Sprint 8: Pipeline Execution And Provider Chaining

Goal: let one transcription job run an ordered provider pipeline while preserving existing single-step jobs.

Tasks:

- Add internal pipeline representation for transcription, diarization, and speaker identification steps.
- Convert existing profile options into a compatibility pipeline.
- Resolve each pipeline step through the provider registry.
- Execute steps serially because providers process one job at a time.
- Merge typed artifacts into canonical transcript JSON in Scriberr.
- Support local transcription plus remote/fake diarization in tests.
- Keep worker queue ownership unchanged.

Acceptance criteria:

- Existing single-model jobs run through a one-step compatibility pipeline.
- Diarization can run on a different provider than transcription.
- Provider step failures stop the pipeline and fail the job with sanitized error metadata.
- Cancellation interrupts the active provider step.
- Canonical transcript output remains compatible with existing transcript API/UI consumers.

Testing focus:

- Single-step compatibility.
- Local/fake transcription plus fake remote diarization.
- Unsupported step kind.
- Step selection failure.
- Mid-pipeline failure.
- Cancellation during second step.
- Canonical transcript speaker merge.

Commit guidance:

- Commit orchestrator pipeline tests before implementation.

## ASRP-Sprint 9: Profile Pipeline Persistence

Goal: persist ordered provider pipeline configuration while preserving legacy profile compatibility.

Tasks:

- Extend profile persistence to store ordered pipeline steps in JSON.
- Keep legacy profile fields populated for existing API consumers where needed.
- Add migration/compatibility logic from current `WhisperXParams` fields into pipeline steps.
- Validate provider-specific options against model-card parameter schemas where supported.
- Keep provider-specific option data bounded and sanitized.
- Avoid schema churn for every provider-specific model option.

Acceptance criteria:

- Existing profiles survive migration and still run.
- New profiles can persist multiple provider steps.
- List/get profile responses remain backwards compatible or explicitly versioned.
- Invalid pipeline shape is rejected by the service before enqueue.

Testing focus:

- Legacy profile read/write.
- Multi-step profile read/write.
- Default profile behavior.
- Provider option validation.
- Per-user profile isolation.

Commit guidance:

- Commit persistence tests before model changes.

## ASRP-Sprint 10: Provider Admin/Diagnostics API

Goal: expose safe provider and model diagnostics for the UI/admin workflows without leaking internals.

Tasks:

- Add service methods for provider list/status/model load/unload.
- Add authenticated/admin-gated endpoints as appropriate for:
  - provider status
  - loaded models
  - model load
  - model unload
- Keep `/api/v1/models/transcription` user-readable for profile selection.
- Ensure load/unload commands do not run inside long handler work if they can block beyond configured limits; use bounded timeouts.
- Sanitize provider status messages.

Acceptance criteria:

- Users can still list selectable transcription models.
- Admin diagnostics show provider state, active operation, and loaded models without paths/secrets.
- Load/unload failures return typed safe errors.
- Route contract tests and security regression tests cover new endpoints.

Testing focus:

- Auth/admin gates.
- Response shape.
- Busy provider load/unload behavior.
- Sanitization.
- Timeout handling.

Commit guidance:

- Commit API route contract tests first.

## ASRP-Sprint 11: Contract Tests, Example Provider, And Hardening

Goal: finish the provider architecture with reusable tests and final guardrails.

Tasks:

- Add provider contract test helpers usable by local, remote fake, and future third-party providers.
- Add a minimal fake/example provider server for development tests.
- Add docs for provider authors:
  - required endpoints
  - model card fields
  - progress events
  - error codes
  - mounted audio path expectations
- Tighten architecture tests from inventory mode to hard enforcement where feasible.
- Review performance hot paths:
  - model-card caching
  - provider status polling
  - normalized audio cache lookup
  - queue claim behavior with multiple providers
- Run the broad validation baseline.

Acceptance criteria:

- Third-party provider implementers have a testable contract.
- Architecture guards prevent regression to API/model/provider coupling.
- Remote provider failures do not destabilize local-only transcription.
- Existing frontend and backend features remain green in focused tests.

Testing focus:

- Contract test suite.
- Broad backend test baseline.
- Security/path-leak regression.
- Deterministic provider registry behavior.

Commit guidance:

- Commit docs/test fixtures separately from guard tightening if the diff is large.

## Deferred Work

Do not include these in this sprint run unless explicitly requested:

- Rewriting `references/engine`.
- Moving local sherpa into a sidecar container.
- gRPC or WebSocket provider transports.
- Live streaming transcription.
- Multi-job concurrent execution inside a single provider.
- Automatic Docker service discovery.
- Cloud/object-storage audio handoff.
