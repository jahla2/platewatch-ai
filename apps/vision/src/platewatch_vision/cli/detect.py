from __future__ import annotations

import argparse
import json
from dataclasses import asdict

from platewatch_vision.application.detection_pipeline import CameraDetectionPipeline
from platewatch_vision.infrastructure.opencv_sink import OpenCVAnnotatedVideoSink
from platewatch_vision.infrastructure.opencv_source import (
    OpenCVFrameSource,
    resolve_video_source,
)
from platewatch_vision.infrastructure.ultralytics_detector import (
    UltralyticsMotorcycleDetector,
)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Run PlateWatch motorcycle detection on a camera or video source."
    )
    parser.add_argument("--source", default="0", help="Camera index, video path, or RTSP URL.")
    parser.add_argument("--model", default="yolo11n.pt", help="Ultralytics model path/name.")
    parser.add_argument("--confidence", type=float, default=0.35)
    parser.add_argument("--image-size", type=int, default=640)
    parser.add_argument("--device", default=None, help="Inference device, e.g. 0, cpu, cuda:0.")
    parser.add_argument("--stride", type=int, default=1, help="Run inference every N frames.")
    parser.add_argument("--output", default=None, help="Optional annotated MP4 output path.")
    parser.add_argument("--output-fps", type=float, default=30.0)
    parser.add_argument("--max-frames", type=int, default=None)
    return parser


def main() -> None:
    args = build_parser().parse_args()

    source = OpenCVFrameSource(resolve_video_source(args.source))
    detector = UltralyticsMotorcycleDetector(
        model_path=args.model,
        confidence=args.confidence,
        image_size=args.image_size,
        device=args.device,
    )
    sink = (
        OpenCVAnnotatedVideoSink(args.output, fps=args.output_fps)
        if args.output
        else None
    )

    metrics = CameraDetectionPipeline(
        source=source,
        detector=detector,
        sink=sink,
        inference_stride=args.stride,
    ).run(max_frames=args.max_frames)

    print(json.dumps(asdict(metrics), indent=2))


if __name__ == "__main__":
    main()
