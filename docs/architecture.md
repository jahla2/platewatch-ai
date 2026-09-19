# PlateWatch Architecture

## System boundary

PlateWatch separates computer-vision processing from application/business logic:

```text
                        ┌──────────────────────┐
Camera / RTSP ─────────►│ Python Vision       │
                        │ latest frame buffer  │
                        │ vehicle detector     │
                        │ tracker              │
                        │ plate detector       │
                        │ OCR + consensus      │
                        └──────────┬───────────┘
                                   │
                          authenticated event
                                   │
                                   ▼
                        ┌──────────────────────┐
                        │ Go Application API   │
                        │ validation           │
                        │ watchlist lookup     │
                        │ persistence          │
                        │ realtime publisher   │
                        └───────┬──────┬───────┘
                                │      │
                           PostgreSQL  SSE
                                       │
                                       ▼
                                React dashboard
```

## Vision design

The application layer depends on narrow ports:

- `FrameSource`
- `VehicleDetector`
- `ObjectTracker`
- `PlateDetector`
- `OCRRecognizer`
- `ImageProcessor`
- `DetectionEventPublisher`

Concrete OpenCV, Ultralytics, OCR, and HTTP implementations live in infrastructure adapters.

For live feeds, `LatestFrameOpenCVSource` continuously captures frames and overwrites the previous unread frame. Inference therefore operates on the newest available frame rather than accumulating an unbounded RTSP queue.

Tracking currently uses a lightweight IoU tracker suitable for the mini-project and fixed cameras. Because tracking is behind a port, ByteTrack/BoT-SORT can replace it without changing the worker/application orchestration.

## Runtime configuration

Model paths, thresholds, camera source, inference stride, OCR thresholds, and tracker settings are environment-backed through `VisionSettings`.

The Docker runtime mounts model files read-only into the vision service. A one-shot model initializer downloads configured weights only when the volume is empty.

## Server design

The Go application layer depends on:

- `DetectionRepository`
- `Watchlist`
- `WatchlistStore`
- `DetectionPublisher`
- `DetectionSubscriber`

PostgreSQL implements persistence ports using `pgxpool`.

The persistence layer uses explicit SQL rather than lazy ORM relations. Detection history is retrieved in one ordered query, and watchlist matching uses an indexed `EXISTS` lookup.

## API/security split

Public/operator-read paths:

```text
GET /api/v1/detections
GET /api/v1/events/stream
```

Internal machine path:

```text
POST /internal/v1/detections
Authorization: Bearer <PLATEWATCH_INTERNAL_TOKEN>
```

Administrative watchlist paths require a separate admin bearer token.

For a public deployment, operator read paths should also be moved behind user/session authentication.

## Web/reverse proxy

Nginx serves the compiled React application and proxies same-origin `/api/*` requests to Go. The SSE route has buffering disabled so detections reach the browser immediately.

This avoids exposing the Go port directly and removes normal browser CORS dependence for the deployed UI.

## Docker dependency graph

```text
model-init ─────────────► vision
                           ▲
postgres ──healthy──► server
                       │   ▲
                       │   │
                       └──► web
```

The vision service waits for both model initialization and a healthy Go server. The Go server waits for healthy PostgreSQL.

## Remaining production hardening

The V1 architecture is intentionally small. Production expansion should add:

- proper operator identities, sessions, and RBAC
- evidence/snapshot object storage with retention controls
- metrics/telemetry and alerting
- GPU-specific runtime/image profile
- validated/fine-tuned Philippine plate model
- multi-camera scheduling and per-camera worker lifecycle
