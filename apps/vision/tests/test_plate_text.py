import json
from pathlib import Path

from platewatch_vision.domain.plate_text import canonicalize_plate_text


def test_shared_country_agnostic_canonicalization_cases() -> None:
    fixture_path = Path(__file__).resolve().parents[3] / "testdata" / "plate_canonicalization.json"
    cases = json.loads(fixture_path.read_text(encoding="utf-8"))

    for case in cases:
        assert canonicalize_plate_text(case["raw"]) == case["key"]
