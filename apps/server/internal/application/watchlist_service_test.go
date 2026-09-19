package application

import (
	"context"
	"testing"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

type watchlistStoreFake struct {
	entries map[string]domain.WatchlistEntry
}

func newWatchlistStoreFake() *watchlistStoreFake {
	return &watchlistStoreFake{entries: map[string]domain.WatchlistEntry{}}
}

func (f *watchlistStoreFake) IsFlagged(_ context.Context, plateKey string) (bool, error) {
	entry, ok := f.entries[plateKey]
	return ok && entry.Active, nil
}

func (f *watchlistStoreFake) ListWatchlist(
	_ context.Context,
	_ int,
) ([]domain.WatchlistEntry, error) {
	result := make([]domain.WatchlistEntry, 0, len(f.entries))
	for _, entry := range f.entries {
		result = append(result, entry)
	}
	return result, nil
}

func (f *watchlistStoreFake) UpsertWatchlist(
	_ context.Context,
	entry domain.WatchlistEntry,
) (domain.WatchlistEntry, error) {
	f.entries[entry.PlateKey] = entry
	return entry, nil
}

func (f *watchlistStoreFake) DeleteWatchlist(_ context.Context, plateKey string) error {
	delete(f.entries, plateKey)
	return nil
}

func TestWatchlistServicePreservesTextAndBuildsCanonicalKey(t *testing.T) {
	store := newWatchlistStoreFake()
	service := NewWatchlistService(store)

	entry, err := service.Upsert(context.Background(), UpsertWatchlistInput{
		PlateText: "abc-1234",
		Reason:    "test",
		Active:    true,
	})
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if entry.PlateText != "abc-1234" {
		t.Fatalf("PlateText = %q, want abc-1234", entry.PlateText)
	}
	if entry.PlateKey != "ABC1234" {
		t.Fatalf("PlateKey = %q, want ABC1234", entry.PlateKey)
	}
}
