package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"touchline/internal/api"
	"touchline/internal/store"
)

func TestNotesEndpoint(t *testing.T) {
	s, _ := store.Open(":memory:")
	t.Cleanup(func() { s.Close() })
	mux := http.NewServeMux()
	api.RegisterNotes(mux, api.Deps{Store: s})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/notes", strings.NewReader(`{"subjectType":"team","subjectId":21,"body":"hi"}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST note = %d body=%s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/notes?subjectType=team&subjectId=21", nil))
	if !strings.Contains(rec.Body.String(), `"body":"hi"`) {
		t.Fatalf("GET notes = %s", rec.Body.String())
	}
}
