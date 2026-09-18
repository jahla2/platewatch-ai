from typing import Protocol

from platewatch_vision.domain.models import PlateDecision


class DetectionEventPublisher(Protocol):
    def publish(self, decision: PlateDecision) -> None:
        """Publish a confirmed plate decision to the application backend."""
