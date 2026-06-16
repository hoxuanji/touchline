package sim_test

import (
	"testing"

	"touchline/internal/hot"
	"touchline/internal/model"
	"touchline/internal/sim"
	"touchline/internal/store"
)

func TestPromoteLiveMakesNScheduledMatchesLive(t *testing.T) {
	st, _ := store.Open(":memory:")
	defer st.Close()
	_ = st.UpsertMatches([]model.Match{
		{ID: 1, Stage: "group", Status: "scheduled"},
		{ID: 2, Stage: "group", Status: "scheduled"},
		{ID: 3, Stage: "group", Status: "scheduled"},
	})
	h := hot.New()
	m, _ := st.Matches()
	h.Hydrate(m, nil)

	n := sim.PromoteLive(st, h, 2)
	if n != 2 {
		t.Fatalf("PromoteLive returned %d, want 2", n)
	}
	live := 0
	for _, mm := range h.Matches() {
		if mm.Status == "live" {
			live++
		}
	}
	if live != 2 {
		t.Fatalf("live matches = %d, want 2", live)
	}
}
