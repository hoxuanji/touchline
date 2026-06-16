# Touchline Phase 2 — Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Make live data real and give the frontend its data/real-time plumbing — a match **simulator** that advances live matches and broadcasts SSE deltas via the hub, `match_events` persistence, and a React foundation (TanStack Query, typed API client, `useLiveStream` EventSource hook, a minimal view shell, Vite dev proxy) — so the four Phase-2 feature slices can be built independently on top.

**Architecture:** A `LIVE_SOURCE` switch selects the live source; only `sim` exists now (real is Phase 3). The simulator runs as a background goroutine: on a tick it advances each "live" match's clock, probabilistically emits events (goal/card), writes through to SQLite + the hot store, and `Broadcast`s typed SSE messages (`match.update`, `match.event`). Determinism for tests comes from an injectable clock + seeded RNG; the production loop uses a real ticker. The frontend wraps the app in a QueryClient, fetches initial state over REST, and a `useLiveStream` hook patches the query cache from SSE with auto-reconnect.

**Tech Stack:** Go stdlib (`math/rand`, `time`, `context`); React + `@tanstack/react-query` (new dep) + EventSource (browser built-in). No router dep yet (minimal view-state shell; full nav is Phase 3).

---

## Environment Note (verification posture)

Network blocked (npm works, Go modules cache-only — do NOT `go get`/`go mod tidy`). TCP bind + Docker + browsers blocked here. Go tested via `go test -race ./...` (httptest/in-memory, no socket). Frontend tested via Vitest (jsdom, no socket). The `useLiveStream` hook is tested against a **mock EventSource** injected in tests (no real network). Live SSE-over-HTTP, `LIVE_SOURCE=sim` end-to-end in a browser, and `docker compose up` are deferred to the user's runtime gate.

---

## Decisions locked for this phase

- **Simulator scope:** advances matches already in `status="live"`; a small "director" promotes a few scheduled matches to live at boot in `sim` mode so the live UX is immediately demoable (clearly a simulation). Goals/cards via seeded RNG. Minute advances on each tick.
- **Determinism:** `Simulator` takes a `*rand.Rand` and a `now func() time.Time`; `Step()` is a pure-ish advance unit that tests call directly. The background `Run(ctx, tick)` calls `Step` on a ticker.
- **SSE message types:** `match.update` (full updated `model.Match`), `match.event` (the new `model.MatchEvent`). Envelope is the existing `sse.Message{Type,Data}`.
- **Persistence:** simulator writes match updates via `store.UpsertMatches` and events via a new `store.AddEvent`; hot store patched via `PatchMatch`. Hot store stays the read source for the API.
- **Frontend dev proxy:** `vite.config.ts` proxies `/api` → `http://localhost:8080` so `npm run dev` (5173) talks to the Go server. Production is same-origin (embedded), so no proxy needed there.
- **No router lib:** `App` holds a `view` state with 4 named views; slices register their page component. Full nav/theme cohesion is Phase 3.

---

## File Structure

| Path | Responsibility |
|------|----------------|
| `internal/store/events.go` | `AddEvent(MatchEvent) (id,err)`, `EventsByMatch(matchID) ([]MatchEvent,error)`. |
| `internal/store/events_test.go` | Tests for event persistence. |
| `internal/sim/simulator.go` | `Simulator` engine: `New(...)`, `Step()` (advance one tick), `Run(ctx, tick)` background loop; emits via a `Broadcaster` interface. |
| `internal/sim/simulator_test.go` | Deterministic Step tests (seeded RNG, fixed clock). |
| `internal/sim/director.go` | `PromoteLive(hot, store, n)` — make N scheduled matches live at boot (sim demo). |
| `internal/server/server.go` | **MODIFY**: none required (SSE already wired); confirm only. |
| `cmd/touchline/main.go` | **MODIFY**: read `LIVE_SOURCE` (default `sim`); in sim mode promote live + start `Simulator.Run` goroutine broadcasting to the hub. |
| `web/package.json` | **MODIFY**: add `@tanstack/react-query`. |
| `web/vite.config.ts` | **MODIFY**: dev proxy `/api` → `:8080`. |
| `web/src/lib/types.ts` | TS mirrors of the API DTOs (Match, Team, Venue, Standing, MatchEvent). |
| `web/src/lib/api.ts` | Typed fetch client: `getFixtures/getTeams/getVenues/getStandings`. |
| `web/src/lib/useLiveStream.ts` | EventSource hook: subscribe, patch query cache, backoff reconnect; injectable EventSource for tests. |
| `web/src/lib/useLiveStream.test.ts` | Hook test with a mock EventSource. |
| `web/src/app/queryClient.ts` | Shared `QueryClient`. |
| `web/src/app/Shell.tsx` | View shell: header + 4 view tabs + live indicator; renders the active view. |
| `web/src/App.tsx` | **MODIFY**: wrap in `QueryClientProvider`, mount `useLiveStream`, render `Shell`. |
| `web/src/App.test.tsx` | **MODIFY**: update for the new shell (still asserts TOUCHLINE wordmark). |

