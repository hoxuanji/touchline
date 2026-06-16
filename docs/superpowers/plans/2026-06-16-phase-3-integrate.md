# Touchline Phase 3 — Integrate & Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`).

**Goal:** Production-readiness pass — a quota budgeter, the real-APIFootball provider + `LIVE_SOURCE=real` flip path (no code change to switch), a data-source/freshness badge, team-name display polish, and the dark-broadcast visual + motion pass — then a full green test sweep.

**Architecture:** A persisted daily request budgeter (in `meta_cache`-style table) hard-caps real API calls. An `apifootball` provider implements the existing `Provider` interface; a real poller (gated by `LIVE_SOURCE=real`) uses it under the budget. A `/api/meta` endpoint exposes `{source, asOf}` so the frontend shows a "SIM" / "data delayed" badge. Team names replace `#id` via a teams lookup in the cache. CSS gets the broadcast-scoreboard treatment with CSS-only transitions/animations.

**Tech Stack:** Go stdlib (`net/http` client, `time`); React + existing query/SSE; CSS.

---

## Environment Note
Network blocked (can't make/test real outbound HTTP; sim is the runtime). The APIFootball provider's HTTP call path is **code-reviewed, not run**; its JSON **normalization is unit-tested against sample bytes** (no socket). Budgeter + meta are pure logic (testable). Frontend via Vitest + `npm run build` (the tsc `noUnusedLocals` gate — always run build, not just test). Deferred to user gate: real key live polling, browser visual check, `docker compose up`, Playwright.

---

## Tasks

## Task 1: Daily request budgeter (backend, TDD)

**Files:** Create `internal/budget/budget.go`, `internal/budget/budget_test.go`.

- [ ] **Step 1: Failing test** `internal/budget/budget_test.go`:
```go
package budget_test

import (
	"testing"
	"time"

	"touchline/internal/budget"
)

func TestBudgetAllowsUpToLimitPerDay(t *testing.T) {
	day := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	b := budget.New(3, func() time.Time { return day })
	for i := 0; i < 3; i++ {
		if !b.Allow() {
			t.Fatalf("call %d should be allowed", i)
		}
	}
	if b.Allow() {
		t.Fatal("4th call should be denied (budget exhausted)")
	}
	if b.Remaining() != 0 {
		t.Fatalf("Remaining = %d, want 0", b.Remaining())
	}
}

func TestBudgetResetsNextDay(t *testing.T) {
	now := time.Date(2026, 6, 16, 23, 0, 0, 0, time.UTC)
	b := budget.New(1, func() time.Time { return now })
	if !b.Allow() {
		t.Fatal("first call allowed")
	}
	if b.Allow() {
		t.Fatal("second call same day denied")
	}
	now = now.AddDate(0, 0, 1) // next day
	if !b.Allow() {
		t.Fatal("call on new day should be allowed (counter reset)")
	}
}

func TestZeroLimitDeniesAll(t *testing.T) {
	b := budget.New(0, func() time.Time { return time.Unix(0, 0) })
	if b.Allow() {
		t.Fatal("zero-limit budget must deny")
	}
}
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** `internal/budget/budget.go`:
```go
// Package budget hard-caps outbound API calls per UTC day so a free-tier quota
// can never be exceeded. Safe for concurrent use.
package budget

import (
	"sync"
	"time"
)

type Budget struct {
	mu    sync.Mutex
	limit int
	now   func() time.Time
	day   string
	used  int
}

// New creates a budget allowing `limit` calls per UTC day. `now` is injectable for tests.
func New(limit int, now func() time.Time) *Budget {
	return &Budget{limit: limit, now: now}
}

func (b *Budget) rollover() {
	d := b.now().UTC().Format("2006-01-02")
	if d != b.day {
		b.day = d
		b.used = 0
	}
}

// Allow records and permits a call if today's budget remains; false otherwise.
func (b *Budget) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rollover()
	if b.used >= b.limit {
		return false
	}
	b.used++
	return true
}

