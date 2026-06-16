// Command touchline serves the World Cup 2026 companion app from a single binary.
package main

import (
	"log"
	"net/http"
	"os"

	"touchline/internal/server"
)

func main() {
	h, err := server.Handler()
	if err != nil {
		log.Fatalf("build handler: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	log.Printf("touchline listening on %s", addr)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatal(err)
	}
}
