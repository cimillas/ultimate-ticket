# ADR 0007: Session-based auth with httpOnly cookies

## Context
We need login/registration to protect holds/confirmations and restrict admin
tools. The system is still a modular monolith, and we want a secure but simple
approach that keeps state server-side. Future architecture plans include moving
to microservices.

## Decision
- Use database-backed sessions and an httpOnly cookie (`ut_session`).
- Hash user passwords with Argon2id.
- Set a 1-hour session TTL with sliding expiration on activity.
- Provide auth endpoints: `POST /auth/register`, `POST /auth/login`,
  `POST /auth/logout`, and `GET /me`.
- Admin user is bootstrapped explicitly via CLI (no auto-provision on startup).
- Usernames cannot include `@` to avoid ambiguity with email-based login.
- Public registration can be disabled via `ALLOW_PUBLIC_REGISTER`.
- Note: when moving to microservices, migrate to JWT-based auth for better
  decoupling.

## Alternatives considered
- Stateless JWT now: rejected to keep the monolith simple and avoid token
  invalidation complexity early.
- Basic auth: rejected due to poor UX and no session management.

## Consequences
- Backend must store `users` and `sessions` tables.
- CORS must allow credentials and frontend must include cookies.
- Admin endpoints are protected via role checks (`admin` only).
- Operational step: bootstrap admin from `ADMIN_*` env vars.
