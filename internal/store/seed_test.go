package store_test

import (
	"context"
	"testing"

	"touchline/internal/provider"
	"touchline/internal/store"
)

func TestSeedFromSnapshotPopulatesWhenEmpty(t *testing.T) {
	s := newTestStore(t)
	p := provider.NewSnapshot()

	seeded, err := store.Seed(context.Background(), s, p)
	if err != nil {
		t.Fatal(err)
	}
	if !seeded {
		t.Fatal("expected Seed to report it seeded an empty db")
	}

	teams, _ := s.Teams()
	if len(teams) != 48 {
		t.Errorf("seeded teams = %d, want 48", len(teams))
	}
	matches, _ := s.Matches()
	if len(matches) != 104 {
		t.Errorf("seeded matches = %d, want 104", len(matches))
	}
	standings, _ := s.Standings()
	if len(standings) != 48 {
		t.Errorf("seeded standings = %d, want 48", len(standings))
	}

	// Second call is a no-op (db already populated).
	seeded2, err := store.Seed(context.Background(), s, p)
	if err != nil {
		t.Fatal(err)
	}
	if seeded2 {
		t.Error("expected second Seed to be a no-op on a populated db")
	}
}
