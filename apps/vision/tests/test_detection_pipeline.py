from datetime import UTC, datetime

from platewatch_vision.application.detection_pipeline import CameraDetectionPipeline
from platewatch_vision.domain.models import BoundingBox, FramePacket, VehicleDetection


class FakeSource:
    def __init__(self, count: int) -> None:
        self.frames = [
            FramePacket(sequence=index, captured_at=datetime.now(UTC), image=object())
            for index in range(1, count + 1)
        ]
        self.closed = False

    def read(self) -> FramePacket | None:
        return self.frames.pop(0) if self.frames else None

    def close(self) -> None:
        self.closed = True


class FakeDetector:
    def __init__(self) -> None:
        self.sequences: list[int] = []

    def detect(self, frame: FramePacket) -> list[VehicleDetection]:
        self.sequences.append(frame.sequence)
        return [
            VehicleDetection(
                label="motorcycle",
                confidence=0.91,
                box=BoundingBox(10, 20, 110, 120),
            )
        ]


class FakeSink:
    def __init__(self) -> None:
        self.frames: list[tuple[int, int]] = []
        self.closed = False

    def write(self, frame: FramePacket, detections: list[VehicleDetection]) -> None:
        self.frames.append((frame.sequence, len(detections)))

    def close(self) -> None:
        self.closed = True


def test_pipeline_applies_inference_stride_and_closes_resources() -> None:
    source = FakeSource(count=5)
    detector = FakeDetector()
    sink = FakeSink()

    metrics = CameraDetectionPipeline(
        source=source,
        detector=detector,
        sink=sink,
        inference_stride=2,
    ).run()

    assert detector.sequences == [1, 3, 5]
    assert sink.frames == [(1, 1), (2, 0), (3, 1), (4, 0), (5, 1)]
    assert metrics.frames_read == 5
    assert metrics.frames_inferred == 3
    assert metrics.detections == 3
    assert source.closed is True
    assert sink.closed is True


def test_pipeline_honors_max_frames() -> None:
    source = FakeSource(count=10)

    metrics = CameraDetectionPipeline(
        source=source,
        detector=FakeDetector(),
    ).run(max_frames=2)

    assert metrics.frames_read == 2
    assert source.closed is True
