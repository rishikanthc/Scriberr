# Sprint Run Tracker: ASR Provider Profile Frontend Convergence

Run ID: `ASR-PROFILE-FE`

Status: completed through ASR-PROFILE-FE-Sprint 4.

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

- [x] Frontend-accessible model-card endpoint returns transcription and diarization models by capability.
- [x] `diarization-default` exposes its parameter schema through that endpoint.

### ASR-PARAM-CONTRACT

Status: completed

Plan:

- `devnotes/v2.0.0/sprint-plans/asr-parameter-contract-sprint-plan.md`

Tracker:

- `devnotes/v2.0.0/sprint-trackers/asr-parameter-contract-sprint-tracker.md`

Required before frontend:

- [x] Provider parameter descriptors can mark read-only values.
- [x] `sherpa.model_type` is read-only for Parakeet models.
- [x] Backend rejects changed read-only parameter values.

### ASR-LEGACY-CLEANUP

Status: completed

Plan:

- `devnotes/v2.0.0/sprint-plans/asr-legacy-profile-cleanup-sprint-plan.md`

Tracker:

- `devnotes/v2.0.0/sprint-trackers/asr-legacy-profile-cleanup-sprint-tracker.md`

Required before frontend:

- [x] Legacy flat ASR profile migration/normalization code is removed.
- [x] Canonical profile shape is `options.pipeline` only.
- [x] Frontend has no legacy flat profile types to preserve.

## ASR-PROFILE-FE-Sprint 0: Contract Verification And Guardrails

Status: completed

Planned tasks:

- [x] Add or update focused tests confirming the canonical model-card endpoint includes `parakeet-v2` for transcription capability.
- [x] Add or update focused tests confirming the canonical model-card endpoint includes `parakeet-v3` for transcription capability.
- [x] Verify both Parakeet model cards include `parameter_schema`, `recommended_defaults`, `chunking`, `dependencies`, and `artifacts`.
- [x] Verify Parakeet parameter schema includes all common ASR parameters plus `sherpa.model_type`.
- [x] Verify Whisper schema includes common ASR parameters plus Whisper-specific fields.
- [x] Verify profile create/update rejects legacy flat fields and accepts `options.pipeline`.
- [x] Inventory legacy flat profile compatibility code paths for deletion.
- [x] Record backend contract gaps before frontend implementation.

Acceptance checks:

- [x] The frontend can rely on the canonical capability-filtered model-card endpoint as the model-card source of truth.
- [x] Parakeet TDT v2 and v3 are visible through the public model-card endpoint.
- [x] Legacy flat ASR profile inputs have no supported save path.
- [x] Dependency sprint contract gaps are fixed.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestASRModelCatalogEndpointFiltersCapabilities|TestProfileCRUDAndDefaultSelection|TestProfileValidationAndAuth'`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider -run 'TestLocalProviderModelDescriptor'`
- [x] `git diff --check -- internal/api/engine_worker_api_test.go internal/transcription/engineprovider/local_provider_test.go devnotes/v2.0.0/sprint-trackers/asr-profile-provider-frontend-sprint-tracker.md`

Artifacts:

- `internal/api/engine_worker_api_test.go`
- `internal/transcription/engineprovider/local_provider_test.go`
- `devnotes/v2.0.0/sprint-trackers/asr-profile-provider-frontend-sprint-tracker.md`

Commit:

- Pending.

## ASR-PROFILE-FE-Sprint 1: Frontend API Contract Rewrite

Status: completed

Planned tasks:

- [x] Update `profilesApi.ts` to model `TranscriptionModel` as the sanitized backend model card.
- [x] Add typed `ParameterDescriptor`, `ParameterOption`, `ActivationRule`, and `ASRStep` types.
- [x] Change `TranscriptionProfileOptions` to `{ pipeline: ASRStep[] }`.
- [x] Remove frontend normalization for legacy flat ASR response fields.
- [x] Save profiles by sending only `options.pipeline`.
- [x] Keep `listTranscriptionModels` filtering by `capabilities.transcription`.

Acceptance checks:

- [x] TypeScript no longer models active ASR profile settings as legacy flat fields.
- [x] Save payloads match backend pipeline validation.
- [x] Legacy flat field normalization is deleted.
- [x] Model-card fields needed for dynamic rendering are preserved.

Verification:

- [x] `npm --prefix web/frontend run build`
- [x] `npm --prefix web/frontend run lint` (passes with existing warnings outside this sprint)
- [x] `git diff --check -- web/frontend/src/features/settings/api/profilesApi.ts web/frontend/src/features/settings/components/ASRProfileDialog.tsx devnotes/v2.0.0/sprint-trackers/asr-profile-provider-frontend-sprint-tracker.md`

Artifacts:

- `web/frontend/src/features/settings/api/profilesApi.ts`
- `web/frontend/src/features/settings/components/ASRProfileDialog.tsx`
- `devnotes/v2.0.0/sprint-trackers/asr-profile-provider-frontend-sprint-tracker.md`

Commit:

- Pending.

## ASR-PROFILE-FE-Sprint 2: Dynamic Parameter Form Core

Status: completed

Planned tasks:

