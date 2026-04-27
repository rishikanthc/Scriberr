# Sprint 6 Notes

## Scope

Finalized the first-pass API surface with contract tests and canonical API docs.

## Decisions

- No temporary legacy singular transcription aliases were added.
- `docs/api/openapi.json` is now the canonical API documentation artifact for this pass.
- Stale generated `docs/api/swagger.json` and `docs/api/undocumented.json` were removed because they documented deleted routes.
- The `web/project-site/public/api` copies were not modified because they are outside the API-only scope.
- Route contract tests now print endpoint names as `method path` subtests under `go test -v`.

## Verification

- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api -run 'TestCanonicalRouteRegistration|TestEndpointContractSmoke|TestAPIDocsContainOnlyCanonicalRoutes'`
- `GOCACHE=/tmp/scriberr-go-cache go test ./internal/api ./cmd/server ./pkg/logger ./pkg/middleware`
- `git diff --check`
