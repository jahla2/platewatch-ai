from dataclasses import dataclass
from datetime import datetime
from typing import Any


@dataclass(frozen=True, slots=True)
class BoundingBox:
    x1: float
    y1: float
    x2: float
    y2: float

    @property
    def width(self) -> float:
        return max(0.0, self.x2 - self.x1)

    @property
    def height(self) -> float:
        return max(0.0, self.y2 - self.y1)


@dataclass(frozen=True, slots=True)
class FramePacket:
    sequence: int
    captured_at: datetime
    image: Any


@dataclass(frozen=True, slots=True)
class VehicleDetection:
    label: str
    confidence: float
    box: BoundingBox


@dataclass(frozen=True, slots=True)
class TrackedVehicle:
    track_id: int
    label: str
    confidence: float
    box: BoundingBox


@dataclass(frozen=True, slots=True)
class PlateDetection:
    confidence: float
    box: BoundingBox


@dataclass(frozen=True, slots=True)
class OCRResult:
    raw_text: str
    confidence: float


@dataclass(frozen=True, slots=True)
class EvidenceRefs:
    snapshot_url: str
    plate_crop_url: str


@dataclass(frozen=True, slots=True)
class PipelineMetrics:
    frames_read: int
    frames_inferred: int
    detections: int
    elapsed_seconds: float
    source_fps: float
    inference_fps: float
    average_inference_ms: float


@dataclass(frozen=True, slots=True)
class PlateCandidate:
    track_id: int
    camera_id: str
    plate_text: str
    ocr_confidence: float
    detection_confidence: float
    image_quality: float
    snapshot_url: str = ""
    plate_crop_url: str = ""


@dataclass(frozen=True, slots=True)
class PlateDecision:
    track_id: int
    camera_id: str
    plate_text: str
    plate_key: str
    confidence: float
    observations: int
    snapshot_url: str = ""
    plate_crop_url: str = ""
