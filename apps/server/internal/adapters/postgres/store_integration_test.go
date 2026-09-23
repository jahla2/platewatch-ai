package postgres

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

func openTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(store.Close)

	if _, err := store.pool.Exec(ctx, "TRUNCATE detection_events, watchlist_entries"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return store, ctx
}

func TestStorePersistsDetectionAndWatchlist(t *testing.T) {
	store, ctx := openTestStore(t)

	entry, err := store.UpsertWatchlist(ctx, domain.WatchlistEntry{
		PlateText: "ABC-1234",
		PlateKey:  "ABC1234",
		Reason:    "test watchlist",
		Active:    true,
	})
	if err != nil {
		t.Fatalf("UpsertWatchlist() error = %v", err)
	}
	if entry.PlateText != "ABC-1234" || entry.PlateKey != "ABC1234" {
		t.Fatalf("watchlist entry = %#v", entry)
	}

	flagged, err := store.IsFlagged(ctx, "ABC1234")
	if err != nil || !flagged {
		t.Fatalf("IsFlagged() = %v, %v; want true, nil", flagged, err)
	}

	event := domain.DetectionEvent{
		ID:           "evt_test",
		CameraID:     "CAM-01",
		TrackID:      42,
		PlateText:    "ABC-1234",
		PlateKey:     "ABC1234",
		Confidence:   0.93,
		Flagged:      true,
		SnapshotURL:  "/evidence/CAM-01/42/vehicle.jpg",
		PlateCropURL: "/evidence/CAM-01/42/plate.jpg",
		DetectedAt:   time.Now().UTC(),
	}
	saved, created, err := store.SaveIdempotent(ctx, "idem-test", event)
	if err != nil {
		t.Fatalf("SaveIdempotent() error = %v", err)
	}
	if !created || saved.ID != event.ID {
		t.Fatalf("SaveIdempotent() = %#v, %v", saved, created)
	}

	replayed, created, err := store.SaveIdempotent(
		ctx,
		"idem-test",
		domain.DetectionEvent{ID: "evt_other"},
	)
	if err != nil {
		t.Fatalf("duplicate SaveIdempotent() error = %v", err)
	}
	if created || replayed.ID != event.ID {
		t.Fatalf("duplicate = %#v, %v; want original event", replayed, created)
	}

	events, err := store.List(ctx, 10, nil)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("List() = %#v", events)
	}
	if events[0].PlateText != event.PlateText || events[0].PlateKey != event.PlateKey {
		t.Fatalf("plate values were not persisted: %#v", events[0])
	}
	if events[0].SnapshotURL != event.SnapshotURL || events[0].PlateCropURL != event.PlateCropURL {
		t.Fatalf("evidence URLs were not persisted: %#v", events[0])
	}
	if events[0].IdempotencyKey != "idem-test" {
		t.Fatalf("idempotency key = %q, want idem-test", events[0].IdempotencyKey)
	}
}

func TestStoreConcurrentIdempotencyCreatesSingleRow(t *testing.T) {
	store, ctx := openTestStore(t)

	const workers = 20
	var createdCount atomic.Int64
	errorsCh := make(chan error, workers)
	var wg sync.WaitGroup

	for index := range workers {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			event := domain.DetectionEvent{
				ID:         fmt.Sprintf("evt_%02d", index),
				CameraID:   "CAM-01",
				TrackID:    int64(index + 1),
				PlateText:  "ABC-1234",
				PlateKey:   "ABC1234",
				Confidence: 0.9,
				DetectedAt: time.Now().UTC(),
			}
			_, created, err := store.SaveIdempotent(ctx, "shared-key", event)
			if err != nil {
				errorsCh <- err
				return
			}
			if created {
				createdCount.Add(1)
			}
		}(index)
	}

	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Errorf("concurrent SaveIdempotent() error = %v", err)
	}

	if createdCount.Load() != 1 {
		t.Fatalf("created count = %d, want 1", createdCount.Load())
	}

	var rowCount int
	if err := store.pool.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM detection_events WHERE idempotency_key = $1",
		"shared-key",
	).Scan(&rowCount); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("row count = %d, want 1", rowCount)
	}
}

func TestStoreHandlesConcurrentDistinctWrites(t *testing.T) {
	store, ctx := openTestStore(t)

	const workers = 100
	errorsCh := make(chan error, workers)
	var wg sync.WaitGroup

	for index := range workers {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			event := domain.DetectionEvent{
				ID:         fmt.Sprintf("evt_load_%03d", index),
				CameraID:   fmt.Sprintf("CAM-%02d", index%4),
				TrackID:    int64(index + 1),
				PlateText:  fmt.Sprintf("LOAD-%03d", index),
				PlateKey:   fmt.Sprintf("LOAD%03d", index),
				Confidence: 0.9,
				DetectedAt: time.Now().UTC(),
			}
			_, created, err := store.SaveIdempotent(
				ctx,
				fmt.Sprintf("load-key-%03d", index),
				event,
			)
			if err != nil {
				errorsCh <- err
				return
			}
			if !created {
				errorsCh <- fmt.Errorf("write %d unexpectedly replayed", index)
			}
		}(index)
	}

	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Errorf("concurrent distinct write error = %v", err)
	}

	var rowCount int
	if err := store.pool.QueryRow(ctx, "SELECT COUNT(*) FROM detection_events").Scan(&rowCount); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if rowCount != workers {
		t.Fatalf("row count = %d, want %d", rowCount, workers)
	}
}

func TestStoreListsLegacyRowsWithoutIdempotencyKey(t *testing.T) {
	store, ctx := openTestStore(t)

	_, err := store.pool.Exec(
		ctx,
		`INSERT INTO detection_events
			(id, camera_id, track_id, plate_number, plate_text, confidence, flagged, detected_at)
		  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		"evt_legacy",
		"CAM-LEGACY",
		int64(1),
		"LEGACY1",
		"LEGACY-1",
		0.8,
		false,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	events, err := store.List(ctx, 10, nil)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(events) != 1 || events[0].ID != "evt_legacy" {
		t.Fatalf("events = %#v", events)
	}
	if events[0].IdempotencyKey != "" {
		t.Fatalf("legacy idempotency key = %q, want empty", events[0].IdempotencyKey)
	}
}
