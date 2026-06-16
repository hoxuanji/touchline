// Package provider abstracts data sources behind one interface. The snapshot
// implementation seeds/falls back from bundled JSON; real and simulated sources
// are added in later phases.
package provider

import (
	"context"

	"touchline/internal/model"
)

type Provider interface {
	Teams(ctx context.Context) ([]model.Team, error)
	Venues(ctx context.Context) ([]model.Venue, error)
	Fixtures(ctx context.Context) ([]model.Match, error)
	Standings(ctx context.Context) ([]model.Standing, error)
}
