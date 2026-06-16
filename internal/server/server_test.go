package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"touchline/internal/server"
)

func TestHealthz(t *testing.T) {
	h, err := server.Handler()
	if err != nil {
		t.Fatalf("Handler() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("body = %q, want it to contain %q", body, `"status":"ok"`)
	}
}