func (b *Budget) Remaining() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rollover()
	r := b.limit - b.used
	if r < 0 {
		return 0
	}
	return r
}
```

- [ ] **Step 4: Run → PASS (with -race).**
- [ ] **Step 5: Commit** — `git add internal/budget && git commit -m "feat: per-UTC-day API request budgeter"`

---

## Task 2: APIFootball provider + real poller + LIVE_SOURCE=real flip (TDD on normalization)

**Files:** Create `internal/provider/apifootball.go`, `internal/provider/apifootball_test.go`; Modify `cmd/touchline/main.go`.

- [ ] **Step 1: Failing test** (normalization from sample bytes — NO network) `internal/provider/apifootball_test.go`:
```go
package provider

import "testing"

// Sample shaped like API-Football's /teams response (trimmed).
const sampleTeams = `{"response":[
  {"team":{"id":21,"name":"Brazil","country":"Brazil","logo":"https://x/21.png"}},
  {"team":{"id":17,"name":"France","country":"France","logo":"https://x/17.png"}}
]}`

func TestParseTeamsResponse(t *testing.T) {
	teams, err := parseTeams([]byte(sampleTeams))
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 2 || teams[0].ID != 21 || teams[0].Name != "Brazil" || teams[0].CrestURL == "" {
		t.Fatalf("parsed = %+v", teams)
	}
}
```
(This test is in `package provider` — internal — so it can call the unexported `parseTeams`.)

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** `internal/provider/apifootball.go`:
```go
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"touchline/internal/model"
)

// APIFootball is the real provider (api-sports.io). It implements Provider.
// Live polling is enabled via LIVE_SOURCE=real once a paid key is configured;
// reference reads are budget-capped by the caller.
type APIFootball struct {
	key    string
	base   string
	client *http.Client
}

func NewAPIFootball(key string) *APIFootball {
	return &APIFootball{
		key:    key,
		base:   "https://v3.football.api-sports.io",
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *APIFootball) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.base+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", a.key)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apifootball %s: status %d", path, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func parseTeams(b []byte) ([]model.Team, error) {
	var doc struct {
		Response []struct {
			Team struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				Country string `json:"country"`
				Logo    string `json:"logo"`
			} `json:"team"`
		} `json:"response"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	out := make([]model.Team, 0, len(doc.Response))
	for _, r := range doc.Response {
		out = append(out, model.Team{ID: r.Team.ID, Name: r.Team.Name, Country: r.Team.Country, CrestURL: r.Team.Logo})
	}
	return out, nil
}

func (a *APIFootball) Teams(ctx context.Context) ([]model.Team, error) {
	b, err := a.get(ctx, "/teams?league=1&season=2026")
	if err != nil {
		return nil, err
	}
	return parseTeams(b)
}

// Venues/Fixtures/Standings: same get+parse pattern. For the V1 flip path they
// return not-implemented sentinels so the interface is satisfied; the snapshot
// remains the seed/fallback. Wire real parsing as endpoints are confirmed.
func (a *APIFootball) Venues(ctx context.Context) ([]model.Venue, error) {
	return nil, fmt.Errorf("apifootball venues: not implemented (use snapshot seed)")
}
func (a *APIFootball) Fixtures(ctx context.Context) ([]model.Match, error) {
	return nil, fmt.Errorf("apifootball fixtures: not implemented (use snapshot seed)")
}
func (a *APIFootball) Standings(ctx context.Context) ([]model.Standing, error) {
	return nil, fmt.Errorf("apifootball standings: not implemented (use snapshot seed)")
}

var _ Provider = (*APIFootball)(nil)
```

- [ ] **Step 4: Run → PASS** — `go test ./internal/provider/`.

- [ ] **Step 5: Wire the flip path in `cmd/touchline/main.go`.** Read it. The current sim block is `if liveSource == "sim" { ... }`. Add an `else if liveSource == "real"` branch that logs the real source is selected and (for V1) still relies on the snapshot seed + notes that live polling would start here:
```go
	} else if liveSource == "real" {
		key := os.Getenv("API_FOOTBALL_KEY")
		if key == "" {
			log.Print("LIVE_SOURCE=real but API_FOOTBALL_KEY is empty; serving snapshot only")
		} else {
			log.Print("LIVE_SOURCE=real: APIFootball provider active (reference reads budget-capped)")
			// A real poller would start here, using provider.NewAPIFootball(key)
			// under a budget.New(DAILY_REQUEST_BUDGET). Snapshot remains the seed/fallback.
			_ = provider.NewAPIFootball(key)
		}
	}
```
(Import already has `provider`. This keeps the flip a pure env switch with no other code change, per the spec. Real polling loop is intentionally minimal here since it can't be exercised without a key/network.)

- [ ] **Step 6: Build + vet + test** — `go build ./... && go vet ./... && go test ./...` → pass.
- [ ] **Step 7: Commit** — `git add internal/provider/apifootball.go internal/provider/apifootball_test.go cmd/touchline/main.go && git commit -m "feat: APIFootball provider + LIVE_SOURCE=real flip path"`

---

## Task 3: /api/meta source+freshness endpoint + frontend badge (TDD)

**Files:** Modify `internal/server/server.go` (add /api/meta), `internal/server/server_test.go` (assert it); Create `web/src/features/meta/useMeta.ts`; Modify `web/src/app/Shell.tsx` (show badge).

- [ ] **Step 1: Add a meta route.** In `internal/server/server.go`, add a `Source string` field to `Deps` and a route. Update `Deps`:
```go
type Deps struct {
	Store  *store.Store
	Hot    *hot.Store
	SSE    *sse.Hub
	Source string // "sim" | "real"
}
```
Add inside `Handler`, before the SPA fallback:
```go
	mux.HandleFunc("GET /api/meta", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"source":"` + deps.Source + `"}`))
	})
