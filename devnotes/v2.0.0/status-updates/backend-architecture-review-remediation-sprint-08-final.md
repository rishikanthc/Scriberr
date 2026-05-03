# Backend Architecture Review Remediation Sprint 08 Final

Date: 2026-05-03

## Scope

Sprint 8 closes the backend architecture review remediation series.

## Changes

- Tightened architecture guard names and failure messages from inventory language to boundary enforcement:
  - production API must not import `internal/database`;
  - only app composition may import `internal/database`;
  - only app composition may import `internal/api`.
- Kept the existing hard guards for:
  - service dependency direction;
  - API-owned LLM provider probing;
  - production API imports of `internal/llm`;
  - production imports of legacy `internal/queue`;
  - product-service usage of unscoped job lookups.
- Fixed a stale settings test expectation so it matches the registered-user default that the adjacent automation settings test already asserts.

## Completed Commits

```txt
6e745c1 backend: plan architecture review remediation
3764c88 backend: enforce admin authorization
8b08d11 backend: scope sse events by user
f1896bf backend: move llm provider probing out of api
af5d8fa backend: protect llm provider credentials
c1ca7ab backend: enforce queue lease ownership
4fee25b backend: move chat generation into service
0860810 backend: quarantine legacy repository paths
```

## Verification

Passed:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestProduction|TestOnlyAppComposition|TestBackendDependencyDirection|TestSettingsPartialUpdateAndValidation'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api ./internal/account ./internal/auth ./internal/chat ./internal/config ./internal/database ./internal/files ./internal/llm ./internal/llmprovider ./internal/recording ./internal/repository ./internal/summarization ./internal/transcription/... ./cmd/server
git diff --check
```

## Residual Debt

- The generic repository base type still exists for repository-internal reuse and non-user-owned system paths.
- The architecture guard for product-service unscoped job lookups is symbol-based, not a full data-flow proof. It protects the known pattern that caused this sprint's finding.
- `references/` and `test-audio/` remain unrelated untracked local paths and were not modified.
