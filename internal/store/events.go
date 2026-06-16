package store

import "touchline/internal/model"

// AddEvent inserts a match event and returns its new autoincrement id.
func (s *Store) AddEvent(e model.MatchEvent) (int, error) {
	res, err := s.db.Exec(
		`INSERT INTO match_events (match_id,minute,type,team_id,player_id,detail) VALUES (?,?,?,?,?,?)`,
		e.MatchID, e.Minute, e.Type, e.TeamID, e.PlayerID, e.Detail)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// EventsByMatch returns a match's events ordered by minute then id.
func (s *Store) EventsByMatch(matchID int) ([]model.MatchEvent, error) {
	rows, err := s.db.Query(
		`SELECT id,match_id,minute,type,team_id,player_id,detail FROM match_events WHERE match_id = ? ORDER BY minute, id`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.MatchEvent
	for rows.Next() {
		var e model.MatchEvent
		if err := rows.Scan(&e.ID, &e.MatchID, &e.Minute, &e.Type, &e.TeamID, &e.PlayerID, &e.Detail); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
