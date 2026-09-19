package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

func TestStorePersistsDetectionAndWatchlist(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	if _, err := store.pool.Exec(ctx, "TRUNCATE detection_events, watchlist_entries"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	entry, err := store.UpsertWatchlist(ctx, domain.WatchlistEntry{
		Plate:  "ABC1234",
		Reason: "test watchlist",
		Active: true,
	})
	if err != nil {
		t.Fatalf("UpsertWatchlist() error = %v", err)
	}
	if entry.Plate != "ABC1234" {
		t.Fatalf("watchlist plate = %q", entry.Plate)
	}

	flagged, err := store.IsFlagged(ctx, "ABC1234")
	if err != nil || !flagged {
		t.Fatalf("IsFlagged() = %v, %v; want true, nil", flagged, err)
	}

	event := domain.DetectionEvent{
		ID:           "evt_test",
		CameraID:     "CAM-01",
		TrackID:      42,
		Plate:        "ABC1234",
		Confidence:   0.93,
		Flagged:      true,
		SnapshotURL:  "/evidence/CAM-01/42/vehicle.jpg",
		PlateCropURL: "/evidence/CAM-01/42/plate.jpg",
		DetectedAt:   time.Now().UTC(),
	}
	if err := store.Save(ctx, event); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	events, err := store.List(ctx, 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(events) != 1 || events[0].Plate != "ABC1234" {
		t.Fatalf("List() = %#v", events)
	}
	if events[0].SnapshotURL != event.SnapshotURL || events[0].PlateCropURL != event.PlateCropURL {
		t.Fatalf("evidence URLs were not persisted: %#v", events[0])
	}
}
