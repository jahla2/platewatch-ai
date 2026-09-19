# PlateWatch

Real-time motorcycle license-plate detection, tracking, OCR consensus, watchlist matching, persistence, and operator monitoring.

## Stack

- **Python** — live camera/RTSP capture, vehicle detection, tracking, plate detection, OCR, temporal consensus
- **Go** — application API, PostgreSQL persistence, watchlist logic, realtime SSE, service authentication
- **React + TypeScript** — operator dashboard
- **PostgreSQL** — detection history and watchlist
- **Docker Compose** — local orchestration

## Runtime flow

```text
Camera / RTSP / video
        ↓
latest-frame capture
        ↓
motorcycle detector
        ↓
tracker
        ↓
license-plate detector
        ↓
plate quality filter
        ↓
OCR
        ↓
temporal consensus
        ↓
authenticated internal Go API
        ↓
PostgreSQL + watchlist lookup
        ↓
SSE
        ↓
React dashboard
```

## Quick start

Create local configuration:

```bash
cp .env.example .env
```

Before running, change at minimum:

```env
PLATEWATCH_INTERNAL_TOKEN=...
PLATEWATCH_ADMIN_TOKEN=...
POSTGRES_PASSWORD=...
DATABASE_URL=postgres://platewatch:<same-password>@postgres:5432/platewatch?sslmode=disable
```

Start the platform:

```bash
docker compose up --build
```

Then open:

```text
http://localhost:3000
```

The web container is the public entry point and reverse-proxies `/api/*` and the SSE stream to the Go service. PostgreSQL, Go, and Python remain on the internal Compose network except for the optional loopback PostgreSQL development port.

## Models

Compose bootstraps two model files into the `platewatch_models` volume when missing:

- a small vehicle detector
- a generic license-plate detector

Model URLs are configurable through:

```env
PLATEWATCH_VEHICLE_MODEL_URL=...
PLATEWATCH_PLATE_MODEL_URL=...
```

Model binaries are deliberately not committed to Git.

The bundled plate-model URL is intended only to make the prototype runnable. For Philippine motorcycle plates, replace it with a validated/fine-tuned model before treating results as field-quality.

## Enable live vision

The default is:

```env
PLATEWATCH_VISION_AUTO_START=false
```

This lets the whole app boot without requiring a camera. To process RTSP:

```env
PLATEWATCH_VISION_AUTO_START=true
PLATEWATCH_CAMERA_ID=GATE-01
PLATEWATCH_VISION_SOURCE=rtsp://user:password@camera/live
```

Keep camera credentials only in your local `.env`; never commit them.

For a directly attached webcam inside Docker, map the camera device to the vision container for your OS/runtime. For initial testing, RTSP is the simplest Docker path.

## Health endpoints

Internal service health checks:

```text
Go:     /healthz
Vision: /healthz
Vision: /readyz
Vision: /v1/status
```

The Compose dependency graph waits for PostgreSQL and Go health before dependent services start.

## Database/query design

Detection history is persisted in PostgreSQL. Important access paths are indexed by:

- detection time
- normalized plate number + time
- camera + time
- active watchlist plate

The current history endpoint is a single ordered query, and watchlist matching is a single indexed `EXISTS` query. There is no ORM/lazy-loading path that produces N+1 queries.

## Security boundaries

Vision sends confirmed plate events only to:

```text
POST /internal/v1/detections
```

using `PLATEWATCH_INTERNAL_TOKEN`.

Watchlist mutation endpoints require `PLATEWATCH_ADMIN_TOKEN`.

The Go server also applies request-size limits, strict JSON decoding, basic security headers, and HTTP timeouts. The current project still needs a real end-user login/session system before internet-facing production deployment.

## Development

Python checks:

```bash
python -m pip install -e "apps/vision[dev]"
ruff check apps/vision
pytest apps/vision/tests
```

Go checks:

```bash
cd apps/server
go mod tidy
gofmt -w .
go test ./...
```

React:

```bash
cd apps/web
npm install
npm run build
```

See [docs/PRD.md](docs/PRD.md), [docs/architecture.md](docs/architecture.md), and [docs/models.md](docs/models.md).
