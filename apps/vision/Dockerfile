FROM python:3.12-slim

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends libgomp1 \
    && rm -rf /var/lib/apt/lists/*

COPY pyproject.toml ./
COPY src ./src

RUN pip install --no-cache-dir ".[vision,ocr]"

EXPOSE 8000

CMD ["uvicorn", "platewatch_vision.api.main:app", "--host", "0.0.0.0", "--port", "8000"]
