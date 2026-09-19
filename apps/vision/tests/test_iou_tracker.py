from platewatch_vision.domain.models import BoundingBox, VehicleDetection
from platewatch_vision.infrastructure.iou_tracker import IoUTracker


def detection(x1: float, y1: float, x2: float, y2: float) -> VehicleDetection:
    return VehicleDetection(
        label="motorcycle",
        confidence=0.9,
        box=BoundingBox(x1, y1, x2, y2),
    )


def test_tracker_keeps_id_for_overlapping_detection() -> None:
    tracker = IoUTracker(iou_threshold=0.3, max_misses=2)

    first = tracker.update([detection(10, 10, 110, 110)])
    second = tracker.update([detection(15, 12, 115, 112)])

    assert len(first) == 1
    assert len(second) == 1
    assert second[0].track_id == first[0].track_id


def test_tracker_assigns_new_id_for_new_vehicle() -> None:
    tracker = IoUTracker(iou_threshold=0.3)

    first = tracker.update([detection(0, 0, 100, 100)])
    second = tracker.update([detection(300, 300, 400, 400)])

    assert first[0].track_id != second[0].track_id
