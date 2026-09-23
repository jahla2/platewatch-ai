ALTER TABLE detection_events
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

UPDATE detection_events
SET idempotency_key = id
WHERE idempotency_key IS NULL OR idempotency_key = '';

ALTER TABLE detection_events
    ALTER COLUMN idempotency_key SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_detection_events_idempotency_key
    ON detection_events (idempotency_key);
