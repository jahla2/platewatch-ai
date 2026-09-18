from __future__ import annotations

from typing import Any

from platewatch_vision.domain.models import BoundingBox, FramePacket, VehicleDetection


class UltralyticsMotorcycleDetector:
    TARGET_LABELS = {"motorcycle", "motorbike"}

    def __init__(
        self,
        model_path: str = "yolo11n.pt",
        confidence: float = 0.35,
        image_size: int = 640,
        device: str | None = None,
        model: Any | None = None,
    ) -> None:
        if not 0 <= confidence <= 1:
            raise ValueError("confidence must be between 0 and 1")
        if image_size < 32:
            raise ValueError("image_size must be at least 32")

        self._model = model or self._load_model(model_path)
        self._confidence = confidence
        self._image_size = image_size
        self._device = device
        self._target_class_ids = self._resolve_target_class_ids(self._model.names)

        if not self._target_class_ids:
            raise RuntimeError("model does not expose a motorcycle/motorbike class")

    def detect(self, frame: FramePacket) -> list[VehicleDetection]:
        prediction = self._model.predict(
            source=frame.image,
            conf=self._confidence,
            imgsz=self._image_size,
            classes=self._target_class_ids,
            device=self._device,
            verbose=False,
        )

        if not prediction:
            return []

        boxes = getattr(prediction[0], "boxes", None)
        if boxes is None:
            return []

        detections: list[VehicleDetection] = []
        names = self._model.names

        for box in boxes:
            class_id = int(box.cls[0].item())
            label = str(names[class_id])
            if label.lower() not in self.TARGET_LABELS:
                continue

            x1, y1, x2, y2 = [float(value) for value in box.xyxy[0].tolist()]
            detections.append(
                VehicleDetection(
                    label=label.lower(),
                    confidence=float(box.conf[0].item()),
                    box=BoundingBox(x1=x1, y1=y1, x2=x2, y2=y2),
                )
            )

        return detections

    @classmethod
    def _resolve_target_class_ids(cls, names: dict[int, str] | list[str]) -> list[int]:
        items = names.items() if isinstance(names, dict) else enumerate(names)
        return [
            int(class_id)
            for class_id, label in items
            if str(label).lower() in cls.TARGET_LABELS
        ]

    @staticmethod
    def _load_model(model_path: str) -> Any:
        try:
            from ultralytics import YOLO
        except ImportError as exc:
            raise RuntimeError(
                "Ultralytics is not installed. Install the vision dependencies with "
                "pip install -e .[vision]"
            ) from exc
        return YOLO(model_path)
