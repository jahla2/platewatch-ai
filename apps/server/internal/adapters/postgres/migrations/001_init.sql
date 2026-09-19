CREATE TABLE IF NOT EXISTS detection_events (
    id TEXT PRIMARY KEY,
    camera_id TEXT NOT NULL,
    track_id BIGINT NOT NULL,
    plate_number TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    flagged BOOLEAN NOT NULL DEFAULT FALSE,
    detected_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_detection_events_detected_at
    ON detection_events (detected_at DESC);

CREATE INDEX IF NOT EXISTS idx_detection_events_plate
    ON detection_events (plate_number, detected_at DESC);

CREATE INDEX IF NOT EXISTS idx_detection_events_camera_detected
    ON detection_events (camera_id, detected_at DESC);

CREATE TABLE IF NOT EXISTS watchlist_entries (
    plate_number TEXT PRIMARY KEY,
    reason TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_watchlist_active_plate
    ON watchlist_entries (plate_number)
    WHERE active = TRUE;
