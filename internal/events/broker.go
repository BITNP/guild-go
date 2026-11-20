package events

import "sync"

// Broker broadcasts non-blocking wake-up hints to live event stream subscribers
// after PostgreSQL transactions commit new rows to the events table.
type Broker struct {
	mu   sync.RWMutex
	subs map[chan int64]struct{}
}

// NewBroker constructs an in-process event wake-up broker.
func NewBroker() *Broker {
	return &Broker{
		subs: make(map[chan int64]struct{}),
	}
}

// Subscribe registers a listener channel and returns an idempotent unsubscribe function.
func (b *Broker) Subscribe() (<-chan int64, func()) {
	ch := make(chan int64, 64)
	if b == nil {
		close(ch)
		return ch, func() {}
	}
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()

	return ch, func() {
		b.mu.Lock()
		if _, ok := b.subs[ch]; ok {
			delete(b.subs, ch)
			close(ch)
		}
		b.mu.Unlock()
	}
}

// Publish notifies all active subscribers that events up to the latest ID have committed.
func (b *Broker) Publish(ids ...int64) {
	if b == nil || len(ids) == 0 {
		return
	}
	latest := ids[len(ids)-1]
	b.mu.RLock()
	for ch := range b.subs {
		select {
		case ch <- latest:
		default:
		}
	}
	b.mu.RUnlock()
}
