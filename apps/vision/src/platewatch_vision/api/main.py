from __future__ import annotations

from contextlib import asynccontextmanager
from dataclasses import asdict

from fastapi import FastAPI, HTTPException, Response, status
from pydantic import BaseModel, Field

from platewatch_vision.application.bootstrap import (
    build_plate_processor,
    build_vision_worker,
)
from platewatch_vision.application.vision_worker import VisionWorker
from platewatch_vision.config.settings import VisionSettings
from platewatch_vision.domain.models import PlateCandidate

_settings = VisionSettings.from_env()
_processor = build_plate_processor(_settings)
_worker: VisionWorker | None = None
_startup_error: str | None = None


@asynccontextmanager
async def lifespan(_: FastAPI):
    global _worker, _startup_error

    if _settings.auto_start:
        try:
            _worker = build_vision_worker(_settings, _processor)
            _worker.start()
        except Exception as exc:
            _startup_error = str(exc)

    try:
        yield
    finally:
        if _worker is not None:
            _worker.stop()


app = FastAPI(title="PlateWatch Vision", version="0.3.0", lifespan=lifespan)


class CandidateRequest(BaseModel):
    camera_id: str = Field(min_length=1)
    plate_text: str = Field(min_length=1)
    ocr_confidence: float = Field(ge=0, le=1)
    detection_confidence: float = Field(ge=0, le=1)
    image_quality: float = Field(ge=0, le=1)


@app.get("/healthz")
def health() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/readyz")
def ready(response: Response) -> dict[str, object]:
    if not _settings.auto_start:
        return {"status": "ready", "vision_worker": "disabled"}

    if _startup_error:
        response.status_code = status.HTTP_503_SERVICE_UNAVAILABLE
        return {"status": "not_ready", "error": _startup_error}

    if _worker is None:
        response.status_code = status.HTTP_503_SERVICE_UNAVAILABLE
        return {"status": "not_ready", "error": "vision worker has not started"}

    snapshot = _worker.snapshot()
    if not snapshot.ready:
        response.status_code = status.HTTP_503_SERVICE_UNAVAILABLE

    return {"status": "ready" if snapshot.ready else "not_ready", **asdict(snapshot)}


@app.get("/v1/status")
def vision_status() -> dict[str, object]:
    worker = asdict(_worker.snapshot()) if _worker is not None else None
    return {
        "camera_id": _settings.camera_id,
        "auto_start": _settings.auto_start,
        "source_configured": bool(_settings.source),
        "vehicle_model": _settings.vehicle_model,
        "plate_model_configured": bool(_settings.plate_model),
        "worker": worker,
        "startup_error": _startup_error,
    }


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
        "plate_text": decision.plate_text,
        "plate_key": decision.plate_key,
        "confidence": decision.confidence,
        "observations": decision.observations,
    }
