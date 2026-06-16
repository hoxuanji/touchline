package store

import (
	"context"

	"touchline/internal/provider"
)

// Seed populates an empty store from the provider. Returns true if it seeded,
// false if the store already had data (no-op). Idempotent and safe on every boot.
func Seed(ctx context.Context, s *Store, p provider.Provider) (bool, error) {
	n, err := s.CountTeams()
	if err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}

	teams, err := p.Teams(ctx)
	if err != nil {
		return false, err
	}
	venues, err := p.Venues(ctx)
	if err != nil {
		return false, err
	}
	fixtures, err := p.Fixtures(ctx)
	if err != nil {
		return false, err
	}
	standings, err := p.Standings(ctx)
	if err != nil {
		return false, err
	}

	if err := s.UpsertTeams(teams); err != nil {
		return false, err
	}
	if err := s.UpsertVenues(venues); err != nil {
		return false, err
	}
	if err := s.UpsertMatches(fixtures); err != nil {
		return false, err
	}
	if err := s.UpsertStandings(standings); err != nil {
		return false, err
	}
	return true, nil
}
