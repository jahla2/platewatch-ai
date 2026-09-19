from __future__ import annotations

from dataclasses import dataclass
from statistics import fmean

from platewatch_vision.domain.plate_text import canonicalize_plate_text


@dataclass(frozen=True, slots=True)
class RecognitionSample:
    ground_truth: str
    predicted_text: str | None
    latency_ms: float


@dataclass(frozen=True, slots=True)
class RecognitionMetrics:
    samples: int
    recognized: int
    exact_matches: int
    recognition_rate: float
    exact_match_rate: float
    average_latency_ms: float
    p95_latency_ms: float


def evaluate_recognition(samples: list[RecognitionSample]) -> RecognitionMetrics:
    if not samples:
        return RecognitionMetrics(
            samples=0,
            recognized=0,
            exact_matches=0,
            recognition_rate=0.0,
            exact_match_rate=0.0,
            average_latency_ms=0.0,
            p95_latency_ms=0.0,
        )

    recognized = 0
    exact_matches = 0
    latencies: list[float] = []

    for sample in samples:
        if sample.latency_ms < 0:
            raise ValueError("latency_ms cannot be negative")
        latencies.append(sample.latency_ms)

        prediction = sample.predicted_text or ""
        if prediction.strip():
            recognized += 1

        expected_key = canonicalize_plate_text(sample.ground_truth)
        predicted_key = canonicalize_plate_text(prediction)
        if expected_key and predicted_key == expected_key:
            exact_matches += 1

    ordered_latencies = sorted(latencies)
    p95_index = max(0, int((len(ordered_latencies) - 1) * 0.95))

    return RecognitionMetrics(
        samples=len(samples),
        recognized=recognized,
        exact_matches=exact_matches,
        recognition_rate=round(recognized / len(samples), 4),
        exact_match_rate=round(exact_matches / len(samples), 4),
        average_latency_ms=round(fmean(latencies), 2),
        p95_latency_ms=round(ordered_latencies[p95_index], 2),
    )
