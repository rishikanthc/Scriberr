# Sprint Run: ASR Parameter Contract Hardening

Run ID: `ASR-PARAM-CONTRACT`

Status: planning only. Do not implement code from this document until the user starts an implementation sprint.

Date: 2026-05-08

## Goal

Make provider parameter mutability explicit in the engine/backend/frontend contract so descriptor-driven UIs can expose every parameter without making internal model identity values editable. This specifically covers Parakeet's `sherpa.model_type`.

## Current State

- Parakeet descriptors advertise `sherpa.model_type` with default `nemo_transducer`.
- That value selects the sherpa recognizer config path and should not be user-editable for a selected Parakeet model.
- Current parameter descriptors expose `advanced` and `requires_reload`, but not read-only/mutability.
- A generic frontend form would render `sherpa.model_type` as an editable string unless special-cased.

## Target Direction

- Add a first-class read-only signal to provider parameter descriptors.
- Mark `sherpa.model_type` read-only in the engine descriptor.
- Preserve the field in model cards so the UI can expose it as read-only metadata.
- Reject profile/job options that try to change read-only parameters away from descriptor values.
- Avoid frontend key-specific behavior except as a short-lived implementation bridge inside this sprint if sequencing requires it.

## Engineering Rules

- Engine remains the source of truth for local parameter descriptors.
- Backend validation enforces descriptor semantics.
- Frontend rendering follows descriptor semantics.
- Do not hide read-only parameters just because they are not editable.
- Do not add model-family conditionals in API/profile/orchestrator code.

## Validation Baseline

```sh
(cd references/engine && GOCACHE=/tmp/scriberr-engine-go-cache go test ./...)
GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/asrcontract ./internal/transcription/engineprovider ./internal/profile ./internal/api
npm --prefix web/frontend run build
git diff --check
```

## ASR-PARAM-CONTRACT-Sprint 0: Contract Design And Engine Descriptor Tests

Goal: define parameter mutability in the provider contract and prove Parakeet marks `sherpa.model_type` read-only.

Tasks:

- Add a read-only/mutability field to engine `ParameterDescriptor`.
- Add descriptor validation tests for the new field.
- Mark `sherpa.model_type` as read-only in Parakeet descriptors.
- Add tests confirming Parakeet v2/v3 descriptors expose `sherpa.model_type` as read-only.
- Confirm Whisper editable parameters remain editable unless explicitly marked read-only.

Acceptance criteria:

- Engine descriptor schema can express read-only parameters.
- Parakeet model type is exposed but not editable by contract.

## ASR-PARAM-CONTRACT-Sprint 1: Backend Model Card Mapping And Validation

Goal: carry read-only semantics into backend model cards and enforce them at profile validation.

Tasks:

- Add the read-only field to `asrcontract.ParameterDescriptor`.
- Map engine descriptor read-only metadata in the local provider adapter.
- Update parameter schema validation to accept the new field.
- Update profile option validation to reject changed read-only parameter values.
- Allow omitted read-only parameters and exact descriptor default/recommended values.
- Add tests for valid omitted/default `sherpa.model_type` and invalid changed values.

Acceptance criteria:

- Backend model cards expose read-only metadata.
- Backend rejects attempts to change `sherpa.model_type`.
- No profile save path relies on frontend-only enforcement.

## ASR-PARAM-CONTRACT-Sprint 2: Frontend Renderer Support

Goal: let the descriptor-driven form render read-only parameters correctly.

Tasks:

- Add `read_only` or equivalent to frontend `ParameterDescriptor` type.
- Render read-only parameters disabled or as metadata values.
- Include read-only parameters in advanced sections when the descriptor exposes them.
- Avoid submitting changed read-only values.
- Add tests for read-only string parameters.

Acceptance criteria:

- The frontend can expose `sherpa.model_type` while preventing edits.
- The generic parameter renderer supports read-only descriptors without key-specific logic.

## Commit Plan

1. Engine descriptor mutability support.
2. Backend model-card mapping and validation.
3. Frontend renderer support.