**Frozen contracts for the slices:**

```go
// internal/sim
type Broadcaster interface{ Broadcast(sse.Message) }
type Simulator struct{ /* ... */ }
func New(st *store.Store, hot *hot.Store, b Broadcaster, rng *rand.Rand, now func() time.Time) *Simulator
func (s *Simulator) Step()                       // advance all live matches one tick
func (s *Simulator) Run(ctx context.Context, tick time.Duration)
```

```ts
// web/src/lib/useLiveStream.ts
export function useLiveStream(opts?: { makeES?: (url: string) => EventSourceLike }): { connected: boolean }
// patches the TanStack Query caches for ['fixtures'] and ['match', id] on match.update/match.event
```

---

## Task 1: match_events store methods (TDD)

**Files:** Create `internal/store/events.go`, `internal/store/events_test.go`.

- [ ] **Step 1: Failing test** — `internal/store/events_test.go`:
```go
package store_test

import (
	"testing"

	"touchline/internal/model"
)

func TestAddAndListEvents(t *testing.T) {
	s := newTestStore(t)
	id, err := s.AddEvent(model.MatchEvent{MatchID: 1, Minute: 23, Type: "goal", TeamID: 1, PlayerID: 0, Detail: "Header"})
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Fatalf("AddEvent id = %d, want > 0", id)
	}
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 1, Minute: 45, Type: "card", TeamID: 2, Detail: "Yellow"})
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 2, Minute: 10, Type: "goal", TeamID: 3})

	evs, err := s.EventsByMatch(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("EventsByMatch(1) = %d, want 2", len(evs))
	}
	if evs[0].Minute != 23 || evs[1].Minute != 45 {
		t.Fatalf("events not ordered by minute: %+v", evs)
	}
}
```

- [ ] **Step 2: Run → FAIL** — `go test ./internal/store/ -run TestAddAndListEvents`.

- [ ] **Step 3: Implement** — `internal/store/events.go`:
```go
package store

import "touchline/internal/model"

// AddEvent inserts a match event and returns its new autoincrement id.
func (s *Store) AddEvent(e model.MatchEvent) (int, error) {
	res, err := s.db.Exec(
		`INSERT INTO match_events (match_id,minute,type,team_id,player_id,detail) VALUES (?,?,?,?,?,?)`,
		e.MatchID, e.Minute, e.Type, e.TeamID, e.PlayerID, e.Detail)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// EventsByMatch returns a match's events ordered by minute then id.
func (s *Store) EventsByMatch(matchID int) ([]model.MatchEvent, error) {
	rows, err := s.db.Query(
		`SELECT id,match_id,minute,type,team_id,player_id,detail FROM match_events WHERE match_id = ? ORDER BY minute, id`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.MatchEvent
	for rows.Next() {
		var e model.MatchEvent
		if err := rows.Scan(&e.ID, &e.MatchID, &e.Minute, &e.Type, &e.TeamID, &e.PlayerID, &e.Detail); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
```

- [ ] **Step 4: Run → PASS** — `go test ./internal/store/`.
- [ ] **Step 5: Commit** — `git add internal/store/events.go internal/store/events_test.go && git commit -m "feat: match_events store methods"`

---

