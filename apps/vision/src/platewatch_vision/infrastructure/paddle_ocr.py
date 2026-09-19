from __future__ import annotations

from typing import Any

from platewatch_vision.domain.models import OCRResult


class PaddleOCRRecognizer:
    def __init__(self, lang: str = "en", engine: Any | None = None) -> None:
        self._lang = lang.strip() or "en"
        self._engine = engine or self._load_engine(self._lang)

    def recognize(self, image: Any) -> OCRResult | None:
        if image is None:
            return None

        result = self._engine.ocr(image, cls=False)
        lines = self._flatten_lines(result)

        candidates: list[OCRResult] = []
        for line in lines:
            if not isinstance(line, (list, tuple)) or len(line) < 2:
                continue
            recognition = line[1]
            if not isinstance(recognition, (list, tuple)) or len(recognition) < 2:
                continue

            raw_text = str(recognition[0]).strip()
            if not raw_text:
                continue
            try:
                confidence = float(recognition[1])
            except (TypeError, ValueError):
                continue
            candidates.append(OCRResult(raw_text=raw_text, confidence=confidence))

        return max(candidates, key=lambda item: item.confidence, default=None)

    @staticmethod
    def _flatten_lines(result: Any) -> list[Any]:
        if not result:
            return []
        if isinstance(result, list) and len(result) == 1 and isinstance(result[0], list):
            return result[0]
        return result if isinstance(result, list) else []

    @staticmethod
    def _load_engine(lang: str) -> Any:
        try:
            from paddleocr import PaddleOCR
        except ImportError as exc:
            raise RuntimeError("PaddleOCR runtime is required for OCR") from exc

        return PaddleOCR(
            lang=lang,
            use_doc_orientation_classify=False,
            use_doc_unwarping=False,
            use_textline_orientation=False,
        )
