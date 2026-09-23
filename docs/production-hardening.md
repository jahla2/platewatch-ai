# Production Hardening

This document records the production controls implemented in PlateWatch and the
controls that are intentionally not applicable to the current product.

## Implemented controls

### API protection

- per-client fixed-window rate limits for public/operator, internal-ingest,
  admin, and login routes
- bounded request bodies
- strict JSON decoding
- request timeouts for non-streaming endpoints
- graceful process shutdown
- panic recovery
- security headers
- exact-origin CORS behavior
- long random secrets validated at startup

### Authentication and authorization

- Python vision -> Go ingest uses a dedicated internal bearer token
- watchlist administration uses a separate admin bearer token
- operator detection history and SSE access require an HttpOnly, SameSite=Strict
  session cookie created from the configured operator token
- the operator token is submitted only during login and is not stored in
  browser JavaScript/localStorage
- production deployments should set `PLATEWATCH_SESSION_SECURE=true` and
  terminate HTTPS before the web service

The current session model is intentionally single-operator/self-hosted. A
multi-user deployment should replace it with an identity provider and RBAC
rather than extending static-token logic.

### Duplicate protection / idempotency

Vision sends a deterministic per-worker detection idempotency key. PostgreSQL
enforces uniqueness using a partial unique index. A retry returns the existing
event instead of inserting another row. Concurrent duplicate requests are
covered by integration tests.

### Database/query performance

- no ORM/lazy loading
- watchlist lookup is one indexed `EXISTS` query
- detection history uses keyset pagination ordered by
  `(detected_at DESC, id DESC)`
- indexes cover detection cursor pagination, plate/time, camera/time, and
  active watchlist lookup
- large history scans do not use growing OFFSET values

### Reliability

- vision -> Go delivery has bounded retry attempts, exponential backoff,
  request timeouts, and stable idempotency keys across retries
- event-delivery failures do not terminate the vision worker
- evidence-write failures degrade to a detection without evidence instead of
  terminating the worker
- OCR candidate history is bounded per track
- SSE includes the event ID and browser state de-duplicates events by ID
- React handles loading, empty, error, retry, reconnecting, and pagination
  states

### Observability

Go exposes:

```text
GET /healthz
GET /readyz
GET /metrics
```

The server logs structured JSON request/error records including status,
duration, path, request ID, and client IP.

Vision exposes:

```text
GET /healthz
GET /readyz
GET /v1/status
```

The vision status includes confirmed plates, worker state, and delivery
failures.

### Concurrency and failure tests

CI runs:

- Python lint and unit/failure-path tests
- Go tests under the race detector
- PostgreSQL integration tests, including simultaneous duplicate detection
  writes
- React TypeScript/Vite production build
- Docker Compose validation and image builds
- PostgreSQL backup + restore verification

## Intentionally not applicable

### External AI/API spending caps

PlateWatch currently runs its detection/OCR models locally and does not call a
metered LLM/image/AI API. There is therefore no external AI spend to meter or
cap. Adding a fake budget subsystem would add complexity without protecting
anything.

If a metered external provider is added later, introduce a provider adapter
with per-tenant/request quotas and a hard budget gate before the first billable
request.

### Payment idempotency

PlateWatch has no payment flow. Detection-event idempotency is implemented
because it is relevant; payment-specific idempotency is not.

### Upload compression and upload limits

There is no public file-upload API. Camera/video ingestion is a configured
runtime source. Request bodies that do exist are size-limited. If an upload
endpoint is introduced later, add MIME/signature validation, size limits,
streaming storage, and media compression at that boundary.

### Application response caching

Detection history and watchlist decisions are realtime/security-sensitive.
PlateWatch intentionally avoids application-level response caching that could
serve stale enforcement/watchlist state. Model downloads are safely cached in
the persistent Docker model volume.

## Remaining deployment responsibilities

- use HTTPS/TLS in front of Nginx and enable secure session cookies
- rotate all secrets from the example values
- configure retention/deletion policy for plate evidence
- connect metrics/logs to the monitoring platform used by the deployment
- run real camera/video load benchmarks on the target hardware
- for multi-instance Go deployment, replace the in-process rate limiter and SSE
  broker with shared infrastructure or gateway-level equivalents
