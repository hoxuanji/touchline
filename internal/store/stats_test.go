package store_test

import (
	"testing"

	"touchline/internal/model"
)

func TestRecomputeStandingsFromFinishedMatches(t *testing.T) {
	s := newTestStore(t)
	_ = s.UpsertTeams([]model.Team{
		{ID: 1, Group: "A"}, {ID: 2, Group: "A"}, {ID: 3, Group: "A"}, {ID: 4, Group: "A"},
	})
	_ = s.UpsertMatches([]model.Match{
		{ID: 1, Stage: "group", Group: "A", HomeID: 1, AwayID: 2, Status: "finished", HomeScore: 2, AwayScore: 0},
		{ID: 2, Stage: "group", Group: "A", HomeID: 3, AwayID: 4, Status: "scheduled"},
	})
	if err := s.RecomputeStandings(); err != nil {
		t.Fatal(err)
	}
	rows, _ := s.Standings()
	byTeam := map[int]model.Standing{}
	for _, r := range rows {
		byTeam[r.TeamID] = r
	}
	if byTeam[1].Pts != 3 || byTeam[1].Won != 1 || byTeam[1].GF != 2 {
		t.Fatalf("team1 standing = %+v", byTeam[1])
	}
	if byTeam[2].Pts != 0 || byTeam[2].Lost != 1 || byTeam[2].GA != 2 {
		t.Fatalf("team2 standing = %+v", byTeam[2])
	}
	if byTeam[3].Played != 0 {
		t.Fatalf("team3 should have 0 played, got %+v", byTeam[3])
	}
}

func TestTopScorersCountsGoals(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 1, Type: "goal", PlayerID: 0, TeamID: 1})
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 1, Type: "goal", PlayerID: 0, TeamID: 1})
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 1, Type: "card", TeamID: 2})
	rows, err := s.TopScorersByTeam()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 || rows[0].TeamID != 1 || rows[0].Goals != 2 {
		t.Fatalf("top scorers = %+v", rows)
	}
}
