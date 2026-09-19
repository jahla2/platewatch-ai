from __future__ import annotations

from typing import Any

from platewatch_vision.domain.models import BoundingBox, PlateDetection


class UltralyticsPlateDetector:
    TARGET_LABELS = {"license_plate", "license-plate", "plate", "number_plate"}

    def __init__(
        self,
        model_path: str,
        confidence: float = 0.50,
        image_size: int = 640,
        device: str | None = None,
        model: Any | None = None,
    ) -> None:
        if not model_path and model is None:
            raise ValueError("plate detector model path is required")
        if not 0 <= confidence <= 1:
            raise ValueError("confidence must be between 0 and 1")
        if image_size < 32:
            raise ValueError("image_size must be at least 32")

        self._model = model or self._load_model(model_path)
        self._confidence = confidence
        self._image_size = image_size
        self._device = device
        self._class_ids = self._resolve_class_ids(self._model.names)

    def detect(self, image: Any) -> list[PlateDetection]:
        prediction = self._model.predict(
            source=image,
            conf=self._confidence,
            imgsz=self._image_size,
            classes=self._class_ids or None,
            device=self._device,
            verbose=False,
        )
        if not prediction:
            return []

        boxes = getattr(prediction[0], "boxes", None)
        if boxes is None:
            return []

        detections: list[PlateDetection] = []
        for box in boxes:
            x1, y1, x2, y2 = [float(value) for value in box.xyxy[0].tolist()]
            detections.append(
                PlateDetection(
                    confidence=float(box.conf[0].item()),
                    box=BoundingBox(x1=x1, y1=y1, x2=x2, y2=y2),
                )
            )

        return detections

    @classmethod
    def _resolve_class_ids(cls, names: dict[int, str] | list[str]) -> list[int]:
        items = list(names.items()) if isinstance(names, dict) else list(enumerate(names))
        if len(items) == 1:
            return [int(items[0][0])]
        return [
            int(class_id)
            for class_id, label in items
            if str(label).lower().replace(" ", "_") in cls.TARGET_LABELS
        ]

    @staticmethod
    def _load_model(model_path: str) -> Any:
        try:
            from ultralytics import YOLO
        except ImportError as exc:
            raise RuntimeError("Ultralytics runtime is required for plate detection") from exc
        return YOLO(model_path)
