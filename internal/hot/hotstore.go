// Package hot is the in-memory current-state cache for fast reads and SSE
// fan-out. It is hydrated from the SQLite store on boot and patched as live
// updates arrive. All methods are safe for concurrent use.
package hot

import (
	"sort"
	"sync"

	"touchline/internal/model"
)

type Store struct {
	mu        sync.RWMutex
	matches   map[int]model.Match
	standings []model.Standing
}

func New() *Store {
	return &Store{matches: map[int]model.Match{}}
}

// Hydrate replaces the in-memory state with the given snapshot.
func (s *Store) Hydrate(matches []model.Match, standings []model.Standing) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.matches = make(map[int]model.Match, len(matches))
	for _, m := range matches {
		s.matches[m.ID] = m
	}
	s.standings = append([]model.Standing(nil), standings...)
}

// Matches returns all matches sorted by kickoff then id (stable for the API).
func (s *Store) Matches() []model.Match {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Match, 0, len(s.matches))
	for _, m := range s.matches {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].KickoffUTC.Equal(out[j].KickoffUTC) {
			return out[i].ID < out[j].ID
		}
		return out[i].KickoffUTC.Before(out[j].KickoffUTC)
	})
	return out
}

func (s *Store) Match(id int) (model.Match, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.matches[id]
	return m, ok
}

// PatchMatch replaces the stored match with the same ID. Returns false if no
// match with that ID exists.
func (s *Store) PatchMatch(m model.Match) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.matches[m.ID]; !ok {
		return false
	}
	s.matches[m.ID] = m
	return true
}

func (s *Store) Standings() []model.Standing {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]model.Standing(nil), s.standings...)
}

func (s *Store) SetStandings(rows []model.Standing) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.standings = append([]model.Standing(nil), rows...)
}
