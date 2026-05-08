# Sprint Run Tracker: ASR Provider Profile Frontend Convergence

Run ID: `ASR-PROFILE-FE`

Status: not started.

This tracker belongs to `devnotes/v2.0.0/sprint-plans/asr-profile-provider-frontend-sprint-plan.md`.

## Run Rules

- Follow `devnotes/v2.0.0/rules/react-architecture-rules.md`.
- Keep implementation sprints independently reviewable.
- Keep ASR profile UI driven by provider model cards and `parameter_schema`.
- Do not reintroduce legacy flat ASR profile fields in active frontend save payloads.
- Do not preserve legacy flat ASR profile compatibility; remove obsolete code instead.
- Do not infer supported parameters from model family strings.
- Update this tracker in the same change set as each completed sprint.
- Run `git diff --check` before closing every sprint.
- Document any skipped validation and the reason.
- Leave unrelated dirty worktree changes untouched and documented.

## Validation Checklist

Before closing each implementation sprint when practical:

- [ ] Focused frontend tests or component checks for the sprint.
- [ ] `npm --prefix web/frontend run build`.
- [ ] `npm --prefix web/frontend run lint`.
- [ ] Focused backend/API tests if the sprint touches provider/profile contracts.
- [ ] `git diff --check`.
- [ ] Desktop Settings > ASR browser check for UI-affecting sprints.
- [ ] Mobile Settings > ASR browser check for UI-affecting sprints.

## Dependency Sprint Tracker

These dependency sprint runs should be completed before this frontend run starts.

### ASR-MODEL-CATALOG

Status: pending

Plan:

- `devnotes/v2.0.0/sprint-plans/asr-model-catalog-endpoints-sprint-plan.md`

Tracker:

- `devnotes/v2.0.0/sprint-trackers/asr-model-catalog-endpoints-sprint-tracker.md`

Required before frontend:

- [ ] Frontend-accessible model-card endpoint returns transcription and diarization models by capability.
- [ ] `diarization-default` exposes its parameter schema through that endpoint.

### ASR-PARAM-CONTRACT

Status: pending

Plan:

- `devnotes/v2.0.0/sprint-plans/asr-parameter-contract-sprint-plan.md`

Tracker:

- `devnotes/v2.0.0/sprint-trackers/asr-parameter-contract-sprint-tracker.md`

Required before frontend:

- [ ] Provider parameter descriptors can mark read-only values.
- [ ] `sherpa.model_type` is read-only for Parakeet models.
- [ ] Backend rejects changed read-only parameter values.

### ASR-LEGACY-CLEANUP

Status: pending

Plan:

- `devnotes/v2.0.0/sprint-plans/asr-legacy-profile-cleanup-sprint-plan.md`

Tracker:

- `devnotes/v2.0.0/sprint-trackers/asr-legacy-profile-cleanup-sprint-tracker.md`

Required before frontend:

- [ ] Legacy flat ASR profile migration/normalization code is removed.
- [ ] Canonical profile shape is `options.pipeline` only.
- [ ] Frontend has no legacy flat profile types to preserve.

## ASR-PROFILE-FE-Sprint 0: Contract Verification And Guardrails

Status: pending

Planned tasks:

- [ ] Add or update focused tests confirming the canonical model-card endpoint includes `parakeet-v2` for transcription capability.
- [ ] Add or update focused tests confirming the canonical model-card endpoint includes `parakeet-v3` for transcription capability.
- [ ] Verify both Parakeet model cards include `parameter_schema`, `recommended_defaults`, `chunking`, `dependencies`, and `artifacts`.
- [ ] Verify Parakeet parameter schema includes all common ASR parameters plus `sherpa.model_type`.
- [ ] Verify Whisper schema includes common ASR parameters plus Whisper-specific fields.
- [ ] Verify profile create/update rejects legacy flat fields and accepts `options.pipeline`.
- [ ] Inventory legacy flat profile compatibility code paths for deletion.
- [ ] Record backend contract gaps before frontend implementation.

Acceptance checks:

- [ ] The frontend can rely on the canonical capability-filtered model-card endpoint as the model-card source of truth.
- [ ] Parakeet TDT v2 and v3 are visible through the public model-card endpoint.
- [ ] Legacy flat ASR profile inputs have no supported save path.
- [ ] Dependency sprint contract gaps are fixed.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-PROFILE-FE-Sprint 1: Frontend API Contract Rewrite

Status: pending

Planned tasks:

- [ ] Update `profilesApi.ts` to model `TranscriptionModel` as the sanitized backend model card.
- [ ] Add typed `ParameterDescriptor`, `ParameterOption`, `ActivationRule`, and `ASRStep` types.
- [ ] Change `TranscriptionProfileOptions` to `{ pipeline: ASRStep[] }`.
- [ ] Remove frontend normalization for legacy flat ASR response fields.
- [ ] Save profiles by sending only `options.pipeline`.
- [ ] Keep `listTranscriptionModels` filtering by `capabilities.transcription`.

