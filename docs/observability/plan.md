# Observability plan (multi-service readiness)

## Goal
Define the minimal observability baseline needed before extracting services.
This is a design plan only; implementation will follow later.

## Guiding principles
- Prefer boring, low-dependency solutions first.
- Ensure request correlation across service boundaries.
- Keep metrics minimal and actionable.

## Baseline requirements
1) **Request correlation**
   - Accept and propagate `X-Request-Id` across services.
   - Generate a new ID if missing at the edge.
   - Include `request_id` in all logs and error responses (where safe).

2) **Structured logs**
   - JSON per line with a stable schema (already in the API).
   - Required fields: `ts`, `request_id`, `service`, `method`, `path`,
     `status`, `duration_ms`, `bytes`, `remote_ip`, `user_agent`.
   - Avoid logging sensitive data (passwords, tokens, cookies).

3) **Metrics (future, minimal)**
   - Request count by route/method/status.
   - Request duration histogram (p50/p95/p99).
   - Error count by code.

4) **Health checks**
   - `/health` stays, but should include dependency status when extracted
     (DB, downstream auth service).

## Current implementation (monolith)
- Request IDs are generated and returned via `X-Request-Id` and included in logs.
- Logs are JSON per line and include `service` via `SERVICE_NAME` (default `api`).
- Prometheus metrics are exposed at `/metrics` (request count + duration).
  See `docs/observability/metrics.md` for details.
- Error responses include `request_id` when available.

## Decisions needed (non-trivial)
1) **Metrics format**
   - Option A (approved): Prometheus `/metrics`.
   - Option B: OpenTelemetry metrics (heavier; better for vendor-neutral stacks).

2) **Tracing**
   - Option A: Defer tracing until after first extraction.
   - Option B: Add OpenTelemetry tracing early to avoid rework.

3) **Log aggregation**
   - Option A: local files + shipping later.
   - Option B: direct stdout -> centralized collector.

## Proposed initial implementation (later)
- Add a small metrics endpoint in the API (Prometheus format).
- Attach service name via env `SERVICE_NAME` (default `api`).
- Ensure `X-Request-Id` is forwarded in any internal HTTP calls.

## Readiness checklist
- All services emit JSON logs with the same schema.
- Request IDs are propagated across all service boundaries.
- Minimal metrics exposed and documented.
- Health endpoints include dependency checks.

## Next step
When approved, implement metrics + service name config in the monolith, then
reuse the same middleware in extracted services.
