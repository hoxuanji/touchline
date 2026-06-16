package sim_test

import (
	"math/rand"
	"testing"
	"time"

	"touchline/internal/hot"
	"touchline/internal/model"
	"touchline/internal/sim"
	"touchline/internal/sse"
	"touchline/internal/store"
)

type capture struct{ msgs []sse.Message }

func (c *capture) Broadcast(m sse.Message) { c.msgs = append(c.msgs, m) }

func newSim(t *testing.T) (*sim.Simulator, *hot.Store, *capture, *store.Store) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	_ = st.UpsertTeams([]model.Team{{ID: 1, Name: "A", Group: "A"}, {ID: 2, Name: "B", Group: "A"}})
	_ = st.UpsertMatches([]model.Match{{ID: 1, Stage: "group", Group: "A", HomeID: 1, AwayID: 2, Status: "live", Minute: 0}})
	h := hot.New()
	m, _ := st.Matches()
	h.Hydrate(m, nil)
	cap := &capture{}
	s := sim.New(st, h, cap, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0) })
	return s, h, cap, st
}

func TestStepAdvancesLiveMatchMinuteAndBroadcasts(t *testing.T) {
	s, h, cap, _ := newSim(t)
	s.Step()
	m, _ := h.Match(1)
	if m.Minute <= 0 {
		t.Fatalf("minute did not advance: %d", m.Minute)
	}
	var sawUpdate bool
	for _, msg := range cap.msgs {
		if msg.Type == "match.update" {
			sawUpdate = true
		}
	}
	if !sawUpdate {
		t.Fatal("expected a match.update broadcast")
	}
}

func TestStepEventuallyFinishesMatch(t *testing.T) {
	s, h, _, _ := newSim(t)
	for i := 0; i < 1000 && func() bool { m, _ := h.Match(1); return m.Status != "finished" }(); i++ {
		s.Step()
	}
	m, _ := h.Match(1)
	if m.Status != "finished" {
		t.Fatalf("match never finished; status=%s minute=%d", m.Status, m.Minute)
	}
}

func TestStepIgnoresNonLiveMatches(t *testing.T) {
	s, h, cap, st := newSim(t)
	_ = st.UpsertMatches([]model.Match{{ID: 1, Status: "scheduled", HomeID: 1, AwayID: 2}})
	m, _ := st.Matches()
	h.Hydrate(m, nil)
	cap.msgs = nil
	s.Step()
	if len(cap.msgs) != 0 {
		t.Fatalf("scheduled match should not broadcast; got %d msgs", len(cap.msgs))
	}
}
