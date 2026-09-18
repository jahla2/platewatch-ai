from dataclasses import dataclass
from datetime import datetime
from typing import Any


@dataclass(frozen=True, slots=True)
class BoundingBox:
    x1: float
    y1: float
    x2: float
    y2: float


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


@dataclass(frozen=True, slots=True)
class PlateDecision:
    track_id: int
    camera_id: str
    plate: str
    confidence: float
    observations: int
