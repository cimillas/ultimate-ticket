# ADR 0004: Availability, event completion, and event start cutoffs

## Status
Accepted

## Date
2025-12-31

## Context
We need to expose zone availability for local tooling, identify when an event
is fully sold out, and prevent new reservations once an event has started.
Confirmations should also stop after the event start time.

## Decision
1. **Zone availability is derived**
   - `available = capacity - confirmed - active_holds` where active holds are
     unexpired (`expires_at > now`).
2. **Event completion is derived**
   - An event is marked complete when it has at least one zone and all zones
     have `available == 0`.
3. **Event start cutoff**
   - Events transition to `closed` once `starts_at` has passed.
   - Creating zones/holds after `starts_at` is blocked.
   - Confirming holds after `starts_at` marks the hold as `invalid` and returns
     an error.

## Consequences
- Availability and completion are computed at read time, avoiding stored
  counters but adding query work.
- A new `invalid` hold status is required to represent holds cancelled after an
  event starts.
- Confirm attempts after `starts_at` return a conflict, and the hold cannot be
  confirmed.
- The `closed` transition is enforced on demand during write operations for
  now; a more robust scheduler can be added later.
