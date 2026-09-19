import json
import urllib.request

from platewatch_vision.domain.models import PlateDecision


class HttpDetectionEventPublisher:
    def __init__(
        self,
        server_url: str,
        internal_token: str = "",
        timeout_seconds: float = 2.0,
    ) -> None:
        self._endpoint = f"{server_url.rstrip('/')}/internal/v1/detections"
        self._internal_token = internal_token.strip()
        self._timeout_seconds = timeout_seconds

    def publish(self, decision: PlateDecision) -> None:
        payload = json.dumps(
            {
                "camera_id": decision.camera_id,
                "track_id": decision.track_id,
                "plate": decision.plate,
                "confidence": decision.confidence,
                "snapshot_url": decision.snapshot_url,
                "plate_crop_url": decision.plate_crop_url,
            }
        ).encode("utf-8")
        headers = {"Content-Type": "application/json"}
        if self._internal_token:
            headers["Authorization"] = f"Bearer {self._internal_token}"

        request = urllib.request.Request(
            self._endpoint,
            data=payload,
            headers=headers,
            method="POST",
        )
        with urllib.request.urlopen(request, timeout=self._timeout_seconds) as response:
            if not 200 <= response.status < 300:
                raise RuntimeError(f"server rejected detection event: {response.status}")
