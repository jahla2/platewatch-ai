# Model Configuration

PlateWatch keeps model weights outside Git and treats license plates as country-agnostic visual/text regions.

## Vehicle detector

The default runtime bootstraps a small Ultralytics YOLO11 nano detector and uses the motorcycle class only.

Environment:

```env
PLATEWATCH_VEHICLE_MODEL_URL=...
PLATEWATCH_VEHICLE_MODEL=/models/yolo11n.pt
PLATEWATCH_VEHICLE_CONFIDENCE=0.35
```

## License-plate detector

The default model URL is a generic public license-plate detector so a developer can exercise the full pipeline without first training a model.

Environment:

```env
PLATEWATCH_PLATE_MODEL_URL=...
PLATEWATCH_PLATE_MODEL=/models/license_plate.pt
PLATEWATCH_PLATE_CONFIDENCE=0.50
```

No country-specific plate format is enforced. Fine-tuning is optional and should only be introduced when a representative benchmark shows that the generic detector is not meeting the required plate-detection recall.

## OCR

The current OCR adapter uses PaddleOCR and combines repeated observations through temporal consensus.

Important settings:

```env
PLATEWATCH_OCR_ENGINE=paddleocr
PLATEWATCH_OCR_LANG=en
PLATEWATCH_OCR_MIN_CONFIDENCE=0.70
PLATEWATCH_OCR_CONFIRMATION_COUNT=3
PLATEWATCH_OCR_INTERVAL_FRAMES=3
PLATEWATCH_PLATE_MIN_QUALITY=0.35
```

`PLATEWATCH_OCR_LANG` controls the configured PaddleOCR recognition language. The application itself is format-agnostic, but OCR accuracy for a particular writing system depends on the OCR model/language selected.

## Plate text contract

PlateWatch stores both values:

```text
plate_text = raw/best OCR representation for display
plate_key  = Unicode-normalized alphanumeric key for matching
```

Example:

```text
plate_text: "B 1234-CD"
plate_key:  "B1234CD"
```

The system does not require a country regex such as a fixed number of letters or digits.

## Licensing

The repository does not redistribute model binaries. Review the license and commercial-use terms of every model/framework URL configured for your deployment.
