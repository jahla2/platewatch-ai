from collections import defaultdict

from platewatch_vision.application.consensus import PlateConsensus
from platewatch_vision.domain.models import PlateCandidate, PlateDecision
from platewatch_vision.domain.ports import DetectionEventPublisher


class TrackPlateProcessor:
    def __init__(
        self,
        consensus: PlateConsensus,
        publisher: DetectionEventPublisher,
        max_candidates_per_track: int = 12,
    ) -> None:
        if max_candidates_per_track < 1:
            raise ValueError("max_candidates_per_track must be positive")

        self._consensus = consensus
        self._publisher = publisher
        self._max_candidates_per_track = max_candidates_per_track
        self._candidates: dict[int, list[PlateCandidate]] = defaultdict(list)
        self._completed_tracks: set[int] = set()

    def add_candidate(self, candidate: PlateCandidate) -> PlateDecision | None:
        if candidate.track_id in self._completed_tracks:
            return None

        self._validate(candidate)
        bucket = self._candidates[candidate.track_id]
        bucket.append(candidate)
        if len(bucket) > self._max_candidates_per_track:
            del bucket[: len(bucket) - self._max_candidates_per_track]

        decision = self._consensus.decide(bucket)
        if decision is None:
            return None

        # Mark the track completed only after the event has been delivered. A
        # transient delivery error leaves the candidates available for retry.
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
