from __future__ import annotations

import argparse
import json
import time
from dataclasses import asdict
from pathlib import Path
from typing import Any

from platewatch_vision.benchmark.metrics import RecognitionSample, evaluate_recognition
from platewatch_vision.infrastructure.paddle_ocr import PaddleOCRRecognizer


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Benchmark PlateWatch OCR against labeled license-plate crops."
    )
    parser.add_argument(
        "--manifest",
        required=True,
        help="JSONL manifest with image and plate_text fields.",
    )
    parser.add_argument("--lang", default="en", help="PaddleOCR language/model selection.")
    return parser


def _load_cv2() -> Any:
    try:
        import cv2
    except ImportError as exc:
        raise RuntimeError(
            "OpenCV is required. Install the vision dependencies with "
            'pip install -e ".[vision,ocr]".'
        ) from exc
    return cv2


def _read_manifest(path: Path) -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    for line_number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), start=1):
        if not line.strip():
            continue

        try:
            row = json.loads(line)
        except json.JSONDecodeError as exc:
            raise ValueError(f"invalid JSON at manifest line {line_number}") from exc

        image = str(row.get("image", "")).strip()
        plate_text = str(row.get("plate_text", "")).strip()
        if not image or not plate_text:
            raise ValueError(
                f"manifest line {line_number} requires image and plate_text"
            )
        rows.append({"image": image, "plate_text": plate_text})

    if not rows:
        raise ValueError("manifest has no benchmark samples")
    return rows


def main() -> None:
    args = build_parser().parse_args()
    manifest_path = Path(args.manifest).resolve()
    cv2 = _load_cv2()
    recognizer = PaddleOCRRecognizer(lang=args.lang)

    samples: list[RecognitionSample] = []
    failures: list[dict[str, str]] = []

    for row in _read_manifest(manifest_path):
        image_path = (manifest_path.parent / row["image"]).resolve()
        image = cv2.imread(str(image_path))
        if image is None:
            raise RuntimeError(f"unable to read benchmark image: {image_path}")

        started_at = time.perf_counter()
        result = recognizer.recognize(image)
        latency_ms = (time.perf_counter() - started_at) * 1000

        predicted_text = result.raw_text if result is not None else None
        sample = RecognitionSample(
            ground_truth=row["plate_text"],
            predicted_text=predicted_text,
            latency_ms=latency_ms,
        )
        samples.append(sample)

        if predicted_text is None:
            failures.append(
                {
                    "image": row["image"],
                    "expected": row["plate_text"],
                    "predicted": "",
                }
            )

    metrics = evaluate_recognition(samples)
    print(
        json.dumps(
            {
                "metrics": asdict(metrics),
                "unrecognized": failures,
            },
            ensure_ascii=False,
            indent=2,
        )
    )


if __name__ == "__main__":
    main()
