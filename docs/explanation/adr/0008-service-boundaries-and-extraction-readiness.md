# ADR 0008: Service boundaries and extraction readiness

## Context
The project is currently a modular monolith. We want to move toward
microservices, but only after the core flows and contracts stabilize.
We need clear service boundaries, extraction order, and readiness criteria so
each step can be validated before moving on.

## Decision
- Keep the monolith as the source of truth until readiness criteria are met.
- Define initial service boundaries:
  - **Auth service**: users, sessions, login/register/logout, and authorization.
  - **Inventory service**: events, zones, holds, availability, and idempotent hold creation.
  - **Orders service**: hold confirmation, order creation, refunds on cancellation.
  - **Admin tooling** remains a thin orchestration layer (initially within the monolith).
- Extraction order: **Auth first**, then **Inventory**, then **Orders**.
- Readiness checklist before extracting any service:
  - Stable API contracts for 2–3 iterations (no breaking changes).
  - E2E tests for critical flows (auth, holds, confirm, cancel, admin).
  - Observability baseline (request IDs, structured logs, basic metrics plan).
  - Clear data ownership and error contract between services.
- Session strategy during transition:
  - Short term: session introspection against Auth service.
  - Future: migrate to JWT when services are fully decoupled.

## Alternatives considered
- Immediate extraction: rejected due to unstable contracts and higher delivery risk.
- Big-bang split into multiple services: rejected due to migration complexity and
  testability concerns.

## Consequences
- Microservice extraction is gated by explicit readiness criteria.
- The monolith remains the primary integration surface until those criteria
  are met.
- Auth will be the first extraction target to minimize coupling and risk.
