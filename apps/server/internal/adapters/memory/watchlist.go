package memory

import (
	"context"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

type Watchlist struct {
	plates map[string]struct{}
}

func NewWatchlist(values []string) *Watchlist {
	plates := make(map[string]struct{}, len(values))
	for _, value := range values {
		plateKey := domain.CanonicalizePlateText(value)
		if plateKey != "" {
			plates[plateKey] = struct{}{}
		}
	}
	return &Watchlist{plates: plates}
}

func (w *Watchlist) IsFlagged(_ context.Context, plateKey string) (bool, error) {
	_, found := w.plates[plateKey]
	return found, nil
}
