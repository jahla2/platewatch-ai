# Model Configuration

PlateWatch keeps model weights outside Git.

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

It is **not** considered the final Philippine motorcycle plate model.

Environment:

```env
PLATEWATCH_PLATE_MODEL_URL=...
PLATEWATCH_PLATE_MODEL=/models/license_plate.pt
PLATEWATCH_PLATE_CONFIDENCE=0.50
```

For field testing, replace this weight file with a model evaluated on a held-out Philippine motorcycle plate dataset including day/night, blur, glare, angle, rain, temporary plates, and different plate generations.

## OCR

The current OCR adapter uses PaddleOCR and combines repeated observations through temporal consensus.

Important settings:

```env
PLATEWATCH_OCR_MIN_CONFIDENCE=0.70
PLATEWATCH_OCR_CONFIRMATION_COUNT=3
PLATEWATCH_OCR_INTERVAL_FRAMES=3
PLATEWATCH_PLATE_MIN_QUALITY=0.35
```

These values are starting points and should be calibrated against a fixed validation set.

## Licensing

The repository does not redistribute model binaries. Review the license and commercial-use terms of every model/framework URL configured for your deployment.
