# PlateWatch

Real-time license plate detection, tracking, OCR consensus, watchlist matching, and operator monitoring.

## Architecture

PlateWatch uses a small-service architecture with clear responsibilities:

- **Python** — computer-vision pipeline and OCR consensus.
- **Go** — validation, watchlist decisions, persistence boundaries, and realtime event delivery.
- **React + TypeScript** — operator dashboard.
- **PostgreSQL** — target persistent store.

The application and domain layers depend on interfaces/protocols rather than concrete HTTP, database, or ML frameworks.

## Repository layout

```text
apps/
  vision/   Python vision service
  server/   Go application backend
  web/      React dashboard

docs/
  PRD.md
  architecture.md

.github/workflows/
  ci.yml
```

## Run locally

```bash
cp .env.example .env
docker compose up --build
```

Open:

- Dashboard: http://localhost:3000
- Go API: http://localhost:8080/healthz
- Python vision API: http://localhost:8000/healthz

## End-to-end development test

Until detector/OCR adapters are added, the Python service accepts OCR candidates so the consensus, Go event flow, watchlist logic, and dashboard can be tested end to end.

Send this request three times:

```bash
curl -X POST http://localhost:8000/v1/tracks/1/plate-candidates \
  -H "Content-Type: application/json" \
  -d '{"camera_id":"CAM-01","plate_text":"ABC-1234","ocr_confidence":0.95,"detection_confidence":0.95,"image_quality":0.95}'
```

On the third matching candidate, Python confirms the plate and sends it to Go. The default development watchlist includes `ABC1234`, so the dashboard receives a realtime **FLAGGED** event.

## Engineering rules

- Domain/application layers do not import database or ML frameworks.
- Services depend on narrow interfaces/protocols for external concerns.
- Slow realtime clients cannot block event ingestion.
- Secrets, RTSP credentials, private plate datasets, and model artifacts are not committed.
- Features are delivered through named branches, tests, pull requests, and review.

## Current milestone

Implemented in the foundation:

- plate normalization and temporal OCR consensus
- one-confirmed-event-per-track processing
- Python-to-Go event publishing
- Go application service with repository/watchlist/publisher ports
- realtime Server-Sent Events (SSE)
- React detection dashboard
- Docker Compose
- Python, Go, and React CI

Next: camera capture, motorcycle detection, tracking, and license-plate detector adapters.

See [docs/PRD.md](docs/PRD.md) and [docs/architecture.md](docs/architecture.md).
