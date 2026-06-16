package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"touchline/internal/model"
)

func RegisterNotes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/notes", func(w http.ResponseWriter, r *http.Request) {
		st := r.URL.Query().Get("subjectType")
		sid, _ := strconv.Atoi(r.URL.Query().Get("subjectId"))
		notes, err := deps.Store.NotesBySubject(st, sid)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, notes)
	})
	mux.HandleFunc("POST /api/notes", func(w http.ResponseWriter, r *http.Request) {
		var n model.Note
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil || n.SubjectType == "" {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		id, err := deps.Store.AddNote(n)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]int{"id": id})
	})
	mux.HandleFunc("DELETE /api/notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := deps.Store.DeleteNote(id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
