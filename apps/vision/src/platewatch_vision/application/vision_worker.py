from __future__ import annotations

import logging
import threading
from dataclasses import dataclass

from platewatch_vision.application.processor import TrackPlateProcessor
from platewatch_vision.domain.errors import EventDeliveryError
from platewatch_vision.domain.models import PlateCandidate
from platewatch_vision.domain.ports import (
    EvidenceStore,
    FrameSource,
    ImageProcessor,
    ObjectTracker,
    OCRRecognizer,
    PlateDetector,
    VehicleDetector,
)

logger = logging.getLogger(__name__)


@dataclass(frozen=True, slots=True)
class VisionWorkerSnapshot:
    running: bool
    ready: bool
    frames_seen: int
    inference_frames: int
    tracked_vehicles: int
    ocr_attempts: int
    confirmed_plates: int
    publish_failures: int
    last_error: str | None


class VisionWorker:
    def __init__(
        self,
        camera_id: str,
        source: FrameSource,
        vehicle_detector: VehicleDetector,
        tracker: ObjectTracker,
        plate_detector: PlateDetector,
        ocr: OCRRecognizer,
        image_processor: ImageProcessor,
        plate_processor: TrackPlateProcessor,
        evidence_store: EvidenceStore | None = None,
        inference_stride: int = 2,
        ocr_interval_frames: int = 3,
        ocr_min_confidence: float = 0.70,
        plate_min_quality: float = 0.35,
    ) -> None:
        if inference_stride < 1:
            raise ValueError("inference_stride must be positive")
        if ocr_interval_frames < 1:
            raise ValueError("ocr_interval_frames must be positive")

        self._camera_id = camera_id
        self._source = source
        self._vehicle_detector = vehicle_detector
        self._tracker = tracker
        self._plate_detector = plate_detector
        self._ocr = ocr
        self._image_processor = image_processor
        self._plate_processor = plate_processor
        self._evidence_store = evidence_store
        self._inference_stride = inference_stride
        self._ocr_interval_frames = ocr_interval_frames
        self._ocr_min_confidence = ocr_min_confidence
        self._plate_min_quality = plate_min_quality

        self._stop_event = threading.Event()
        self._thread: threading.Thread | None = None
        self._lock = threading.Lock()
        self._snapshot = VisionWorkerSnapshot(
            running=False,
            ready=False,
            frames_seen=0,
            inference_frames=0,
            tracked_vehicles=0,
            ocr_attempts=0,
            confirmed_plates=0,
            publish_failures=0,
            last_error=None,
        )
        self._last_ocr_sequence: dict[int, int] = {}

    def start(self) -> None:
        if self._thread is not None and self._thread.is_alive():
            return
        self._stop_event.clear()
        self._thread = threading.Thread(target=self._run, daemon=True, name="platewatch-vision")
        self._thread.start()

    def stop(self) -> None:
        self._stop_event.set()
        self._source.close()
        if self._thread is not None and self._thread.is_alive():
            self._thread.join(timeout=3.0)

    def snapshot(self) -> VisionWorkerSnapshot:
        with self._lock:
            return self._snapshot

    def _run(self) -> None:
        self._set(running=True, ready=True, last_error=None)

        try:
            while not self._stop_event.is_set():
                frame = self._source.read()
                if frame is None:
                    continue

                self._increment(frames_seen=1)
                if (frame.sequence - 1) % self._inference_stride != 0:
                    continue

                detections = self._vehicle_detector.detect(frame)
                tracked = self._tracker.update(detections)
                self._increment(
                    inference_frames=1,
                    tracked_vehicles=len(tracked),
                )

                for vehicle in tracked:
                    last_sequence = self._last_ocr_sequence.get(vehicle.track_id, -10**9)
                    if frame.sequence - last_sequence < self._ocr_interval_frames:
                        continue
                    self._last_ocr_sequence[vehicle.track_id] = frame.sequence

                    vehicle_crop = self._image_processor.crop(frame.image, vehicle.box)
                    if vehicle_crop is None:
                        continue

                    plates = self._plate_detector.detect(vehicle_crop)
                    if not plates:
                        continue

                    plate = max(plates, key=lambda item: item.confidence)
                    plate_crop = self._image_processor.crop(vehicle_crop, plate.box)
                    if plate_crop is None:
                        continue

                    quality = self._image_processor.quality(plate_crop)
                    if quality < self._plate_min_quality:
                        continue

                    self._increment(ocr_attempts=1)
                    ocr_result = self._ocr.recognize(plate_crop)
                    if ocr_result is None or ocr_result.confidence < self._ocr_min_confidence:
                        continue

                    snapshot_url = ""
                    plate_crop_url = ""
                    if self._evidence_store is not None:
                        try:
                            refs = self._evidence_store.save(
                                camera_id=self._camera_id,
                                track_id=vehicle.track_id,
                                snapshot=vehicle_crop,
                                plate_crop=plate_crop,
                            )
                            snapshot_url = refs.snapshot_url
                            plate_crop_url = refs.plate_crop_url
                        except (OSError, RuntimeError) as exc:
                            logger.warning(
                                "evidence_save_failed",
                                extra={
                                    "camera_id": self._camera_id,
                                    "track_id": vehicle.track_id,
                                    "error": str(exc),
                                },
                            )
                            self._set(last_error=f"evidence: {exc}")

                    try:
                        decision = self._plate_processor.add_candidate(
                            PlateCandidate(
                                track_id=vehicle.track_id,
                                camera_id=self._camera_id,
                                plate_text=ocr_result.raw_text,
                                ocr_confidence=ocr_result.confidence,
                                detection_confidence=plate.confidence,
                                image_quality=quality,
                                snapshot_url=snapshot_url,
                                plate_crop_url=plate_crop_url,
                            )
                        )
                    except EventDeliveryError as exc:
                        logger.warning(
                            "detection_delivery_failed",
                            extra={
                                "camera_id": self._camera_id,
                                "track_id": vehicle.track_id,
                                "error": str(exc),
                            },
                        )
                        self._increment(publish_failures=1)
                        self._set(last_error=f"delivery: {exc}")
                        continue

                    if decision is not None:
                        self._increment(confirmed_plates=1)
                        self._set(last_error=None)
        except Exception as exc:
            logger.exception(
                "vision_worker_failed",
                extra={"camera_id": self._camera_id},
            )
            self._set(ready=False, last_error=str(exc))
        finally:
            self._source.close()
            self._set(running=False)

    def _increment(self, **values: int) -> None:
        with self._lock:
            current = self._snapshot
            payload = {
                "running": current.running,
                "ready": current.ready,
                "frames_seen": current.frames_seen,
                "inference_frames": current.inference_frames,
                "tracked_vehicles": current.tracked_vehicles,
                "ocr_attempts": current.ocr_attempts,
                "confirmed_plates": current.confirmed_plates,
                "publish_failures": current.publish_failures,
                "last_error": current.last_error,
            }
            for key, value in values.items():
                payload[key] += value
            self._snapshot = VisionWorkerSnapshot(**payload)

    def _set(self, **values: object) -> None:
        with self._lock:
            current = self._snapshot
            payload = {
                "running": current.running,
                "ready": current.ready,
                "frames_seen": current.frames_seen,
                "inference_frames": current.inference_frames,
                "tracked_vehicles": current.tracked_vehicles,
                "ocr_attempts": current.ocr_attempts,
                "confirmed_plates": current.confirmed_plates,
                "publish_failures": current.publish_failures,
                "last_error": current.last_error,
            }
            payload.update(values)
            self._snapshot = VisionWorkerSnapshot(**payload)
