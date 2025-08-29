API documentation workflow

- Source of truth: handler annotations in `internal/api/*.go` (`@Summary`, `@Description`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Router`, etc.).
- Generator: swag (github.com/swaggo/swag) parses annotations and emits Swagger/OpenAPI JSON/YAML.
- Landing site: reads the static spec at `/api/swagger.json` and renders a searchable, developer‑friendly reference with parameters, request bodies, responses, curl examples, permalinks, and copy‑to‑clipboard.

Regenerate the spec

1) Install swag (one time):

   go install github.com/swaggo/swag/cmd/swag@latest

2) Generate into `docs/` from the server entrypoint:

   swag init -g cmd/server/main.go -o docs

3) Run the landing site (copies the spec on dev/build):

   cd web/landing && npm run dev

Notes

- The landing app copies `docs/swagger.json` to `web/landing/public/api/swagger.json` via `npm run sync:spec` (invoked on `dev` and `build`).
- The renderer supports Swagger 2.0 and OpenAPI 3.0. For Swagger 2.0, request bodies and responses fall back to `parameters[in=body]` and `responses[*].schema` automatically.
- Keep annotations up to date when adding/changing endpoints. Tags control grouping in the sidebar.

