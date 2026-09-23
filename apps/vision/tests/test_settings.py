import pytest

from platewatch_vision.config.settings import VisionSettings


def test_settings_load_runtime_configuration(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("PLATEWATCH_VISION_AUTO_START", "true")
    monkeypatch.setenv("PLATEWATCH_VISION_SOURCE", "rtsp://camera/live")
    monkeypatch.setenv("PLATEWATCH_PLATE_MODEL", "/models/plate.pt")
    monkeypatch.setenv("PLATEWATCH_INFERENCE_STRIDE", "3")
    monkeypatch.setenv("PLATEWATCH_OCR_LANG", "en")
    monkeypatch.setenv("PLATEWATCH_PUBLISH_TIMEOUT_SECONDS", "1.5")
    monkeypatch.setenv("PLATEWATCH_PUBLISH_MAX_ATTEMPTS", "4")
    monkeypatch.setenv("PLATEWATCH_PUBLISH_BACKOFF_SECONDS", "0.2")

    settings = VisionSettings.from_env()

    assert settings.auto_start is True
    assert settings.source == "rtsp://camera/live"
    assert settings.plate_model == "/models/plate.pt"
    assert settings.inference_stride == 3
    assert settings.ocr_lang == "en"
    assert settings.publisher_timeout_seconds == 1.5
    assert settings.publisher_max_attempts == 4
    assert settings.publisher_backoff_seconds == 0.2


def test_settings_reject_invalid_threshold(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("PLATEWATCH_VEHICLE_CONFIDENCE", "1.2")

    with pytest.raises(ValueError, match="VEHICLE_CONFIDENCE"):
        VisionSettings.from_env()
