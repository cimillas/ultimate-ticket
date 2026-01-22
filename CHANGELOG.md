# Changelog

All notable changes to this project will be documented in this file.

This project follows Semantic Versioning (SemVer).

## [Unreleased]

## [0.6.0]
- Added user-scoped orders endpoint (`GET /orders`) with hold context.
- Added password change endpoint (`POST /auth/password`) that keeps the current session and invalidates others.
- Updated frontend labels to “Dashboard” and added “My activity” + “Account” sections.
- Expanded E2E script to cover user holds/orders, password change, and optional auth reset (`E2E_RESET_AUTH=1`).
- Updated API docs, error codes, and examples for new endpoints.
## [0.5.0]
- Added hold ownership (user_id) and enforced ownership on confirmations.
- Scoped hold idempotency per user and added migration/backfill for legacy holds.
- Added ADR 0010 documenting hold ownership and idempotency scope.
- Split backend and frontend env files with new examples and config updates.
- Added documentation index (`docs/README.md`) and updated references.

## [0.4.1]
- Added service boundaries ADR and extraction readiness docs.
- Added API inventory, compatibility checklist, and breaking changes log.
- Added observability plan, E2E plan, and extraction roadmap.
- Added API versioning ADR.
- Added curl-based E2E validation script and Makefile target.
- Documented E2E and logging usage in READMEs.

## [0.4.0]
- Added session-based auth with user registration, login/logout, and admin role enforcement.
- Added public event/zone listing endpoints and protected holds/confirmations.
- Updated frontend to include login/register UI and cookie-based requests.
- Removed automatic admin creation; added `authctl` for bootstrap/reset with local-only guard.
- Session cookie secure flag now defaults to true outside `APP_ENV=local`.
- Disallowed `@` in usernames and removed login identifier ambiguity.
- Added `ALLOW_PUBLIC_REGISTER` flag to disable public registration outside local.
- Added request ID middleware and JSON request logs (status, latency, bytes, IP, agent).

## [0.3.0]
- Added event cancellation flow with refunds, cancellation timestamps, and admin endpoint.
- Added event `closed` status that blocks new zones/holds/confirmations once the start time passes.
- Added admin tooling to list active holds and confirmed orders per zone.
- Expanded frontend console/admin UI with event selectors and improved inventory tooling.
- Added frontend test suite (Vitest) and expanded backend coverage around inventory lifecycle.
- Updated API error codes, ADRs, and domain documentation for new lifecycle states.

## [0.2.0]
- Added admin endpoints for managing events/zones in local tooling.
- Added CORS allow-list support via `CORS_ORIGINS`.
- Added shared `.env.example`, `.env` auto-loading, and warnings for defaults.
- Added a minimal frontend and improved layout for clarity on desktop.
- Upgraded Vite to v7 and added `.nvmrc` for frontend tooling consistency.
- Standardized API errors as JSON with `error` and `code`, including not-found responses.
- Documented API error codes and domain concepts.
- Removed QA naming from docs and UI.

## [0.1.0]
- Initial project structure and API skeleton (holds, confirms, migrations, tests).
