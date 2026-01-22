# ADR 0010: Hold ownership and user-scoped idempotency

## Status
Accepted

## Date
2026-01-08

## Context
Holds are currently global: any authenticated user can confirm a hold if they
know its ID. Idempotency keys for holds are also scoped only by event and zone,
so two users can collide when reusing the same key.

This creates a security risk (hold takeover) and awkward conflicts between
different users during retries.

## Decision
1. **Add ownership to holds**
   - Add `holds.user_id` (non-null, FK to users).
   - `CreateHold` stores the authenticated user ID.
   - `ConfirmHold` verifies `hold.user_id == session.user.id`.
   - If the user does not own the hold, return `ErrHoldNotFound` (HTTP 404) to
     avoid leaking existence.

2. **Scope hold idempotency per user**
   - Unique index: `(event_id, zone_id, user_id, idempotency_key)`.
   - Reads for idempotency checks include `user_id`.

3. **Legacy holds during migration**
   - If existing holds lack a user, they are assigned to the earliest user.
   - If no users exist, create a `legacy-holds` user and assign them.

## Consequences
### Positive
- Prevents hold confirmation by other users.
- Idempotency keys are isolated per user and no longer collide.

### Negative
- Requires a schema migration and backfill.
- Legacy holds become owned by a synthetic or earliest user.

## Alternatives Considered
- Store ownership in sessions only (rejected: sessions expire, no long-term trace).
- Keep idempotency global (rejected: collisions between users).
