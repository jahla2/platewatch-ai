import urllib.error

import pytest

from platewatch_vision.domain.errors import EventDeliveryError
from platewatch_vision.domain.models import PlateDecision
from platewatch_vision.infrastructure.http_publisher import HttpDetectionEventPublisher


class Response:
    def __init__(self, status: int) -> None:
        self.status = status

    def __enter__(self) -> "Response":
        return self

    def __exit__(self, *_args: object) -> None:
        return None


def decision() -> PlateDecision:
    return PlateDecision(
        track_id=7,
        camera_id="CAM-01",
        plate_text="ABC-1234",
        plate_key="ABC1234",
        confidence=0.92,
        observations=3,
    )


def test_publisher_retries_network_failure_with_stable_idempotency_key() -> None:
    requests = []
    sleeps: list[float] = []
    attempts = 0

    def opener(request: object, timeout: float) -> Response:
        nonlocal attempts
        attempts += 1
        requests.append((request, timeout))
        if attempts == 1:
            raise urllib.error.URLError("temporary failure")
        return Response(201)

    publisher = HttpDetectionEventPublisher(
        server_url="http://server:8080",
        internal_token="secret",
        timeout_seconds=1.5,
        max_attempts=3,
        backoff_seconds=0.1,
        opener=opener,
        sleeper=sleeps.append,
        instance_id="worker-1",
    )

    publisher.publish(decision())

    assert attempts == 2
    assert sleeps == [0.1]
    first_key = requests[0][0].get_header("Idempotency-key")
    second_key = requests[1][0].get_header("Idempotency-key")
    assert first_key
    assert first_key == second_key
    assert requests[0][1] == 1.5


def test_publisher_does_not_retry_non_retryable_http_error() -> None:
    attempts = 0

    def opener(request: object, timeout: float) -> Response:
        del request, timeout
        nonlocal attempts
        attempts += 1
        raise urllib.error.HTTPError(
            url="http://server/internal/v1/detections",
            code=400,
            msg="bad request",
            hdrs=None,
            fp=None,
        )

    publisher = HttpDetectionEventPublisher(
        server_url="http://server:8080",
        max_attempts=3,
        opener=opener,
        sleeper=lambda _seconds: None,
        instance_id="worker-1",
    )

    with pytest.raises(EventDeliveryError, match="status 400"):
        publisher.publish(decision())

    assert attempts == 1


def test_publisher_fails_after_retry_budget_exhausted() -> None:
    attempts = 0

    def opener(request: object, timeout: float) -> Response:
        del request, timeout
        nonlocal attempts
        attempts += 1
        raise urllib.error.URLError("offline")

    publisher = HttpDetectionEventPublisher(
        server_url="http://server:8080",
        max_attempts=2,
        backoff_seconds=0,
        opener=opener,
        sleeper=lambda _seconds: None,
    )

    with pytest.raises(EventDeliveryError, match="2 attempts"):
        publisher.publish(decision())

    assert attempts == 2
