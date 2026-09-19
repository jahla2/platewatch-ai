from typing import Any, Protocol

from platewatch_vision.domain.models import (
    BoundingBox,
    EvidenceRefs,
    FramePacket,
    OCRResult,
    PlateDecision,
    PlateDetection,
    TrackedVehicle,
    VehicleDetection,
)


class FrameSource(Protocol):
    def read(self) -> FramePacket | None:
        """Return the next/current frame, or None when unavailable."""

    def close(self) -> None:
        """Release the underlying video/camera resource."""


class VehicleDetector(Protocol):
    def detect(self, frame: FramePacket) -> list[VehicleDetection]:
        """Detect target vehicles in a frame."""


class ObjectTracker(Protocol):
    def update(self, detections: list[VehicleDetection]) -> list[TrackedVehicle]:
        """Assign stable track IDs to vehicle detections."""


class PlateDetector(Protocol):
    def detect(self, image: Any) -> list[PlateDetection]:
        """Detect license plates within a vehicle crop."""


class OCRRecognizer(Protocol):
    def recognize(self, image: Any) -> OCRResult | None:
        """Read normalized candidate text from a license-plate crop."""


class ImageProcessor(Protocol):
    def crop(self, image: Any, box: BoundingBox) -> Any | None:
        """Crop a bounded region from an image."""

    def quality(self, image: Any) -> float:
        """Return a normalized 0..1 image quality score."""


class EvidenceStore(Protocol):
    def save(
        self,
        camera_id: str,
        track_id: int,
        snapshot: Any,
        plate_crop: Any,
    ) -> EvidenceRefs:
        """Persist the latest evidence for a track and return browser-safe URLs."""


class FrameAnalysisSink(Protocol):
    def write(self, frame: FramePacket, detections: list[VehicleDetection]) -> None:
        """Persist or render a processed frame."""

    def close(self) -> None:
        """Release sink resources."""


class DetectionEventPublisher(Protocol):
    def publish(self, decision: PlateDecision) -> None:
        """Publish a confirmed plate decision to the application backend."""
