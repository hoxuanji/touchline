package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"touchline/internal/api"
	"touchline/internal/hot"
	"touchline/internal/store"
)

func TestRegisterWiresSliceRoutes(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	h := hot.New()
	h.Hydrate(nil, nil)
	mux := http.NewServeMux()
	api.Register(mux, api.Deps{Store: s, Hot: h})

	// each slice route should be reachable (not 404 from an unregistered pattern)
	for _, path := range []string{"/api/follows", "/api/stats/scorers", "/api/reminders", "/api/notes?subjectType=team&subjectId=1"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code == http.StatusNotFound {
			t.Errorf("route %s not registered (404)", path)
		}
	}
}
