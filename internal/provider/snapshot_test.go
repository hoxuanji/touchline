package provider_test

import (
	"context"
	"testing"

	"touchline/internal/provider"
)

func TestSnapshotProvider(t *testing.T) {
	p := provider.NewSnapshot()
	ctx := context.Background()

	venues, err := p.Venues(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(venues) != 16 {
		t.Errorf("venues = %d, want 16", len(venues))
	}

	teams, err := p.Teams(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 48 {
		t.Errorf("teams = %d, want 48", len(teams))
	}

	fixtures, err := p.Fixtures(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 12 groups * 6 round-robin matches = 72 group matches, + 32 knockout = 104.
	if len(fixtures) != 104 {
		t.Errorf("fixtures = %d, want 104", len(fixtures))
	}
	groupCount := 0
	for _, m := range fixtures {
		if m.Stage == "group" {
			groupCount++
		}
		if m.VenueID < 1 || m.VenueID > 16 {
			t.Errorf("fixture %d has venueId %d out of range", m.ID, m.VenueID)
		}
	}
	if groupCount != 72 {
		t.Errorf("group matches = %d, want 72", groupCount)
	}

	standings, err := p.Standings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(standings) != 48 {
		t.Errorf("standings = %d, want 48 (one per team)", len(standings))
	}
}