## Task 2: Simulator engine (TDD, deterministic)

**Files:** Create `internal/sim/simulator.go`, `internal/sim/simulator_test.go`.

- [ ] **Step 1: Failing test** — `internal/sim/simulator_test.go`:
```go
package sim_test

import (
	"math/rand"
	"testing"
	"time"

	"touchline/internal/hot"
	"touchline/internal/model"
	"touchline/internal/sim"
	"touchline/internal/sse"
	"touchline/internal/store"
)

type capture struct{ msgs []sse.Message }

func (c *capture) Broadcast(m sse.Message) { c.msgs = append(c.msgs, m) }

func newSim(t *testing.T) (*sim.Simulator, *hot.Store, *capture, *store.Store) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	_ = st.UpsertTeams([]model.Team{{ID: 1, Name: "A", Group: "A"}, {ID: 2, Name: "B", Group: "A"}})
	_ = st.UpsertMatches([]model.Match{{ID: 1, Stage: "group", Group: "A", HomeID: 1, AwayID: 2, Status: "live", Minute: 0}})
	h := hot.New()
	m, _ := st.Matches()
	h.Hydrate(m, nil)
	cap := &capture{}
	s := sim.New(st, h, cap, rand.New(rand.NewSource(1)), func() time.Time { return time.Unix(0, 0) })
	return s, h, cap, st
}

func TestStepAdvancesLiveMatchMinuteAndBroadcasts(t *testing.T) {
	s, h, cap, _ := newSim(t)
	s.Step()
	m, _ := h.Match(1)
	if m.Minute <= 0 {
		t.Fatalf("minute did not advance: %d", m.Minute)
	}
	// at least one match.update broadcast for the live match
	var sawUpdate bool
	for _, msg := range cap.msgs {
		if msg.Type == "match.update" {
			sawUpdate = true
		}
	}
	if !sawUpdate {
		t.Fatal("expected a match.update broadcast")
	}
}

func TestStepEventuallyFinishesMatch(t *testing.T) {
	s, h, _, _ := newSim(t)
	for i := 0; i < 1000 && func() bool { m, _ := h.Match(1); return m.Status != "finished" }(); i++ {
		s.Step()
	}
	m, _ := h.Match(1)
	if m.Status != "finished" {
		t.Fatalf("match never finished after many steps; status=%s minute=%d", m.Status, m.Minute)
	}
}

func TestStepIgnoresNonLiveMatches(t *testing.T) {
	s, h, cap, st := newSim(t)
	_ = st.UpsertMatches([]model.Match{{ID: 1, Status: "scheduled", HomeID: 1, AwayID: 2}})
	m, _ := st.Matches()
	h.Hydrate(m, nil)
	cap.msgs = nil
	s.Step()
	if len(cap.msgs) != 0 {
		t.Fatalf("scheduled match should not broadcast; got %d msgs", len(cap.msgs))
	}
}
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** — `internal/sim/simulator.go`:
```go
// Package sim is the match simulator: it advances live matches, emits events,
// persists changes, and broadcasts SSE deltas. It is the LIVE_SOURCE=sim source.
package sim

import (
	"context"
	"math/rand"
	"time"

	"touchline/internal/hot"
	"touchline/internal/model"
	"touchline/internal/sse"
	"touchline/internal/store"
)

// Broadcaster is satisfied by *sse.Hub.
type Broadcaster interface{ Broadcast(sse.Message) }

const fullTime = 90

type Simulator struct {
	store *store.Store
	hot   *hot.Store
	bc    Broadcaster
	rng   *rand.Rand
	now   func() time.Time
}

func New(st *store.Store, h *hot.Store, bc Broadcaster, rng *rand.Rand, now func() time.Time) *Simulator {
	return &Simulator{store: st, hot: h, bc: bc, rng: rng, now: now}
}

