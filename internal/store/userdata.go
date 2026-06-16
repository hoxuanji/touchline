package store

import (
	"database/sql"
	"errors"

	"touchline/internal/model"
)

func (s *Store) SetPref(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO prefs (key,value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

// GetPref returns the value, or "" if the key is absent.
func (s *Store) GetPref(key string) (string, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM prefs WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return v, nil
}

func (s *Store) AddReminder(matchID, leadMinutes int) (int, error) {
	res, err := s.db.Exec(`INSERT INTO reminders (match_id,lead_minutes,fired) VALUES (?,?,0)`, matchID, leadMinutes)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) Reminders() ([]model.Reminder, error) {
	rows, err := s.db.Query(`SELECT id,match_id,lead_minutes,fired FROM reminders ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Reminder
	for rows.Next() {
		var r model.Reminder
		var fired int
		if err := rows.Scan(&r.ID, &r.MatchID, &r.LeadMinutes, &fired); err != nil {
			return nil, err
		}
		r.Fired = fired != 0
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DeleteReminder(id int) error {
	_, err := s.db.Exec(`DELETE FROM reminders WHERE id = ?`, id)
	return err
}
