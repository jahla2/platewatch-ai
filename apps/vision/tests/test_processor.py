import pytest

from platewatch_vision.application.consensus import PlateConsensus
from platewatch_vision.application.processor import TrackPlateProcessor
from platewatch_vision.domain.errors import EventDeliveryError
from platewatch_vision.domain.models import PlateCandidate, PlateDecision


class FakePublisher:
    def __init__(self) -> None:
        self.published: list[PlateDecision] = []

    def publish(self, decision: PlateDecision) -> None:
        self.published.append(decision)


def test_processor_publishes_track_only_once() -> None:
    publisher = FakePublisher()
    processor = TrackPlateProcessor(
        PlateConsensus(min_observations=2, min_confidence=0.70),
        publisher,
    )
    sample = PlateCandidate(
        track_id=1,
        camera_id="CAM-01",
        plate_text="ABC1234",
        ocr_confidence=0.95,
        detection_confidence=0.95,
        image_quality=0.95,
    )

    assert processor.add_candidate(sample) is None
    assert processor.add_candidate(sample) is not None
    assert processor.add_candidate(sample) is None
    assert len(publisher.published) == 1


def test_processor_retries_exact_frozen_decision_after_delivery_failure() -> None:
    class FlakyPublisher:
        def __init__(self) -> None:
            self.attempts: list[PlateDecision] = []

        def publish(self, decision: PlateDecision) -> None:
            self.attempts.append(decision)
            if len(self.attempts) == 1:
                raise EventDeliveryError("temporary")

    publisher = FlakyPublisher()
    processor = TrackPlateProcessor(
        PlateConsensus(min_observations=1, min_confidence=0.70),
        publisher,
        max_candidates_per_track=2,
    )
    first = PlateCandidate(
        track_id=2,
        camera_id="CAM-01",
        plate_text="XYZ-987",
        ocr_confidence=0.95,
        detection_confidence=0.95,
        image_quality=0.95,
        snapshot_url="/evidence/first.jpg",
    )
    later = PlateCandidate(
        track_id=2,
        camera_id="CAM-01",
        plate_text="XYZ987",
        ocr_confidence=0.99,
        detection_confidence=0.99,
        image_quality=0.99,
        snapshot_url="/evidence/later.jpg",
    )

    with pytest.raises(EventDeliveryError):
        processor.add_candidate(first)

    decision = processor.add_candidate(later)

    assert decision is not None
    assert len(publisher.attempts) == 2
    assert publisher.attempts[0] == publisher.attempts[1]
    assert decision.plate_text == "XYZ-987"
    assert decision.snapshot_url == "/evidence/first.jpg"
    assert processor.add_candidate(later) is None
