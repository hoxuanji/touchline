// Command touchline serves the World Cup 2026 companion app from a single binary.
package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"touchline/internal/budget"
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
	} else if liveSource == "real" {
		key := os.Getenv("API_FOOTBALL_KEY")
		if key == "" {
			log.Print("LIVE_SOURCE=real but API_FOOTBALL_KEY is empty; serving snapshot only")
		} else {
			dailyLimit, _ := strconv.Atoi(os.Getenv("DAILY_REQUEST_BUDGET"))
			if dailyLimit <= 0 {
				dailyLimit = 50
			}
			b := budget.New(dailyLimit, time.Now)
			realProvider := provider.NewAPIFootball(key)
			log.Printf("LIVE_SOURCE=real: APIFootball provider active, budget %d req/day (remaining: %d)",
				dailyLimit, b.Remaining())
			// reference-data refresh under budget (runs once at boot):
			if b.Allow() {
				if teams, err := realProvider.Teams(context.Background()); err == nil {
					_ = st.UpsertTeams(teams)
					log.Printf("refreshed %d teams from APIFootball", len(teams))
				} else {
					log.Printf("team refresh failed (snapshot remains): %v", err)
				}
			}
		}
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
