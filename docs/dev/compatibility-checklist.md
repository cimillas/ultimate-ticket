# Compatibility checklist (pre-extraction)

## Goal
Freeze public contracts and inter-service expectations before any service
extraction to avoid cascading changes.

## What to freeze (must be stable for 2–3 iterations)
1) **HTTP API surface**
   - Routes, methods, and auth requirements.
   - Status codes and error codes.
   - Required headers (e.g., `Idempotency-Key`, `X-Request-Id`).

2) **Request/response shapes**
   - JSON fields and types.
   - Error payload schema: `{"error":"...","code":"...","request_id":"..."}`.
   - Auth response schema (`user`, `expires_at`).

3) **Auth semantics**
   - Session TTL + sliding renewal behavior.
   - Cookie name and properties (`httpOnly`, `SameSite`, `Secure`).
   - Registration behavior (`ALLOW_PUBLIC_REGISTER`).

4) **Idempotency behavior**
   - Holds creation and confirm retry behavior.
   - Conflict semantics (409) and error codes.

5) **Domain states**
   - Event: `active`, `closed`, `cancelled`.
   - Hold: `active`, `confirmed`, `expired`, `invalid`.
   - Order: `confirmed`, `refunded`.

6) **Time rules**
   - `starts_at` closes events.
   - Hold expiry and confirmation rules.

## Freeze status
Checked items are considered **frozen** for the next 2–3 iterations.

- [x] API inventory captured in `docs/reference/api/api-inventory.md`
- [x] Error codes defined in `docs/reference/api/error-codes.md`
- [x] Error payload schema (`{"error","code","request_id"}`) unchanged
- [x] Error/status mapping reviewed against handlers
- [x] Breaking changes log created (`docs/dev/breaking-changes.md`)
- [x] Auth cookie name + semantics (`ut_session`)
- [x] Idempotency rules for holds/confirm (409 on conflicts)
- [x] Domain states (`event`, `hold`, `order`)
- [x] Time rules (`starts_at`, hold expiry)
- [x] Versioning approach decided (ADR 0009)

## Non-trivial decisions to validate
1) **API versioning**
   - Option A (recommended): implicit versioning + changelog discipline.
   - Option B: `/v1` path or `Accept` header versioning (higher overhead).

2) **Error contract evolution**
   - Option A: additive only (new codes allowed, old codes stable).
   - Option B: strict freeze (no new codes without version bump).

3) **Auth boundary**
   - Option A: session introspection (short-term).
   - Option B: JWT (future microservice target).

## Readiness checklist
- No breaking API changes in the last 2–3 iterations.
- Error codes and status codes documented and unchanged.
- E2E critical flows pass reliably.
- Observability plan is agreed and ready.

## Next step
Once validated, document a versioning approach and keep a “breaking changes”
log before extracting any service.

## Operational note (freeze window)
For the next 2–3 iterations:
- No breaking changes without ADR + entry in `docs/dev/breaking-changes.md`.
- Run `scripts/e2e/run.sh` (or `make test`) to validate critical flows.
- Update `docs/reference/api/api-inventory.md` only for additive changes.
