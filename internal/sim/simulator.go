// Package sim is the match simulator: it advances live matches, emits events,
// persists changes, and broadcasts SSE deltas. It is the LIVE_SOURCE=sim source.
package sim

import (
	"context"
	"math/rand"
	"time"

	"touchline/internal/hot"
	"touchline/internal/model"
	"touchline/internal/sse"
	"touchline/internal/store"
)

// Broadcaster is satisfied by *sse.Hub.
type Broadcaster interface{ Broadcast(sse.Message) }

const fullTime = 90

type Simulator struct {
	store *store.Store
	hot   *hot.Store
	bc    Broadcaster
	rng   *rand.Rand
	now   func() time.Time
}

func New(st *store.Store, h *hot.Store, bc Broadcaster, rng *rand.Rand, now func() time.Time) *Simulator {
	return &Simulator{store: st, hot: h, bc: bc, rng: rng, now: now}
}

// Step advances every live match by one simulated tick (~3 match-minutes),
// possibly emits an event, persists, and broadcasts. Finished matches are skipped.
func (s *Simulator) Step() {
	for _, m := range s.hot.Matches() {
		if m.Status != "live" && m.Status != "ht" {
			continue
		}
		m.Status = "live"
		m.Minute += 3
		roll := s.rng.Float64()
		if roll < 0.12 {
			side, teamID := s.pickSide(m)
			if side == 0 {
				m.HomeScore++
			} else {
				m.AwayScore++
			}
			s.emitEvent(m, "goal", teamID)
		} else if roll < 0.20 {
			_, teamID := s.pickSide(m)
			s.emitEvent(m, "card", teamID)
		}
		if m.Minute >= fullTime {
			m.Minute = fullTime
			m.Status = "finished"
		}
		s.hot.PatchMatch(m)
		_ = s.store.UpsertMatches([]model.Match{m})
		s.bc.Broadcast(sse.Message{Type: "match.update", Data: m})
	}
}

// pickSide returns (0=home / 1=away, teamID).
func (s *Simulator) pickSide(m model.Match) (int, int) {
	if s.rng.Intn(2) == 0 {
		return 0, m.HomeID
	}
	return 1, m.AwayID
}

func (s *Simulator) emitEvent(m model.Match, typ string, teamID int) {
	e := model.MatchEvent{MatchID: m.ID, Minute: m.Minute, Type: typ, TeamID: teamID}
	if id, err := s.store.AddEvent(e); err == nil {
		e.ID = id
	}
	s.bc.Broadcast(sse.Message{Type: "match.event", Data: e})
}

// Run advances the simulation on a ticker until ctx is cancelled.
func (s *Simulator) Run(ctx context.Context, tick time.Duration) {
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.Step()
		}
	}
}
