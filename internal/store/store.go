// Package store is the SQLite persistence layer (pure-Go modernc driver) plus
// typed query/command methods over the domain model.
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"touchline/internal/model"
)

type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at dsn and runs migrations.
// Use ":memory:" for tests. A file path uses WAL for better concurrency.
func Open(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) CountTeams() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM teams`).Scan(&n)
	return n, err
}

func (s *Store) UpsertTeams(teams []model.Team) error {
	return s.tx(func(tx *sql.Tx) error {
		for _, t := range teams {
			if _, err := tx.Exec(
				`INSERT INTO teams (id,name,country,group_name,crest_url) VALUES (?,?,?,?,?)
				 ON CONFLICT(id) DO UPDATE SET name=excluded.name,country=excluded.country,group_name=excluded.group_name,crest_url=excluded.crest_url`,
				t.ID, t.Name, t.Country, t.Group, t.CrestURL); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) UpsertVenues(venues []model.Venue) error {
	return s.tx(func(tx *sql.Tx) error {
		for _, v := range venues {
			if _, err := tx.Exec(
				`INSERT INTO venues (id,name,city,country,capacity) VALUES (?,?,?,?,?)
				 ON CONFLICT(id) DO UPDATE SET name=excluded.name,city=excluded.city,country=excluded.country,capacity=excluded.capacity`,
				v.ID, v.Name, v.City, v.Country, v.Capacity); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) UpsertMatches(matches []model.Match) error {
	return s.tx(func(tx *sql.Tx) error {
		for _, m := range matches {
			if _, err := tx.Exec(
				`INSERT INTO matches (id,stage,group_name,venue_id,home_id,away_id,kickoff_utc,status,minute,home_score,away_score)
				 VALUES (?,?,?,?,?,?,?,?,?,?,?)
				 ON CONFLICT(id) DO UPDATE SET stage=excluded.stage,group_name=excluded.group_name,venue_id=excluded.venue_id,
				 home_id=excluded.home_id,away_id=excluded.away_id,kickoff_utc=excluded.kickoff_utc,status=excluded.status,
				 minute=excluded.minute,home_score=excluded.home_score,away_score=excluded.away_score`,
				m.ID, m.Stage, m.Group, m.VenueID, m.HomeID, m.AwayID, m.KickoffUTC.UTC().Format(time.RFC3339),
				m.Status, m.Minute, m.HomeScore, m.AwayScore); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) UpsertStandings(rows []model.Standing) error {
	return s.tx(func(tx *sql.Tx) error {
		for _, r := range rows {
			if _, err := tx.Exec(
				`INSERT INTO standings (group_name,team_id,played,won,drawn,lost,gf,ga,pts,form)
				 VALUES (?,?,?,?,?,?,?,?,?,?)
				 ON CONFLICT(group_name,team_id) DO UPDATE SET played=excluded.played,won=excluded.won,drawn=excluded.drawn,
				 lost=excluded.lost,gf=excluded.gf,ga=excluded.ga,pts=excluded.pts,form=excluded.form`,
				r.Group, r.TeamID, r.Played, r.Won, r.Drawn, r.Lost, r.GF, r.GA, r.Pts, r.Form); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) Teams() ([]model.Team, error) {
	rows, err := s.db.Query(`SELECT id,name,country,group_name,crest_url FROM teams ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Team
	for rows.Next() {
		var t model.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Country, &t.Group, &t.CrestURL); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) Venues() ([]model.Venue, error) {
	rows, err := s.db.Query(`SELECT id,name,city,country,capacity FROM venues ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Venue
	for rows.Next() {
		var v model.Venue
		if err := rows.Scan(&v.ID, &v.Name, &v.City, &v.Country, &v.Capacity); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) Matches() ([]model.Match, error) {
	rows, err := s.db.Query(`SELECT id,stage,group_name,venue_id,home_id,away_id,kickoff_utc,status,minute,home_score,away_score FROM matches ORDER BY kickoff_utc, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Match
	for rows.Next() {
		var m model.Match
		var kickoff string
		if err := rows.Scan(&m.ID, &m.Stage, &m.Group, &m.VenueID, &m.HomeID, &m.AwayID, &kickoff, &m.Status, &m.Minute, &m.HomeScore, &m.AwayScore); err != nil {
			return nil, err
		}
		m.KickoffUTC, err = time.Parse(time.RFC3339, kickoff)
		if err != nil {
			return nil, fmt.Errorf("parse kickoff for match %d: %w", m.ID, err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) Standings() ([]model.Standing, error) {
	rows, err := s.db.Query(`SELECT group_name,team_id,played,won,drawn,lost,gf,ga,pts,form FROM standings ORDER BY group_name, pts DESC, (gf-ga) DESC, team_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Standing
	for rows.Next() {
		var r model.Standing
		if err := rows.Scan(&r.Group, &r.TeamID, &r.Played, &r.Won, &r.Drawn, &r.Lost, &r.GF, &r.GA, &r.Pts, &r.Form); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) AddFollow(teamID int) error {
	_, err := s.db.Exec(`INSERT INTO follows (team_id) VALUES (?) ON CONFLICT(team_id) DO NOTHING`, teamID)
	return err
}

func (s *Store) RemoveFollow(teamID int) error {
	_, err := s.db.Exec(`DELETE FROM follows WHERE team_id = ?`, teamID)
	return err
}

func (s *Store) Follows() ([]int, error) {
	rows, err := s.db.Query(`SELECT team_id FROM follows ORDER BY team_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) tx(fn func(*sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
