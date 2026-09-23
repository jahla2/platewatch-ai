package memory

import (
	"context"
	"sync"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

type DetectionRepository struct {
	mu          sync.RWMutex
	events      []domain.DetectionEvent
	idempotency map[string]domain.DetectionEvent
}

func NewDetectionRepository() *DetectionRepository {
	return &DetectionRepository{
		idempotency: make(map[string]domain.DetectionEvent),
	}
}

func (r *DetectionRepository) SaveIdempotent(
	_ context.Context,
	idempotencyKey string,
	event domain.DetectionEvent,
) (domain.DetectionEvent, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, found := r.idempotency[idempotencyKey]; found {
		return existing, false, nil
	}

	event.IdempotencyKey = idempotencyKey
	r.events = append(r.events, event)
	r.idempotency[idempotencyKey] = event
	return event, true, nil
}

func (r *DetectionRepository) List(
	_ context.Context,
	limit int,
	cursor *domain.DetectionCursor,
) ([]domain.DetectionEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.DetectionEvent, 0, limit)
	for index := len(r.events) - 1; index >= 0 && len(result) < limit; index-- {
		event := r.events[index]
		if cursor != nil {
			if event.DetectedAt.After(cursor.DetectedAt) {
				continue
			}
			if event.DetectedAt.Equal(cursor.DetectedAt) && event.ID >= cursor.ID {
				continue
			}
		}
		result = append(result, event)
	}
	return result, nil
}
