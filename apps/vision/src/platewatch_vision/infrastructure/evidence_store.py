from __future__ import annotations

import os
import re
from pathlib import Path
from typing import Any

from platewatch_vision.domain.models import EvidenceRefs

_SAFE_COMPONENT = re.compile(r"[^A-Za-z0-9_.-]+")


class OpenCVEvidenceStore:
    def __init__(
        self,
        root_dir: str,
        url_prefix: str = "/evidence",
        cv2_module: Any | None = None,
    ) -> None:
        self._root = Path(root_dir)
        self._url_prefix = "/" + url_prefix.strip("/")
        self._cv2 = cv2_module or self._load_cv2()

    def save(
        self,
        camera_id: str,
        track_id: int,
        snapshot: Any,
        plate_crop: Any,
    ) -> EvidenceRefs:
        safe_camera = _SAFE_COMPONENT.sub("_", camera_id.strip()) or "camera"
        directory = self._root / safe_camera / str(track_id)
        directory.mkdir(parents=True, exist_ok=True)

        snapshot_path = directory / "vehicle.jpg"
        plate_path = directory / "plate.jpg"

        self._write_atomic(snapshot_path, snapshot)
        self._write_atomic(plate_path, plate_crop)

        base_url = f"{self._url_prefix}/{safe_camera}/{track_id}"
        return EvidenceRefs(
            snapshot_url=f"{base_url}/vehicle.jpg",
            plate_crop_url=f"{base_url}/plate.jpg",
        )

    def _write_atomic(self, path: Path, image: Any) -> None:
        temporary = path.with_suffix(path.suffix + ".tmp.jpg")
        if not self._cv2.imwrite(str(temporary), image):
            raise RuntimeError(f"failed to write evidence image: {path}")
        os.replace(temporary, path)

    @staticmethod
    def _load_cv2() -> Any:
        try:
            import cv2
        except ImportError as exc:
            raise RuntimeError("OpenCV runtime is required for evidence storage") from exc
        return cv2
