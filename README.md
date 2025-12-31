# Ultimate-ticket

High-scale ticketing prototype focused on:
- Zone-based inventory (no seat selection)
- Holds with TTL
- Eventually: waiting room, payments, anti-bot, SRE hardening

## Domain concepts (short)
- Event: the ticketed experience (concert/show), groups zones.
- Zone: a sellable area within an event, with a fixed capacity.
- Hold: temporary reservation of tickets in a zone (has TTL).
- Confirmation (order): finalizes a hold into a purchase.

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
  - `PORT` (default: `8080`)
  - `DATABASE_URL` (default: `postgres://ultimate_ticket:ultimate_ticket@localhost:5432/ultimate_ticket?sslmode=disable`)
  - `CORS_ORIGINS` (comma-separated, e.g. `http://localhost:5173`)
- Endpoints:
  - `GET /health` → `ok`
  - `POST /holds` with JSON `{event_id, zone_id, quantity, idempotency_key}` (409 on capacity or idempotency conflict)
  - `POST /holds/{id}/confirm` with header `Idempotency-Key` (201 created, 200 idempotent retry)
  - Admin (local tooling only):
    - `POST /admin/events` + `GET /admin/events`
    - `POST /admin/events/{event_id}/zones` + `GET /admin/events/{event_id}/zones`

Migrations:
- Applied on startup and recorded in `schema_migrations`.

Manual test examples (curl)
```bash
# Create hold (201)
curl -s -X POST http://localhost:8080/holds \
  -H 'Content-Type: application/json' \
  -d '{"event_id":"EVENT_ID","zone_id":"ZONE_ID","quantity":2,"idempotency_key":"hold-req-1"}'
```
Expected response (201):
```json
{"id":"<hold_id>","status":"active","expires_at":"<expires_at>"}
```

```bash
# Idempotent retry (201, same hold_id)
curl -s -X POST http://localhost:8080/holds \
  -H 'Content-Type: application/json' \
  -d '{"event_id":"EVENT_ID","zone_id":"ZONE_ID","quantity":2,"idempotency_key":"hold-req-1"}'
```
Expected response (201):
```json
{"id":"<hold_id>","status":"active","expires_at":"<expires_at>"}
```

```bash
# Idempotency conflict (409)
curl -s -X POST http://localhost:8080/holds \
  -H 'Content-Type: application/json' \
  -d '{"event_id":"EVENT_ID","zone_id":"ZONE_ID","quantity":3,"idempotency_key":"hold-req-1"}'
```
Expected response (409):
```json
{"error":"idempotency conflict","code":"idempotency_conflict"}
```

```bash
# Confirm hold (201)
curl -s -X POST http://localhost:8080/holds/<hold_id>/confirm \
  -H 'Idempotency-Key: confirm-req-1'
```
Expected response (201):
```json
{"id":"<order_id>","hold_id":"<hold_id>","status":"confirmed","created_at":"<created_at>"}
```

```bash
# Idempotent confirm retry (200, same order_id)
curl -s -X POST http://localhost:8080/holds/<hold_id>/confirm \
  -H 'Idempotency-Key: confirm-req-1'
```
Expected response (200):
```json
{"id":"<order_id>","hold_id":"<hold_id>","status":"confirmed","created_at":"<created_at>"}
```

Error format:
```json
{"error":"<message>","code":"<code>"}
```

Full reference: `docs/api/error-codes.md`

## Frontend (local)
This frontend is intentionally minimal and decoupled. It reads variables from the
repo root `.env` (see `.env.example`).

Setup:
```bash
cp .env.example .env
```

Run backend (from repo root; `.env` is auto-loaded if present):
```bash
make run
```

Run frontend:
```bash
make frontend-install
make frontend-run
```

Frontend env variables (from `.env`):
- `VITE_API_BASE_URL` (e.g. `http://localhost:8080`)
- `FRONTEND_PORT` (default: `5173`)
## Common commands (from repo root)
```bash
make test
make run
make fmt
make vet
make tidy
make lint
make build
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
