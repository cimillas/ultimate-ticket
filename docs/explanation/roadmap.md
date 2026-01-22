# Roadmap

This document combines the product roadmap with the service extraction roadmap.

## Product phases
### Phase 1
- Monolithic Go service
- PostgreSQL
- Holds with TTL
- No overselling
- TDD only

### Phase 2
- Admission control / simple queue
- Observability basics

### Phase 3
- Inventory scaling experiments
- DynamoDB vs Postgres comparison

## Service extraction roadmap
### Phase 1: Stabilize contracts
**Exit criteria**
- No breaking API changes in 2-3 iterations.
- Error codes stable and documented.
- E2E critical flows pass on clean DB.
- Observability plan approved.

**Actions**
- Adopt the compatibility checklist in `docs/dev/compatibility-checklist.md`.
- Freeze API + error codes; update docs only for additive changes.
- Keep a breaking-changes log in `docs/dev/breaking-changes.md`.

**Status**
- Complete. See `docs/dev/phase-1-checklist.md`.

### Phase 2: Observability baseline
**Exit criteria**
- JSON logs consistent across services.
- Request IDs propagated across boundaries.
- Minimal metrics exposed and documented.

**Actions**
- Ensure `SERVICE_NAME` is set for request logs.
- Expose `/metrics` with request count and duration.
- Decide tracing and log aggregation approach (see `docs/explanation/observability-plan.md`).

**Status**
- In progress. Logs include `service`, metrics are exposed, and error payloads include `request_id` when available.

### Phase 3: Auth extraction (first service)
**Exit criteria**
- Auth API contract stable.
- Session introspection plan approved.
- Frontend and backend accept new auth endpoint base URL.

**Actions**
- Split auth code into a separate repo/module.
- Add an Auth client in the monolith (HTTP calls).
- Migrate cookies to be issued by Auth service; backend uses introspection.

### Phase 4: Inventory extraction
**Exit criteria**
- Holds/availability logic stable and tested for concurrency.
- Idempotency semantics validated.

**Actions**
- Extract event/zone/hold logic.
- Expose inventory endpoints; monolith calls them.

### Phase 5: Orders extraction
**Exit criteria**
- Confirm flow stable and covered by E2E tests.
- Refund/cancel rules unchanged for multiple iterations.

**Actions**
- Extract order creation/confirmation/refunds.
- Connect orders with inventory (holds) via IDs and validations.

### Phase 6: Admin tooling
**Exit criteria**
- Admin endpoints stable.
- Orchestration path clear across services.

**Actions**
- Move admin orchestration into a thin gateway or keep in monolith.

## Risks and mitigations
- **Contract drift**: mitigate with checklist + ADRs.
- **Data ownership ambiguity**: mitigate with explicit ownership per service.
- **Operational overhead**: delay extraction until metrics/logs ready.
