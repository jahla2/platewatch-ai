import os

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

from platewatch_vision.application.consensus import PlateConsensus
from platewatch_vision.application.processor import TrackPlateProcessor
from platewatch_vision.domain.models import PlateCandidate
from platewatch_vision.infrastructure.http_publisher import HttpDetectionEventPublisher

app = FastAPI(title="PlateWatch Vision", version="0.1.0")

_publisher = HttpDetectionEventPublisher(
    os.getenv("PLATEWATCH_SERVER_URL", "http://localhost:8080")
)
_processor = TrackPlateProcessor(PlateConsensus(), _publisher)


class CandidateRequest(BaseModel):
    camera_id: str = Field(min_length=1)
    plate_text: str = Field(min_length=1)
    ocr_confidence: float = Field(ge=0, le=1)
    detection_confidence: float = Field(ge=0, le=1)
    image_quality: float = Field(ge=0, le=1)


@app.get("/healthz")
def health() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/v1/tracks/{track_id}/plate-candidates")
def add_plate_candidate(track_id: int, request: CandidateRequest) -> dict[str, object]:
    try:
        decision = _processor.add_candidate(
            PlateCandidate(
                track_id=track_id,
                camera_id=request.camera_id,
                plate_text=request.plate_text,
                ocr_confidence=request.ocr_confidence,
                detection_confidence=request.detection_confidence,
                image_quality=request.image_quality,
            )
        )
    except (ValueError, OSError, RuntimeError) as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc

    if decision is None:
        return {"status": "collecting"}

    return {
        "status": "confirmed",
        "plate": decision.plate,
        "confidence": decision.confidence,
        "observations": decision.observations,
    }
