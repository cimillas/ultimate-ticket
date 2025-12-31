# ADR 0003: Admin endpoints, CORS, shared env, and local frontend

## Status
Accepted

## Date
2025-12-31

## Context
We need a minimal, functional frontend to exercise the backend locally.
The frontend must be **decoupled** from the API and should avoid extra tooling
or local servers beyond what is required. The current API lacks endpoints to
create/list events and zones, so we cannot set up full flows without touching
the database directly.

We also need a simple, explicit way to configure ports and base URLs without
hardcoding them in code, and the API must allow cross-origin requests from the
frontend during local development.

## Decision
We will:

1. **Add admin-only endpoints under `/admin`**
   - `POST /admin/events` + `GET /admin/events`
   - `POST /admin/events/{event_id}/zones` + `GET /admin/events/{event_id}/zones`
   These endpoints are for local tooling, not for end users.

2. **Enable CORS via a simple allow-list**
   - Add a middleware that reads `CORS_ORIGINS` (comma-separated).
   - Only allowed origins receive CORS headers; others are rejected for preflight.

3. **Use a shared `.env` file for backend and frontend**
   - Define `PORT`, `DATABASE_URL`, `CORS_ORIGINS`, `VITE_API_BASE_URL`, and `FRONTEND_PORT`.
   - Configure Vite to read the repo root `.env` so both apps share the same file.

4. **Create a minimal Vite + Vanilla JS frontend**
   - No UX work; just forms for admin/event/zone creation and hold flows.
   - Intended for local use only.

## Consequences

### Positive
- Local testing can validate the full flow without touching the database directly.
- Frontend and backend stay decoupled while sharing configuration.
- CORS is explicit and controlled via environment configuration.

### Negative
- Admin endpoints must be protected or removed for production use.
- Adds a small Node/Vite toolchain to the repository.

## Alternatives Considered
- Serve a static HTML page from the API to avoid CORS (rejected: not decoupled).
- Use file:// or a Python dev server (rejected: CORS issues / extra tooling).
- HTMX for zero-JS (rejected: additional dependency without clear benefit).
