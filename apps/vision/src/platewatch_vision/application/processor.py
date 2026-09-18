from collections import defaultdict

from platewatch_vision.application.consensus import PlateConsensus
from platewatch_vision.domain.models import PlateCandidate, PlateDecision
from platewatch_vision.domain.ports import DetectionEventPublisher


class TrackPlateProcessor:
    def __init__(
        self,
        consensus: PlateConsensus,
        publisher: DetectionEventPublisher,
    ) -> None:
        self._consensus = consensus
        self._publisher = publisher
        self._candidates: dict[int, list[PlateCandidate]] = defaultdict(list)
        self._completed_tracks: set[int] = set()

    def add_candidate(self, candidate: PlateCandidate) -> PlateDecision | None:
        if candidate.track_id in self._completed_tracks:
            return None

        self._validate(candidate)
        bucket = self._candidates[candidate.track_id]
        bucket.append(candidate)

        decision = self._consensus.decide(bucket)
        if decision is None:
            return None

        self._publisher.publish(decision)
        self._completed_tracks.add(candidate.track_id)
        self._candidates.pop(candidate.track_id, None)
        return decision

    @staticmethod
    def _validate(candidate: PlateCandidate) -> None:
        if candidate.track_id <= 0:
            raise ValueError("track_id must be positive")
        if not candidate.camera_id.strip():
            raise ValueError("camera_id is required")
        for name, value in (
            ("ocr_confidence", candidate.ocr_confidence),
            ("detection_confidence", candidate.detection_confidence),
            ("image_quality", candidate.image_quality),
        ):
            if not 0 <= value <= 1:
                raise ValueError(f"{name} must be between 0 and 1")
