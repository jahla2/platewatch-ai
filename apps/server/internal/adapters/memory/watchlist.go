package memory

import (
	"context"
	"regexp"
	"strings"
)

var nonAlphaNumeric = regexp.MustCompile(`[^A-Z0-9]`)

type Watchlist struct {
	plates map[string]struct{}
}

func NewWatchlist(values []string) *Watchlist {
	plates := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := nonAlphaNumeric.ReplaceAllString(strings.ToUpper(strings.TrimSpace(value)), "")
		if normalized != "" {
			plates[normalized] = struct{}{}
		}
	}
	return &Watchlist{plates: plates}
}

func (w *Watchlist) IsFlagged(_ context.Context, plate string) (bool, error) {
	_, found := w.plates[plate]
	return found, nil
}
