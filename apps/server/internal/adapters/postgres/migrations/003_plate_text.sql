ALTER TABLE detection_events
    ADD COLUMN IF NOT EXISTS plate_text TEXT NOT NULL DEFAULT '';

UPDATE detection_events
SET plate_text = plate_number
WHERE plate_text = '';

ALTER TABLE watchlist_entries
    ADD COLUMN IF NOT EXISTS plate_text TEXT NOT NULL DEFAULT '';

UPDATE watchlist_entries
SET plate_text = plate_number
WHERE plate_text = '';
