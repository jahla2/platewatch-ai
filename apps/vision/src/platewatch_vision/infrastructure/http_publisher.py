import json
import urllib.request

from platewatch_vision.domain.models import PlateDecision


class HttpDetectionEventPublisher:
    def __init__(self, server_url: str, timeout_seconds: float = 2.0) -> None:
        self._endpoint = f"{server_url.rstrip('/')}/api/v1/detections"
        self._timeout_seconds = timeout_seconds

    def publish(self, decision: PlateDecision) -> None:
        payload = json.dumps(
            {
                "camera_id": decision.camera_id,
                "track_id": decision.track_id,
                "plate": decision.plate,
                "confidence": decision.confidence,
            }
        ).encode("utf-8")
        request = urllib.request.Request(
            self._endpoint,
            data=payload,
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with urllib.request.urlopen(request, timeout=self._timeout_seconds) as response:
            if not 200 <= response.status < 300:
                raise RuntimeError(f"server rejected detection event: {response.status}")
