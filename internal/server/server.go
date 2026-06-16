// Package server wires the HTTP routes for Touchline.
package server

import (
	"net/http"

	"touchline/internal/api"
	"touchline/internal/hot"
	"touchline/internal/sse"
	"touchline/internal/store"
	"touchline/web"
)

// Deps are the runtime dependencies the handler needs.
type Deps struct {
	Store *store.Store
	Hot   *hot.Store
	SSE   *sse.Hub
}

// Handler builds the application's HTTP handler: health, REST API, SSE stream,
// and the embedded SPA.
func Handler(deps Deps) (http.Handler, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	api.Register(mux, api.Deps{Store: deps.Store, Hot: deps.Hot})

	mux.HandleFunc("GET /api/stream", deps.SSE.ServeHTTP)

	mux.Handle("/", http.FileServerFS(web.Assets))

	return mux, nil
}