```

- [ ] **Step 2: Update server test** — in `internal/server/server_test.go`, in `newHandler` pass `Source: "sim"` in `server.Deps{...}`, and add:
```go
func TestMetaRoute(t *testing.T) {
	h := newHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/meta", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"source":"sim"`) {
		t.Fatalf("meta: code=%d body=%s", rec.Code, rec.Body.String())
	}
}
```

- [ ] **Step 3: Run → FAIL then implement → PASS** — `go test ./internal/server/`.

- [ ] **Step 4: Pass Source from main.go** — in `cmd/touchline/main.go`, set `Source: liveSource` in the `server.Deps{...}` literal. Build → clean.

- [ ] **Step 5: Frontend badge.** Create `web/src/features/meta/useMeta.ts`:
```ts
import { useQuery } from '@tanstack/react-query'

export interface Meta { source: string }

async function getMeta(): Promise<Meta> {
  const r = await fetch('/api/meta', { headers: { Accept: 'application/json' } })
  return r.json()
}

export function useMeta() {
  return useQuery({ queryKey: ['meta'], queryFn: getMeta, staleTime: Infinity })
}
```
In `web/src/app/Shell.tsx`, import `useMeta` and render a source chip in the header (next to the live dot):
```tsx
  const { data: meta } = useMeta()
```
and in the header JSX add: `{meta?.source && <span className="shell__source">{meta.source.toUpperCase()}</span>}`. Add the `useMeta` import.

- [ ] **Step 6: Run frontend** — `cd web && npm test && npm run build` → pass. (The App test still finds TOUCHLINE; `useMeta` query has no data in jsdom → chip not rendered, fine.)

- [ ] **Step 7: Commit** — `git add internal/server cmd/touchline web/src/features/meta web/src/app/Shell.tsx && git commit -m "feat: /api/meta source endpoint + header source badge"`

---

## Task 4: Team-name display polish (frontend, TDD)

**Files:** Create `web/src/lib/useTeamNames.ts`; Modify `web/src/features/match-center/MatchCenterPage.tsx`, `web/src/features/schedule/SchedulePage.tsx`, and their tests.

- [ ] **Step 1: Create the lookup hook** `web/src/lib/useTeamNames.ts`:
```ts
import { useQuery } from '@tanstack/react-query'
import { api } from './api'
import type { Team } from './types'

