# ADR 0005: Admin zone inventory listings

## Status
Accepted

## Date
2025-12-31

## Context
Local tooling needs visibility into in-flight reservations and confirmed
orders for a specific zone. This supports QA and debugging without touching
the database directly.

## Decision
Expose two admin-only endpoints under a zone:
- `GET /admin/events/{event_id}/zones/{zone_id}/holds` for **active, unexpired** holds.
- `GET /admin/events/{event_id}/zones/{zone_id}/orders` for confirmed orders.

These endpoints are read-only and intended for local tooling.

## Consequences
- Adds two admin endpoints and corresponding frontend forms.
- Requires event + zone validation to distinguish `event_not_found` vs
  `zone_not_found`.
