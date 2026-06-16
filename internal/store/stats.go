package store

import "touchline/internal/model"

// RecomputeStandings derives the group standings table from all finished group
// matches and writes it back. Idempotent: recomputes from scratch each call.
func (s *Store) RecomputeStandings() error {
	teams, err := s.Teams()
	if err != nil {
		return err
	}
	matches, err := s.Matches()
	if err != nil {
		return err
	}
	tab := map[int]*model.Standing{}
	for _, t := range teams {
		tab[t.ID] = &model.Standing{Group: t.Group, TeamID: t.ID}
	}
	for _, m := range matches {
		if m.Stage != "group" || m.Status != "finished" {
			continue
		}
		h, a := tab[m.HomeID], tab[m.AwayID]
		if h == nil || a == nil {
			continue
		}
		h.Played++
		a.Played++
		h.GF += m.HomeScore
		h.GA += m.AwayScore
		a.GF += m.AwayScore
		a.GA += m.HomeScore
		switch {
		case m.HomeScore > m.AwayScore:
			h.Won++
			h.Pts += 3
			a.Lost++
		case m.HomeScore < m.AwayScore:
			a.Won++
			a.Pts += 3
			h.Lost++
		default:
			h.Drawn++
			a.Drawn++
			h.Pts++
			a.Pts++
		}
	}
	rows := make([]model.Standing, 0, len(tab))
	for _, st := range tab {
		rows = append(rows, *st)
	}
	return s.UpsertStandings(rows)
}

// ScorerRow aggregates goals; PlayerID 0 means team-attributed (no player data yet).
type ScorerRow struct {
	TeamID   int `json:"teamId"`
	PlayerID int `json:"playerId"`
	Goals    int `json:"goals"`
}

// TopScorersByTeam counts goal events grouped by team, ordered by goals desc.
func (s *Store) TopScorersByTeam() ([]ScorerRow, error) {
	rows, err := s.db.Query(`SELECT team_id, COUNT(*) AS goals FROM match_events WHERE type = 'goal' GROUP BY team_id ORDER BY goals DESC, team_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScorerRow
	for rows.Next() {
		var r ScorerRow
		if err := rows.Scan(&r.TeamID, &r.Goals); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
