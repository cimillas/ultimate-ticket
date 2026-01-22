# Phase 1 checklist: stabilize contracts

## Purpose
Track readiness to move from the monolith to the first service extraction by
freezing contracts and keeping critical behavior stable.

## Checklist
### Contracts and compatibility
- [x] Confirm `docs/contracts/api-inventory.md` reflects all current endpoints.
- [x] Confirm error codes in `docs/api/error-codes.md` match handler behavior.
- [x] Confirm `docs/contracts/compatibility-checklist.md` is still valid.
- [x] Record any breaking change in `docs/contracts/breaking-changes.md`.

### Tests and reliability
- [x] `scripts/e2e/run.sh` passes on a clean database.
- [x] `make backend-test` passes without flakes.
- [x] `make frontend-test` passes without flakes.

### Observability plan
- [x] `docs/observability/plan.md` reviewed and approved for Phase 2 work.
- [x] Request IDs are generated and returned by the API.

### Operational agreement
- [x] No breaking API changes in the last 2-3 iterations.
- [x] Document any additive contract changes (fields, endpoints, codes).

## Status log
Use this section to capture review dates and outcomes.

- Date: 2026-01-22
  Result: Partial
  Notes: Ran `make test` (backend, frontend, E2E). Did not verify clean DB state.
- Date: 2026-01-22
  Result: Partial
  Notes: Verified API inventory + error codes; updated compatibility checklist for hold `expired`.
- Date: 2026-01-22
  Result: Partial
  Notes: Reset Postgres volume and ran `make test` with E2E passing on a clean DB.
- Date: 2026-01-22
  Result: Partial
  Notes: Confirmed docs align with current handlers; no additive contract changes found.
- Date: 2026-01-22
  Result: Complete
  Notes: Approved observability plan for Phase 2; no breaking changes in recent iterations.
