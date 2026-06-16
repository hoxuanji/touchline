package snapshotdata_test

import (
	"encoding/json"
	"testing"

	snapshotdata "touchline/data/snapshot"
)

func TestVenuesSnapshotHas16(t *testing.T) {
	var venues []map[string]any
	if err := json.Unmarshal(snapshotdata.VenuesJSON, &venues); err != nil {
		t.Fatal(err)
	}
	if len(venues) != 16 {
		t.Fatalf("venues = %d, want 16", len(venues))
	}
}

func TestTeamsSnapshotHas48In12Groups(t *testing.T) {
	var doc struct {
		Teams []struct {
			ID    int    `json:"id"`
			Group string `json:"group"`
		} `json:"teams"`
	}
	if err := json.Unmarshal(snapshotdata.TeamsJSON, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Teams) != 48 {
		t.Fatalf("teams = %d, want 48", len(doc.Teams))
	}
	byGroup := map[string]int{}
	for _, tm := range doc.Teams {
		byGroup[tm.Group]++
	}
	if len(byGroup) != 12 {
		t.Fatalf("groups = %d, want 12", len(byGroup))
	}
	for g, n := range byGroup {
		if n != 4 {
			t.Errorf("group %s has %d teams, want 4", g, n)
		}
	}
}
