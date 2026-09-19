from collections import defaultdict

from platewatch_vision.domain.models import PlateCandidate, PlateDecision
from platewatch_vision.domain.plate_text import canonicalize_plate_text


class PlateConsensus:
    def __init__(self, min_observations: int = 3, min_confidence: float = 0.80) -> None:
        if min_observations < 1:
            raise ValueError("min_observations must be positive")
        if not 0 <= min_confidence <= 1:
            raise ValueError("min_confidence must be between 0 and 1")
        self._min_observations = min_observations
        self._min_confidence = min_confidence

    def decide(self, candidates: list[PlateCandidate]) -> PlateDecision | None:
        if len(candidates) < self._min_observations:
            return None

        scores: dict[str, float] = defaultdict(float)
        counts: dict[str, int] = defaultdict(int)
        weighted_candidates: list[tuple[PlateCandidate, str, float]] = []

        for candidate in candidates:
            plate_key = canonicalize_plate_text(candidate.plate_text)
            if not plate_key:
                continue

            weight = (
                candidate.ocr_confidence
                * candidate.detection_confidence
                * candidate.image_quality
            )
            scores[plate_key] += weight
            counts[plate_key] += 1
            weighted_candidates.append((candidate, plate_key, weight))

        if not scores:
            return None

        best_key = max(scores, key=scores.get)
        observations = counts[best_key]
        if observations < self._min_observations:
            return None

        confidence = scores[best_key] / observations
        if confidence < self._min_confidence:
            return None

        representative, _, _ = max(
            (
                item
                for item in weighted_candidates
                if item[1] == best_key
            ),
            key=lambda item: item[2],
        )

        return PlateDecision(
            track_id=representative.track_id,
            camera_id=representative.camera_id,
            plate_text=representative.plate_text.strip(),
            plate_key=best_key,
            confidence=round(confidence, 4),
            observations=observations,
            snapshot_url=representative.snapshot_url,
            plate_crop_url=representative.plate_crop_url,
        )
