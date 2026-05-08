# Sprint Run: ASR Provider Profile Frontend Convergence

Run ID: `ASR-PROFILE-FE`

Status: planning only. Do not implement code from this document until the user starts an implementation sprint.

Date: 2026-05-08

## Goal

Revamp the Settings > ASR > Transcription profiles flow so it is driven by provider model cards and the engine parameter schema instead of legacy hard-coded profile fields.

The frontend must expose every parameter advertised by the selected engine model, hide unsupported parameters, and save profile options as descriptor-keyed `pipeline[].options` maps. The backend provider work is assumed complete; this run is for verifying the exposed contract and updating the frontend to consume it correctly.

## Current Findings

- `references/engine/speech/providers/sherpa/catalog/catalog.go` already registers `parakeet-v2` and `parakeet-v3`.
- Both Parakeet models use the `parakeetDescriptor`, advertise transcription, English fixed language, word and segment timestamps, fixed/VAD chunking, batching, CPU/CUDA runtime backends, and model artifact downloads.
- The local provider adapter maps engine descriptors into `asrcontract.ModelCard` through `internal/transcription/engineprovider/local_provider.go`.
- `GET /api/v1/models/transcription` returns sanitized model cards, including `parameter_schema`, `chunking`, `dependencies`, `artifacts`, and `recommended_defaults`.
- `web/frontend/src/features/settings/api/profilesApi.ts` and `ASRProfileDialog.tsx` still use legacy flat options such as `language`, `task`, `threads`, `tail_paddings`, `chunking_strategy`, and diarization fields.
- Backend profile handlers now reject legacy top-level ASR option fields and require `options.pipeline`.

## Source-Of-Truth Parameter Inventory

Transcription models advertise these common ASR parameters:

- `runtime.provider`
- `runtime.num_threads`
- `decoding.method`
- `chunking.mode`
- `chunking.chunk_seconds`
- `chunking.overlap_seconds`
- `batching.batch_size`
- `output.timestamps`
- `vad.threshold`
- `vad.min_silence_seconds`
- `vad.min_speech_seconds`
- `vad.max_speech_seconds`
- `vad.window_size`
- `vad.buffer_seconds`
- `vad.feed_seconds`

Whisper models additionally advertise:

- `sherpa.whisper.language`
- `sherpa.whisper.task`
- `sherpa.whisper.tail_paddings`
- `sherpa.whisper.enable_token_timestamps`
- `sherpa.whisper.enable_segment_timestamps`

Parakeet TDT v2 and v3 additionally advertise:

- `sherpa.model_type`

Diarization models advertise:

- `runtime.provider`
- `runtime.num_threads`
- `diarization.num_clusters`
- `diarization.threshold`
- `diarization.min_duration_on`
- `diarization.min_duration_off`

The frontend must not maintain separate hard-coded allowlists for these fields. This inventory is here to guide contract tests and implementation review only.

## Engineering Rules

- Follow `devnotes/v2.0.0/rules/react-architecture-rules.md`.
- Use `features/settings/api` for typed endpoint contracts and normalization.
- Use `features/settings/hooks` for TanStack Query hooks and mutation invalidation.
- Keep profile dialog state local to the dialog.
- Keep backend field names in API types unless intentionally normalized at the boundary.
- Do not use legacy flat ASR fields in new save payloads.
- Do not preserve legacy flat ASR profile compatibility. Remove old compatibility code instead of migrating or normalizing it.
- Do not infer supported parameters from model family strings.
- Prefer rendering from `parameter_schema`, `recommended_defaults`, and existing profile `pipeline[].options`.
- Hide fields only when the descriptor says they are advanced or `visible_when` rules are not satisfied.
- Never send unsupported parameter keys for the selected model.
- Keep the settings UI compact, scannable, and aligned with existing settings patterns.

## Validation Baseline

Run before closing implementation sprints when practical:

```sh
npm --prefix web/frontend run build
npm --prefix web/frontend run lint
GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./internal/profile ./internal/transcription/engineprovider ./internal/transcription/asrcontract
(cd references/engine && GOCACHE=/tmp/scriberr-engine-go-cache go test ./...)
git diff --check
```

