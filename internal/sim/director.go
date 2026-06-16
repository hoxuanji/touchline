package sim

import (
	"touchline/internal/hot"
	"touchline/internal/model"
	"touchline/internal/store"
)

// PromoteLive flips up to n scheduled matches to "live" (in both the hot store
// and SQLite) so the simulated live experience is immediately visible in sim
// mode. Returns how many were promoted.
func PromoteLive(st *store.Store, h *hot.Store, n int) int {
	promoted := 0
	for _, m := range h.Matches() {
		if promoted >= n {
			break
		}
		if m.Status == "scheduled" {
			m.Status = "live"
			m.Minute = 0
			h.PatchMatch(m)
			_ = st.UpsertMatches([]model.Match{m})
			promoted++
		}
	}
	return promoted
}
