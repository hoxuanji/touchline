package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"touchline/internal/model"
)

func TestMatchJSONFieldNames(t *testing.T) {
	m := model.Match{
		ID: 7, Stage: "group", Group: "A", VenueID: 3,
		HomeID: 1, AwayID: 2, KickoffUTC: time.Date(2026, 6, 11, 20, 0, 0, 0, time.UTC),
		Status: "scheduled", Minute: 0, HomeScore: 0, AwayScore: 0,
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "stage", "group", "venueId", "homeId", "awayId", "kickoffUtc", "status", "minute", "homeScore", "awayScore"} {
		if _, ok := got[key]; !ok {
			t.Errorf("Match JSON missing key %q; got %v", key, got)
		}
	}
}

func TestTeamJSONFieldNames(t *testing.T) {
	b, _ := json.Marshal(model.Team{ID: 1, Name: "Brazil", Country: "Brazil", Group: "A", CrestURL: ""})
	var got map[string]any
	_ = json.Unmarshal(b, &got)
	for _, key := range []string{"id", "name", "country", "group", "crestUrl"} {
		if _, ok := got[key]; !ok {
			t.Errorf("Team JSON missing key %q", key)
		}
	}
}