If frontend tests are added, run the focused test command for the affected package before the production build.

## ASR-PROFILE-FE-Sprint 0: Contract Verification And Guardrails

Goal: prove that the backend and local engine expose the complete contract the frontend needs.

Tasks:

- Add or update focused tests confirming the canonical model-card endpoint includes both `parakeet-v2` and `parakeet-v3` for transcription capability.
- Verify both Parakeet model cards include `parameter_schema`, `recommended_defaults`, `chunking`, `dependencies`, and `artifacts`.
- Verify Parakeet parameter schema includes all common ASR parameters plus `sherpa.model_type`.
- Verify Whisper schema includes common ASR parameters plus Whisper-specific fields.
- Verify profile create/update rejects legacy flat fields and accepts `options.pipeline`.
- Inventory and mark frontend/backend legacy flat profile compatibility code for deletion.
- Record any backend contract gaps before frontend implementation.

Acceptance criteria:

- The frontend can rely on the canonical capability-filtered model-card endpoint as the model-card source of truth.
- Parakeet TDT v2 and v3 are visible through the public model-card endpoint.
- Legacy flat ASR profile inputs have no supported save path.
- No frontend implementation starts until dependency sprint contract gaps are fixed.

## ASR-PROFILE-FE-Sprint 1: Frontend API Contract Rewrite

Goal: replace legacy profile types with model-card and pipeline-aware types.

Tasks:

- Update `profilesApi.ts` to model `TranscriptionModel` as the sanitized backend model card, including:
  - `display_name`
  - `provider`
  - `model_type`
  - `capabilities`
  - `chunking`
  - `dependencies`
  - `parameter_schema`
  - `recommended_defaults`
- Add typed `ParameterDescriptor`, `ParameterOption`, `ActivationRule`, and `ASRStep` types.
- Change `TranscriptionProfileOptions` to `{ pipeline: ASRStep[] }`.
- Remove frontend normalization for legacy flat ASR response fields.
- Save profiles by sending only `options.pipeline`, never legacy flat fields.
- Keep `listTranscriptionModels` filtering by `capabilities.transcription`.

Acceptance criteria:

- TypeScript no longer models active ASR profile settings as `language`, `task`, `threads`, `chunking_strategy`, or diarization tuning fields.
- Save payloads match backend pipeline validation.
- Legacy flat field normalization is deleted rather than preserved.
- Unknown model-card fields are not discarded if they are needed for future rendering.

## ASR-PROFILE-FE-Sprint 2: Dynamic Parameter Form Core

Goal: build a reusable descriptor-driven form renderer for model parameters.

Tasks:

- Add a feature-local component for rendering one model card's `parameter_schema`.
- Support all parameter types currently defined by the provider contract:
  - `boolean`
  - `integer`
  - `number`
  - `string`
  - `enum`
  - `duration`
  - `path_ref` as read-only/unsupported for user entry unless a future provider explicitly needs it.
- Use descriptor `label`, `default`, `min`, `max`, `step`, `options`, `scope`, `advanced`, `requires_reload`, and `visible_when`.
- Initialize values from existing step options, falling back to `recommended_defaults`, then descriptor defaults.
- Omit parameters whose value matches the descriptor default only if this does not lose intended `recommended_defaults`; otherwise save explicit recommended defaults.
- Implement `visible_when` evaluation for VAD-only controls.
- Group controls by scope: model, runtime, decoding, chunking, VAD, output, and postprocess.
- Add an Advanced disclosure that shows all `advanced` parameters.

Acceptance criteria:

- Selecting a model changes the rendered controls entirely from that model's descriptor.
- Unsupported model-specific controls disappear automatically.
- VAD settings appear only when `chunking.mode` is `vad`.
- Boolean fields use checkboxes/toggles, enum fields use selects, and numeric fields use bounded number inputs.

## ASR-PROFILE-FE-Sprint 3: Profile Dialog Revamp

Goal: replace the hard-coded ASR profile modal with a pipeline editor aligned to backend provider semantics.

Tasks:

