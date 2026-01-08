# Ultimate-ticket

High-scale ticketing prototype focused on:
- Zone-based inventory (no seat selection)
- Holds with TTL
- Eventually: waiting room, payments, anti-bot, SRE hardening

## Domain concepts (short)
- Event: the ticketed experience (concert/show), groups zones, and can be complete when all zones are sold out.
- Zone: a sellable area within an event, with a fixed capacity and derived availability.
- Hold: temporary reservation of tickets in a zone (has TTL); blocked once an event is closed after start.
- Confirmation (order): finalizes a hold into a purchase; rejected once the event is closed.

Full reference: `docs/concepts.md`

## Quickstart (local)
Requirements:
- Docker + Docker Compose
- Go 1.22+ (optional if you run only via Docker)

Run:
```bash
docker compose -f deployments/local/docker-compose.yml up --build
```

Services:
- Postgres on `localhost:5432` (user/password/db: `ultimate_ticket`)

API (default config)
- Base URL: `http://localhost:8080`
- Env:
  - `APP_ENV` (set to `local` to allow auth resets)
  - `PORT` (default: `8080`)
  - `DATABASE_URL` (default: `postgres://ultimate_ticket:ultimate_ticket@localhost:5432/ultimate_ticket?sslmode=disable`)
  - `CORS_ORIGINS` (comma-separated, e.g. `http://localhost:5173`)
  - `SESSION_TTL` (default: `1h`)
  - `ADMIN_USERNAME` (used by auth bootstrap/reset)
  - `ADMIN_PASSWORD` (used by auth bootstrap/reset)
  - `ADMIN_EMAIL` (used by auth bootstrap/reset)
  - `SESSION_COOKIE_SECURE` (default: `false` in `APP_ENV=local`, `true` otherwise)
  - `ALLOW_PUBLIC_REGISTER` (default: `true` in `APP_ENV=local`, `false` otherwise)
- Endpoints:
  - `GET /health` → `ok`
  - Auth:
    - `POST /auth/register` with JSON `{username, email, password}`
    - `POST /auth/login` with JSON `{identifier, password}`
    - `POST /auth/logout`
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
    - `POST /holds/{id}/confirm` with header `Idempotency-Key` (requires login)
  - Admin (local tooling only, requires admin login):
    - `POST /admin/events` + `GET /admin/events`
    - `POST /admin/events/{event_id}/cancel`
    - `POST /admin/events/{event_id}/zones` + `GET /admin/events/{event_id}/zones`
    - `GET /admin/events/{event_id}/zones/{zone_id}/holds`
    - `GET /admin/events/{event_id}/zones/{zone_id}/orders`

Migrations:
- Applied on startup and recorded in `schema_migrations`.

Manual test examples (curl)
```bash
# Login (stores session cookie)
curl -s -c cookies.txt -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"admin","password":"admin"}'
```

```bash
# Create hold (201)
curl -s -b cookies.txt -X POST http://localhost:8080/holds \
  -H 'Content-Type: application/json' \
  -d '{"event_id":"EVENT_ID","zone_id":"ZONE_ID","quantity":2,"idempotency_key":"hold-req-1"}'
```
Expected response (201):
```json
{"id":"<hold_id>","status":"active","expires_at":"<expires_at>"}
```

```bash
# Idempotent retry (201, same hold_id)
curl -s -b cookies.txt -X POST http://localhost:8080/holds \
  -H 'Content-Type: application/json' \
  -d '{"event_id":"EVENT_ID","zone_id":"ZONE_ID","quantity":2,"idempotency_key":"hold-req-1"}'
```
Expected response (201):
```json
{"id":"<hold_id>","status":"active","expires_at":"<expires_at>"}
```

```bash
# Idempotency conflict (409)
curl -s -b cookies.txt -X POST http://localhost:8080/holds \
  -H 'Content-Type: application/json' \
  -d '{"event_id":"EVENT_ID","zone_id":"ZONE_ID","quantity":3,"idempotency_key":"hold-req-1"}'
```
Expected response (409):
```json
{"error":"idempotency conflict","code":"idempotency_conflict"}
```

```bash
# Confirm hold (201)
curl -s -b cookies.txt -X POST http://localhost:8080/holds/<hold_id>/confirm \
  -H 'Idempotency-Key: confirm-req-1'
```
Expected response (201):
```json
{"id":"<order_id>","hold_id":"<hold_id>","status":"confirmed","created_at":"<created_at>"}
```

```bash
# Idempotent confirm retry (200, same order_id)
curl -s -b cookies.txt -X POST http://localhost:8080/holds/<hold_id>/confirm \
  -H 'Idempotency-Key: confirm-req-1'
```
Expected response (200):
```json
{"id":"<order_id>","hold_id":"<hold_id>","status":"confirmed","created_at":"<created_at>"}
```
Note: `status` can be `confirmed` or `refunded` if the event was cancelled after confirmation.

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

## Frontend (local)
This frontend is intentionally minimal and decoupled. It reads variables from the
repo root `.env` (see `.env.example`).

Setup:
```bash
cp .env.example .env
```

Run backend (from repo root; `.env` is auto-loaded if present):
```bash
make backend-run
```

Run frontend:
```bash
make frontend-install
make frontend-run
```

Frontend tests:
```bash
make frontend-test
```

Frontend pages:
- Console: `http://localhost:5173/`
- Admin (requires admin login): `http://localhost:5173/admin/`
- Login: `http://localhost:5173/login/`
- Register: `http://localhost:5173/register/`
Idempotency keys are auto-generated in the UI and can be edited or regenerated for debugging.

Frontend env variables (from `.env`):
- `VITE_API_BASE_URL` (e.g. `http://localhost:8080`)
- `FRONTEND_PORT` (default: `5173`)
## Common commands (from repo root)
`make test` runs backend + frontend; use the scoped targets below if needed.
```bash
make test # requires API running and ALLOW_PUBLIC_REGISTER=true for E2E
make backend-test
make frontend-test
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

## Repository layout (initial)
- `services/api/` — Go modular monolith (see `docs/adr/0002-repo-structure.md`)
  - `cmd/api/` — entrypoint
  - `internal/domain/` — domain model and invariants
  - `internal/app/` — application services/use cases
  - `internal/storage/postgres/` — storage adapters
  - `internal/transport/http/` — HTTP handlers
  - `internal/clock/` — time abstractions
  - `migrations/` — database migrations
- `deployments/local/` — Docker Compose for local dependencies
- `docs/adr/` — architecture decisions
