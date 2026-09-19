from pathlib import Path

from platewatch_vision.infrastructure.evidence_store import OpenCVEvidenceStore


class FakeCV2:
    def imwrite(self, path: str, _image: object) -> bool:
        Path(path).write_bytes(b"image")
        return True


def test_evidence_store_writes_immutable_browser_paths(tmp_path: Path) -> None:
    store = OpenCVEvidenceStore(
        root_dir=str(tmp_path),
        url_prefix="/evidence",
        cv2_module=FakeCV2(),
    )

    refs = store.save(
        camera_id="Gate / 01",
        track_id=7,
        snapshot=object(),
        plate_crop=object(),
    )

    assert refs.snapshot_url.startswith("/evidence/Gate_01/7/")
    assert refs.snapshot_url.endswith("-vehicle.jpg")
    assert refs.plate_crop_url.startswith("/evidence/Gate_01/7/")
    assert refs.plate_crop_url.endswith("-plate.jpg")
    assert len(list((tmp_path / "Gate_01" / "7").glob("*.jpg"))) == 2
