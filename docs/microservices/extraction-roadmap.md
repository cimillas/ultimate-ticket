# Microservices extraction roadmap

## Goal
Provide a step-by-step, verifiable path to extract services without breaking
existing contracts or slowing delivery.

## Phase 1: Stabilize contracts
**Exit criteria**
- No breaking API changes in 2–3 iterations.
- Error codes stable and documented.
- E2E critical flows pass on clean DB.
- Observability plan approved.

**Actions**
- Adopt the compatibility checklist in `docs/contracts/compatibility-checklist.md`.
- Freeze API + error codes; update docs when changes are additive only.
- Keep a simple breaking-changes log.

## Phase 2: Observability baseline
**Exit criteria**
- JSON logs consistent across services.
- Request IDs propagated across boundaries.
- Minimal metrics format decided (Prometheus vs OTel).

**Actions**
- Add `SERVICE_NAME` env to logs (monolith first).
- Decide metrics format and draft `/metrics` spec (no implementation yet).

## Phase 3: Auth extraction (first service)
**Exit criteria**
- Auth API contract stable.
- Session introspection plan approved.
- Frontend and backend accept new auth endpoint base URL.

**Actions**
- Split auth code into a separate repo/module.
- Add an Auth client in the monolith (HTTP calls).
- Migrate cookies to be issued by Auth service; backend uses introspection.

## Phase 4: Inventory extraction
**Exit criteria**
- Holds/availability logic stable and tested for concurrency.
- Idempotency semantics validated.

**Actions**
- Extract event/zone/hold logic.
- Expose inventory endpoints; monolith calls them.

## Phase 5: Orders extraction
**Exit criteria**
- Confirm flow stable and covered by E2E tests.
- Refund/cancel rules unchanged for multiple iterations.

**Actions**
- Extract order creation/confirmation/refunds.
- Connect orders with inventory (holds) via IDs and validations.

## Phase 6: Admin tooling
**Exit criteria**
- Admin endpoints stable.
- Orchestration path clear across services.

**Actions**
- Move admin orchestration into a thin gateway or keep in monolith.

## Risks and mitigations
- **Contract drift**: mitigate with checklist + ADRs.
- **Data ownership ambiguity**: mitigate with explicit ownership per service.
- **Operational overhead**: delay extraction until metrics/logs ready.

## Next step
Use `docs/microservices/phase-1-checklist.md` to track Phase 1 completion.
