# PlateWatch V1 Product Requirements

## Objective

Build a near-real-time motorcycle license-plate monitoring prototype that detects and tracks motorcycles, recognizes visible license plates, checks confirmed plates against a configurable watchlist, and surfaces events on a web dashboard.

## V1 scope

- one test video, webcam, or RTSP source
- motorcycle detection
- object tracking
- plate detection
- OCR
- multi-frame OCR consensus
- normalized plate number and confidence
- watchlist lookup
- detection-event storage
- realtime dashboard events
- operator review status

## Out of scope for V1

- automatic legal penalties
- owner identification
- facial recognition
- government-database integration
- speed/red-light/counterflow enforcement
- multi-camera re-identification

## Functional flow

```text
frame
 -> motorcycle detection
 -> track ID
 -> plate crop
 -> OCR candidates
 -> temporal consensus
 -> confirmed plate
 -> Go API
 -> watchlist decision
 -> store event
 -> realtime browser event
```

## Quality rules

- do not OCR every frame
- do not query the watchlist repeatedly for the same finalized track
- normalize OCR before matching
- low-confidence detections require human review
- the vision service must drop stale frames instead of building an unbounded queue
- secrets, RTSP credentials, and private datasets must never be committed

## Initial acceptance criteria

- services build independently
- Python unit tests pass
- Go unit tests pass
- React production build passes
- confirmed plate candidates can be sent from Python to Go
- Go normalizes and checks the plate against a watchlist
- detection events are emitted to the browser through the realtime stream
