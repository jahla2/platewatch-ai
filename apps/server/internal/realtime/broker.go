package realtime

import (
	"context"
	"sync"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

const subscriberBufferSize = 64

type Broker struct {
	mu          sync.Mutex
	subscribers map[chan domain.DetectionEvent]struct{}
}

func NewBroker() *Broker {
	return &Broker{subscribers: make(map[chan domain.DetectionEvent]struct{})}
}

func (b *Broker) Publish(_ context.Context, event domain.DetectionEvent) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for subscriber := range b.subscribers {
		select {
		case subscriber <- event:
		default:
			// Never block detection ingestion on a slow browser. Closing the
			// overloaded stream makes EventSource reconnect; the web client then
			// resynchronizes from PostgreSQL and de-duplicates by event ID.
			delete(b.subscribers, subscriber)
			close(subscriber)
		}
	}
	return nil
}

func (b *Broker) Subscribe() (<-chan domain.DetectionEvent, func()) {
	channel := make(chan domain.DetectionEvent, subscriberBufferSize)

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
