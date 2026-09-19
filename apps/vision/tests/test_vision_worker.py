import time
from datetime import UTC, datetime

from platewatch_vision.application.consensus import PlateConsensus
from platewatch_vision.application.processor import TrackPlateProcessor
from platewatch_vision.application.vision_worker import VisionWorker
from platewatch_vision.domain.models import (
    BoundingBox,
    FramePacket,
    OCRResult,
    PlateDecision,
    PlateDetection,
    TrackedVehicle,
    VehicleDetection,
)


class Source:
    def __init__(self) -> None:
        self.sent = False
        self.closed = False

    def read(self) -> FramePacket | None:
        if self.sent:
            time.sleep(0.01)
            return None
        self.sent = True
        return FramePacket(1, datetime.now(UTC), image="frame")

    def close(self) -> None:
        self.closed = True


class Detector:
    def detect(self, _frame: FramePacket) -> list[VehicleDetection]:
        return [VehicleDetection("motorcycle", 0.95, BoundingBox(0, 0, 100, 100))]


class Tracker:
    def update(self, _detections: list[VehicleDetection]) -> list[TrackedVehicle]:
        return [TrackedVehicle(7, "motorcycle", 0.95, BoundingBox(0, 0, 100, 100))]


class PlateDetectorFake:
    def detect(self, _image: object) -> list[PlateDetection]:
        return [PlateDetection(0.92, BoundingBox(0, 0, 50, 20))]


class OCR:
    def recognize(self, _image: object) -> OCRResult:
        return OCRResult(raw_text="ABC-1234", confidence=0.96)


class Images:
    def crop(self, _image: object, _box: BoundingBox) -> object:
        return object()

    def quality(self, _image: object) -> float:
        return 0.95


class Publisher:
    def __init__(self) -> None:
        self.events: list[PlateDecision] = []

    def publish(self, decision: PlateDecision) -> None:
        self.events.append(decision)


def test_worker_runs_full_plate_pipeline() -> None:
    publisher = Publisher()
    processor = TrackPlateProcessor(
        PlateConsensus(min_observations=1, min_confidence=0.5),
        publisher,
    )
    source = Source()
    worker = VisionWorker(
        camera_id="CAM-01",
        source=source,
        vehicle_detector=Detector(),
        tracker=Tracker(),
        plate_detector=PlateDetectorFake(),
        ocr=OCR(),
        image_processor=Images(),
        plate_processor=processor,
        inference_stride=1,
        ocr_interval_frames=1,
        ocr_min_confidence=0.7,
        plate_min_quality=0.3,
    )

    worker.start()
    deadline = time.time() + 1.0
    while not publisher.events and time.time() < deadline:
        time.sleep(0.01)
    worker.stop()

    assert publisher.events[0].plate_text == "ABC-1234"\n    assert publisher.events[0].plate_key == "ABC1234"
    assert publisher.events[0].track_id == 7
    assert source.closed is True
    assert worker.snapshot().confirmed_plates == 1
