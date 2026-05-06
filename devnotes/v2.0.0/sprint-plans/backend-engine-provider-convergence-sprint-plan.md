# Sprint Run: Backend Engine Provider Convergence

Run ID: `BE-ENG-PROVIDER`

Status: planning only. Do not implement code from this document until the user starts an implementation sprint.

## Purpose

Streamline the Scriberr backend and local speech engine so they share one provider/model contract, one parameter schema, and one execution-planning authority. The bundled sherpa-onnx provider must stay in-process and be called directly through the Go API. REST is only for future remote providers.

## Target Architecture

Ownership:

- Engine owns local model descriptors, parameter validation, sherpa config mapping, model lifecycle, chunking, VAD, batching, timestamp stitching, canonical local results, metrics, and sanitized plan summaries.
- Backend owns durable jobs, profiles, pipeline ordering, authorization, audio preprocessing, queueing, retries, progress persistence, transcript persistence, and public API DTOs.
- Local provider adapter is a thin in-process Go adapter from backend provider contract to `scriberr-engine`.
- Remote providers use the backend provider contract over REST later; they do not affect the local sherpa Go path.

Source of truth:

- Model descriptors come from the engine for the bundled local provider.
- Profile options are stored as `pipeline[].options` parameter maps using descriptor keys.
- Backend does not synthesize local model schemas, chunking defaults, batching defaults, or sherpa-specific model parameters.
- Backend stores engine-returned plan summaries and metrics instead of recomputing them.

## Engineering Rules

- No backward compatibility shims for removed legacy provider fields.
- Keep the local sherpa provider in-process through Go APIs.
- Do not add REST between Scriberr backend and `scriberr-engine`.
- Do not let API handlers or profile services import `scriberr-engine`; only the local provider adapter may import it.
- Remove duplicated planning instead of keeping parallel implementations.
- Prefer generic parameter maps over model-specific request fields.
- Add tests before cleanup for every removed compatibility path.
- Keep external provider extensibility through backend contract types, not through local sherpa assumptions.
- Commit each sprint separately when implemented.

## Validation Baseline

Run before closing implementation sprints when practical:

```sh
GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/... ./internal/profile ./internal/api
GOCACHE=/tmp/scriberr-go-cache go vet ./internal/transcription/... ./internal/profile ./internal/api
GOCACHE=/tmp/scriberr-engine-go-cache go test ./... 
```

Architecture checks:

```sh
rg -n 'scriberr-engine' internal --glob '*.go'
rg -n 'chunking\\.mode|chunking\\.chunk_seconds|batching\\.batch_size|vad\\.' internal/transcription/orchestrator internal/transcription/engineprovider --glob '*.go'
```

Expected:

- `scriberr-engine` imports exist only in the local provider adapter.
- Orchestrator does not implement fixed/VAD/batch planning for local engine execution.
- Local provider adapter forwards parameter maps instead of legacy typed fields.

## BE-ENG-PROVIDER-Sprint 0: Contract Duplication Inventory

Goal: freeze the current duplication map and add guardrails before deleting code.

Tasks:

- Inventory duplicated provider/model/status/progress/result types across:
  - `internal/transcription/asrcontract`
  - `internal/transcription/engineprovider`
  - `references/engine/speech/engine`
  - `references/engine/speech/providers`
  - `references/engine/speech/results`
- Add architecture tests that enforce:
  - only local provider adapter imports `scriberr-engine`
  - backend orchestrator does not import engine packages
  - local provider adapter does not synthesize local model schemas
  - root tests compile against the current engine API
- Document the final target contract shape.

Acceptance criteria:

- Duplication is documented with file-level cleanup decisions.
- Guardrails fail on reintroduced legacy engine request fields.
- No runtime behavior changes.

## BE-ENG-PROVIDER-Sprint 1: Backend Request Contract Collapse

Goal: replace backend provider request structs with generic parameter maps.

Tasks:

- Replace `engineprovider.TranscriptionRequest` model-specific fields with:
  - `JobID`
  - `UserID`
  - `AudioPath`
  - `ModelID`
  - `Parameters map[string]any`
  - `Progress`
- Replace `engineprovider.DiarizationRequest` tuning fields with `Parameters map[string]any`.
- Update orchestrator to pass selected `pipeline[].options` directly.
- Move legacy `ASRParams` flat fields into profile migration/normalization only long enough to convert to pipeline options, then delete them.
- Remove backend-owned `ChunkingStrategy`, `ChunkSize`, timestamp booleans, tail padding, decoding method, thread count, and diarization tuning fields from active execution flow.

