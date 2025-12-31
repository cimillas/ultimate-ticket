# API service (modular monolith)

This folder will host the Go module for the initial modular monolith.
Initial layout:

- Module path: `github.com/cimillas/ultimate-ticket/services/api`
- `cmd/api/` — entrypoint
- `internal/domain/` — domain model and invariants
- `internal/app/` — application services/use cases
- `internal/storage/postgres/` — storage adapters
- `internal/transport/http/` — HTTP handlers
- `internal/clock/` — time abstractions
- `migrations/` — database migrations

Domain concepts reference: `docs/concepts.md`

Run locally:
```bash
cd services/api
go test ./...
go run ./cmd/api
```

Or from repo root:
```bash
make test
make run
make fmt
make vet
make tidy
make lint
make build
```

Configuration:
- `PORT` (default: `8080`)
- `DATABASE_URL` (default: `postgres://ultimate_ticket:ultimate_ticket@localhost:5432/ultimate_ticket?sslmode=disable`)
- `CORS_ORIGINS` (comma-separated allow list, e.g. `http://localhost:5173`)

The API loads `.env` automatically when present (current dir or parent directories).

Endpoints:
- `GET /health` → `ok`
- `POST /holds` with JSON `{event_id, zone_id, quantity, idempotency_key}`; returns `201` with hold data or `409` on capacity/idempotency conflict.
- `POST /holds/{id}/confirm` with header `Idempotency-Key`; returns `201` or `200` on idempotent retry.
- Admin (local tooling only):
  - `POST /admin/events` + `GET /admin/events`
  - `POST /admin/events/{event_id}/zones` + `GET /admin/events/{event_id}/zones`

Error format:
```json
{"error":"<message>","code":"<code>"}
```

Full reference: `docs/api/error-codes.md`

Migrations:
- Applied on startup and recorded in `schema_migrations`.
