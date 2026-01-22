# ADR 0006: Event cancellation with refunds

## Context
We need an admin action to cancel events without deleting them. When an event is
cancelled, confirmed reservations must be refunded, and active holds must be
invalidated. We also need to store the cancellation timestamp for auditing.

## Decision
- Add `events.status` (`active|cancelled`) and `events.cancelled_at`.
- Add `orders.status` (`confirmed|refunded`).
- Reuse `holds.status=invalid` for cancelled events.
- Provide `POST /admin/events/{event_id}/cancel` to perform the operation.
- Block new holds and confirmations for cancelled events.

## Alternatives considered
- Introduce a dedicated `holds.status=cancelled` to differentiate reasons.
  We skipped it to keep early-state complexity low.

## Consequences
- Event cancellation is idempotent and keeps historical data.
- Admin "list confirmed orders" excludes refunded orders by design.
- Idempotent confirm retries after cancellation return the existing order
  (which may be `refunded`), while new confirms return `event_cancelled`.
