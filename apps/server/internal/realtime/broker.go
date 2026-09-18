package realtime

import (
	"context"
	"sync"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

type Broker struct {
	mu          sync.RWMutex
	subscribers map[chan domain.DetectionEvent]struct{}
}

func NewBroker() *Broker {
	return &Broker{subscribers: make(map[chan domain.DetectionEvent]struct{})}
}

func (b *Broker) Publish(_ context.Context, event domain.DetectionEvent) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for subscriber := range b.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
	return nil
}

func (b *Broker) Subscribe() (<-chan domain.DetectionEvent, func()) {
	channel := make(chan domain.DetectionEvent, 16)

	b.mu.Lock()
	b.subscribers[channel] = struct{}{}
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, exists := b.subscribers[channel]; exists {
			delete(b.subscribers, channel)
			close(channel)
		}
	}
	return channel, cancel
}