Acceptance criteria:

- Backend compiles against current engine request API.
- Local provider adapter builds `speechengine.TranscriptionRequest{Parameters: ...}` only.
- Legacy flat ASR fields are not used for new execution planning.

## BE-ENG-PROVIDER-Sprint 2: Descriptor Passthrough For Local Provider

Goal: stop synthesizing local model metadata in the backend.

Tasks:

- Extend engine public model listing to expose full descriptors or descriptor-equivalent model cards.
- Map engine descriptors into `asrcontract.ModelCard` mechanically.
- Delete backend functions that manually invent local schemas/defaults:
  - `chunkingCapabilitiesForModel`
  - `parameterSchemaForModel`
  - `recommendedDefaultsForModel`
  - local Parakeet/Whisper-specific schema helpers
- Preserve backend model aggregation for future remote providers.

Acceptance criteria:

- Local provider model cards match engine descriptors.
- Whisper language/task options come from engine descriptors.
- Parakeet chunk/batch defaults come from engine descriptors.
- Backend has no local sherpa model-family heuristics for schema/defaults.

## BE-ENG-PROVIDER-Sprint 3: Delete Backend Chunking And Batching Planner

Goal: make the engine the sole local chunking/VAD/batching planner.

Tasks:

- Remove `orchestrator.ExecutionPlan` chunking/batching/runtime planning for local engine execution.
- Keep only backend pipeline sequencing: transcription, diarization, speaker identification, audio tagging later.
- Persist engine-returned plan summary from provider result metadata.
- Remove VAD settings from backend planner unless they are just validated parameter keys passed to provider.
- Delete backend boundary progress events that duplicate engine chunk/batch progress.

Acceptance criteria:

- No backend code computes local chunk count from duration/chunk size.
- No backend code chooses fixed vs VAD for local engine after profile validation.
- Engine progress events drive transcription progress.

## BE-ENG-PROVIDER-Sprint 4: Canonical Result And Metrics Alignment

Goal: remove duplicate result/metadata computation.

Tasks:

- Map engine transcript result into backend canonical transcript once.
- Preserve engine metrics and plan summary in execution metadata.
- Delete backend estimated audio duration and recomputed RTF for local engine results.
- Normalize speaker fields only in canonical transcript builder.
- Add tests for metrics and plan summary persistence.

Acceptance criteria:

- Execution metadata uses engine metrics where available.
- Backend does not recalculate local decode metrics from words/segments.
- Transcript JSON remains stable for API consumers.

## BE-ENG-PROVIDER-Sprint 5: Profile Parameter Model Cleanup

Goal: make ASR profile settings descriptor-driven.

Tasks:

- Store profile settings as pipeline steps with `options`.
- Validate options against provider model descriptor schema.
- Remove active use of flat `ASRParams` provider/model fields except migration reads.
- Add profile API responses that include descriptor schemas for frontend dynamic controls.
- Ensure unsupported options fail at profile save time and job execution time.

Acceptance criteria:

- Frontend can render ASR profile controls from model descriptors.
- Adding a provider model generally means exposing a descriptor, not editing backend profile structs.
- Profile validation is provider/model-specific but not hardcoded to sherpa.

## BE-ENG-PROVIDER-Sprint 6: Provider Interface Slimming

Goal: make future providers easy to implement.

Tasks:

- Split provider interface by task capability:
  - model registry/status/lifecycle
  - transcription
  - diarization
  - speaker identification
  - future audio tagging/language detection
- Remove `Capabilities()` if `Models()` provides enough information.
- Keep `Prepare()` only if it has a concrete provider lifecycle role; otherwise delete.
- Keep local provider direct-Go; add remote provider interface notes only.

Acceptance criteria:

- A transcription-only provider can implement a small interface.
- Unsupported task methods are not required unless the provider advertises the task.
- Registry selection remains capability-driven.

## BE-ENG-PROVIDER-Sprint 7: Hardening And Documentation

Goal: lock in the simplified architecture.

Tasks:

- Update ASR provider specs and author guide with final source-of-truth rules.
- Update backend and engine architecture docs.
- Add architecture tests preventing schema synthesis, duplicate chunk planning, and REST use for local sherpa.
- Run full backend and engine validation.

Acceptance criteria:

- Docs match implemented architecture.
- Local sherpa path is confirmed direct Go API only.
- Future remote providers have a clear, separate extension path.
