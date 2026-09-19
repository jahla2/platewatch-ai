from __future__ import annotations

import os
from dataclasses import dataclass


def _env_bool(name: str, default: bool) -> bool:
    value = os.getenv(name)
    if value is None:
        return default
    return value.strip().lower() in {"1", "true", "yes", "on"}


def _env_int(name: str, default: int) -> int:
    value = os.getenv(name)
    return int(value) if value not in (None, "") else default


def _env_float(name: str, default: float) -> float:
    value = os.getenv(name)
    return float(value) if value not in (None, "") else default


@dataclass(frozen=True, slots=True)
class VisionSettings:
    server_url: str
    internal_token: str
    auto_start: bool
    camera_id: str
    source: str
    vehicle_model: str
    vehicle_confidence: float
    plate_model: str
    plate_confidence: float
    image_size: int
    device: str | None
    inference_stride: int
    track_iou_threshold: float
    track_max_misses: int
    ocr_engine: str
    ocr_min_confidence: float
    ocr_confirmation_count: int
    ocr_interval_frames: int
    plate_min_quality: float
    evidence_dir: str
    evidence_url_prefix: str

    @classmethod
    def from_env(cls) -> VisionSettings:
        device = os.getenv("PLATEWATCH_VISION_DEVICE", "").strip() or None
        settings = cls(
            server_url=os.getenv("PLATEWATCH_SERVER_URL", "http://localhost:8080").strip(),
            internal_token=os.getenv("PLATEWATCH_INTERNAL_TOKEN", "").strip(),
            auto_start=_env_bool("PLATEWATCH_VISION_AUTO_START", False),
            camera_id=os.getenv("PLATEWATCH_CAMERA_ID", "CAM-01").strip(),
            source=os.getenv("PLATEWATCH_VISION_SOURCE", "0").strip(),
            vehicle_model=os.getenv("PLATEWATCH_VEHICLE_MODEL", "yolo11n.pt").strip(),
            vehicle_confidence=_env_float("PLATEWATCH_VEHICLE_CONFIDENCE", 0.35),
            plate_model=os.getenv("PLATEWATCH_PLATE_MODEL", "").strip(),
            plate_confidence=_env_float("PLATEWATCH_PLATE_CONFIDENCE", 0.50),
            image_size=_env_int("PLATEWATCH_VISION_IMAGE_SIZE", 640),
            device=device,
            inference_stride=_env_int("PLATEWATCH_INFERENCE_STRIDE", 2),
            track_iou_threshold=_env_float("PLATEWATCH_TRACK_IOU_THRESHOLD", 0.30),
            track_max_misses=_env_int("PLATEWATCH_TRACK_MAX_MISSES", 8),
            ocr_engine=os.getenv("PLATEWATCH_OCR_ENGINE", "paddleocr").strip().lower(),
            ocr_min_confidence=_env_float("PLATEWATCH_OCR_MIN_CONFIDENCE", 0.70),
            ocr_confirmation_count=_env_int("PLATEWATCH_OCR_CONFIRMATION_COUNT", 3),
            ocr_interval_frames=_env_int("PLATEWATCH_OCR_INTERVAL_FRAMES", 3),
            plate_min_quality=_env_float("PLATEWATCH_PLATE_MIN_QUALITY", 0.35),
            evidence_dir=os.getenv("PLATEWATCH_EVIDENCE_DIR", "/evidence").strip(),
            evidence_url_prefix=os.getenv("PLATEWATCH_EVIDENCE_URL_PREFIX", "/evidence").strip(),
        )
        settings.validate()
        return settings

    def validate(self) -> None:
        if not self.server_url:
            raise ValueError("PLATEWATCH_SERVER_URL is required")
        if not self.camera_id:
            raise ValueError("PLATEWATCH_CAMERA_ID is required")
        if not self.source:
            raise ValueError("PLATEWATCH_VISION_SOURCE is required")
        if not 0 <= self.vehicle_confidence <= 1:
            raise ValueError("PLATEWATCH_VEHICLE_CONFIDENCE must be between 0 and 1")
        if not 0 <= self.plate_confidence <= 1:
            raise ValueError("PLATEWATCH_PLATE_CONFIDENCE must be between 0 and 1")
        if self.image_size < 32:
            raise ValueError("PLATEWATCH_VISION_IMAGE_SIZE must be at least 32")
        if self.inference_stride < 1:
            raise ValueError("PLATEWATCH_INFERENCE_STRIDE must be positive")
        if not 0 <= self.track_iou_threshold <= 1:
            raise ValueError("PLATEWATCH_TRACK_IOU_THRESHOLD must be between 0 and 1")
        if self.track_max_misses < 0:
            raise ValueError("PLATEWATCH_TRACK_MAX_MISSES cannot be negative")
        if not 0 <= self.ocr_min_confidence <= 1:
            raise ValueError("PLATEWATCH_OCR_MIN_CONFIDENCE must be between 0 and 1")
        if self.ocr_confirmation_count < 1:
            raise ValueError("PLATEWATCH_OCR_CONFIRMATION_COUNT must be positive")
        if self.ocr_interval_frames < 1:
            raise ValueError("PLATEWATCH_OCR_INTERVAL_FRAMES must be positive")
        if not 0 <= self.plate_min_quality <= 1:
            raise ValueError("PLATEWATCH_PLATE_MIN_QUALITY must be between 0 and 1")
        if not self.evidence_dir:
            raise ValueError("PLATEWATCH_EVIDENCE_DIR is required")
        if not self.evidence_url_prefix.startswith("/"):
            raise ValueError("PLATEWATCH_EVIDENCE_URL_PREFIX must start with /")
