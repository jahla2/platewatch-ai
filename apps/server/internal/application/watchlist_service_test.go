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

func (f *watchlistStoreFake) IsFlagged(_ context.Context, plate string) (bool, error) {
	entry, ok := f.entries[plate]
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
	f.entries[entry.Plate] = entry
	return entry, nil
}

func (f *watchlistStoreFake) DeleteWatchlist(_ context.Context, plate string) error {
	delete(f.entries, plate)
	return nil
}

func TestWatchlistServiceNormalizesPlate(t *testing.T) {
	store := newWatchlistStoreFake()
	service := NewWatchlistService(store)

	entry, err := service.Upsert(context.Background(), UpsertWatchlistInput{
		Plate: "abc-1234",
		Reason: "test",
		Active: true,
	})
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if entry.Plate != "ABC1234" {
		t.Fatalf("Plate = %q, want ABC1234", entry.Plate)
	}
}