- [x] Add a feature-local component for rendering one model card's `parameter_schema`.
- [x] Support `boolean`, `integer`, `number`, `string`, `enum`, `duration`, and `path_ref`.
- [x] Use descriptor metadata: `label`, `default`, `min`, `max`, `step`, `options`, `scope`, `advanced`, `requires_reload`, and `visible_when`.
- [x] Initialize values from existing step options, then `recommended_defaults`, then descriptor defaults.
- [x] Implement `visible_when` evaluation for VAD-only controls.
- [x] Group controls by scope.
- [x] Add an Advanced disclosure for `advanced` parameters.

Acceptance checks:

- [x] Selecting a model changes rendered controls from that model's descriptor.
- [x] Unsupported model-specific controls disappear automatically.
- [x] VAD settings appear only when `chunking.mode` is `vad`.
- [x] Boolean, enum, and numeric fields use appropriate controls.

Verification:

- [x] `npm --prefix web/frontend run build`
- [x] `npm --prefix web/frontend run lint` (passes with existing warnings outside this sprint)
- [x] `git diff --check -- web/frontend/src/features/settings/components/ASRParameterForm.tsx web/frontend/src/features/settings/components/asrParameterValues.ts devnotes/v2.0.0/sprint-trackers/asr-profile-provider-frontend-sprint-tracker.md`

Artifacts:

- `web/frontend/src/features/settings/components/ASRParameterForm.tsx`
- `web/frontend/src/features/settings/components/asrParameterValues.ts`
- `devnotes/v2.0.0/sprint-trackers/asr-profile-provider-frontend-sprint-tracker.md`

Commit:

- Pending.

## ASR-PROFILE-FE-Sprint 3: Profile Dialog Revamp

Status: completed

Planned tasks:

- [x] Keep profile name, description, and default toggle.
- [x] Render a transcription step selector using transcription model cards.
- [x] Save the first pipeline step as `kind`, `provider`, `model`, `model_family` if required, and `options`.
- [x] Add an optional diarization step toggle.
- [x] Render diarization parameters from a diarization model schema where available.
- [x] Remove hard-coded language/task/thread/tail-padding/chunking/diarization controls from `ASRProfileDialog.tsx`.
- [x] Add compact summaries for selected model, installed/download state, reload-required changes, and advanced fields.

Acceptance checks:

- [x] The dialog exposes every parameter in the selected transcription model schema.
- [x] Parakeet TDT v2/v3 show Parakeet-supported parameters and no Whisper-only parameters.
- [x] Whisper models show Whisper-specific language/task/timestamp controls.
- [x] Diarization is descriptor-driven or explicitly blocked pending a backend model-card endpoint.

Verification:

- [x] `npm --prefix web/frontend run build`
- [x] `npm --prefix web/frontend run lint` (passes with existing warnings outside this sprint)
- [x] `git diff --check -- web/frontend/src/features/settings/components/ASRProfileDialog.tsx web/frontend/src/features/settings/pages/SettingsPage.tsx devnotes/v2.0.0/sprint-trackers/asr-profile-provider-frontend-sprint-tracker.md`

Artifacts:

- `web/frontend/src/features/settings/components/ASRProfileDialog.tsx`
- `web/frontend/src/features/settings/pages/SettingsPage.tsx`
- `devnotes/v2.0.0/sprint-trackers/asr-profile-provider-frontend-sprint-tracker.md`

Commit:

- Pending.

## ASR-PROFILE-FE-Sprint 4: Profile List And Legacy Cleanup

Status: completed

Planned tasks:

- [x] Update `ProfileRow` summaries to read from `options.pipeline`.
- [x] Show transcription provider/model, key selected parameters, and diarization state.
- [x] Use `parameter_schema.expose_in_summary` when model cards are available.
- [x] Show a clear missing-model error for profiles whose stored model is no longer available.
- [x] Treat profiles without a valid transcription pipeline step as invalid data, not frontend-repairable data.
- [x] Remove remaining frontend reads of legacy flat fields from profile summaries and dialogs.
- [x] Avoid mutating profile options during render.

Acceptance checks:

- [x] Existing pipeline profiles display useful summaries.
- [x] Missing model cards produce clear errors instead of blank dialogs.
- [x] Profiles without a valid pipeline are not silently repaired or defaulted by the frontend.
- [x] Profile rows no longer depend on legacy flat option names.

Verification:

- [x] `npm --prefix web/frontend run build`
- [x] `npm --prefix web/frontend run lint` (passes with existing warnings outside this sprint)
- [x] `rg -n "profile\.options\.(model|language|task|diarize|chunking_strategy)|tail_paddings|decoding_method|chunking_strategy|defaultProfileParams|normalizeParams|familyForModel" web/frontend/src/features/settings -g '*.ts' -g '*.tsx'`
- [x] `git diff --check -- web/frontend/src/features/settings/pages/SettingsPage.tsx devnotes/v2.0.0/sprint-trackers/asr-profile-provider-frontend-sprint-tracker.md`

Artifacts:

- `web/frontend/src/features/settings/pages/SettingsPage.tsx`
- `devnotes/v2.0.0/sprint-trackers/asr-profile-provider-frontend-sprint-tracker.md`

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
