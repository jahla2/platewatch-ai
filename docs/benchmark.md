# Plate Recognition Benchmark

Use benchmarks before deciding whether the generic detector or OCR model needs tuning.

## OCR crop benchmark

Create a directory containing labeled plate crops and a JSONL manifest:

```text
benchmark/
  manifest.jsonl
  images/
    001.jpg
    002.jpg
```

Example `manifest.jsonl`:

```jsonl
{"image":"images/001.jpg","plate_text":"ABC-1234"}
{"image":"images/002.jpg","plate_text":"B 1234 CD"}
```

Install the runtime:

```bash
cd apps/vision
python -m pip install -e ".[vision,ocr,dev]"
```

Run:

```bash
platewatch-benchmark-ocr --manifest ../../benchmark/manifest.jsonl --lang en
```

The report includes:

- recognition rate
- exact canonical plate-match rate
- average OCR latency
- p95 OCR latency
- unrecognized samples

Matching is country/format agnostic. For example, `ABC-1234` and `ABC 1234`
have the same canonical matching key, while the original ground-truth and OCR
text remain available for inspection.

## Recommended evaluation set

Use representative plate crops from the intended camera environment:

- near and distant plates
- front and rear views
- motorcycles and other required vehicle classes
- day and night
- motion blur
- glare and shadows
- angled plates
- different plate layouts and writing systems relevant to the deployment

Do not train or fine-tune only because a plate format is different. First measure
detector/OCR performance. Tune or replace a model only when the benchmark shows
a measurable gap.

## Full-pipeline benchmark

The OCR crop benchmark isolates recognition quality. A later field test should
also measure the complete camera pipeline:

```text
camera frame
→ vehicle detection
→ tracking
→ plate detection
→ OCR
→ consensus
→ Go API
→ PostgreSQL
→ SSE/dashboard
```

Track:

- plate detection recall
- track-level final exact-match accuracy
- false alerts
- end-to-end detection-to-dashboard latency
- GPU/CPU utilization
- dropped-frame rate

A fixed recorded video with labeled expected plate events is recommended for
repeatable regression testing.
