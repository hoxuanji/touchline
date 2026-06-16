package store_test

import (
	"testing"

	"touchline/internal/model"
	"touchline/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestMigrateAndCountEmpty(t *testing.T) {
	s := newTestStore(t)
	n, err := s.CountTeams()
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("CountTeams on fresh db = %d, want 0", n)
	}
}

func TestUpsertAndReadTeamsVenuesMatches(t *testing.T) {
	s := newTestStore(t)
	if err := s.UpsertTeams([]model.Team{{ID: 1, Name: "Brazil", Country: "Brazil", Group: "A"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertVenues([]model.Venue{{ID: 1, Name: "Azteca", City: "Mexico City", Country: "Mexico", Capacity: 87000}}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertMatches([]model.Match{{ID: 1, Stage: "group", Group: "A", VenueID: 1, HomeID: 1, AwayID: 1, Status: "scheduled"}}); err != nil {
		t.Fatal(err)
	}

	teams, err := s.Teams()
	if err != nil || len(teams) != 1 || teams[0].Name != "Brazil" {
		t.Fatalf("Teams() = %v, err %v", teams, err)
	}
	venues, err := s.Venues()
	if err != nil || len(venues) != 1 {
		t.Fatalf("Venues() = %v, err %v", venues, err)
	}
	matches, err := s.Matches()
	if err != nil || len(matches) != 1 || matches[0].Group != "A" {
		t.Fatalf("Matches() = %v, err %v", matches, err)
	}
}

func TestUpsertIsIdempotentOnConflict(t *testing.T) {
	s := newTestStore(t)
	_ = s.UpsertTeams([]model.Team{{ID: 1, Name: "Brazil", Group: "A"}})
	_ = s.UpsertTeams([]model.Team{{ID: 1, Name: "Brasil", Group: "A"}}) // same id, new name
	teams, _ := s.Teams()
	if len(teams) != 1 || teams[0].Name != "Brasil" {
		t.Fatalf("expected upsert to update in place, got %v", teams)
	}
}

func TestFollowsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	if err := s.AddFollow(7); err != nil {
		t.Fatal(err)
	}
	if err := s.AddFollow(7); err != nil { // idempotent
		t.Fatalf("re-follow should be idempotent: %v", err)
	}
	follows, err := s.Follows()
	if err != nil || len(follows) != 1 || follows[0] != 7 {
		t.Fatalf("Follows() = %v, err %v", follows, err)
	}
	if err := s.RemoveFollow(7); err != nil {
		t.Fatal(err)
	}
	follows, _ = s.Follows()
	if len(follows) != 0 {
		t.Fatalf("after remove, follows = %v", follows)
	}
}
