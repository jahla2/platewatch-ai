package domain

import "time"

type DetectionEvent struct {
	ID             string    `json:"id"`
	IdempotencyKey string    `json:"-"`
	CameraID       string    `json:"camera_id"`
	TrackID        int64     `json:"track_id"`
	PlateText      string    `json:"plate_text"`
	PlateKey       string    `json:"plate_key"`
	Confidence     float64   `json:"confidence"`
	Flagged        bool      `json:"flagged"`
	SnapshotURL    string    `json:"snapshot_url"`
	PlateCropURL   string    `json:"plate_crop_url"`
	DetectedAt     time.Time `json:"detected_at"`
}

type DetectionCursor struct {
	DetectedAt time.Time
	ID         string
}
