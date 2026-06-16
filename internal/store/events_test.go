package store_test

import (
	"testing"

	"touchline/internal/model"
)

func TestAddAndListEvents(t *testing.T) {
	s := newTestStore(t)
	id, err := s.AddEvent(model.MatchEvent{MatchID: 1, Minute: 23, Type: "goal", TeamID: 1, PlayerID: 0, Detail: "Header"})
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Fatalf("AddEvent id = %d, want > 0", id)
	}
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 1, Minute: 45, Type: "card", TeamID: 2, Detail: "Yellow"})
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 2, Minute: 10, Type: "goal", TeamID: 3})

	evs, err := s.EventsByMatch(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("EventsByMatch(1) = %d, want 2", len(evs))
	}
	if evs[0].Minute != 23 || evs[1].Minute != 45 {
		t.Fatalf("events not ordered by minute: %+v", evs)
	}
}
