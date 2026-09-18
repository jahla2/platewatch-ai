from __future__ import annotations

import time

from platewatch_vision.domain.models import PipelineMetrics
from platewatch_vision.domain.ports import FrameAnalysisSink, FrameSource, VehicleDetector


class CameraDetectionPipeline:
    def __init__(
        self,
        source: FrameSource,
        detector: VehicleDetector,
        sink: FrameAnalysisSink | None = None,
        inference_stride: int = 1,
    ) -> None:
        if inference_stride < 1:
            raise ValueError("inference_stride must be positive")

        self._source = source
        self._detector = detector
        self._sink = sink
        self._inference_stride = inference_stride

    def run(self, max_frames: int | None = None) -> PipelineMetrics:
        if max_frames is not None and max_frames < 1:
            raise ValueError("max_frames must be positive")

        started_at = time.perf_counter()
        frames_read = 0
        frames_inferred = 0
        detections_count = 0
        inference_seconds = 0.0

        try:
            while max_frames is None or frames_read < max_frames:
                frame = self._source.read()
                if frame is None:
                    break

                frames_read += 1
                detections = []

                if (frames_read - 1) % self._inference_stride == 0:
                    inference_started_at = time.perf_counter()
                    detections = self._detector.detect(frame)
                    inference_seconds += time.perf_counter() - inference_started_at
                    frames_inferred += 1
                    detections_count += len(detections)

                if self._sink is not None:
                    self._sink.write(frame, detections)
        finally:
            self._source.close()
            if self._sink is not None:
                self._sink.close()

        elapsed_seconds = max(time.perf_counter() - started_at, 1e-9)
        source_fps = frames_read / elapsed_seconds
        inference_fps = (
            frames_inferred / inference_seconds if inference_seconds > 0 else 0.0
        )
        average_inference_ms = (
            (inference_seconds / frames_inferred) * 1000 if frames_inferred else 0.0
        )

        return PipelineMetrics(
            frames_read=frames_read,
            frames_inferred=frames_inferred,
            detections=detections_count,
            elapsed_seconds=round(elapsed_seconds, 4),
            source_fps=round(source_fps, 2),
            inference_fps=round(inference_fps, 2),
            average_inference_ms=round(average_inference_ms, 2),
        )
