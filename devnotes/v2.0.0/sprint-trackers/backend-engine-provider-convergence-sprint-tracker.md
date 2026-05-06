# Sprint Run Tracker: Backend Engine Provider Convergence

Run ID: `BE-ENG-PROVIDER`

Status: in progress.

This tracker belongs to `devnotes/v2.0.0/sprint-plans/backend-engine-provider-convergence-sprint-plan.md`.

## Run Rules

- Follow the sprint plan engineering rules.
- Keep implementation sprints independently reviewable.
- Keep the bundled sherpa-onnx provider connected through direct Go API calls.
- Do not add REST between Scriberr backend and `scriberr-engine`.
- Remove redundant legacy code instead of preserving compatibility.
- Update this tracker in the same change set as each completed sprint.

## Validation Checklist

Before closing each implementation sprint when practical:

- [ ] Focused tests for the changed package.
- [ ] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/... ./internal/profile ./internal/api`
- [ ] `GOCACHE=/tmp/scriberr-go-cache go vet ./internal/transcription/... ./internal/profile ./internal/api`
- [ ] `GOCACHE=/tmp/scriberr-engine-go-cache go test ./...` in `references/engine`
- [ ] `git diff --check`
- [ ] Architecture import checks.

## BE-ENG-PROVIDER-Sprint 0: Contract Duplication Inventory

Status: complete

Planned tasks:

- [x] Inventory duplicated backend/engine provider contracts.
- [x] Confirm existing import guardrails for the local direct-Go engine adapter.
- [x] Document final target contract shape.

Acceptance checks:

- [x] Duplication map exists.
- [x] Existing architecture guardrails cover direct-Go import boundaries.
- [x] No runtime behavior changes.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider` fails at compile time because the backend adapter still targets removed engine request fields. This is the intentional Sprint 1 cleanup target, documented in `devnotes/v2.0.0/status-updates/be-eng-provider-sprint-00-contract-inventory.md`.

Commit:

- Pending.

## BE-ENG-PROVIDER-Sprint 1: Backend Request Contract Collapse

Status: complete

Planned tasks:

- [x] Replace backend transcription request typed model fields with `Parameters map[string]any`.
- [x] Replace backend diarization request typed tuning fields with `Parameters map[string]any`.
- [x] Pass pipeline step options directly to local engine provider.
- [x] Remove active use of legacy flat ASR fields in execution.

Acceptance checks:

- [x] Backend compiles against current engine API.
- [x] Local provider uses `speechengine.TranscriptionRequest.Parameters`.
- [x] No new execution path uses legacy flat ASR fields.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api`
- [x] `GOCACHE=/tmp/scriberr-engine-go-cache go test ./...` in `references/engine`

Commit:

- Pending.

## BE-ENG-PROVIDER-Sprint 2: Descriptor Passthrough For Local Provider

Status: complete

Planned tasks:

- [x] Expose full engine descriptors or descriptor-equivalent model cards.
- [x] Map engine descriptors mechanically into backend `asrcontract.ModelCard`.
- [x] Delete local backend schema/default synthesis.

Acceptance checks:

- [x] Local model cards match engine descriptors.
- [x] Whisper and Parakeet options/defaults come from engine.
- [x] Backend no longer hardcodes local sherpa model schemas.

Verification:

- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/...`
- [x] `GOCACHE=/tmp/scriberr-go-cache go test ./internal/profile ./internal/api`
- [x] `GOCACHE=/tmp/scriberr-engine-go-cache go test ./...` in `references/engine`

Commit:

- Pending.

## BE-ENG-PROVIDER-Sprint 3: Delete Backend Chunking And Batching Planner

Status: planned

Planned tasks:

- [ ] Remove local execution chunk/batch planning from orchestrator.
- [ ] Keep pipeline sequencing only.
- [ ] Persist provider/engine plan summary.
- [ ] Remove duplicated boundary progress events.

Acceptance checks:

- [ ] Backend does not compute local chunk count.
- [ ] Backend does not choose fixed vs VAD for local engine execution.
- [ ] Engine progress drives transcription progress.

Verification:

- [ ] Not started.

Commit:

- Pending.

## BE-ENG-PROVIDER-Sprint 4: Canonical Result And Metrics Alignment

Status: planned

Planned tasks:

- [ ] Preserve engine metrics and plan summary in execution metadata.
- [ ] Delete local metric recomputation from backend provider adapter.
- [ ] Keep canonical transcript merge in orchestrator.

Acceptance checks:

- [ ] Execution metadata uses engine metrics where available.
- [ ] Backend does not infer local audio duration from words/segments.
- [ ] Transcript JSON remains stable.

Verification:

- [ ] Not started.

Commit:

- Pending.

## BE-ENG-PROVIDER-Sprint 5: Profile Parameter Model Cleanup

Status: planned

Planned tasks:

- [ ] Make profiles store pipeline step options as descriptor-keyed maps.
- [ ] Validate against provider model descriptors.
- [ ] Remove active flat `ASRParams` execution usage.
- [ ] Return schemas for frontend dynamic ASR controls.

Acceptance checks:

- [ ] Frontend can render ASR profile controls from descriptors.
- [ ] Unsupported options fail validation.
- [ ] Adding local model support does not require backend schema edits.

Verification:

- [ ] Not started.

Commit:

- Pending.

## BE-ENG-PROVIDER-Sprint 6: Provider Interface Slimming

Status: planned

Planned tasks:

- [ ] Split provider capabilities by task interface.
- [ ] Remove redundant `Capabilities()` if covered by `Models()`.
- [ ] Remove or justify `Prepare()`.
- [ ] Keep local Go provider separate from future remote REST adapter.

Acceptance checks:

- [ ] Transcription-only provider interface is small.
- [ ] Providers implement only advertised task methods.
- [ ] Registry selection remains capability-driven.

Verification:

- [ ] Not started.

Commit:

- Pending.

## BE-ENG-PROVIDER-Sprint 7: Hardening And Documentation

Status: planned

Planned tasks:

- [ ] Update provider specs and author guide.
- [ ] Update backend and engine architecture docs.
- [ ] Add final architecture guardrails.
- [ ] Run full validation.

Acceptance checks:

- [ ] Docs match implementation.
- [ ] Local sherpa path is direct Go API only.
- [ ] Remote provider extension path is clear.

Verification:

- [ ] Not started.

Commit:

- Pending.
