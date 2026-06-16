// Package server wires the HTTP routes for Touchline.
package server

import (
	"net/http"

	"touchline/web"
)

// Handler builds the application's HTTP handler: a health endpoint plus the
// embedded SPA. Later phases extend this with API and SSE routes.
func Handler() (http.Handler, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.Handle("/", http.FileServerFS(web.Assets))

	return mux, nil
}
