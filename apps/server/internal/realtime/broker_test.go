package realtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jahla2/platewatch-ai/apps/server/internal/domain"
)

func TestBrokerConcurrentPublishAndSubscribe(t *testing.T) {
	broker := NewBroker()
	const subscribers = 20
	const events = 100

	var cancels []func()
	var channels []<-chan domain.DetectionEvent
	for range subscribers {
		channel, cancel := broker.Subscribe()
		channels = append(channels, channel)
		cancels = append(cancels, cancel)
	}
	defer func() {
		for _, cancel := range cancels {
			cancel()
		}
	}()

	var wg sync.WaitGroup
	for index := range events {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_ = broker.Publish(context.Background(), domain.DetectionEvent{
				ID:         "evt",
				TrackID:    int64(index + 1),
				DetectedAt: time.Now().UTC(),
			})
		}(index)
	}
	wg.Wait()

	received := 0
	for _, channel := range channels {
		for {
			select {
			case <-channel:
				received++
			default:
				goto nextChannel
			}
		}
	nextChannel:
	}

	if received == 0 {
		t.Fatal("expected at least one event across subscribers")
	}
}
