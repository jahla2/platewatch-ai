package domain

import "time"

type WatchlistEntry struct {
	PlateText string    `json:"plate_text"`
	PlateKey  string    `json:"plate_key"`
	Reason    string    `json:"reason"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
