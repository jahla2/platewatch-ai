from __future__ import annotations

from datetime import UTC, datetime
from typing import Any

from platewatch_vision.domain.models import FramePacket


def resolve_video_source(value: str) -> int | str:
    stripped = value.strip()
    return int(stripped) if stripped.isdigit() else stripped


class OpenCVFrameSource:
    def __init__(self, source: int | str, cv2_module: Any | None = None) -> None:
        self._cv2 = cv2_module or self._load_cv2()
        self._capture = self._cv2.VideoCapture(source)
        self._sequence = 0

        if hasattr(self._cv2, "CAP_PROP_BUFFERSIZE"):
            self._capture.set(self._cv2.CAP_PROP_BUFFERSIZE, 1)

        if not self._capture.isOpened():
            self._capture.release()
            raise RuntimeError(f"unable to open video source: {source}")

    def read(self) -> FramePacket | None:
        ok, image = self._capture.read()
        if not ok:
            return None

        self._sequence += 1
        return FramePacket(
            sequence=self._sequence,
            captured_at=datetime.now(UTC),
            image=image,
        )

    def close(self) -> None:
        self._capture.release()

    @staticmethod
    def _load_cv2() -> Any:
        try:
            import cv2
        except ImportError as exc:
            raise RuntimeError(
                "OpenCV is not installed. Install the vision dependencies with "
                "pip install -e .[vision]"
            ) from exc
        return cv2
