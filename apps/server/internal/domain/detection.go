package domain

import "time"

type DetectionEvent struct {
	ID           string    `json:"id"`
	CameraID     string    `json:"camera_id"`
	TrackID      int64     `json:"track_id"`
	Plate        string    `json:"plate"`
	Confidence   float64   `json:"confidence"`
	Flagged      bool      `json:"flagged"`
	SnapshotURL  string    `json:"snapshot_url"`
	PlateCropURL string    `json:"plate_crop_url"`
	DetectedAt   time.Time `json:"detected_at"`
}
