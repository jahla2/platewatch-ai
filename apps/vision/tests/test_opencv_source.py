from platewatch_vision.infrastructure.opencv_source import OpenCVFrameSource, resolve_video_source


class FakeCapture:
    def __init__(self, opened: bool = True) -> None:
        self.opened = opened
        self.released = False
        self.set_calls: list[tuple[int, int]] = []
        self.frames = ["frame-1"]

    def isOpened(self) -> bool:
        return self.opened

    def set(self, prop: int, value: int) -> None:
        self.set_calls.append((prop, value))

    def read(self) -> tuple[bool, object | None]:
        if not self.frames:
            return False, None
        return True, self.frames.pop(0)

    def release(self) -> None:
        self.released = True


class FakeCV2:
    CAP_PROP_BUFFERSIZE = 38

    def __init__(self, capture: FakeCapture) -> None:
        self.capture = capture
        self.sources: list[int | str] = []

    def VideoCapture(self, source: int | str) -> FakeCapture:
        self.sources.append(source)
        return self.capture


def test_resolve_video_source_supports_camera_indexes_and_urls() -> None:
    assert resolve_video_source("0") == 0
    assert resolve_video_source(" 2 ") == 2
    assert resolve_video_source("rtsp://camera/live") == "rtsp://camera/live"


def test_opencv_source_reads_and_releases_capture() -> None:
    capture = FakeCapture()
    cv2 = FakeCV2(capture)
    source = OpenCVFrameSource(0, cv2_module=cv2)

    frame = source.read()
    source.close()

    assert frame is not None
    assert frame.sequence == 1
    assert frame.image == "frame-1"
    assert cv2.sources == [0]
    assert capture.set_calls == [(FakeCV2.CAP_PROP_BUFFERSIZE, 1)]
    assert capture.released is True
