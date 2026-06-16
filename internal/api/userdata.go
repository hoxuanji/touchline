package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func RegisterUserdata(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/follows", func(w http.ResponseWriter, r *http.Request) {
		ids, err := deps.Store.Follows()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ids)
	})
	mux.HandleFunc("POST /api/follows/{teamId}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("teamId"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := deps.Store.AddFollow(id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE /api/follows/{teamId}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("teamId"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := deps.Store.RemoveFollow(id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/prefs/{key}", func(w http.ResponseWriter, r *http.Request) {
		v, err := deps.Store.GetPref(r.PathValue("key"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"key": r.PathValue("key"), "value": v})
	})
	mux.HandleFunc("PUT /api/prefs/{key}", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if err := deps.Store.SetPref(r.PathValue("key"), body.Value); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/reminders", func(w http.ResponseWriter, r *http.Request) {
		rs, err := deps.Store.Reminders()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, rs)
	})
	mux.HandleFunc("POST /api/reminders", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			MatchID     int `json:"matchId"`
			LeadMinutes int `json:"leadMinutes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if body.LeadMinutes == 0 {
			body.LeadMinutes = 30
		}
		id, err := deps.Store.AddReminder(body.MatchID, body.LeadMinutes)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]int{"id": id})
	})
	mux.HandleFunc("DELETE /api/reminders/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := deps.Store.DeleteReminder(id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
