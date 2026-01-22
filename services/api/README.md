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
make test # requires API running and ALLOW_PUBLIC_REGISTER=true for E2E
make backend-test
make backend-run
make backend-fmt
make backend-vet
make backend-tidy
make backend-lint
make backend-build
make backend-auth-bootstrap
APP_ENV=local CONFIRM=YES make backend-auth-reset
ALLOW_PUBLIC_REGISTER=true make backend-e2e
```

Configuration:
- Env file: `services/api/.env` (copy from `services/api/.env.example`)
- `APP_ENV` (set to `local` to allow auth resets)
- `PORT` (default: `8080`)
- `DATABASE_URL` (default: `postgres://ultimate_ticket:ultimate_ticket@localhost:5432/ultimate_ticket?sslmode=disable`)
- `CORS_ORIGINS` (comma-separated allow list, e.g. `http://localhost:5173`)
- `SESSION_TTL` (default: `1h`)
- `ADMIN_USERNAME` (used by auth bootstrap/reset)
- `ADMIN_PASSWORD` (used by auth bootstrap/reset)
- `ADMIN_EMAIL` (used by auth bootstrap/reset)
- `SESSION_COOKIE_SECURE` (default: `false` in `APP_ENV=local`, `true` otherwise)
- `ALLOW_PUBLIC_REGISTER` (default: `true` in `APP_ENV=local`, `false` otherwise)

The API loads `.env` automatically when present (current dir or parent directories).
For local runs, keep `services/api/.env` alongside the Go module.

Endpoints:
- `GET /health` → `ok`
- Auth:
  - `POST /auth/register` with JSON `{username, email, password}`
  - `POST /auth/login` with JSON `{identifier, password}`
  - `POST /auth/logout`
  - `POST /auth/password` with JSON `{current_password, new_password}` (requires login)
  - `GET /me`
  - Note: usernames cannot contain `@` (use the email field instead).
  - Public registration can be disabled via `ALLOW_PUBLIC_REGISTER=false`.
  - Bootstrap admin (required for admin panel):
    - `make backend-auth-bootstrap`
  - Reset auth (local only):
    - `APP_ENV=local CONFIRM=YES make backend-auth-reset`
- Public:
  - `GET /events`
  - `GET /events/{event_id}/zones`
- Protected:
  - `POST /holds` with JSON `{event_id, zone_id, quantity, idempotency_key}` (requires login)
  - `GET /holds` returns active holds for the current user (requires login)
  - `GET /orders` returns orders for the current user (requires login)
  - `POST /holds/{id}/confirm` with header `Idempotency-Key` (requires login)
- Admin (local tooling only, requires admin login):
  - `POST /admin/events` + `GET /admin/events`
  - `POST /admin/events/{event_id}/cancel`
  - `POST /admin/events/{event_id}/zones` + `GET /admin/events/{event_id}/zones`

Error format:
```json
{"error":"<message>","code":"<code>"}
```

Full reference: `docs/api/error-codes.md`

Logging:
- Each request logs a single JSON line with `ts`, `request_id`, `method`, `path`, `status`, `duration_ms`, `bytes`, `remote_ip`, and `user_agent`.
- `request_id` is taken from `X-Request-Id` or generated if missing; the response echoes `X-Request-Id`.
Example:
```json
{"ts":"2026-01-07T22:49:06Z","request_id":"7b6c2d...","method":"GET","path":"/events","status":200,"duration_ms":12,"bytes":123,"remote_ip":"127.0.0.1:54321","user_agent":"curl/8.1.0"}
```

Migrations:
- Applied on startup and recorded in `schema_migrations`.
