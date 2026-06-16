package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"touchline/internal/api"
	"touchline/internal/hot"
	"touchline/internal/model"
	"touchline/internal/store"
)

func setup(t *testing.T) http.Handler {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	_ = s.UpsertTeams([]model.Team{{ID: 1, Name: "Brazil", Group: "F"}})
	_ = s.UpsertVenues([]model.Venue{{ID: 1, Name: "Azteca"}})
	_ = s.UpsertMatches([]model.Match{{ID: 1, Stage: "group", Group: "F", HomeID: 1, AwayID: 1, Status: "scheduled"}})
	h := hot.New()
	matches, _ := s.Matches()
	h.Hydrate(matches, nil)
	mux := http.NewServeMux()
	api.Register(mux, api.Deps{Store: s, Hot: h})
	return mux
}

func TestGetFixtures(t *testing.T) {
	h := setup(t)
	req := httptest.NewRequest(http.MethodGet, "/api/fixtures", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var matches []model.Match
	if err := json.Unmarshal(rec.Body.Bytes(), &matches); err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].ID != 1 {
		t.Fatalf("fixtures = %v", matches)
	}
}

func TestGetTeamsAndVenues(t *testing.T) {
	h := setup(t)
	for _, path := range []string{"/api/teams", "/api/venues"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s status = %d", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("%s content-type = %q", path, ct)
		}
	}
}

func TestGetStandingsEmptyReturnsArrayNotNull(t *testing.T) {
	h := setup(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/standings", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	// hot store hydrated with nil standings; must serialize as [] (or at least valid JSON array), never bare null
	if body == "null\n" || body == "null" {
		t.Fatalf("standings empty body = %q, want a JSON array", body)
	}
}
