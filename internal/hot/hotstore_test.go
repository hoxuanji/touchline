package hot_test

import (
	"sync"
	"testing"

	"touchline/internal/hot"
	"touchline/internal/model"
)

func TestHotStoreHydrateAndRead(t *testing.T) {
	h := hot.New()
	h.Hydrate(
		[]model.Match{{ID: 1, Status: "scheduled", HomeScore: 0}},
		[]model.Standing{{Group: "A", TeamID: 1}},
	)
	got := h.Matches()
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("Matches() = %v", got)
	}
}

func TestHotStorePatchMatch(t *testing.T) {
	h := hot.New()
	h.Hydrate([]model.Match{{ID: 1, Status: "scheduled"}}, nil)
	ok := h.PatchMatch(model.Match{ID: 1, Status: "live", Minute: 12, HomeScore: 1})
	if !ok {
		t.Fatal("PatchMatch should report it updated an existing match")
	}
	m, found := h.Match(1)
	if !found || m.Status != "live" || m.HomeScore != 1 {
		t.Fatalf("after patch, Match(1) = %+v found=%v", m, found)
	}
	if h.PatchMatch(model.Match{ID: 999}) {
		t.Error("PatchMatch on unknown id should report false")
	}
}

func TestHotStoreConcurrentReadsWrites(t *testing.T) {
	h := hot.New()
	h.Hydrate([]model.Match{{ID: 1}}, nil)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); h.PatchMatch(model.Match{ID: 1, Minute: 1}) }()
		go func() { defer wg.Done(); _ = h.Matches() }()
	}
	wg.Wait()
}
