package ports

import (
	"context"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

type DetectionRepository interface {
	Save(ctx context.Context, event domain.DetectionEvent) error
	List(ctx context.Context, limit int) ([]domain.DetectionEvent, error)
}

type Watchlist interface {
	IsFlagged(ctx context.Context, plate string) (bool, error)
}

type WatchlistStore interface {
	Watchlist
	ListWatchlist(ctx context.Context, limit int) ([]domain.WatchlistEntry, error)
	UpsertWatchlist(
		ctx context.Context,
		entry domain.WatchlistEntry,
	) (domain.WatchlistEntry, error)
	DeleteWatchlist(ctx context.Context, plate string) error
}

type DetectionPublisher interface {
	Publish(ctx context.Context, event domain.DetectionEvent) error
}

type DetectionSubscriber interface {
	Subscribe() (<-chan domain.DetectionEvent, func())
}
