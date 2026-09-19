from platewatch_vision.infrastructure.paddle_ocr import PaddleOCRRecognizer


class FakeEngine:
    def ocr(self, _image: object, cls: bool = False) -> list[list[object]]:
        assert cls is False
        return [
            [
                [[0, 0], [1, 0], [1, 1], [0, 1]],
                ("abc-1234", 0.91),
            ]
        ]


def test_ocr_normalizes_plate_text() -> None:
    result = PaddleOCRRecognizer(engine=FakeEngine()).recognize(object())

    assert result is not None
    assert result.text == "ABC1234"
    assert result.confidence == 0.91
