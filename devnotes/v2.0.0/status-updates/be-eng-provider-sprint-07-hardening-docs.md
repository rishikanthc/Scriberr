# BE-ENG-PROVIDER Sprint 7: Hardening And Documentation

Status: complete.

## Final Architecture Guardrails

- Local sherpa execution stays in-process through the `scriberr-engine` Go API.
- Backend providers expose model cards through `Models()`; selectable capabilities are derived by the registry.
- Execution methods are task-specific provider interfaces.
- Profiles store ordered `pipeline` steps with descriptor-keyed `options`.
- Backend validates step options against provider model descriptors.
- Backend does not pre-chunk ASR audio; local long-form planning belongs to `scriberr-engine`, and external providers own their remote decode plan.
- Execution metadata may preserve provider/engine metrics and plan summaries, but must remain sanitized.

## Updated Docs

- `devnotes/v2.0.0/specs/asr-provider-author-guide.md`
- `devnotes/v2.0.0/specs/asr-provider-backend-architecture.md`
- `references/engine/devnotes/specs/engine-provider-architecture.md`
- `references/engine/devnotes/rules/engine-core-rules.md`

## Verification

- `GOCACHE=/private/tmp/scriberr-go-cache go test ./...`
- `GOCACHE=/private/tmp/scriberr-go-cache go vet ./internal/transcription/... ./internal/profile ./internal/api`
- `GOCACHE=/private/tmp/scriberr-engine-go-cache go test ./...`
- `GOCACHE=/private/tmp/scriberr-engine-go-cache go vet ./...`
- `git diff --check`
