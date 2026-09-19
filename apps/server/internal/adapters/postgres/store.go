package postgres

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

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

func (s *Store) Save(ctx context.Context, event domain.DetectionEvent) error {
	_, err := s.pool.Exec(
		ctx,
		`INSERT INTO detection_events
			(id, camera_id, track_id, plate_number, confidence, flagged, detected_at)
		  VALUES ($1, $2, $3, $4, $5, $6, $7)
		  ON CONFLICT (id) DO NOTHING`,
		event.ID,
		event.CameraID,
		event.TrackID,
		event.Plate,
		event.Confidence,
		event.Flagged,
		event.DetectedAt,
	)
	return err
}

func (s *Store) List(ctx context.Context, limit int) ([]domain.DetectionEvent, error) {
	rows, err := s.pool.Query(
		ctx,
		`SELECT id, camera_id, track_id, plate_number, confidence, flagged, detected_at
		   FROM detection_events
		  ORDER BY detected_at DESC
		  LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]domain.DetectionEvent, 0, limit)
	for rows.Next() {
		var event domain.DetectionEvent
		if err := rows.Scan(
			&event.ID,
			&event.CameraID,
			&event.TrackID,
			&event.Plate,
			&event.Confidence,
			&event.Flagged,
			&event.DetectedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) IsFlagged(ctx context.Context, plate string) (bool, error) {
	var flagged bool
	err := s.pool.QueryRow(
		ctx,
		`SELECT EXISTS (
			SELECT 1 FROM watchlist_entries
			 WHERE plate_number = $1 AND active = TRUE
		)`,
		plate,
	).Scan(&flagged)
	return flagged, err
}

func (s *Store) ListWatchlist(
	ctx context.Context,
	limit int,
) ([]domain.WatchlistEntry, error) {
	rows, err := s.pool.Query(
		ctx,
		`SELECT plate_number, reason, active, created_at, updated_at
		   FROM watchlist_entries
		  ORDER BY updated_at DESC
		  LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]domain.WatchlistEntry, 0, limit)
	for rows.Next() {
		var entry domain.WatchlistEntry
		if err := rows.Scan(
			&entry.Plate,
			&entry.Reason,
			&entry.Active,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) UpsertWatchlist(
	ctx context.Context,
	entry domain.WatchlistEntry,
) (domain.WatchlistEntry, error) {
	var saved domain.WatchlistEntry
	err := s.pool.QueryRow(
		ctx,
		`INSERT INTO watchlist_entries (plate_number, reason, active)
		  VALUES ($1, $2, $3)
		  ON CONFLICT (plate_number) DO UPDATE
		  SET reason = EXCLUDED.reason,
		      active = EXCLUDED.active,
		      updated_at = NOW()
		  RETURNING plate_number, reason, active, created_at, updated_at`,
		entry.Plate,
		entry.Reason,
		entry.Active,
	).Scan(
		&saved.Plate,
		&saved.Reason,
		&saved.Active,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	)
	return saved, err
}

func (s *Store) DeleteWatchlist(ctx context.Context, plate string) error {
	command, err := s.pool.Exec(
		ctx,
		`DELETE FROM watchlist_entries WHERE plate_number = $1`,
		plate,
	)
	if err != nil {
		return err
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
		if _, err := s.pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}