// Step advances every live match by one simulated tick (~3 match-minutes),
// possibly emits an event, persists, and broadcasts. Finished matches are skipped.
func (s *Simulator) Step() {
	for _, m := range s.hot.Matches() {
		if m.Status != "live" && m.Status != "ht" {
			continue
		}
		m.Status = "live"
		m.Minute += 3
		// ~12% chance of a goal this tick, ~8% a card.
		roll := s.rng.Float64()
		if roll < 0.12 {
			scorer, teamID := s.pickSide(m)
			if scorer == 0 {
				m.HomeScore++
			} else {
				m.AwayScore++
			}
			s.emitEvent(m, "goal", teamID)
		} else if roll < 0.20 {
			_, teamID := s.pickSide(m)
			s.emitEvent(m, "card", teamID)
		}
		if m.Minute >= fullTime {
			m.Minute = fullTime
			m.Status = "finished"
		}
		s.hot.PatchMatch(m)
		_ = s.store.UpsertMatches([]model.Match{m})
		s.bc.Broadcast(sse.Message{Type: "match.update", Data: m})
	}
}

// pickSide returns (0=home/1=away, teamID).
func (s *Simulator) pickSide(m model.Match) (int, int) {
	if s.rng.Intn(2) == 0 {
		return 0, m.HomeID
	}
	return 1, m.AwayID
}

func (s *Simulator) emitEvent(m model.Match, typ string, teamID int) {
	e := model.MatchEvent{MatchID: m.ID, Minute: m.Minute, Type: typ, TeamID: teamID}
	if id, err := s.store.AddEvent(e); err == nil {
		e.ID = id
	}
	s.bc.Broadcast(sse.Message{Type: "match.event", Data: e})
}

// Run advances the simulation on a ticker until ctx is cancelled.
func (s *Simulator) Run(ctx context.Context, tick time.Duration) {
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.Step()
		}
	}
}
```

- [ ] **Step 4: Run → PASS (with -race)** — `go test -race ./internal/sim/`.
- [ ] **Step 5: Commit** — `git add internal/sim/simulator.go internal/sim/simulator_test.go && git commit -m "feat: deterministic match simulator with SSE broadcasts"`

---

## Task 3: Director + boot wiring (sim live loop)

**Files:** Create `internal/sim/director.go`, `internal/sim/director_test.go`; Modify `cmd/touchline/main.go`.

- [ ] **Step 1: Failing test** — `internal/sim/director_test.go`:
```go
package sim_test

import (
	"testing"

	"touchline/internal/hot"
	"touchline/internal/model"
	"touchline/internal/sim"
	"touchline/internal/store"
)

func TestPromoteLiveMakesNScheduledMatchesLive(t *testing.T) {
	st, _ := store.Open(":memory:")
	defer st.Close()
	_ = st.UpsertMatches([]model.Match{
		{ID: 1, Stage: "group", Status: "scheduled"},
		{ID: 2, Stage: "group", Status: "scheduled"},
		{ID: 3, Stage: "group", Status: "scheduled"},
	})
	h := hot.New()
	m, _ := st.Matches()
	h.Hydrate(m, nil)

	n := sim.PromoteLive(st, h, 2)
	if n != 2 {
		t.Fatalf("PromoteLive returned %d, want 2", n)
	}
	live := 0
	for _, mm := range h.Matches() {
		if mm.Status == "live" {
			live++
		}
	}
	if live != 2 {
		t.Fatalf("live matches = %d, want 2", live)
	}
}
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** — `internal/sim/director.go`:
```go
package sim

import (
	"touchline/internal/hot"
	"touchline/internal/store"
)

// PromoteLive flips up to n scheduled matches to "live" (in both the hot store
// and SQLite) so the simulated live experience is immediately visible in sim
// mode. Returns how many were promoted.
func PromoteLive(st *store.Store, h *hot.Store, n int) int {
	promoted := 0
	for _, m := range h.Matches() {
		if promoted >= n {
			break
		}
		if m.Status == "scheduled" {
			m.Status = "live"
			m.Minute = 0
			h.PatchMatch(m)
			_ = st.UpsertMatches([]model.Match{m})
			promoted++
		}
	}
	return promoted
}
```

- [ ] **Step 4: Run → PASS.**