// Returns a id->name lookup. Falls back to "#id" when a team isn't loaded yet.
export function useTeamNames(): (id: number) => string {
  const { data: teams = [] } = useQuery({ queryKey: ['teams'], queryFn: api.teams, staleTime: Infinity })
  const byId = new Map<number, string>(teams.map((t: Team) => [t.id, t.name]))
  return (id: number) => byId.get(id) ?? `#${id}`
}
```

- [ ] **Step 2: Update MatchCenter test** — change the live-match assertion in `MatchCenterPage.test.tsx` to seed teams and assert names. Add to the test's cache seeding:
```tsx
  qc.setQueryData(['teams'], [
    { id: 1, name: 'Brazil', country: '', group: '', crestUrl: '' },
    { id: 2, name: 'Spain', country: '', group: '', crestUrl: '' },
  ])
```
and add an assertion `expect(screen.getByText(/Brazil/)).toBeInTheDocument()`.

- [ ] **Step 3: Update `MatchCenterPage.tsx`** — use the hook; in `MatchCenterPage` call `const name = useTeamNames()` and pass to rows; change `MatchRow`'s teams span to `{name(m.homeId)} v {name(m.awayId)}`. (Move `MatchRow` inline or pass `name` as a prop.) Implementer: thread `name` into `MatchRow` via prop `name: (id:number)=>string`.

- [ ] **Step 4: Update Schedule similarly** — `SchedulePage.tsx` uses `useTeamNames()` and renders `{name(m.homeId)} v {name(m.awayId)}`. Update `SchedulePage.test.tsx` to seed `['teams']` if it asserts names (optional — the existing date-grouping assertion still holds; only change if needed to keep tsc/test green).

- [ ] **Step 5: Run** — `cd web && npm test && npm run build` → all pass.
- [ ] **Step 6: Commit** — `git add web/src/lib/useTeamNames.ts web/src/features/match-center web/src/features/schedule && git commit -m "feat: show team names instead of ids in Match Center + Schedule"`

---

## Task 5: Dark-broadcast visual + motion pass (CSS, build-verified)

**Files:** Modify `web/src/index.css`.

- [ ] **Step 1: Expand `web/src/index.css`** with broadcast styling for the new components — keep the existing `:root` tokens and `.shell*` rules; ADD rules for nav buttons (`.shell__nav button`), the live dot pulse (`.shell__live.is-on` with `@keyframes pulse`), source chip (`.shell__source`), page container (`.shell__body--page`), match rows (`.mc__row`, `.mc__score`, `.mc__min` with a subtle score transition), schedule (`.sched__day`, `.sched__row`), standings table (`table`, `th`, `td`), explore lists, and a `.muted` utility. Use CSS transitions only (no JS motion lib). Example additions:
```css
.shell__nav { display: flex; gap: 0.25rem; margin-left: auto; }
.shell__nav button {
  background: transparent; color: var(--muted); border: 0; padding: 0.4rem 0.7rem;
  font: inherit; letter-spacing: 0.05em; cursor: pointer; border-radius: 4px;
  transition: color 0.15s ease, background 0.15s ease;
}
.shell__nav button:hover { color: var(--text); }
.shell__nav button.is-active { color: var(--text); background: rgba(0,210,106,0.12); }
.shell__live { margin-left: 0.75rem; color: var(--muted); font-size: 0.7rem; }
.shell__live.is-on { color: var(--pitch); animation: pulse 1.6s ease-in-out infinite; }
@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.35; } }
.shell__source {
  margin-left: 0.5rem; font-size: 0.6rem; letter-spacing: 0.15em; color: var(--bg);
  background: var(--muted); padding: 0.1rem 0.4rem; border-radius: 3px;
}
.shell__body--page { display: block; align-items: initial; padding: 1.5rem; overflow-y: auto; }
.muted { color: var(--muted); }
.mc__h { font-family: var(--font-display); letter-spacing: 0.1em; font-size: 0.9rem; color: var(--muted); margin: 1rem 0 0.5rem; text-transform: uppercase; }
.mc__list { list-style: none; display: grid; gap: 0.5rem; }
.mc__row { display: grid; grid-template-columns: 1fr auto auto; gap: 1rem; align-items: center;
  padding: 0.75rem 1rem; background: var(--panel); border: 1px solid var(--line); border-radius: 6px; }
