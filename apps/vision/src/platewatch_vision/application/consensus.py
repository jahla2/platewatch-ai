import re
from collections import defaultdict

from platewatch_vision.domain.models import PlateCandidate, PlateDecision

_NON_ALPHANUMERIC = re.compile(r"[^A-Z0-9]")


def normalize_plate(value: str) -> str:
    return _NON_ALPHANUMERIC.sub("", value.upper().strip())


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

        for candidate in candidates:
            plate = normalize_plate(candidate.plate_text)
            if not plate:
                continue
            weight = (
                candidate.ocr_confidence
                * candidate.detection_confidence
                * candidate.image_quality
            )
            scores[plate] += weight
            counts[plate] += 1

        if not scores:
            return None

        best_plate = max(scores, key=scores.get)
        observations = counts[best_plate]
        if observations < self._min_observations:
            return None

        confidence = scores[best_plate] / observations
        if confidence < self._min_confidence:
            return None

        representative = next(
            item for item in candidates if normalize_plate(item.plate_text) == best_plate
        )
        return PlateDecision(
            track_id=representative.track_id,
            camera_id=representative.camera_id,
            plate=best_plate,
            confidence=round(confidence, 4),
            observations=observations,
        )
