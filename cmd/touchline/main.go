// Command touchline serves the World Cup 2026 companion app from a single binary.
package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"touchline/internal/hot"
	"touchline/internal/provider"
	"touchline/internal/server"
	"touchline/internal/sim"
	"touchline/internal/sse"
	"touchline/internal/store"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "touchline.db"
	}

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	seeded, err := store.Seed(context.Background(), st, provider.NewSnapshot())
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	if seeded {
		log.Print("seeded database from bundled WC2026 snapshot")
	}

	hotStore := hot.New()
	matches, err := st.Matches()
	if err != nil {
		log.Fatalf("load matches: %v", err)
	}
	standings, err := st.Standings()
	if err != nil {
		log.Fatalf("load standings: %v", err)
	}
	hotStore.Hydrate(matches, standings)

	hub := sse.NewHub()

	// Live source. LIVE_SOURCE=sim (default) runs the built-in simulator;
	// "real" polling is added in Phase 3 with no change to the rest.
	liveSource := os.Getenv("LIVE_SOURCE")
	if liveSource == "" {
		liveSource = "sim"
	}
	if liveSource == "sim" {
		promoted := sim.PromoteLive(st, hotStore, 4)
		log.Printf("sim mode: promoted %d matches to live", promoted)
		simulator := sim.New(st, hotStore, hub, rand.New(rand.NewSource(time.Now().UnixNano())), time.Now)
		go simulator.Run(context.Background(), 5*time.Second)
	}

	h, err := server.Handler(server.Deps{Store: st, Hot: hotStore, SSE: hub})
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
