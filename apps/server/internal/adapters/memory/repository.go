package memory

import (
	"context"
	"sync"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

type DetectionRepository struct {
	mu     sync.RWMutex
	events []domain.DetectionEvent
}

func NewDetectionRepository() *DetectionRepository {
	return &DetectionRepository{}
}

func (r *DetectionRepository) Save(_ context.Context, event domain.DetectionEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return nil
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
