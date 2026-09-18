# PlateWatch Architecture

## Goal

PlateWatch is split into three application services so each part has one clear responsibility:

- **Vision (Python):** frame processing, detection, tracking, OCR, and OCR consensus.
- **Server (Go):** validation, watchlist decisions, persistence boundary, and realtime event delivery.
- **Web (React):** operator dashboard and event visualization.

This keeps computer-vision workloads independent from API and UI concerns.

## V1 data flow

```text
Camera / test video
        |
        v
Python vision pipeline
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

The initial repository uses in-memory Go adapters so the application boundary is executable and testable immediately. PostgreSQL is included in local infrastructure and will replace the in-memory repository through the existing repository interface.

## Clean architecture boundaries

### Python

The Python application depends on protocols rather than concrete model implementations. Detector, recognizer, tracker, and event publisher adapters can be changed without modifying plate consensus rules.

### Go

The Go application service depends on interfaces for:

- detection repository
- watchlist
- event publisher

Transport and persistence are adapters around the application layer. Domain objects contain no HTTP or database code.

### React

The frontend isolates API/event-stream code from rendering. Components receive domain-shaped data instead of knowing transport details.

## Why SSE for V1

Realtime detection delivery is server-to-browser only. Server-Sent Events provide automatic browser reconnection and require no third-party Go dependency. If the product later needs bidirectional realtime commands, the realtime port can be backed by WebSockets without changing the detection application service.

## Planned adapters

1. YOLO motorcycle detector
2. ByteTrack/BoT-SORT tracker
3. fine-tuned license-plate detector
4. OCR adapter
5. PostgreSQL repository
6. snapshot/object-storage adapter
7. RTSP/GStreamer frame source
