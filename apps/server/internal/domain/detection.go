package domain

import "time"

type DetectionEvent struct {
	ID         string    `json:"id"`
	CameraID   string    `json:"camera_id"`
	TrackID    int64     `json:"track_id"`
	Plate      string    `json:"plate"`
	Confidence float64   `json:"confidence"`
	Flagged    bool      `json:"flagged"`
	DetectedAt time.Time `json:"detected_at"`
}
