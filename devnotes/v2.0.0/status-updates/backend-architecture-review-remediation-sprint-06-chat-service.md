# Backend Architecture Review Remediation Sprint 06 Chat Service

Date: 2026-05-03

## Scope

Sprint 6 addresses:

- Finding 5: chat generation ran inside the HTTP handler.

## Changes

- Moved chat model listing, model availability checks, LLM client construction, context assembly, streaming, message persistence, and generation-run terminal handling into `internal/chat`.
- Added a service-level streaming test with a fake LLM client.
- Kept the API handler focused on auth/session lookup, request binding, error mapping, and SSE serialization.
- Removed production API imports of `internal/llm`.
- Added an architecture guard that fails if production API code imports `internal/llm`.

## TDD Evidence

The new chat service streaming test failed before implementation because the service did not expose the streaming workflow:

```txt
service.SetLLMClientFactory undefined
service.StreamMessage undefined
StreamMessageCommand undefined
```

After implementation, the service test and existing API chat lifecycle tests passed.

## Verification

Passed:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/chat ./internal/api -run 'TestChat|TestProduction|TestBackendDependencyDirection'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/chat
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/llm ./internal/llmprovider ./cmd/server
git diff --check
```

## Notes

The first non-escalated provider verification hit the sandbox local-port restriction in `httptest.NewServer`. The same command passed after rerunning with permission to bind local test ports.
