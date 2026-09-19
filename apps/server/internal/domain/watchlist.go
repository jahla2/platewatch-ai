package domain

import "time"

type WatchlistEntry struct {
	Plate     string    `json:"plate"`
	Reason    string    `json:"reason"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
