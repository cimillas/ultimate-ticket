# API error codes

## Error format
All error responses are JSON with a stable code:

```json
{"error":"<message>","code":"<code>"}
```

## Code reference
- `method_not_allowed` - HTTP method is not supported for the endpoint.
- `not_found` - Endpoint path does not match a known route.
- `invalid_request_body` - Request JSON is invalid or cannot be parsed.
- `missing_required_field` - Required field(s) are missing.
- `invalid_starts_at` - Invalid RFC3339 timestamp for `starts_at`.
- `invalid_id` - Provided ID is invalid (path or body).
- `event_name_required` - Event name is required.
- `zone_name_required` - Zone name is required.
- `invalid_quantity` - Quantity must be greater than zero.
- `invalid_capacity` - Capacity must be greater than zero.
- `idempotency_key_required` - Idempotency key is required.
- `idempotency_conflict` - Idempotency key already used with different payload.
- `insufficient_capacity` - Not enough inventory available in the zone.
- `zone_not_found` - Zone does not exist for the event.
- `event_not_found` - Event does not exist.
- `zone_already_exists` - Zone with same name already exists for the event.
- `hold_not_found` - Hold does not exist.
- `hold_expired` - Hold has expired.
- `hold_already_confirmed` - Hold is already confirmed.
- `forbidden` - Request is blocked by CORS allow-list.
- `internal_error` - Unexpected server error.

## Endpoint mapping

### `POST /holds`
- 400 `invalid_request_body`, `missing_required_field`, `idempotency_key_required`, `invalid_quantity`, `invalid_id`
- 404 `zone_not_found`
- 409 `idempotency_conflict`, `insufficient_capacity`
- 500 `internal_error`
- 405 `method_not_allowed`

### `POST /holds/{hold_id}/confirm`
- 400 `idempotency_key_required`
- 404 `not_found`, `invalid_id`, `hold_not_found`
- 409 `hold_expired`, `hold_already_confirmed`
- 500 `internal_error`
- 405 `method_not_allowed`

### `POST /admin/events`
- 400 `invalid_request_body`, `event_name_required`, `invalid_starts_at`
- 500 `internal_error`
- 405 `method_not_allowed`

### `GET /admin/events`
- 500 `internal_error`
- 405 `method_not_allowed`

### `POST /admin/events/{event_id}/zones`
- 400 `invalid_request_body`, `zone_name_required`, `invalid_capacity`
- 404 `not_found`, `invalid_id`, `event_not_found`
- 409 `zone_already_exists`
- 500 `internal_error`
- 405 `method_not_allowed`

### `GET /admin/events/{event_id}/zones`
- 404 `not_found`, `invalid_id`, `event_not_found`
- 500 `internal_error`
- 405 `method_not_allowed`

### `OPTIONS` (CORS preflight)
- 403 `forbidden`
