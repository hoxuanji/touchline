// Package sse implements a Server-Sent Events hub: one upstream update is fanned
// out to all connected subscribers. Slow subscribers drop messages rather than
// blocking the broadcaster.
package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Message is the typed SSE envelope sent to clients.
type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Subscriber struct {
	C chan Message
}

type Hub struct {
	mu   sync.RWMutex
	subs map[*Subscriber]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[*Subscriber]struct{}{}}
}

func (h *Hub) Subscribe() *Subscriber {
	s := &Subscriber{C: make(chan Message, 16)}
	h.mu.Lock()
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	return s
}

func (h *Hub) Unsubscribe(s *Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subs[s]; ok {
		delete(h.subs, s)
		close(s.C)
	}
}

// Broadcast sends to every subscriber. A subscriber whose buffer is full is
// skipped (message dropped) so one slow client can't stall the hub.
func (h *Hub) Broadcast(msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for s := range h.subs {
		select {
		case s.C <- msg:
		default:
		}
	}
}

func (h *Hub) SubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}

// ServeHTTP streams broadcasts to one client as SSE until the request context
// is cancelled (client disconnect).
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sub := h.Subscribe()
	defer h.Unsubscribe(sub)

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-sub.C:
			if !ok {
				return
			}
			b, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}
