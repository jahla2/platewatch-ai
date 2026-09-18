from __future__ import annotations

from pathlib import Path
from typing import Any

from platewatch_vision.domain.models import FramePacket, VehicleDetection


class OpenCVAnnotatedVideoSink:
    def __init__(
        self,
        output_path: str,
        fps: float = 30.0,
        cv2_module: Any | None = None,
    ) -> None:
        if fps <= 0:
            raise ValueError("fps must be positive")

        self._cv2 = cv2_module or self._load_cv2()
        self._output_path = Path(output_path)
        self._fps = fps
        self._writer: Any | None = None

    def write(self, frame: FramePacket, detections: list[VehicleDetection]) -> None:
        image = frame.image.copy()

        for detection in detections:
            box = detection.box
            start = (int(box.x1), int(box.y1))
            end = (int(box.x2), int(box.y2))
            label = f"{detection.label} {detection.confidence:.2f}"

            self._cv2.rectangle(image, start, end, (255, 255, 255), 2)
            self._cv2.putText(
                image,
                label,
                (start[0], max(20, start[1] - 8)),
                self._cv2.FONT_HERSHEY_SIMPLEX,
                0.6,
                (255, 255, 255),
                2,
                self._cv2.LINE_AA,
            )

        if self._writer is None:
            height, width = image.shape[:2]
            self._output_path.parent.mkdir(parents=True, exist_ok=True)
            codec = self._cv2.VideoWriter_fourcc(*"mp4v")
            self._writer = self._cv2.VideoWriter(
                str(self._output_path),
                codec,
                self._fps,
                (width, height),
            )
            if not self._writer.isOpened():
                raise RuntimeError(f"unable to open output video: {self._output_path}")

        self._writer.write(image)

    def close(self) -> None:
        if self._writer is not None:
            self._writer.release()
            self._writer = None

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
