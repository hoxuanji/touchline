package provider

import (
	"sort"
	"time"

	"touchline/internal/model"
)

// tournamentStart is the WC2026 opener date (UTC). The generated skeleton spreads
// matches deterministically across the group stage and a knockout bracket.
var tournamentStart = time.Date(2026, 6, 11, 19, 0, 0, 0, time.UTC)

// generateFixtures builds a structural 104-match skeleton: a single round-robin
// (6 matches) within each group of 4, followed by a 32-match knockout bracket.
// Output is deterministic and stable across runs (sorted, fixed scheduling rule).
func generateFixtures(teams []model.Team) []model.Match {
	groups := map[string][]model.Team{}
	var groupNames []string
	for _, t := range teams {
		if _, ok := groups[t.Group]; !ok {
			groupNames = append(groupNames, t.Group)
		}
		groups[t.Group] = append(groups[t.Group], t)
	}
	sort.Strings(groupNames)

	var matches []model.Match
	id := 1
	venue := 1
	nextVenue := func() int {
		v := venue
		venue++
		if venue > 16 {
			venue = 1
		}
		return v
	}
	day := 0
	pairs := [][2]int{{0, 1}, {2, 3}, {0, 2}, {1, 3}, {0, 3}, {1, 2}}
	for _, g := range groupNames {
		gt := groups[g]
		if len(gt) != 4 {
			continue
		}
		for _, p := range pairs {
			matches = append(matches, model.Match{
				ID:         id,
				Stage:      "group",
				Group:      g,
				VenueID:    nextVenue(),
				HomeID:     gt[p[0]].ID,
				AwayID:     gt[p[1]].ID,
				KickoffUTC: tournamentStart.AddDate(0, 0, day).Add(time.Duration(id%3) * 3 * time.Hour),
				Status:     "scheduled",
			})
			id++
			day = (day + 1) % 24
		}
	}

	knockout := []struct {
		stage string
		n     int
	}{
		{"r32", 16}, {"r16", 8}, {"qf", 4}, {"sf", 2}, {"third", 1}, {"final", 1},
	}
	koDay := 25
	for _, k := range knockout {
		for i := 0; i < k.n; i++ {
			matches = append(matches, model.Match{
				ID:         id,
				Stage:      k.stage,
				Group:      "",
				VenueID:    nextVenue(),
				HomeID:     0,
				AwayID:     0,
				KickoffUTC: tournamentStart.AddDate(0, 0, koDay).Add(time.Duration(i%2) * 4 * time.Hour),
				Status:     "scheduled",
			})
			id++
		}
		koDay += 3
	}
	return matches
}