Acceptance checks:

- [ ] TypeScript no longer models active ASR profile settings as legacy flat fields.
- [ ] Save payloads match backend pipeline validation.
- [ ] Legacy flat field normalization is deleted.
- [ ] Model-card fields needed for dynamic rendering are preserved.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-PROFILE-FE-Sprint 2: Dynamic Parameter Form Core

Status: pending

Planned tasks:

- [ ] Add a feature-local component for rendering one model card's `parameter_schema`.
- [ ] Support `boolean`, `integer`, `number`, `string`, `enum`, `duration`, and `path_ref`.
- [ ] Use descriptor metadata: `label`, `default`, `min`, `max`, `step`, `options`, `scope`, `advanced`, `requires_reload`, and `visible_when`.
- [ ] Initialize values from existing step options, then `recommended_defaults`, then descriptor defaults.
- [ ] Implement `visible_when` evaluation for VAD-only controls.
- [ ] Group controls by scope.
- [ ] Add an Advanced disclosure for `advanced` parameters.

Acceptance checks:

- [ ] Selecting a model changes rendered controls from that model's descriptor.
- [ ] Unsupported model-specific controls disappear automatically.
- [ ] VAD settings appear only when `chunking.mode` is `vad`.
- [ ] Boolean, enum, and numeric fields use appropriate controls.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-PROFILE-FE-Sprint 3: Profile Dialog Revamp

Status: pending

Planned tasks:

- [ ] Keep profile name, description, and default toggle.
- [ ] Render a transcription step selector using transcription model cards.
- [ ] Save the first pipeline step as `kind`, `provider`, `model`, `model_family` if required, and `options`.
- [ ] Add an optional diarization step toggle.
- [ ] Render diarization parameters from a diarization model schema where available.
- [ ] Remove hard-coded language/task/thread/tail-padding/chunking/diarization controls from `ASRProfileDialog.tsx`.
- [ ] Add compact summaries for selected model, installed/download state, reload-required changes, and advanced fields.

Acceptance checks:

- [ ] The dialog exposes every parameter in the selected transcription model schema.
- [ ] Parakeet TDT v2/v3 show Parakeet-supported parameters and no Whisper-only parameters.
- [ ] Whisper models show Whisper-specific language/task/timestamp controls.
- [ ] Diarization is descriptor-driven or explicitly blocked pending a backend model-card endpoint.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-PROFILE-FE-Sprint 4: Profile List And Legacy Cleanup

Status: pending

Planned tasks:

- [ ] Update `ProfileRow` summaries to read from `options.pipeline`.
- [ ] Show transcription provider/model, key selected parameters, and diarization state.
- [ ] Use `parameter_schema.expose_in_summary` when model cards are available.
- [ ] Show a clear missing-model error for profiles whose stored model is no longer available.
- [ ] Treat profiles without a valid transcription pipeline step as invalid data, not frontend-repairable data.
- [ ] Remove remaining frontend reads of legacy flat fields from profile summaries and dialogs.
- [ ] Avoid mutating profile options during render.

Acceptance checks:

- [ ] Existing pipeline profiles display useful summaries.
- [ ] Missing model cards produce clear errors instead of blank dialogs.
- [ ] Profiles without a valid pipeline are not silently repaired or defaulted by the frontend.
- [ ] Profile rows no longer depend on legacy flat option names.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.

## ASR-PROFILE-FE-Sprint 5: Frontend Tests And Browser QA

Status: pending

Planned tasks:

- [ ] Add focused unit tests for model-card normalization.
- [ ] Add focused unit tests for pipeline save payload construction.
- [ ] Add focused unit tests for descriptor default resolution.
- [ ] Add focused unit tests for `visible_when` evaluation.
- [ ] Add focused unit tests for unsupported parameter stripping.
- [ ] Add component tests for Parakeet v2/v3 controls.
- [ ] Add component tests for Whisper controls.
- [ ] Add component tests for VAD advanced controls.
- [ ] Add component tests for diarization toggle behavior.
- [ ] Run production build and lint.
- [ ] Verify Settings > ASR at desktop and mobile widths.

Acceptance checks:

- [ ] The profile dialog can create and edit a Parakeet TDT v2 or v3 profile.
- [ ] The save payload contains only descriptor-supported keys under `pipeline[].options`.
- [ ] No unsupported legacy flat fields are sent.
- [ ] Controls fit on desktop and mobile without overlap.

Verification:

- [ ] Pending.

Artifacts:

- Pending.

Commit:

- Pending.
