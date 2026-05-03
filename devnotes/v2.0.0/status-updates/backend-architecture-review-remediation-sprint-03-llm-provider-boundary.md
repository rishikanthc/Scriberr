# Backend Architecture Review Remediation Sprint 03 LLM Provider Boundary

Date: 2026-05-03

## Scope

Sprint 3 addresses Finding 6: concrete external LLM provider probing lived in `internal/api`.

## Changes

- Moved OpenAI-compatible `/models` probing and Ollama `/api/tags` fallback probing into `internal/llmprovider`.
- Added `llmprovider.HTTPConnectionTester` as the concrete service adapter.
- Updated `internal/app` composition to wire `llmprovider.Service` with `llmprovider.HTTPConnectionTester`.
- Kept API handlers focused on auth, validation, service calls, response mapping, and events.
- Added an API architecture guard preventing provider probing symbols from returning to production API code.
- Moved provider probing behavior tests into `internal/llmprovider`.

## TDD Evidence

The new architecture guard failed before the move:

```txt
TestProductionAPIDoesNotOwnLLMProviderConnectionTester:
production API owns LLM provider probing symbol "LLMProviderConnectionTester" in llm_provider_handlers.go
```

After moving the implementation, the guard passed.

## Verification

Passed:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/llmprovider ./internal/api -run 'TestHTTPConnectionTester|TestLLMProvider|TestProduction|TestBackendDependencyDirection'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/app ./cmd/server
git diff --check
```

## Notes

The LLM provider/API verification command was rerun with loopback allowed because sandboxed `httptest.NewServer` binds are denied in this environment.
