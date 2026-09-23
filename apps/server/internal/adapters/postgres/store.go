package postgres

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	store := &Store{pool: pool}
	if err := store.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) SaveIdempotent(
	ctx context.Context,
	idempotencyKey string,
	event domain.DetectionEvent,
) (domain.DetectionEvent, bool, error) {
	event.IdempotencyKey = idempotencyKey

	row := s.pool.QueryRow(
		ctx,
		`INSERT INTO detection_events
			(id, idempotency_key, camera_id, track_id, plate_number, plate_text, confidence,
			 flagged, snapshot_url, plate_crop_url, detected_at)
		  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		  ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL AND idempotency_key <> ''
		  DO NOTHING
		  RETURNING id, idempotency_key, camera_id, track_id, plate_text, plate_number,
		            confidence, flagged, snapshot_url, plate_crop_url, detected_at`,
		event.ID,
		idempotencyKey,
		event.CameraID,
		event.TrackID,
		event.PlateKey,
		event.PlateText,
		event.Confidence,
		event.Flagged,
		event.SnapshotURL,
		event.PlateCropURL,
		event.DetectedAt,
	)

	var saved domain.DetectionEvent
	err := scanDetection(row, &saved)
	if err == nil {
		return saved, true, nil
	}
	if err != pgx.ErrNoRows {
		return domain.DetectionEvent{}, false, fmt.Errorf("insert detection: %w", err)
	}

	row = s.pool.QueryRow(
		ctx,
		`SELECT id, idempotency_key, camera_id, track_id, plate_text, plate_number,
		        confidence, flagged, snapshot_url, plate_crop_url, detected_at
		   FROM detection_events
		  WHERE idempotency_key = $1`,
		idempotencyKey,
	)
	if err := scanDetection(row, &saved); err != nil {
		return domain.DetectionEvent{}, false, fmt.Errorf("load idempotent detection: %w", err)
	}
	return saved, false, nil
}

func (s *Store) List(
	ctx context.Context,
	limit int,
	cursor *domain.DetectionCursor,
) ([]domain.DetectionEvent, error) {
	const baseQuery = `SELECT id, idempotency_key, camera_id, track_id, plate_text, plate_number,
	                           confidence, flagged, snapshot_url, plate_crop_url, detected_at
	                      FROM detection_events`

	var (
		rows pgx.Rows
		err  error
	)
	if cursor == nil {
		rows, err = s.pool.Query(
			ctx,
			baseQuery+` ORDER BY detected_at DESC, id DESC LIMIT $1`,
			limit,
		)
	} else {
		rows, err = s.pool.Query(
			ctx,
			baseQuery+`
			 WHERE (detected_at, id) < ($1, $2)
			 ORDER BY detected_at DESC, id DESC
			 LIMIT $3`,
			cursor.DetectedAt,
			cursor.ID,
			limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("list detections: %w", err)
	}
	defer rows.Close()

	events := make([]domain.DetectionEvent, 0, limit)
	for rows.Next() {
		var event domain.DetectionEvent
		if err := scanDetection(rows, &event); err != nil {
			return nil, fmt.Errorf("scan detection: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate detections: %w", err)
	}
	return events, nil
}

func scanDetection(row pgx.Row, event *domain.DetectionEvent) error {
	return row.Scan(
		&event.ID,
		&event.IdempotencyKey,
		&event.CameraID,
		&event.TrackID,
		&event.PlateText,
		&event.PlateKey,
		&event.Confidence,
		&event.Flagged,
		&event.SnapshotURL,
		&event.PlateCropURL,
		&event.DetectedAt,
	)
}

func (s *Store) IsFlagged(ctx context.Context, plateKey string) (bool, error) {
	var flagged bool
	err := s.pool.QueryRow(
		ctx,
		`SELECT EXISTS (
			SELECT 1 FROM watchlist_entries
			 WHERE plate_number = $1 AND active = TRUE
		)`,
		plateKey,
	).Scan(&flagged)
	if err != nil {
		return false, fmt.Errorf("watchlist lookup: %w", err)
	}
	return flagged, nil
}

func (s *Store) ListWatchlist(
	ctx context.Context,
	limit int,
) ([]domain.WatchlistEntry, error) {
	rows, err := s.pool.Query(
		ctx,
		`SELECT plate_text, plate_number, reason, active, created_at, updated_at
		   FROM watchlist_entries
		  ORDER BY updated_at DESC
		  LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list watchlist: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.WatchlistEntry, 0, limit)
	for rows.Next() {
		var entry domain.WatchlistEntry
		if err := rows.Scan(
			&entry.PlateText,
			&entry.PlateKey,
			&entry.Reason,
			&entry.Active,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan watchlist: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watchlist: %w", err)
	}
	return entries, nil
}

func (s *Store) UpsertWatchlist(
	ctx context.Context,
	entry domain.WatchlistEntry,
) (domain.WatchlistEntry, error) {
	var saved domain.WatchlistEntry
	err := s.pool.QueryRow(
		ctx,
		`INSERT INTO watchlist_entries (plate_number, plate_text, reason, active)
		  VALUES ($1, $2, $3, $4)
		  ON CONFLICT (plate_number) DO UPDATE
		  SET plate_text = EXCLUDED.plate_text,
		      reason = EXCLUDED.reason,
		      active = EXCLUDED.active,
		      updated_at = NOW()
		  RETURNING plate_text, plate_number, reason, active, created_at, updated_at`,
		entry.PlateKey,
		entry.PlateText,
		entry.Reason,
		entry.Active,
	).Scan(
		&saved.PlateText,
		&saved.PlateKey,
		&saved.Reason,
		&saved.Active,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	)
	if err != nil {
		return domain.WatchlistEntry{}, fmt.Errorf("upsert watchlist: %w", err)
	}
	return saved, nil
}

func (s *Store) DeleteWatchlist(ctx context.Context, plateKey string) error {
	command, err := s.pool.Exec(
		ctx,
		`DELETE FROM watchlist_entries WHERE plate_number = $1`,
		plateKey,
	)
	if err != nil {
		return fmt.Errorf("delete watchlist: %w", err)
	}
	if command.RowsAffected() == 0 {
		return fmt.Errorf("watchlist plate not found")
	}
	return nil
}

func (s *Store) migrate(ctx context.Context) error {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		sql, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}
