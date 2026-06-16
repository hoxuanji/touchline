package api

import "net/http"

func RegisterStats(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/stats/scorers", func(w http.ResponseWriter, r *http.Request) {
		rows, err := deps.Store.TopScorersByTeam()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, rows)
	})
}