- Keep profile name, description, and default toggle.
- Render a transcription step selector using transcription model cards.
- Save the first pipeline step as:
  - `kind: "transcription"`
  - `provider`
  - `model`
  - `model_family` only if it remains part of the canonical pipeline step contract
  - `options`
- Add an optional diarization step toggle.
- When diarization is enabled, select the default diarization-capable model from available provider cards if the frontend has them; otherwise use a backend-confirmed default only if the API still exposes it.
- Render diarization parameters from that model's schema, not hard-coded fields.
- Remove hard-coded language/task/thread/tail-padding/chunking/diarization controls from `ASRProfileDialog.tsx`.
- Add compact summaries for selected model, installed/download state, reload-required changes, and advanced fields.

Acceptance criteria:

- The dialog exposes every parameter in the selected transcription model schema.
- Parakeet TDT v2/v3 show Parakeet-supported parameters and do not show Whisper-only parameters.
- Whisper models show Whisper-specific language/task/timestamp controls.
- Diarization parameters are descriptor-driven or explicitly blocked until the backend exposes diarization model cards to the frontend.

## ASR-PROFILE-FE-Sprint 4: Profile List And Legacy Cleanup

Goal: make pipeline profiles readable and delete legacy flat-profile UI assumptions.

Tasks:

- Update `ProfileRow` summaries to read from `options.pipeline`.
- Show transcription provider/model, key selected parameter summaries, and whether diarization is enabled.
- Use `parameter_schema.expose_in_summary` when model cards are available.
- For profiles whose stored model is no longer available, show a clear missing-model error.
- Treat profiles without a valid transcription pipeline step as invalid data and require backend/API cleanup, not frontend repair.
- Remove remaining frontend reads of legacy flat fields from profile summaries and dialogs.
- Avoid mutating profile options during render.

Acceptance criteria:

- Existing pipeline profiles display useful summaries.
- Missing model cards produce clear errors instead of blank dialogs.
- Profiles without a valid pipeline are not silently repaired or defaulted by the frontend.
- Profile rows no longer depend on legacy flat option names.

## ASR-PROFILE-FE-Sprint 5: Frontend Tests And Browser QA

Goal: lock the new profile workflow against regressions.

Tasks:

- Add focused unit tests for:
  - model-card normalization
  - pipeline save payload construction
  - descriptor default resolution
  - `visible_when` evaluation
  - unsupported parameter stripping
- Add component tests for:
  - Parakeet v2/v3 controls
  - Whisper controls
  - VAD advanced controls visibility
  - diarization toggle behavior
- Run production build and lint.
- Use browser QA on Settings > ASR at desktop and mobile widths.

Acceptance criteria:

- The profile dialog can create and edit a Parakeet TDT v2 or v3 profile.
- The save payload contains only descriptor-supported keys under `pipeline[].options`.
- No unsupported legacy flat fields are sent.
- Controls fit on desktop and mobile without overlap.

## Related Sprint Runs

The backend/API gaps that block a clean descriptor-driven frontend are split into separate sprint runs instead of being tracked as deferred questions:

- `ASR-MODEL-CATALOG`: `devnotes/v2.0.0/sprint-plans/asr-model-catalog-endpoints-sprint-plan.md`
- `ASR-PARAM-CONTRACT`: `devnotes/v2.0.0/sprint-plans/asr-parameter-contract-sprint-plan.md`
- `ASR-LEGACY-CLEANUP`: `devnotes/v2.0.0/sprint-plans/asr-legacy-profile-cleanup-sprint-plan.md`

Run order:

1. Complete `ASR-MODEL-CATALOG` so the frontend can fetch transcription and diarization model cards.
2. Complete `ASR-PARAM-CONTRACT` so read-only parameters such as `sherpa.model_type` have first-class contract semantics.
3. Complete `ASR-LEGACY-CLEANUP` so frontend work does not carry old flat profile compatibility.
4. Implement this frontend run against the cleaned provider/profile contract.

## Commit Plan

1. Contract verification tests and any backend endpoint gap fixes.
2. Frontend API type and pipeline payload rewrite.
3. Dynamic descriptor-driven parameter form.
4. Profile dialog and profile row revamp.
5. Tests, responsive QA, and cleanup of legacy ASR field references.
