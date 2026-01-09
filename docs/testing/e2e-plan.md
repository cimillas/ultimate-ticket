# E2E test plan (critical flows)

## Goal
Validate the highest-risk user flows end-to-end against a running system before
any microservice extraction.

## Scope (v1)
Critical flows to cover:
1) Auth: register -> login -> /me -> logout
2) Holds: create -> idempotent retry -> idempotency conflict
3) Confirm: confirm -> idempotent retry -> conflict
4) Cancel: cancel event -> holds invalid -> orders refunded
5) Admin access control: user vs admin permissions

Out of scope (for now):
- Load/performance tests
- Browser UI automation
- Chaos/fault injection

## Environment
Recommended default: local Docker dependencies + local API/frontend.
Required env vars:
- Backend (`services/api/.env`): `DATABASE_URL`, `CORS_ORIGINS`, `ADMIN_USERNAME`,
  `ADMIN_PASSWORD`, `ADMIN_EMAIL`, `ALLOW_PUBLIC_REGISTER` (true for local testing).
- Frontend (`frontend/.env`): `VITE_API_BASE_URL`.

## Data strategy
Recommended default: seed via API calls (not raw SQL) to validate real behavior.
Keep ids recorded in a temporary file (or exported env vars) for follow-up calls.

Alternatives:
- SQL seeding for speed (faster but bypasses API validation).
- Fixture import scripts (useful later for large datasets).

## Test execution options
Recommended default: curl + small bash helpers.
Pros: no new dependencies, quick iteration, explicit requests.

Alternatives:
- Postman collection (good for manual QA, less automatable).
- Playwright tests (good for UI flows, heavier to maintain).
- k6 (later, for load testing).

Script:
- `scripts/e2e/run.sh` runs the critical flows using curl.
- Requires the API to be running locally and `ALLOW_PUBLIC_REGISTER=true`.

## Critical scenarios

### 1) Auth: register -> login -> /me -> logout
Given: `ALLOW_PUBLIC_REGISTER=true`
When:
1. POST `/auth/register` with `{username,email,password}`
2. POST `/auth/login` with `{identifier,password}`
3. GET `/me` with session cookie
4. POST `/auth/logout` (after all user actions to keep session valid)
Then:
- Registration returns 200 with user.
- Login returns 200 and sets cookie.
- `/me` returns 200 with same user.
- Logout returns 200 and clears cookie.

### 2) Holds: create -> idempotent retry -> conflict
Given: event + zone exist and user is logged in.
When:
1. POST `/holds` with `{event_id, zone_id, quantity, idempotency_key}`
2. Repeat same request with same idempotency key
3. Repeat with same idempotency key but different quantity
Then:
- First call returns 201 with hold id.
- Second call returns 201 with same hold id.
- Third call returns 409 `idempotency_conflict`.

### 3) Confirm: confirm -> idempotent retry -> conflict
Given: active hold exists for a logged-in user.
When:
1. POST `/holds/{id}/confirm` with `Idempotency-Key`
2. Repeat with same idempotency key
3. Repeat with same key but after hold changed (quantity or invalid)
Then:
- First call returns 201 order.
- Second call returns 200 with same order.
- Third call returns 409 conflict.

### 4) Cancel: cancel event -> holds invalid -> orders refunded
Given: event with active and confirmed holds.
When:
1. POST `/admin/events/{event_id}/cancel` as admin
2. List holds/orders for the event zone
Then:
- Event status becomes cancelled.
- Active holds become invalid.
- Confirmed orders become refunded.

### 5) Admin access control
Given: normal user and admin user exist.
When:
1. Call `/admin/events` as normal user
2. Call `/admin/events` as admin
Then:
- Normal user gets 403.
- Admin gets 200.

## Success criteria
- All flows pass with correct status codes and error codes.
- No unexpected 500s.
- Idempotency behavior is stable and repeatable.

## Validation checklist (pre-extraction)
- E2E flows pass on clean DB.
- Repeatable with new IDs every run.
- Observability logs include `request_id` for each request.

## Next step
Implement a minimal bash-based runner under `scripts/e2e/` once the plan is accepted.
