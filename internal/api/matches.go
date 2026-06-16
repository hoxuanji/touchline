package api

import (
	"net/http"
	"strconv"

	"touchline/internal/model"
)

// RegisterMatches adds the match-detail route.
func RegisterMatches(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/matches/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		m, ok := deps.Hot.Match(id)
		if !ok {
			http.Error(w, "match not found", http.StatusNotFound)
			return
		}
		events, err := deps.Store.EventsByMatch(id)
		if err != nil {
			writeError(w, err)
			return
		}
		teams, err := deps.Store.Teams()
		if err != nil {
			writeError(w, err)
			return
		}
		names := map[int]string{}
		for _, t := range teams {
			names[t.ID] = t.Name
		}
		writeJSON(w, http.StatusOK, struct {
			Match    model.Match        `json:"match"`
			Events   []model.MatchEvent `json:"events"`
			HomeName string             `json:"homeName"`
			AwayName string             `json:"awayName"`
		}{Match: m, Events: events, HomeName: names[m.HomeID], AwayName: names[m.AwayID]})
	})
}
