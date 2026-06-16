// Package server wires the HTTP routes for Touchline.
package server

import (
	"encoding/json"
	"net/http"
	"os"

	"touchline/internal/api"
	"touchline/internal/hot"
	"touchline/internal/provider"
	"touchline/internal/sse"
	"touchline/internal/store"
	"touchline/web"
)

// Deps are the runtime dependencies the handler needs.
type Deps struct {
	Store  *store.Store
	Hot    *hot.Store
	SSE    *sse.Hub
	Source string // "sim" | "real"
}

// Handler builds the application's HTTP handler.
func Handler(deps Deps) (http.Handler, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	source := deps.Source
	if source == "" {
		source = "sim"
	}
	mux.HandleFunc("GET /api/meta", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"source":"` + source + `"}`))
	})

	// Diagnostic: calls the real API status endpoint so we can see
	// connectivity + quota from inside the running container.
	mux.HandleFunc("GET /api/apistatus", func(w http.ResponseWriter, r *http.Request) {
		key := os.Getenv("API_FOOTBALL_KEY")
		if key == "" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"error":"API_FOOTBALL_KEY not set"}`))
			return
		}
		p := provider.NewAPIFootball(key)
		result, err := p.Status(r.Context())
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	})

	api.Register(mux, api.Deps{Store: deps.Store, Hot: deps.Hot})

	mux.HandleFunc("GET /api/stream", deps.SSE.ServeHTTP)

	mux.Handle("/", http.FileServerFS(web.Assets))

	return mux, nil
}
