import pytest

from platewatch_vision.benchmark.metrics import (
    RecognitionSample,
    evaluate_recognition,
)


def test_metrics_use_canonical_plate_matching() -> None:
    metrics = evaluate_recognition(
        [
            RecognitionSample("ABC-1234", "ABC 1234", 10.0),
            RecognitionSample("ав-123", "АВ123", 20.0),
            RecognitionSample("XYZ9", None, 30.0),
        ]
    )

    assert metrics.samples == 3
    assert metrics.recognized == 2
    assert metrics.exact_matches == 2
    assert metrics.recognition_rate == pytest.approx(0.6667)
    assert metrics.exact_match_rate == pytest.approx(0.6667)
    assert metrics.average_latency_ms == 20.0
    assert metrics.p95_latency_ms == 20.0


def test_metrics_reject_negative_latency() -> None:
    with pytest.raises(ValueError, match="latency_ms"):
        evaluate_recognition(
            [RecognitionSample("ABC123", "ABC123", -1.0)]
        )


def test_metrics_handle_empty_dataset() -> None:
    metrics = evaluate_recognition([])

    assert metrics.samples == 0
    assert metrics.exact_match_rate == 0.0
