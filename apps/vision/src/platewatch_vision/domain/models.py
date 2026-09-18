from dataclasses import dataclass


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
