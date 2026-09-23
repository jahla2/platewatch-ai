ALTER TABLE detection_events
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS uq_detection_events_idempotency_key
    ON detection_events (idempotency_key)
    WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';

CREATE INDEX IF NOT EXISTS idx_detection_events_cursor
    ON detection_events (detected_at DESC, id DESC);