- [ ] **Step 5: Wire into `cmd/touchline/main.go`** — after `hub := sse.NewHub()` and before building the handler, add the sim live source. Insert these imports (`math/rand`, `time`, `touchline/internal/sim`) and this block:
```go
	// Live source. LIVE_SOURCE=sim (default) runs the built-in simulator;
	// "real" polling is added in Phase 3 with no code change to the rest.
	liveSource := os.Getenv("LIVE_SOURCE")
	if liveSource == "" {
		liveSource = "sim"
	}
	if liveSource == "sim" {
		n := sim.PromoteLive(st, hotStore, 4)
		log.Printf("sim mode: promoted %d matches to live", n)
		simulator := sim.New(st, hotStore, hub, rand.New(rand.NewSource(time.Now().UnixNano())), time.Now)
		go simulator.Run(context.Background(), 5*time.Second)
	}
```
(Place after hot-store hydration so live matches exist in the hot store. `context` is already imported; add `math/rand` and `time` and `touchline/internal/sim`. NOTE: `time.Now()` is the real wall clock here — fine for production; tests use the injected clock.)

- [ ] **Step 6: Verify build + full suite** — `go build ./... && go vet ./... && go test -race ./...` → all pass.
- [ ] **Step 7: Commit** — `git add internal/sim cmd/touchline && git commit -m "feat: sim director + boot live loop (LIVE_SOURCE=sim)"`

---

## Task 4: Frontend data foundation — QueryClient, types, API client

**Files:** Modify `web/package.json`; Create `web/src/lib/types.ts`, `web/src/lib/api.ts`, `web/src/app/queryClient.ts`.

- [ ] **Step 1: Add the dependency** — in `web/package.json` dependencies add `"@tanstack/react-query": "^5.59.0"`. Run `cd web && npm install` (npm network works).

- [ ] **Step 2: Create `web/src/lib/types.ts`** (mirror the Go DTOs):
```ts
export interface Team { id: number; name: string; country: string; group: string; crestUrl: string }
export interface Venue { id: number; name: string; city: string; country: string; capacity: number }
export interface Match {
  id: number; stage: string; group: string; venueId: number;
  homeId: number; awayId: number; kickoffUtc: string;
  status: string; minute: number; homeScore: number; awayScore: number;
}
export interface Standing {
  group: string; teamId: number; played: number; won: number; drawn: number;
  lost: number; gf: number; ga: number; pts: number; form: string;
}
export interface MatchEvent {
  id: number; matchId: number; minute: number; type: string;
  teamId: number; playerId: number; detail: string;
}
```

- [ ] **Step 3: Create `web/src/lib/api.ts`**:
```ts
import type { Match, Standing, Team, Venue } from './types'

async function get<T>(path: string): Promise<T> {
  const res = await fetch(path, { headers: { Accept: 'application/json' } })
  if (!res.ok) throw new Error(`${path} -> ${res.status}`)
  return res.json() as Promise<T>
}

export const api = {
  fixtures: () => get<Match[]>('/api/fixtures'),
  standings: () => get<Standing[]>('/api/standings'),
  teams: () => get<Team[]>('/api/teams'),
  venues: () => get<Venue[]>('/api/venues'),
}
```

- [ ] **Step 4: Create `web/src/app/queryClient.ts`**:
```ts
import { QueryClient } from '@tanstack/react-query'

export const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 30_000, refetchOnWindowFocus: false } },
})
```

- [ ] **Step 5: Type-check** — `cd web && npx tsc --noEmit` → no errors.
- [ ] **Step 6: Commit** — `git add web/package.json web/package-lock.json web/src/lib web/src/app && git commit -m "feat: frontend data layer (query client, API client, types)"`

---

## Task 5: useLiveStream hook (TDD with mock EventSource)

**Files:** Create `web/src/lib/useLiveStream.ts`, `web/src/lib/useLiveStream.test.tsx`.

