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

func setupMatches(t *testing.T) (http.Handler, *store.Store) {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	_ = s.UpsertTeams([]model.Team{{ID: 1, Name: "Brazil"}, {ID: 2, Name: "Spain"}})
	_ = s.UpsertMatches([]model.Match{{ID: 10, Stage: "group", Group: "F", HomeID: 1, AwayID: 2, Status: "live", Minute: 30, HomeScore: 1}})
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 10, Minute: 12, Type: "goal", TeamID: 1})
	h := hot.New()
	m, _ := s.Matches()
	h.Hydrate(m, nil)
	mux := http.NewServeMux()
	api.RegisterMatches(mux, api.Deps{Store: s, Hot: h})
	return mux, s
}

func TestGetMatchDetail(t *testing.T) {
	h, _ := setupMatches(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/matches/10", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got struct {
		Match    model.Match        `json:"match"`
		Events   []model.MatchEvent `json:"events"`
		HomeName string             `json:"homeName"`
		AwayName string             `json:"awayName"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Match.ID != 10 || got.Match.HomeScore != 1 {
		t.Fatalf("match = %+v", got.Match)
	}
	if len(got.Events) != 1 || got.Events[0].Type != "goal" {
		t.Fatalf("events = %+v", got.Events)
	}
	if got.HomeName != "Brazil" || got.AwayName != "Spain" {
		t.Fatalf("names = %q vs %q", got.HomeName, got.AwayName)
	}
}

func TestGetMatchDetailNotFound(t *testing.T) {
	h, _ := setupMatches(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/matches/999", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
