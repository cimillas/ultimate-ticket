# Metrics reference

## Goal
Document the Prometheus metrics exposed by the API so they can be validated and
kept stable through extraction.

## Endpoint
- `GET /metrics`
- Auth: none
- Format: Prometheus text (`text/plain; version=0.0.4`)

## Metrics
### `http_requests_total`
- Type: counter
- Description: total number of HTTP requests handled.
- Labels:
  - `method`: HTTP method (e.g., `GET`, `POST`).
  - `route`: normalized route pattern.
  - `status`: HTTP status code as string (e.g., `200`, `401`).

### `http_request_duration_seconds`
- Type: histogram
- Description: request duration in seconds.
- Labels:
  - `method`: HTTP method.
  - `route`: normalized route pattern.
- Buckets: Prometheus default buckets.

## Route normalization
Metrics use normalized routes to avoid high cardinality. Unknown routes are
grouped under `unknown`.

Normalized patterns:
- `/health`
- `/metrics`
- `/events`
- `/events/{event_id}/zones`
- `/holds`
- `/holds/{hold_id}/confirm`
- `/orders`
- `/auth/register`
- `/auth/login`
- `/auth/logout`
- `/auth/password`
- `/me`
- `/admin/events`
- `/admin/events/{event_id}/cancel`
- `/admin/events/{event_id}/zones`
- `/admin/events/{event_id}/zones/{zone_id}/holds`
- `/admin/events/{event_id}/zones/{zone_id}/orders`

## Notes
- Metrics are collected by middleware and reflect post-handler status codes.
- Route normalization is path-based; method is tracked in a separate label.

## Example output
```text
# HELP http_request_duration_seconds Duration of HTTP requests in seconds.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",route="/health",le="0.005"} 1
http_request_duration_seconds_bucket{method="GET",route="/health",le="0.01"} 1
http_request_duration_seconds_bucket{method="GET",route="/health",le="0.025"} 1
http_request_duration_seconds_bucket{method="GET",route="/health",le="0.05"} 1
http_request_duration_seconds_bucket{method="GET",route="/health",le="0.1"} 1
http_request_duration_seconds_bucket{method="GET",route="/health",le="0.25"} 1
http_request_duration_seconds_bucket{method="GET",route="/health",le="0.5"} 1
http_request_duration_seconds_bucket{method="GET",route="/health",le="1"} 1
http_request_duration_seconds_bucket{method="GET",route="/health",le="2.5"} 1
http_request_duration_seconds_bucket{method="GET",route="/health",le="5"} 1
http_request_duration_seconds_bucket{method="GET",route="/health",le="10"} 1
http_request_duration_seconds_bucket{method="GET",route="/health",le="+Inf"} 1
http_request_duration_seconds_sum{method="GET",route="/health"} 0.001
http_request_duration_seconds_count{method="GET",route="/health"} 1
# HELP http_requests_total Total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="GET",route="/health",status="200"} 1
```
