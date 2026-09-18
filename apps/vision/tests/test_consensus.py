from platewatch_vision.application.consensus import PlateConsensus, normalize_plate
from platewatch_vision.domain.models import PlateCandidate


def candidate(text: str, confidence: float = 0.95) -> PlateCandidate:
    return PlateCandidate(
        track_id=10,
        camera_id="CAM-01",
        plate_text=text,
        ocr_confidence=confidence,
        detection_confidence=0.95,
        image_quality=0.95,
    )


def test_normalize_plate_removes_spaces_and_symbols() -> None:
    assert normalize_plate(" abc-1234 ") == "ABC1234"


def test_consensus_confirms_repeated_plate() -> None:
    decision = PlateConsensus(min_observations=3, min_confidence=0.80).decide(
        [
            candidate("ABC 1234"),
            candidate("ABC-1234"),
            candidate("ABC1234"),
            candidate("ABCI234", confidence=0.60),
        ]
    )
    assert decision is not None
    assert decision.plate == "ABC1234"
    assert decision.observations == 3
    assert decision.confidence >= 0.80


def test_consensus_waits_for_minimum_observations() -> None:
    consensus = PlateConsensus(min_observations=3)
    assert consensus.decide([candidate("ABC1234"), candidate("ABC1234")]) is None
