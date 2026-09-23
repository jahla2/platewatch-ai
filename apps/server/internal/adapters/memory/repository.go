package memory

import (
	"context"
	"sync"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

type DetectionRepository struct {
	mu              sync.RWMutex
	events          []domain.DetectionEvent
	idempotencyKeys map[string]struct{}
}

func NewDetectionRepository() *DetectionRepository {
	return &DetectionRepository{idempotencyKeys: make(map[string]struct{})}
}

func (r *DetectionRepository) SaveIfAbsent(
	_ context.Context,
	event domain.DetectionEvent,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.idempotencyKeys[event.IdempotencyKey]; exists {
		return false, nil
	}

	r.idempotencyKeys[event.IdempotencyKey] = struct{}{}
	r.events = append(r.events, event)
	return true, nil
}

func (r *DetectionRepository) List(_ context.Context, limit int) ([]domain.DetectionEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit > len(r.events) {
		limit = len(r.events)
	}
	result := make([]domain.DetectionEvent, 0, limit)
	for index := len(r.events) - 1; index >= len(r.events)-limit; index-- {
		result = append(result, r.events[index])
	}
	return result, nil
}
