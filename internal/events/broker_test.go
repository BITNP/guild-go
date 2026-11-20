package events

import (
	"testing"
	"time"
)

func TestBrokerPublishAndUnsubscribe(t *testing.T) {
	broker := NewBroker()
	ch1, unsub1 := broker.Subscribe()
	ch2, unsub2 := broker.Subscribe()
	defer unsub2()

	broker.Publish() // Empty publish must be a no-op.
	select {
	case got := <-ch1:
		t.Fatalf("unexpected notification on empty publish: %d", got)
	default:
	}

	broker.Publish(10, 11)
	for _, ch := range []<-chan int64{ch1, ch2} {
		select {
		case got := <-ch:
			if got != 11 {
				t.Fatalf("expected latest id 11, got %d", got)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for broker publish")
		}
	}

	// Unsubscribe must be idempotent and close the subscriber channel cleanly.
	unsub1()
	unsub1()
	if _, ok := <-ch1; ok {
		t.Fatal("expected unsubscribed channel to be closed")
	}

	// Publishing to remaining subscribers continues working without panicking.
	broker.Publish(12)
	select {
	case got := <-ch2:
		if got != 12 {
			t.Fatalf("expected 12, got %d", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second publish")
	}
}

func TestBrokerNilSafe(t *testing.T) {
	var broker *Broker
	broker.Publish(1)
	ch, unsub := broker.Subscribe()
	defer unsub()
	if _, ok := <-ch; ok {
		t.Fatal("expected closed channel from nil broker")
	}
}