- [ ] **Step 1: Failing test** — `web/src/lib/useLiveStream.test.tsx`:
```tsx
import { renderHook, act, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useLiveStream, type EventSourceLike } from './useLiveStream'
import type { Match } from './types'

class MockES implements EventSourceLike {
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  onmessage: ((ev: { data: string }) => void) | null = null
  close() {}
  emit(obj: unknown) { this.onmessage?.({ data: JSON.stringify(obj) }) }
  open() { this.onopen?.() }
}

function wrapper(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

test('patches fixtures cache on match.update', async () => {
  const qc = new QueryClient()
  qc.setQueryData<Match[]>(['fixtures'], [
    { id: 1, stage: 'group', group: 'A', venueId: 0, homeId: 1, awayId: 2, kickoffUtc: '', status: 'live', minute: 0, homeScore: 0, awayScore: 0 },
  ])
  let es!: MockES
  const { result } = renderHook(() => useLiveStream({ makeES: () => (es = new MockES()) }), { wrapper: wrapper(qc) })
  act(() => es.open())
  await waitFor(() => expect(result.current.connected).toBe(true))

  act(() => es.emit({ type: 'match.update', data: { id: 1, status: 'live', minute: 30, homeScore: 1, awayScore: 0, stage: 'group', group: 'A', venueId: 0, homeId: 1, awayId: 2, kickoffUtc: '' } }))

  const fixtures = qc.getQueryData<Match[]>(['fixtures'])!
  expect(fixtures.find(m => m.id === 1)!.minute).toBe(30)
  expect(fixtures.find(m => m.id === 1)!.homeScore).toBe(1)
})
```

- [ ] **Step 2: Run → FAIL** — `cd web && npx vitest run src/lib/useLiveStream.test.ts`.

- [ ] **Step 3: Implement** — `web/src/lib/useLiveStream.ts`:
```ts
import { useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { Match } from './types'

export interface EventSourceLike {
  onopen: (() => void) | null
  onerror: (() => void) | null
  onmessage: ((ev: { data: string }) => void) | null
  close(): void
}

type Msg = { type: string; data: unknown }

export function useLiveStream(opts?: { makeES?: (url: string) => EventSourceLike }): { connected: boolean } {
  const qc = useQueryClient()
  const [connected, setConnected] = useState(false)
  const retchRef = useRef(0)

  useEffect(() => {
    let stopped = false
    let es: EventSourceLike | null = null

    const connect = () => {
      if (stopped) return
      const make = opts?.makeES ?? ((url: string) => new EventSource(url) as unknown as EventSourceLike)
      es = make('/api/stream')
      es.onopen = () => { setConnected(true); retchRef.current = 0 }
      es.onerror = () => {
        setConnected(false)
        es?.close()
        // exponential backoff, capped at 10s
        const delay = Math.min(1000 * 2 ** retchRef.current++, 10_000)
        if (!stopped) setTimeout(connect, delay)
      }
      es.onmessage = (ev) => {
        let msg: Msg
        try { msg = JSON.parse(ev.data) } catch { return }
        if (msg.type === 'match.update') {
          const m = msg.data as Match
          qc.setQueryData<Match[]>(['fixtures'], (old) =>
            old ? old.map((x) => (x.id === m.id ? m : x)) : old)
          qc.setQueryData<Match>(['match', m.id], m)
        }
        // match.event handled by Match Center slice (appends to ['events', id])
      }
    }
    connect()
    return () => { stopped = true; es?.close() }
  }, [qc, opts])

  return { connected }
}
```

- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `git add web/src/lib/useLiveStream.ts web/src/lib/useLiveStream.test.tsx && git commit -m "feat: useLiveStream SSE hook with cache patching + backoff"`

---

## Task 6: View shell + App wiring + Vite dev proxy

**Files:** Modify `web/vite.config.ts`, `web/src/App.tsx`, `web/src/App.test.tsx`; Create `web/src/app/Shell.tsx`.

- [ ] **Step 1: Add the dev proxy to `web/vite.config.ts`** — add a `server.proxy` block (keep existing `base`, `plugins`, `test`):
```ts
  server: {
    proxy: { '/api': 'http://localhost:8080' },
  },
```

