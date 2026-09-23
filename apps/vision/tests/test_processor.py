from platewatch_vision.application.consensus import PlateConsensus
from platewatch_vision.application.processor import TrackPlateProcessor
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


def test_processor_keeps_track_retryable_after_delivery_failure() -> None:
    from platewatch_vision.domain.errors import EventDeliveryError

    class FlakyPublisher:
        def __init__(self) -> None:
            self.attempts = 0

        def publish(self, _decision: PlateDecision) -> None:
            self.attempts += 1
            if self.attempts == 1:
                raise EventDeliveryError("temporary")

    publisher = FlakyPublisher()
    processor = TrackPlateProcessor(
        PlateConsensus(min_observations=1, min_confidence=0.70),
        publisher,
        max_candidates_per_track=2,
    )
    sample = PlateCandidate(
        track_id=2,
        camera_id="CAM-01",
        plate_text="XYZ987",
        ocr_confidence=0.95,
        detection_confidence=0.95,
        image_quality=0.95,
    )

    import pytest

    with pytest.raises(EventDeliveryError):
        processor.add_candidate(sample)

    decision = processor.add_candidate(sample)
    assert decision is not None
    assert publisher.attempts == 2
    assert processor.add_candidate(sample) is None
