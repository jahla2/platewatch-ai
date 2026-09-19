from __future__ import annotations

from typing import Any

from platewatch_vision.domain.models import BoundingBox


class OpenCVImageProcessor:
    def __init__(self, cv2_module: Any | None = None) -> None:
        self._cv2 = cv2_module or self._load_cv2()

    def crop(self, image: Any, box: BoundingBox) -> Any | None:
        if image is None or not hasattr(image, "shape") or len(image.shape) < 2:
            return None

        height, width = image.shape[:2]
        x1 = max(0, min(width, int(box.x1)))
        y1 = max(0, min(height, int(box.y1)))
        x2 = max(0, min(width, int(box.x2)))
        y2 = max(0, min(height, int(box.y2)))

        if x2 <= x1 or y2 <= y1:
            return None
        return image[y1:y2, x1:x2]

    def quality(self, image: Any) -> float:
        if image is None or not hasattr(image, "shape") or len(image.shape) < 2:
            return 0.0

        height, width = image.shape[:2]
        if width < 20 or height < 8:
            return 0.0

        gray = self._cv2.cvtColor(image, self._cv2.COLOR_BGR2GRAY)
        blur_variance = float(self._cv2.Laplacian(gray, self._cv2.CV_64F).var())

        sharpness = min(1.0, blur_variance / 250.0)
        width_score = min(1.0, width / 120.0)
        height_score = min(1.0, height / 30.0)
        return round(sharpness * width_score * height_score, 4)

    @staticmethod
    def _load_cv2() -> Any:
        try:
            import cv2
        except ImportError as exc:
            raise RuntimeError("OpenCV runtime is required for image processing") from exc
        return cv2
