// Package api holds the REST handlers. Reference reads (teams/venues) come from
// the store; live-updating reads (fixtures/standings) come from the hot store.
package api

import (
	"encoding/json"
	"net/http"
	"reflect"

	"touchline/internal/hot"
	"touchline/internal/store"
)

type Deps struct {
	Store *store.Store
	Hot   *hot.Store
}

// Register attaches the /api/* routes to mux.
func Register(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/fixtures", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Hot.Matches())
	})
	mux.HandleFunc("GET /api/standings", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Hot.Standings())
	})
	mux.HandleFunc("GET /api/teams", func(w http.ResponseWriter, r *http.Request) {
		teams, err := deps.Store.Teams()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, teams)
	})
	mux.HandleFunc("GET /api/venues", func(w http.ResponseWriter, r *http.Request) {
		venues, err := deps.Store.Venues()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, venues)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		_, _ = w.Write([]byte("[]\n"))
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}
