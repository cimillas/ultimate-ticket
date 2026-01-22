# API contract inventory

## Goal
List the current API surface, auth requirements, required headers, and error
codes so we can freeze contracts before extraction.

Source of truth for error codes: `docs/reference/api/error-codes.md`.

## Public endpoints (no auth)
- `GET /health`
  - Response: `ok` (text)

- `GET /metrics`
  - Response: Prometheus metrics (text)

- `GET /events`
  - Response: JSON array of events

- `GET /events/{event_id}/zones`
  - Response: JSON array of zones for event

## Auth endpoints
- `POST /auth/register`
  - Auth: none (controlled by `ALLOW_PUBLIC_REGISTER`)
  - Body: `{ "username", "email", "password" }`
  - Notes: `username` cannot contain `@`

- `POST /auth/login`
  - Body: `{ "identifier", "password" }`
  - Notes: `identifier` is email if it contains `@`, otherwise username

- `POST /auth/logout`
  - Uses session cookie when present; returns ok even if missing

- `POST /auth/password`
  - Requires session cookie
  - Body: `{ "current_password", "new_password" }`

- `GET /me`
  - Requires session cookie

## Protected endpoints (session required)
- `POST /holds`
  - Body: `{ "event_id", "zone_id", "quantity", "idempotency_key" }`

- `GET /holds`
  - Response: JSON array of active holds for the current user

- `GET /orders`
  - Response: JSON array of orders for the current user

- `POST /holds/{id}/confirm`
  - Header: `Idempotency-Key: <string>`

## Admin endpoints (session + admin role)
- `POST /admin/events`
  - Body: `{ "name", "starts_at" }`

- `GET /admin/events`

- `POST /admin/events/{event_id}/cancel`

- `POST /admin/events/{event_id}/zones`
  - Body: `{ "name", "capacity" }`

- `GET /admin/events/{event_id}/zones`

- `GET /admin/events/{event_id}/zones/{zone_id}/holds`

- `GET /admin/events/{event_id}/zones/{zone_id}/orders`

## Common headers
- `X-Request-Id` (optional, echoed if provided; generated if missing)
- `Idempotency-Key` (required only for `POST /holds/{id}/confirm`)

## Error contract
- Error payload: `{"error":"<message>","code":"<code>","request_id":"<id>"}` (request_id optional)
- Status codes and error codes are documented in `docs/reference/api/error-codes.md`.

## Notes
- Session cookie name: `ut_session`
- Idempotency rules apply to hold creation and confirmation (see error codes doc).