- [ ] **Step 2: Create `web/src/app/Shell.tsx`** (minimal 4-view shell; slices fill the views):
```tsx
import { useState } from 'react'
import { useLiveStream } from '../lib/useLiveStream'

const VIEWS = ['Match Center', 'Schedule', 'Standings', 'Explore'] as const
type View = (typeof VIEWS)[number]

export default function Shell() {
  const [view, setView] = useState<View>('Match Center')
  const { connected } = useLiveStream()
  return (
    <main className="shell">
      <header className="shell__bar">
        <span className="shell__mark" aria-hidden="true">▲</span>
        <h1 className="shell__title">TOUCHLINE</h1>
        <span className="shell__tag">WORLD CUP 2026</span>
        <nav className="shell__nav">
          {VIEWS.map((v) => (
            <button key={v} className={v === view ? 'is-active' : ''} onClick={() => setView(v)}>{v}</button>
          ))}
        </nav>
        <span className={`shell__live ${connected ? 'is-on' : ''}`} title={connected ? 'live' : 'offline'}>●</span>
      </header>
      <section className="shell__body">
        <p className="shell__note">{view} coming online…</p>
      </section>
    </main>
  )
}
```

- [ ] **Step 3: Update `web/src/App.tsx`**:
```tsx
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from './app/queryClient'
import Shell from './app/Shell'

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Shell />
    </QueryClientProvider>
  )
}
```

- [ ] **Step 4: Update `web/src/App.test.tsx`** (still asserts the wordmark renders within providers):
```tsx
import { render, screen } from '@testing-library/react'
import App from './App'

test('renders the Touchline shell title and tournament tag', () => {
  render(<App />)
  expect(screen.getByText('TOUCHLINE')).toBeInTheDocument()
  expect(screen.getByText('WORLD CUP 2026')).toBeInTheDocument()
})
```
**Note:** `App` now mounts `useLiveStream`, which in jsdom will call `new EventSource('/api/stream')`. jsdom has no EventSource → it throws. Guard the hook: in `useLiveStream`, the default `makeES` references `EventSource`; wrap creation so that if `typeof EventSource === 'undefined'` the hook no-ops (stays disconnected) instead of throwing. Update the default branch:
```ts
      if (typeof EventSource === 'undefined' && !opts?.makeES) { return }
```
Place this guard at the top of `connect()`. This keeps the App render test green in jsdom without a real EventSource.

- [ ] **Step 5: Run frontend tests + build** — `cd web && npm test && npm run build` → all pass; build emits dist (with `.gitkeep` preserved).
- [ ] **Step 6: Commit** — `git add web/vite.config.ts web/src && git commit -m "feat: view shell, App providers, live indicator, vite dev proxy"`

---

## Final Verification (whole-phase)

- [ ] `go build ./... && go vet ./... && go test -race ./...` → all pass (adds `internal/sim`, store events).
- [ ] **Simulator determinism:** sim Step test advances minute + broadcasts; eventually finishes; ignores non-live — all pass with seeded RNG.
- [ ] `cd web && npm test` → App render + useLiveStream cache-patch tests pass.
- [ ] `cd web && npm run build` → succeeds; `.gitkeep` survives.
- [ ] No DB committed; tree clean.

### Deferred to user runtime gate
```bash
# real live experience end-to-end
docker compose up --build      # or: cd web && npm run build && go run ./cmd/touchline
# open http://localhost:8080/ : 4 promoted matches show as LIVE; scores/minutes tick every ~5s
curl -N -s localhost:8080/api/stream    # observe match.update / match.event frames
```

---

## Spec Coverage (Phase 2 Foundation portion)

| Item | Task |
|---|---|
| Match simulator (clock, goals, cards) | Task 2 |
| LIVE_SOURCE=sim|real switch (sim impl) | Task 3 |
| SSE deltas match.update/match.event from live source | Tasks 2, 3 |
| match_events persistence | Task 1 |
| TanStack Query for REST | Task 4 |
| useLiveStream EventSource hook + backoff + cache patch | Task 5 |
| Frontend shell/views + dev proxy | Task 6 |

## Out of scope (the 4 slices, planned next)

- (a) Match Center page (match detail, event timeline, lineups) — consumes `['fixtures']`, `['match',id]`, `['events',id]`, live patches.
- (b) Schedule/Calendar + follows/reminders endpoints + UI.
- (c) Standings & Stats page + standings computation from results + top scorers.
- (d) Explore hub (countries/players/venues) + notes endpoints + UI.
- Real APIFootball provider + quota guardrails + stale badges + full nav/theme cohesion → Phase 3.
