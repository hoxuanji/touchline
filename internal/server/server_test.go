package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"touchline/internal/hot"
	"touchline/internal/model"
	"touchline/internal/server"
	"touchline/internal/sse"
	"touchline/internal/store"
)

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	_ = s.UpsertMatches([]model.Match{{ID: 1, Stage: "group", Group: "A", Status: "scheduled"}})
	h := hot.New()
	matches, _ := s.Matches()
	h.Hydrate(matches, nil)
	handler, err := server.Handler(server.Deps{Store: s, Hot: h, SSE: sse.NewHub(), Source: "sim"})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestHealthz(t *testing.T) {
	h := newHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("healthz: code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestFixturesRouteWired(t *testing.T) {
	h := newHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/fixtures", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("fixtures route status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"id":1`) {
		t.Fatalf("fixtures body = %s", rec.Body.String())
	}
}

func TestMetaRoute(t *testing.T) {
	h := newHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/meta", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"source":"sim"`) {
		t.Fatalf("meta: code=%d body=%s", rec.Code, rec.Body.String())
	}
}
