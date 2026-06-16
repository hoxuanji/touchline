package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"touchline/internal/api"
	"touchline/internal/store"
)

func setupUserdata(t *testing.T) http.Handler {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	mux := http.NewServeMux()
	api.RegisterUserdata(mux, api.Deps{Store: s})
	return mux
}

func TestFollowLifecycle(t *testing.T) {
	h := setupUserdata(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/follows/7", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST follow status = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/follows", nil))
	if !strings.Contains(rec.Body.String(), "7") {
		t.Fatalf("follows list = %s", rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/follows/7", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE follow status = %d", rec.Code)
	}
}

func TestPrefsEndpoint(t *testing.T) {
	h := setupUserdata(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/prefs/timezone", strings.NewReader(`{"value":"Europe/London"}`)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PUT pref status = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/prefs/timezone", nil))
	if !strings.Contains(rec.Body.String(), "Europe/London") {
		t.Fatalf("GET pref = %s", rec.Body.String())
	}
}