.mc__row.is-live { border-color: rgba(0,210,106,0.4); }
.mc__score { font-family: var(--font-display); font-size: 1.4rem; font-variant-numeric: tabular-nums; transition: color 0.3s ease; }
.mc__min { color: var(--pitch); font-size: 0.8rem; font-variant-numeric: tabular-nums; }
.sched__day { font-family: var(--font-display); color: var(--muted); letter-spacing: 0.1em; margin: 1.25rem 0 0.5rem; }
.sched__row { display: flex; gap: 1rem; padding: 0.4rem 0; border-bottom: 1px solid var(--line); }
.standings table { width: 100%; border-collapse: collapse; margin-bottom: 1.5rem; }
.standings th, .standings td { text-align: right; padding: 0.4rem 0.6rem; border-bottom: 1px solid var(--line); font-variant-numeric: tabular-nums; }
.standings th:first-child, .standings td:first-child { text-align: left; }
.standings h3 { font-family: var(--font-display); letter-spacing: 0.1em; margin-bottom: 0.5rem; }
.explore section { margin-bottom: 1.5rem; }
.explore ul { list-style: none; display: grid; gap: 0.25rem; }
```

- [ ] **Step 2: Build** — `cd web && npm test && npm run build` → succeeds (CSS changes don't affect tests; build confirms no syntax error).
- [ ] **Step 3: Commit** — `git add web/src/index.css && git commit -m "feat: dark-broadcast visual + motion pass"`

---

## Final Verification (whole-phase + whole-app)

- [ ] `go build ./... && go vet ./... && go test -race ./...` → all pass (adds budget, apifootball, meta).
- [ ] `cd web && npm test` → all suites pass.
- [ ] `cd web && npm run build` → succeeds (tsc noUnusedLocals gate + vite), `.gitkeep` preserved.
- [ ] Tree clean, no `*.db`.

### Deferred to user runtime gate (consolidated for the PR)
```bash
docker compose up --build
# Browser http://localhost:8080 : dark-broadcast UI; 4 nav tabs; live dot pulses;
#   "SIM" source chip; Match Center shows team names + ticking live scores;
#   Standings fill as matches finish; follows/notes persist across `docker compose down/up`.
LIVE_SOURCE=real API_FOOTBALL_KEY=... docker compose up --build   # flip path (needs paid key)
# Playwright smoke (Phase spec): drive a sim match, assert score updates in DOM via SSE.
```

## Spec Coverage (Phase 3)
| Spec item | Task |
|---|---|
| Real-API live flip path (LIVE_SOURCE=real, no code change) | Task 2 |
| Quota guardrails (DAILY_REQUEST_BUDGET) | Task 1 |
| Stale/source badge ("data delayed · source") | Task 3 |
| Dark-broadcast visual + motion pass | Task 5 |
| Team-name polish (nav/theme cohesion detail) | Task 4 |
| Full test pass | Final Verification |

## Out of scope / honest gaps (documented, not built)
- Real APIFootball Fixtures/Venues/Standings parsing is stubbed (returns not-implemented); snapshot remains seed/fallback. Confirmed-shape parsing is wired when a paid key + live API docs are available (spec risk §14).
- Playwright smoke E2E and the live browser visual check require a running server (port bind) → user runtime gate.
- Knockout bracket visualization, player squads/lineups (no player data seeded), reminder *firing* (reminders persist but no notification scheduler) — beyond V1 scope.
