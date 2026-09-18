# PlateWatch Architecture

## Goal

PlateWatch is split into three application services so each part has one clear responsibility:

- **Vision (Python):** frame processing, detection, tracking, OCR, and OCR consensus.
- **Server (Go):** validation, watchlist decisions, persistence boundary, and realtime event delivery.
- **Web (React):** operator dashboard and event visualization.

This keeps computer-vision workloads independent from API and UI concerns.

## V1 data flow

```text
Camera / video / RTSP
        |
        v
FrameSource protocol
        |
        v
CameraDetectionPipeline
        |
        v
VehicleDetector protocol
        |
        +--> Ultralytics motorcycle adapter
        |
        v
tracking (next milestone)
        |
        v
plate detector + OCR
        |
        | POST confirmed plate event
        v
Go application service
        |
        +--> watchlist lookup
        +--> repository
        +--> realtime publisher
                    |
                    v
                  SSE
                    |
                    v
               React UI
```

The initial Go repository adapter is in memory so the application boundary is executable and testable immediately. PostgreSQL is included in local infrastructure and will replace it through the existing repository interface.

## Python vision boundaries

The core pipeline knows only these ports:

- `FrameSource`
- `VehicleDetector`
- `FrameAnalysisSink`
- `DetectionEventPublisher`

Concrete camera and model libraries remain in infrastructure adapters.

Current adapters:

- `OpenCVFrameSource` — webcam, local video, or RTSP source
- `UltralyticsMotorcycleDetector` — small configurable detector
- `OpenCVAnnotatedVideoSink` — optional bounding-box MP4 output

Heavy CV dependencies are optional in `pyproject.toml`. This keeps normal unit tests and CI lightweight while the Docker image installs the full vision runtime.

The detection loop can skip inference using `inference_stride`. For example, stride 2 reads every frame but runs the detector on frames 1, 3, 5, and so on. Tracking will later carry identities across skipped frames.

## Go boundaries

The Go application service depends on interfaces for:

- detection repository
- watchlist
- event publisher

Transport and persistence are adapters around the application layer. Domain objects contain no HTTP or database code.

## React boundary

The frontend isolates API/event-stream code from rendering. Components receive domain-shaped data instead of knowing transport details.

## Why SSE for V1

Realtime detection delivery is server-to-browser only. Server-Sent Events provide automatic browser reconnection and require no third-party Go dependency. If the product later needs bidirectional realtime commands, the realtime port can be backed by WebSockets without changing the detection application service.

## Remaining V1 adapters

1. object tracker
2. fine-tuned license-plate detector
3. OCR adapter
4. PostgreSQL repository
5. snapshot/object-storage adapter
6. low-latency latest-frame RTSP reader for multi-camera deployment
