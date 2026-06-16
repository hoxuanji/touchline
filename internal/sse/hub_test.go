package sse_test

import (
	"testing"
	"time"

	"touchline/internal/sse"
)

func TestHubBroadcastReachesSubscriber(t *testing.T) {
	h := sse.NewHub()
	sub := h.Subscribe()
	defer h.Unsubscribe(sub)

	h.Broadcast(sse.Message{Type: "match.update", Data: map[string]any{"id": 1}})

	select {
	case msg := <-sub.C:
		if msg.Type != "match.update" {
			t.Fatalf("got type %q", msg.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive broadcast")
	}
}

func TestHubUnsubscribeStopsDelivery(t *testing.T) {
	h := sse.NewHub()
	sub := h.Subscribe()
	h.Unsubscribe(sub)
	h.Broadcast(sse.Message{Type: "x"}) // must not panic on closed channel
	if got := h.SubscriberCount(); got != 0 {
		t.Fatalf("SubscriberCount = %d, want 0", got)
	}
}

func TestHubSlowSubscriberDoesNotBlock(t *testing.T) {
	h := sse.NewHub()
	_ = h.Subscribe() // never drains
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			h.Broadcast(sse.Message{Type: "x"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Broadcast blocked on a slow subscriber")
	}
}
