from __future__ import annotations

import threading
import time
from datetime import UTC, datetime
from typing import Any

from platewatch_vision.domain.models import FramePacket


class LatestFrameOpenCVSource:
    """Continuously captures frames and exposes only the most recent one.

    This prevents RTSP latency from growing when inference is slower than the camera FPS.
    """

    def __init__(
        self,
        source: int | str,
        reconnect_delay_seconds: float = 1.0,
        cv2_module: Any | None = None,
    ) -> None:
        self._cv2 = cv2_module or self._load_cv2()
        self._source = source
        self._reconnect_delay_seconds = reconnect_delay_seconds
        self._condition = threading.Condition()
        self._capture: Any | None = None
        self._latest: FramePacket | None = None
        self._last_delivered_sequence = 0
        self._sequence = 0
        self._closed = False
        self._thread = threading.Thread(target=self._capture_loop, daemon=True)
        self._thread.start()

    def read(self) -> FramePacket | None:
        with self._condition:
            self._condition.wait_for(
                lambda: self._closed
                or (
                    self._latest is not None
                    and self._latest.sequence > self._last_delivered_sequence
                ),
                timeout=2.0,
            )
            if self._closed:
                return None
            if self._latest is None or self._latest.sequence <= self._last_delivered_sequence:
                return None

            self._last_delivered_sequence = self._latest.sequence
            return self._latest

    def close(self) -> None:
        with self._condition:
            self._closed = True
            self._condition.notify_all()

        if self._capture is not None:
            self._capture.release()
        if self._thread.is_alive():
            self._thread.join(timeout=2.0)

    def _capture_loop(self) -> None:
        while not self._closed:
            capture = self._open_capture()
            if capture is None:
                time.sleep(self._reconnect_delay_seconds)
                continue

            self._capture = capture
            while not self._closed:
                ok, image = capture.read()
                if not ok:
                    break

                self._sequence += 1
                packet = FramePacket(
                    sequence=self._sequence,
                    captured_at=datetime.now(UTC),
                    image=image,
                )
                with self._condition:
                    self._latest = packet
                    self._condition.notify_all()

            capture.release()
            self._capture = None
            if not self._closed:
                time.sleep(self._reconnect_delay_seconds)

    def _open_capture(self) -> Any | None:
        capture = self._cv2.VideoCapture(self._source)
        if hasattr(self._cv2, "CAP_PROP_BUFFERSIZE"):
            capture.set(self._cv2.CAP_PROP_BUFFERSIZE, 1)
        if not capture.isOpened():
            capture.release()
            return None
        return capture

    @staticmethod
    def _load_cv2() -> Any:
        try:
            import cv2
        except ImportError as exc:
            raise RuntimeError("OpenCV runtime is required for live capture") from exc
        return cv2
