# ADR 0002: Repository structure for initial phase

## Status
Accepted

## Date
2025-XX-XX

## Context
We need an initial repository layout that:
- supports a modular monolith in Go,
- keeps clear boundaries for future service extraction,
- aligns with the documented local commands, and
- stays simple for early-stage iteration.

## Decision
We will:
1. Place the Go module under `services/api`.
2. Use `services/api/cmd/api` as the entrypoint and keep domain logic under
   `services/api/internal`.
3. Keep storage adapters under `services/api/internal/storage`, with PostgreSQL
   as the initial adapter path.
4. Keep HTTP transport under `services/api/internal/transport/http`.
5. Use `deployments/local` for Docker Compose files that run local dependencies.

## Consequences
### Positive
- Matches the existing local command conventions in the README.
- Preserves clean boundaries that make future service extraction easier.
- Keeps top-level project docs and deployment artifacts separate from code.

### Negative
- Adds one extra level of nesting compared to a root-level Go module.
- Requires `cd services/api` for Go tooling until a root-level workspace is
  introduced.

## Alternatives Considered
### Root-level Go module
Rejected because it would mix code with top-level repo artifacts and make future
service extraction noisier.
