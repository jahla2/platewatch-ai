from __future__ import annotations

import hashlib
import json
import logging
import time
import urllib.error
import urllib.request
import uuid
from collections.abc import Callable
from typing import Any

from platewatch_vision.domain.errors import EventDeliveryError
from platewatch_vision.domain.models import PlateDecision

logger = logging.getLogger(__name__)


class HttpDetectionEventPublisher:
    def __init__(
        self,
        server_url: str,
        internal_token: str = "",
        timeout_seconds: float = 2.0,
        max_attempts: int = 3,
        backoff_seconds: float = 0.25,
        opener: Callable[..., Any] = urllib.request.urlopen,
        sleeper: Callable[[float], None] = time.sleep,
        instance_id: str | None = None,
    ) -> None:
        if timeout_seconds <= 0:
            raise ValueError("timeout_seconds must be positive")
        if max_attempts < 1:
            raise ValueError("max_attempts must be positive")
        if backoff_seconds < 0:
            raise ValueError("backoff_seconds cannot be negative")

        self._endpoint = f"{server_url.rstrip('/')}/internal/v1/detections"
        self._internal_token = internal_token.strip()
        self._timeout_seconds = timeout_seconds
        self._max_attempts = max_attempts
        self._backoff_seconds = backoff_seconds
        self._opener = opener
        self._sleeper = sleeper
        self._instance_id = instance_id or uuid.uuid4().hex

    def publish(self, decision: PlateDecision) -> None:
        payload = json.dumps(
            {
                "camera_id": decision.camera_id,
                "track_id": decision.track_id,
                "plate_text": decision.plate_text,
                "plate_key": decision.plate_key,
                "confidence": decision.confidence,
                "snapshot_url": decision.snapshot_url,
                "plate_crop_url": decision.plate_crop_url,
            }
        ).encode("utf-8")

        idempotency_key = self._idempotency_key(decision)
        headers = {
            "Content-Type": "application/json",
            "Idempotency-Key": idempotency_key,
        }
        if self._internal_token:
            headers["Authorization"] = f"Bearer {self._internal_token}"

        last_error: BaseException | None = None
        for attempt in range(1, self._max_attempts + 1):
            request = urllib.request.Request(
                self._endpoint,
                data=payload,
                headers=headers,
                method="POST",
            )

            try:
                with self._opener(request, timeout=self._timeout_seconds) as response:
                    if 200 <= response.status < 300:
                        return
                    if not self._should_retry_status(response.status):
                        raise EventDeliveryError(
                            f"server rejected detection event with status {response.status}"
                        )
                    last_error = RuntimeError(
                        f"retryable server status {response.status}"
                    )
            except urllib.error.HTTPError as exc:
                if not self._should_retry_status(exc.code):
                    raise EventDeliveryError(
                        f"server rejected detection event with status {exc.code}"
                    ) from exc
                last_error = exc
            except (urllib.error.URLError, TimeoutError, OSError) as exc:
                last_error = exc

            if attempt < self._max_attempts:
                delay = min(
                    self._backoff_seconds * (2 ** (attempt - 1)),
                    5.0,
                )
                logger.warning(
                    "detection_publish_retry",
                    extra={
                        "attempt": attempt,
                        "max_attempts": self._max_attempts,
                        "delay_seconds": delay,
                    },
                )
                self._sleeper(delay)

        raise EventDeliveryError(
            f"failed to deliver detection after {self._max_attempts} attempts"
        ) from last_error

    def _idempotency_key(self, decision: PlateDecision) -> str:
        material = "|".join(
            (
                self._instance_id,
                decision.camera_id,
                str(decision.track_id),
                decision.plate_key,
            )
        )
        digest = hashlib.sha256(material.encode("utf-8")).hexdigest()
        return f"vision_{digest}"

    @staticmethod
    def _should_retry_status(status: int) -> bool:
        return status in {408, 425, 429} or status >= 500
