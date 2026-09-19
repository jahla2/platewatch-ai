from __future__ import annotations

from platewatch_vision.application.consensus import PlateConsensus
from platewatch_vision.application.processor import TrackPlateProcessor
from platewatch_vision.application.vision_worker import VisionWorker
from platewatch_vision.config.settings import VisionSettings
from platewatch_vision.infrastructure.evidence_store import OpenCVEvidenceStore
from platewatch_vision.infrastructure.http_publisher import HttpDetectionEventPublisher
from platewatch_vision.infrastructure.image_processor import OpenCVImageProcessor
from platewatch_vision.infrastructure.iou_tracker import IoUTracker
from platewatch_vision.infrastructure.latest_frame_source import LatestFrameOpenCVSource
from platewatch_vision.infrastructure.opencv_source import resolve_video_source
from platewatch_vision.infrastructure.paddle_ocr import PaddleOCRRecognizer
from platewatch_vision.infrastructure.ultralytics_detector import (
    UltralyticsMotorcycleDetector,
)
from platewatch_vision.infrastructure.ultralytics_plate_detector import (
    UltralyticsPlateDetector,
)


def build_plate_processor(settings: VisionSettings) -> TrackPlateProcessor:
    publisher = HttpDetectionEventPublisher(
        server_url=settings.server_url,
        internal_token=settings.internal_token,
    )
    consensus = PlateConsensus(
        min_observations=settings.ocr_confirmation_count,
        min_confidence=settings.ocr_min_confidence,
    )
    return TrackPlateProcessor(consensus=consensus, publisher=publisher)


def build_vision_worker(
    settings: VisionSettings,
    plate_processor: TrackPlateProcessor,
) -> VisionWorker:
    if not settings.plate_model:
        raise RuntimeError(
            "PLATEWATCH_PLATE_MODEL is required when PLATEWATCH_VISION_AUTO_START=true"
        )
    if settings.ocr_engine != "paddleocr":
        raise RuntimeError(f"unsupported OCR engine: {settings.ocr_engine}")

    return VisionWorker(
        camera_id=settings.camera_id,
        source=LatestFrameOpenCVSource(resolve_video_source(settings.source)),
        vehicle_detector=UltralyticsMotorcycleDetector(
            model_path=settings.vehicle_model,
            confidence=settings.vehicle_confidence,
            image_size=settings.image_size,
            device=settings.device,
        ),
        tracker=IoUTracker(
            iou_threshold=settings.track_iou_threshold,
            max_misses=settings.track_max_misses,
        ),
        plate_detector=UltralyticsPlateDetector(
            model_path=settings.plate_model,
            confidence=settings.plate_confidence,
            image_size=settings.image_size,
            device=settings.device,
        ),
        ocr=PaddleOCRRecognizer(),
        image_processor=OpenCVImageProcessor(),
        plate_processor=plate_processor,
        evidence_store=OpenCVEvidenceStore(
            root_dir=settings.evidence_dir,
            url_prefix=settings.evidence_url_prefix,
        ),
        inference_stride=settings.inference_stride,
        ocr_interval_frames=settings.ocr_interval_frames,
        ocr_min_confidence=settings.ocr_min_confidence,
        plate_min_quality=settings.plate_min_quality,
    )
