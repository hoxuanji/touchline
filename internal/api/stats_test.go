package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"touchline/internal/api"
	"touchline/internal/model"
	"touchline/internal/store"
)

func TestGetTopScorers(t *testing.T) {
	s, _ := store.Open(":memory:")
	t.Cleanup(func() { s.Close() })
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 1, Type: "goal", TeamID: 5})
	mux := http.NewServeMux()
	api.RegisterStats(mux, api.Deps{Store: s})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stats/scorers", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"teamId":5`) {
		t.Fatalf("scorers: code=%d body=%s", rec.Code, rec.Body.String())
	}
}
