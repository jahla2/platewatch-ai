from platewatch_vision.application.consensus import PlateConsensus
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


def test_consensus_votes_by_canonical_key_and_preserves_best_raw_text() -> None:
    decision = PlateConsensus(min_observations=3, min_confidence=0.80).decide(
        [
            candidate("ABC 1234", confidence=0.90),
            candidate("ABC-1234", confidence=0.99),
            candidate("ABC1234", confidence=0.95),
            candidate("ABCI234", confidence=0.60),
        ]
    )

    assert decision is not None
    assert decision.plate_key == "ABC1234"
    assert decision.plate_text == "ABC-1234"
    assert decision.observations == 3
    assert decision.confidence >= 0.80


def test_consensus_handles_non_ascii_plate_text_without_country_rules() -> None:
    decision = PlateConsensus(min_observations=2, min_confidence=0.70).decide(
        [
            candidate("ав-123", confidence=0.95),
            candidate("АВ 123", confidence=0.94),
        ]
    )

    assert decision is not None
    assert decision.plate_key == "АВ123"


def test_consensus_waits_for_minimum_observations() -> None:
    consensus = PlateConsensus(min_observations=3)
    assert consensus.decide([candidate("ABC1234"), candidate("ABC1234")]) is None
