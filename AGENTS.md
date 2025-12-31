# AGENTS.md — AI-assisted development guide

This repository is developed with AI assistance. The AI acts as a guided pair-programmer, not an autonomous developer.

## 0) Operating mode: guided, not automatic
- The AI MUST propose a plan before making non-trivial changes.
- The AI MUST wait for explicit developer approval for any **non-trivial decision** (see section 8).
- The AI SHOULD prefer small, reviewable diffs and incremental progress.
- If uncertain, the AI MUST ask rather than guess.

## 1) Project goals (high level)
Build a high-scale ticketing prototype focused on:
- Zone-based inventory (e.g., floor/stands/amphitheater) — **no exact seat selection**
- Holds with TTL that prevent overselling
- Evolution path: toy prototype → robust services → queue/waiting room → payments → anti-abuse → SRE hardening

## 2) Strict TDD workflow (mandatory)
We apply TDD. The AI must follow this loop:

1. **Write/Update a test first** describing the desired behavior.
2. Run tests and confirm the new test fails for the right reason.
3. Implement the smallest change to make the test pass.
4. Refactor only if tests are green.
5. Repeat.

Rules:
- No production code without a corresponding test (except wiring/bootstrapping).
- Prefer **table-driven tests** in Go when sensible.
- Prefer **deterministic tests**: avoid sleeps/timeouts; use fake clocks where needed.
- When fixing a bug: add a regression test first.

## 3) Local development: Docker-first
During development we run dependencies via Docker.

### Setup / run commands (keep updated)
- Start local stack: `docker compose -f deployments/local/docker-compose.yml up --build`
- Stop: `docker compose -f deployments/local/docker-compose.yml down -v`
- Run unit tests: `cd services/api && go test ./...`
- Run API: `cd services/api && go run ./cmd/api`
- Format: `cd services/api && go fmt ./...`
- Vet: `cd services/api && go vet ./...`
- Tidy: `cd services/api && go mod tidy`
- Lint (if configured): `cd services/api && golangci-lint run`
Alternative from repo root:
- Run unit tests: `make test`
- Run API: `make run`
- Frontend install: `make frontend-install`
- Frontend run: `make frontend-run`
- Format: `make fmt`
- Vet: `make vet`
- Tidy: `make tidy`
- Lint (if configured): `make lint`
- Build: `make build`

Local dependencies:
- Postgres @ `localhost:5432` (user/password/db: `ultimate_ticket`)
- API expects `DATABASE_URL` (defaults to the local Postgres DSN above)
- CORS allow-list via `CORS_ORIGINS` (comma-separated)
 - `.env` is auto-loaded when present (current dir or parent directories)

API endpoints:
- `POST /holds` expects `idempotency_key` in the JSON body.
- `POST /holds/{id}/confirm` expects `Idempotency-Key` header.
- Admin (local tooling only):
  - `POST /admin/events` + `GET /admin/events`
  - `POST /admin/events/{event_id}/zones` + `GET /admin/events/{event_id}/zones`

If any command changes, update this file and the main README.

## 4) Architecture & repository structure
Start as a modular monolith (Go) and evolve into services by extraction.

Guidelines:
- Keep domain logic in `internal/domain` (or equivalent) with minimal dependencies.
- Storage adapters live under `internal/storage`.
- HTTP handlers should be thin and delegate to domain/application services.
- Keep boundaries clean so future splitting into services is straightforward.

## 5) Code style & simplicity (avoid “tricky” solutions initially)
Early-stage rule: **prefer clarity over cleverness**.

- Prefer straightforward control flow over metaprogramming/reflection.
- Avoid complex concurrency patterns unless necessary.
- Avoid premature optimization and over-engineering.
- Choose boring, well-known approaches first.
- Keep functions small, names explicit, errors wrapped with context.
- Use UTC everywhere. Treat time explicitly (inject clock where needed).

## 6) Correctness rules (invariants)
Inventory invariants (must always hold):
- Never oversell: `sold + held <= capacity` per (event_id, zone_id).
- A hold expires at `expires_at` and must release inventory exactly once.
- Confirm/commit must be idempotent (safe to retry).
- Mutating endpoints must accept an idempotency key.

If a change might affect these invariants, it is **non-trivial** and requires approval.

## 7) Git & PR workflow (small, reviewable changes)
- Prefer PR-sized increments: one concern per change.
- Each PR should include:
  - tests
  - a brief description of behavior change
  - how to run/verify locally
- Update docs when behavior/commands change.

Commit message style (suggested):
- `feat: ...`, `fix: ...`, `test: ...`, `docs: ...`, `chore: ...`

## 8) “Non-trivial decisions” requiring developer approval
The AI MUST ask for approval before:
- Adding a new external dependency (Go module) or major tool.
- Introducing async/event-driven components (queues, buses, outbox).
- Introducing a new persistence technology (e.g., DynamoDB, Redis, Kafka) or changing DB schema strategy significantly.
- Changing public API contracts (endpoints, request/response shape) beyond additive changes.
- Implementing non-obvious concurrency/locking schemes.
- Making security-related choices (authN/authZ, tokens, cryptography).
- Large refactors, re-architectures, or rewrites.
- Any change that affects the inventory invariants in section 6.

Approval protocol:
- The AI proposes: (a) options, (b) trade-offs, (c) recommended default, (d) rollback plan.
- Then the AI waits for explicit "approved" from the developer.

## 9) Testing strategy (scope & expectations)
We aim for a pyramid:
- Unit tests for domain logic (majority)
- Integration tests for DB interactions (few but meaningful)
- Load tests later (k6) once core invariants are stable

For integration tests:
- Prefer running dependencies via docker-compose
- Keep tests hermetic and reproducible

## 10) Security & secrets
- NEVER commit secrets, tokens, credentials, private keys.
- Use env vars + local `.env` (gitignored) for development.
- Security-sensitive features will be designed explicitly and reviewed.

## 11) Documentation: ADRs
Non-trivial design decisions must be captured in an ADR under `docs/adr/`:
- Context, decision, consequences, alternatives
- Keep ADRs short and practical

## 12) AI behavior expectations (what “good” looks like)
- Ask clarifying questions only when genuinely needed, but do not block progress unnecessarily.
- Prefer explicit diffs, explain intent, keep changes minimal.
- Always keep the project runnable and tests passing.
- If you break tests, fix them in the same change before proposing more work.
