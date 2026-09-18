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

type DetectionPublisher interface {
	Publish(ctx context.Context, event domain.DetectionEvent) error
}

type DetectionSubscriber interface {
	Subscribe() (<-chan domain.DetectionEvent, func())
}
