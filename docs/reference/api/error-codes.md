# API error codes

## Error format
All error responses are JSON with a stable code:

```json
{"error":"<message>","code":"<code>","request_id":"<id>"}
```

`request_id` is omitted when unavailable.

## Code reference
- `method_not_allowed` - HTTP method is not supported for the endpoint.
- `not_found` - Endpoint path does not match a known route.
- `invalid_request_body` - Request JSON is invalid or cannot be parsed.
- `missing_required_field` - Required field(s) are missing.
- `invalid_starts_at` - Invalid RFC3339 timestamp for `starts_at`.
- `invalid_id` - Provided ID is invalid (path or body).
- `event_name_required` - Event name is required.
- `zone_name_required` - Zone name is required.
- `username_required` - Username is required.
- `username_invalid` - Username format is invalid.
- `email_required` - Email is required.
- `password_required` - Password is required.
- `username_taken` - Username already exists.
- `email_taken` - Email already exists.
- `invalid_quantity` - Quantity must be greater than zero.
- `invalid_capacity` - Capacity must be greater than zero.
- `idempotency_key_required` - Idempotency key is required.
- `idempotency_conflict` - Idempotency key already used with different payload.
- `insufficient_capacity` - Not enough inventory available in the zone.
- `event_closed` - Event has started and is now closed for new actions.
- `event_cancelled` - Event has been cancelled, action is no longer allowed.
- `invalid_credentials` - Credentials are invalid.
- `unauthorized` - Authentication is required or session is invalid.
- `registration_disabled` - Public registration is disabled.
- `zone_not_found` - Zone does not exist for the event.
- `event_not_found` - Event does not exist.
- `zone_already_exists` - Zone with same name already exists for the event.
- `hold_not_found` - Hold does not exist.
- `hold_expired` - Hold has expired.
- `hold_already_confirmed` - Hold is already confirmed.
- `hold_invalid` - Hold is no longer valid.
- `forbidden` - Request is blocked (CORS or admin access).
- `internal_error` - Unexpected server error.

## Endpoint mapping

### `POST /holds`
- 400 `invalid_request_body`, `missing_required_field`, `idempotency_key_required`, `invalid_quantity`, `invalid_id`
- 401 `unauthorized`
- 404 `zone_not_found`
- 409 `idempotency_conflict`, `insufficient_capacity`
- 409 `event_closed`, `event_cancelled`
- 500 `internal_error`
- 405 `method_not_allowed`

### `GET /holds`
- 400 `invalid_id`
- 401 `unauthorized`
- 500 `internal_error`
- 405 `method_not_allowed`

### `POST /holds/{hold_id}/confirm`
- 400 `idempotency_key_required`
- 401 `unauthorized`
- 404 `not_found`, `invalid_id`, `hold_not_found`
- 409 `hold_expired`, `hold_already_confirmed`, `hold_invalid`, `event_closed`, `event_cancelled`
- 500 `internal_error`
- 405 `method_not_allowed`

### `GET /events`
- 500 `internal_error`
- 405 `method_not_allowed`

### `GET /events/{event_id}/zones`
- 404 `not_found`, `invalid_id`, `event_not_found`
- 500 `internal_error`
- 405 `method_not_allowed`

### `GET /orders`
- 400 `invalid_id`
- 401 `unauthorized`
- 500 `internal_error`
- 405 `method_not_allowed`

### `POST /auth/register`
- 400 `invalid_request_body`, `username_required`, `username_invalid`, `email_required`, `password_required`
- 403 `registration_disabled`
- 409 `username_taken`, `email_taken`
- 500 `internal_error`
- 405 `method_not_allowed`

### `POST /auth/login`
- 400 `invalid_request_body`
- 401 `invalid_credentials`
- 500 `internal_error`
- 405 `method_not_allowed`

### `POST /auth/logout`
- 500 `internal_error`
- 405 `method_not_allowed`

### `POST /auth/password`
- 400 `invalid_request_body`, `password_required`
- 401 `unauthorized`, `invalid_credentials`
- 500 `internal_error`
- 405 `method_not_allowed`

### `GET /me`
- 401 `unauthorized`
- 500 `internal_error`

### `POST /admin/events`
- 401 `unauthorized`
- 403 `forbidden`
- 400 `invalid_request_body`, `event_name_required`, `invalid_starts_at`
- 500 `internal_error`
- 405 `method_not_allowed`

### `GET /admin/events`
- 401 `unauthorized`
- 403 `forbidden`
- 500 `internal_error`
- 405 `method_not_allowed`

### `POST /admin/events/{event_id}/cancel`
- 401 `unauthorized`
- 403 `forbidden`
- 404 `not_found`, `invalid_id`, `event_not_found`
- 500 `internal_error`
- 405 `method_not_allowed`

### `POST /admin/events/{event_id}/zones`
- 401 `unauthorized`
- 403 `forbidden`
- 400 `invalid_request_body`, `zone_name_required`, `invalid_capacity`
- 404 `not_found`, `invalid_id`, `event_not_found`
- 409 `zone_already_exists`, `event_closed`, `event_cancelled`
- 500 `internal_error`
- 405 `method_not_allowed`

### `GET /admin/events/{event_id}/zones`
- 401 `unauthorized`
- 403 `forbidden`
- 404 `not_found`, `invalid_id`, `event_not_found`
- 500 `internal_error`
- 405 `method_not_allowed`

### `GET /admin/events/{event_id}/zones/{zone_id}/holds`
- 401 `unauthorized`
- 403 `forbidden`
- 404 `not_found`, `invalid_id`, `event_not_found`, `zone_not_found`
- 500 `internal_error`
- 405 `method_not_allowed`

### `GET /admin/events/{event_id}/zones/{zone_id}/orders`
- 401 `unauthorized`
- 403 `forbidden`
- 404 `not_found`, `invalid_id`, `event_not_found`, `zone_not_found`
- 500 `internal_error`
- 405 `method_not_allowed`

### `OPTIONS` (CORS preflight)
- 403 `forbidden`
