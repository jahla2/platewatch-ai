# PlateWatch

Real-time license plate detection, tracking, OCR consensus, watchlist matching, and operator monitoring.

## Architecture

PlateWatch uses a small-service architecture with clear responsibilities:

- **Python** — camera/video ingestion, computer vision, tracking, plate OCR, and OCR consensus.
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

## Run the platform locally

```bash
cp .env.example .env
docker compose up --build
```

Open:

- Dashboard: http://localhost:3000
- Go API: http://localhost:8080/healthz
- Python vision API: http://localhost:8000/healthz

## Run motorcycle detection

The detector accepts a webcam index, video file, or RTSP URL. Heavy computer-vision dependencies are isolated in the optional `vision` extra so normal API/unit-test development stays lightweight.

```bash
cd apps/vision
python -m pip install -e ".[vision,dev]"
```

Webcam:

```bash
platewatch-detect --source 0 --output ../../artifacts/webcam.mp4
```

Recorded video:

```bash
platewatch-detect \
  --source ./sample.mp4 \
  --model yolo11n.pt \
  --confidence 0.35 \
  --stride 2 \
  --output ../../artifacts/detected.mp4
```

RTSP:

```bash
platewatch-detect --source "rtsp://USER:PASSWORD@CAMERA/live"
```

Do not commit RTSP credentials. Use local environment/configuration when connecting to real cameras.

The command reports processing metrics such as frames read, frames inferred, detections, effective FPS, and average inference latency. With `--output`, detected motorcycles are written to an annotated MP4 with bounding boxes.

## End-to-end plate-event development test

Until the plate detector/OCR adapter is connected to the camera pipeline, the Python API also accepts OCR candidates so consensus, Go event flow, watchlist logic, and the dashboard can be tested end to end.

Send this request three times:

```bash
curl -X POST http://localhost:8000/v1/tracks/1/plate-candidates \
  -H "Content-Type: application/json" \
  -d '{"camera_id":"CAM-01","plate_text":"ABC-1234","ocr_confidence":0.95,"detection_confidence":0.95,"image_quality":0.95}'
```

On the third matching candidate, Python confirms the plate and sends it to Go. The default development watchlist includes `ABC1234`, so the dashboard receives a realtime **FLAGGED** event.

## Engineering rules

- Domain/application layers do not import database or ML frameworks.
- Camera and detector implementations sit behind narrow Python protocols.
- Heavy CV dependencies are optional for fast CI and backend/UI development.
- Slow realtime clients cannot block event ingestion.
- Secrets, RTSP credentials, private plate datasets, and model artifacts are not committed.
- Features are delivered through named branches, tests, pull requests, and review.

## Current milestone

Implemented:

- OpenCV camera/video/RTSP frame source
- small-model motorcycle detection adapter
- configurable confidence, image size, device, and inference stride
- optional annotated MP4 output
- pipeline FPS and inference-latency metrics
- plate normalization and temporal OCR consensus
- Python-to-Go event publishing
- Go clean application boundary and realtime SSE
- React detection dashboard
- Docker Compose and CI

Next: object tracking so each motorcycle receives a stable track ID before license-plate detection and OCR.

See [docs/PRD.md](docs/PRD.md) and [docs/architecture.md](docs/architecture.md).
