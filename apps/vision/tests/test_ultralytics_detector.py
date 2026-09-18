from datetime import UTC, datetime

import pytest

from platewatch_vision.domain.models import FramePacket
from platewatch_vision.infrastructure.ultralytics_detector import (
    UltralyticsMotorcycleDetector,
)


class Scalar:
    def __init__(self, value: float) -> None:
        self.value = value

    def item(self) -> float:
        return self.value


class Vector:
    def __init__(self, values: list[float]) -> None:
        self.values = values

    def tolist(self) -> list[float]:
        return self.values


class FakeBox:
    def __init__(self) -> None:
        self.cls = [Scalar(3)]
        self.conf = [Scalar(0.93)]
        self.xyxy = [Vector([10.0, 20.0, 100.0, 120.0])]


class FakeResult:
    boxes = [FakeBox()]


class FakeModel:
    names = {0: "person", 2: "car", 3: "motorcycle"}

    def __init__(self) -> None:
        self.kwargs: dict[str, object] | None = None

    def predict(self, **kwargs: object) -> list[FakeResult]:
        self.kwargs = kwargs
        return [FakeResult()]


def test_detector_filters_to_motorcycle_class() -> None:
    model = FakeModel()
    detector = UltralyticsMotorcycleDetector(
        confidence=0.4,
        image_size=512,
        device="cpu",
        model=model,
    )
    frame = FramePacket(
        sequence=1,
        captured_at=datetime.now(UTC),
        image="image",
    )

    detections = detector.detect(frame)

    assert len(detections) == 1
    assert detections[0].label == "motorcycle"
    assert detections[0].confidence == pytest.approx(0.93)
    assert detections[0].box.x2 == 100.0
    assert model.kwargs is not None
    assert model.kwargs["classes"] == [3]
    assert model.kwargs["conf"] == 0.4
    assert model.kwargs["imgsz"] == 512
    assert model.kwargs["device"] == "cpu"


def test_detector_rejects_model_without_motorcycle_class() -> None:
    model = FakeModel()
    model.names = {0: "person", 2: "car"}

    with pytest.raises(RuntimeError, match="motorcycle"):
        UltralyticsMotorcycleDetector(model=model)
