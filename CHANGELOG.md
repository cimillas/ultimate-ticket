# Changelog

All notable changes to this project will be documented in this file.

This project follows Semantic Versioning (SemVer).

## [Unreleased]

## [0.4.1]
- Added session-based auth with user registration, login/logout, and admin role enforcement.
- Added public event/zone listing endpoints and protected holds/confirmations.
- Updated frontend to include login/register UI and cookie-based requests.
- Removed automatic admin creation; added `authctl` for bootstrap/reset with local-only guard.
- Session cookie secure flag now defaults to true outside `APP_ENV=local`.
- Disallowed `@` in usernames and removed login identifier ambiguity.
- Added `ALLOW_PUBLIC_REGISTER` flag to disable public registration outside local.
- Added request ID middleware and JSON request logs (status, latency, bytes, IP, agent).
- Added ADR for service boundaries and extraction readiness.
- Added E2E test plan for critical flows.
- Added observability readiness plan (multi-service).
- Added contract compatibility checklist for pre-extraction readiness.
- Added microservices extraction roadmap.
- Added API contract inventory for extraction readiness.
- Added ADR for API versioning strategy.
- Added curl-based E2E validation script.
- Added breaking changes log for API contract governance.

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
