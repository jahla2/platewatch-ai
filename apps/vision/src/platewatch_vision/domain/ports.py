from typing import Protocol

from platewatch_vision.domain.models import (
    FramePacket,
    PlateDecision,
    VehicleDetection,
)


class FrameSource(Protocol):
    def read(self) -> FramePacket | None:
        """Return the next frame, or None when the source is exhausted."""

    def close(self) -> None:
        """Release the underlying video/camera resource."""


class VehicleDetector(Protocol):
    def detect(self, frame: FramePacket) -> list[VehicleDetection]:
        """Detect target vehicles in a frame."""


class FrameAnalysisSink(Protocol):
    def write(self, frame: FramePacket, detections: list[VehicleDetection]) -> None:
        """Persist or render a processed frame."""

    def close(self) -> None:
        """Release sink resources."""


class DetectionEventPublisher(Protocol):
    def publish(self, decision: PlateDecision) -> None:
        """Publish a confirmed plate decision to the application backend."""
