package provider

import (
	"context"
	"encoding/json"

	snapshotdata "touchline/data/snapshot"
	"touchline/internal/model"
)

// Snapshot serves the bundled structural WC2026 data. Fixtures and standings are
// generated deterministically from the team/group structure.
type Snapshot struct {
	teams  []model.Team
	venues []model.Venue
}

func NewSnapshot() *Snapshot {
	s := &Snapshot{}
	_ = json.Unmarshal(snapshotdata.VenuesJSON, &s.venues)
	var doc struct {
		Teams []model.Team `json:"teams"`
	}
	_ = json.Unmarshal(snapshotdata.TeamsJSON, &doc)
	s.teams = doc.Teams
	return s
}

func (s *Snapshot) Teams(ctx context.Context) ([]model.Team, error)   { return s.teams, nil }
func (s *Snapshot) Venues(ctx context.Context) ([]model.Venue, error) { return s.venues, nil }

func (s *Snapshot) Fixtures(ctx context.Context) ([]model.Match, error) {
	return generateFixtures(s.teams), nil
}

func (s *Snapshot) Standings(ctx context.Context) ([]model.Standing, error) {
	out := make([]model.Standing, 0, len(s.teams))
	for _, t := range s.teams {
		out = append(out, model.Standing{Group: t.Group, TeamID: t.ID, Form: ""})
	}
	return out, nil
}

var _ Provider = (*Snapshot)(nil)
