from __future__ import annotations

from dataclasses import dataclass

from platewatch_vision.domain.models import BoundingBox, TrackedVehicle, VehicleDetection


def _iou(left: BoundingBox, right: BoundingBox) -> float:
    x1 = max(left.x1, right.x1)
    y1 = max(left.y1, right.y1)
    x2 = min(left.x2, right.x2)
    y2 = min(left.y2, right.y2)

    intersection = max(0.0, x2 - x1) * max(0.0, y2 - y1)
    if intersection <= 0:
        return 0.0

    left_area = left.width * left.height
    right_area = right.width * right.height
    union = left_area + right_area - intersection
    return intersection / union if union > 0 else 0.0


@dataclass(slots=True)
class _Track:
    track_id: int
    detection: VehicleDetection
    misses: int = 0


class IoUTracker:
    """Lightweight tracker optimized for fixed traffic cameras and small deployments."""

    def __init__(self, iou_threshold: float = 0.30, max_misses: int = 8) -> None:
        if not 0 <= iou_threshold <= 1:
            raise ValueError("iou_threshold must be between 0 and 1")
        if max_misses < 0:
            raise ValueError("max_misses cannot be negative")

        self._iou_threshold = iou_threshold
        self._max_misses = max_misses
        self._next_id = 1
        self._tracks: dict[int, _Track] = {}

    def update(self, detections: list[VehicleDetection]) -> list[TrackedVehicle]:
        unmatched_detection_indexes = set(range(len(detections)))
        matched_track_ids: set[int] = set()

        candidates: list[tuple[float, int, int]] = []
        for track_id, track in self._tracks.items():
            for detection_index, detection in enumerate(detections):
                score = _iou(track.detection.box, detection.box)
                if score >= self._iou_threshold:
                    candidates.append((score, track_id, detection_index))

        for _, track_id, detection_index in sorted(candidates, reverse=True):
            if track_id in matched_track_ids or detection_index not in unmatched_detection_indexes:
                continue
            self._tracks[track_id].detection = detections[detection_index]
            self._tracks[track_id].misses = 0
            matched_track_ids.add(track_id)
            unmatched_detection_indexes.remove(detection_index)

        for track_id in list(self._tracks):
            if track_id not in matched_track_ids:
                self._tracks[track_id].misses += 1
                if self._tracks[track_id].misses > self._max_misses:
                    del self._tracks[track_id]

        for detection_index in unmatched_detection_indexes:
            detection = detections[detection_index]
            track_id = self._next_id
            self._next_id += 1
            self._tracks[track_id] = _Track(track_id=track_id, detection=detection)
            matched_track_ids.add(track_id)

        return [
            TrackedVehicle(
                track_id=track.track_id,
                label=track.detection.label,
                confidence=track.detection.confidence,
                box=track.detection.box,
            )
            for track in self._tracks.values()
            if track.misses == 0
        ]
