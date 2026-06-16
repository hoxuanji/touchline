# Touchline Phase 2 — Feature Slices Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`).

**Goal:** Build the four feature areas on the Phase-2 foundation — (a) Match Center, (b) Schedule/Calendar + follows/reminders, (c) Standings & Stats, (d) Explore + notes — each as isolated backend handlers + store methods + a React feature page, then a small serial task wiring pages into the shell and handlers into the router.

**Architecture:** Each slice adds *new* files only (parallel-safe): a Go handler file `internal/api/<slice>.go` exposing a `RegisterXxx(mux, deps)` func, optional new store-methods file `internal/store/<topic>.go`, and a React feature folder `web/src/features/<slice>/`. A final **wiring task** edits the two shared seams (`internal/api/api.go` to call the registrars, `web/src/app/Shell.tsx` to render the pages) and extends `useLiveStream` for `match.event`.

**Tech Stack:** Go stdlib + existing store/hot/sse; React + TanStack Query (`useQuery`) + the existing `api` client and `useLiveStream`.

---

## Environment Note
Network blocked (npm works; Go cache-only — no `go get`/`tidy`). No socket/Docker/browser. Go via `go test -race ./...` (httptest/`:memory:`). Frontend via Vitest (jsdom). Deferred to user gate: live browser experience, `docker compose up`.

## Conventions (established, reuse)
- API JSON via the existing `api` package helpers. Each slice handler file lives in `package api` and exposes `func RegisterXxx(mux *http.ServeMux, deps Deps)`. Reuse the package-private `writeJSON`/`writeError` (same package).
- Path params via Go 1.22 routing: `mux.HandleFunc("GET /api/matches/{id}", ...)`, read with `r.PathValue("id")`.
- Frontend: each page is `web/src/features/<slice>/<Name>Page.tsx`, fetching with `useQuery({ queryKey: [...], queryFn: ... })`. Feature-specific fetches go in `web/src/features/<slice>/api.ts` (do NOT edit the shared `web/src/lib/api.ts`). A Vitest test per page renders it inside a `QueryClientProvider` with seeded cache (no network).
- Empty collections must serialize `[]` not `null` (the api `writeJSON` already guards nil slices).

---

## Shared seam contract (so slices stay parallel-safe)
- Each slice's Go handler file adds `func Register<Slice>(mux *http.ServeMux, deps Deps)`. The wiring task (Task W) adds the calls inside `api.Register`. Until then the funcs are unused — that's fine, they compile (exported funcs need no caller). **However** Go flags unused *imports*, not unused exported funcs, so this is safe.
- Each slice's page is the default export of its `*Page.tsx`. Task W imports them into `Shell.tsx` and maps `view → page`.
- No two slices edit the same file. Store methods: slice (b) → `internal/store/userdata.go`; slice (c) → `internal/store/stats.go`; slice (d) → `internal/store/notes.go`. Slice (a) needs no new store method (uses existing `EventsByMatch`, `Teams`, hot `Match`).

---

# SLICE A — Match Center

**Files:** Create `internal/api/matches.go`, `internal/api/matches_test.go`, `web/src/features/match-center/api.ts`, `web/src/features/match-center/MatchCenterPage.tsx`, `web/src/features/match-center/MatchCenterPage.test.tsx`. Modify `web/src/lib/useLiveStream.ts` (add `match.event` handling).

### Task A1: Match detail endpoint (backend, TDD)

- [ ] **Step 1: Failing test** `internal/api/matches_test.go`:
```go
package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"touchline/internal/api"
	"touchline/internal/hot"
	"touchline/internal/model"
	"touchline/internal/store"
)

func setupMatches(t *testing.T) (http.Handler, *store.Store) {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	_ = s.UpsertTeams([]model.Team{{ID: 1, Name: "Brazil"}, {ID: 2, Name: "Spain"}})
	_ = s.UpsertMatches([]model.Match{{ID: 10, Stage: "group", Group: "F", HomeID: 1, AwayID: 2, Status: "live", Minute: 30, HomeScore: 1}})
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 10, Minute: 12, Type: "goal", TeamID: 1})
	h := hot.New()
	m, _ := s.Matches()
	h.Hydrate(m, nil)
	mux := http.NewServeMux()
	api.RegisterMatches(mux, api.Deps{Store: s, Hot: h})
	return mux, s
}

func TestGetMatchDetail(t *testing.T) {
	h, _ := setupMatches(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/matches/10", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got struct {
		Match    model.Match        `json:"match"`
		Events   []model.MatchEvent `json:"events"`
		HomeName string             `json:"homeName"`
		AwayName string             `json:"awayName"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Match.ID != 10 || got.Match.HomeScore != 1 {
		t.Fatalf("match = %+v", got.Match)
	}
	if len(got.Events) != 1 || got.Events[0].Type != "goal" {
		t.Fatalf("events = %+v", got.Events)
	}
	if got.HomeName != "Brazil" || got.AwayName != "Spain" {
		t.Fatalf("names = %q vs %q", got.HomeName, got.AwayName)
	}
}

func TestGetMatchDetailNotFound(t *testing.T) {
	h, _ := setupMatches(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/matches/999", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** `internal/api/matches.go`:
```go
package api

import (
	"net/http"
	"strconv"

	"touchline/internal/model"
)

// RegisterMatches adds the match-detail route.
func RegisterMatches(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/matches/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		m, ok := deps.Hot.Match(id)
		if !ok {
			http.Error(w, "match not found", http.StatusNotFound)
			return
		}
		events, err := deps.Store.EventsByMatch(id)
		if err != nil {
			writeError(w, err)
			return
		}
		teams, err := deps.Store.Teams()
		if err != nil {
			writeError(w, err)
			return
		}
		names := map[int]string{}
		for _, t := range teams {
			names[t.ID] = t.Name
		}
		writeJSON(w, http.StatusOK, struct {
			Match    model.Match        `json:"match"`
			Events   []model.MatchEvent `json:"events"`
			HomeName string             `json:"homeName"`
			AwayName string             `json:"awayName"`
		}{Match: m, Events: events, HomeName: names[m.HomeID], AwayName: names[m.AwayID]})
	})
}
```

- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `git add internal/api/matches.go internal/api/matches_test.go && git commit -m "feat(match-center): match detail endpoint"`

### Task A2: Match Center page + match.event live patch (frontend, TDD)

- [ ] **Step 1:** Extend `web/src/lib/useLiveStream.ts` — inside `onmessage`, after the `match.update` branch, add `match.event` handling:
```ts
        if (msg.type === 'match.event') {
          const ev = msg.data as { matchId: number }
          qc.setQueryData<unknown[]>(['events', ev.matchId], (old) => (old ? [...old, msg.data] : [msg.data]))
        }
```
(Keep the existing `match.update` branch. The `MatchEvent` import is not required since we use a structural type for the id.)

- [ ] **Step 2: Failing test** `web/src/features/match-center/MatchCenterPage.test.tsx`:
```tsx
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import MatchCenterPage from './MatchCenterPage'
import type { Match } from '../../lib/types'

function wrap(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

test('lists live matches with score', () => {
  const qc = new QueryClient()
  qc.setQueryData<Match[]>(['fixtures'], [
    { id: 1, stage: 'group', group: 'F', venueId: 0, homeId: 1, awayId: 2, kickoffUtc: '2026-06-11T19:00:00Z', status: 'live', minute: 30, homeScore: 2, awayScore: 1 },
  ])
  render(<MatchCenterPage />, { wrapper: wrap(qc) })
  expect(screen.getByText(/2\s*[-–]\s*1/)).toBeInTheDocument()
  expect(screen.getByText(/30'/)).toBeInTheDocument()
})
```

- [ ] **Step 3: Run → FAIL** — `cd web && npx vitest run src/features/match-center/MatchCenterPage.test.tsx`.

- [ ] **Step 4: Implement** `web/src/features/match-center/MatchCenterPage.tsx`:
```tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '../../lib/api'
import type { Match } from '../../lib/types'

const LIVE = new Set(['live', 'ht'])

export default function MatchCenterPage() {
  const { data: matches = [], isLoading } = useQuery({ queryKey: ['fixtures'], queryFn: api.fixtures })
  if (isLoading) return <p className="muted">Loading matches…</p>
  const live = matches.filter((m) => LIVE.has(m.status))
  const upcoming = matches.filter((m) => m.status === 'scheduled').slice(0, 12)
  return (
    <div className="mc">
      <h2 className="mc__h">Live</h2>
      {live.length === 0 && <p className="muted">No live matches right now.</p>}
      <ul className="mc__list">
        {live.map((m) => <MatchRow key={m.id} m={m} live />)}
      </ul>
      <h2 className="mc__h">Upcoming</h2>
      <ul className="mc__list">
        {upcoming.map((m) => <MatchRow key={m.id} m={m} />)}
      </ul>
    </div>
  )
}

function MatchRow({ m, live }: { m: Match; live?: boolean }) {
  return (
    <li className={`mc__row ${live ? 'is-live' : ''}`}>
      <span className="mc__teams">#{m.homeId} v #{m.awayId}</span>
      <span className="mc__score">{m.homeScore} – {m.awayScore}</span>
      {live ? <span className="mc__min">{m.minute}&apos;</span> : <span className="mc__time">{new Date(m.kickoffUtc).toLocaleString()}</span>}
    </li>
  )
}
```
(Team names are shown as `#id` here; the wiring/polish phase can map names. The test asserts score `2 – 1` and minute `30'`.)

- [ ] **Step 5: Run → PASS.**
- [ ] **Step 6: Commit** — `git add web/src/lib/useLiveStream.ts web/src/features/match-center && git commit -m "feat(match-center): live + upcoming matches page, match.event cache patch"`

---

# SLICE B — Schedule + follows/reminders

**Files:** Create `internal/store/userdata.go`, `internal/store/userdata_test.go`, `internal/api/userdata.go`, `internal/api/userdata_test.go`, `web/src/features/schedule/api.ts`, `web/src/features/schedule/SchedulePage.tsx`, `web/src/features/schedule/SchedulePage.test.tsx`.

### Task B1: reminders + prefs store methods (TDD)

- [ ] **Step 1: Failing test** `internal/store/userdata_test.go`:
```go
package store_test

import "testing"

func TestPrefsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	if err := s.SetPref("timezone", "America/New_York"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPref("timezone", "Europe/London"); err != nil { // upsert
		t.Fatal(err)
	}
	v, err := s.GetPref("timezone")
	if err != nil || v != "Europe/London" {
		t.Fatalf("GetPref = %q err %v", v, err)
	}
	missing, err := s.GetPref("nope")
	if err != nil || missing != "" {
		t.Fatalf("missing pref should be empty string, got %q err %v", missing, err)
	}
}

func TestRemindersRoundTrip(t *testing.T) {
	s := newTestStore(t)
	id, err := s.AddReminder(10, 30)
	if err != nil || id <= 0 {
		t.Fatalf("AddReminder id=%d err=%v", id, err)
	}
	rs, err := s.Reminders()
	if err != nil || len(rs) != 1 || rs[0].MatchID != 10 || rs[0].LeadMinutes != 30 {
		t.Fatalf("Reminders = %+v err %v", rs, err)
	}
	if err := s.DeleteReminder(id); err != nil {
		t.Fatal(err)
	}
	rs, _ = s.Reminders()
	if len(rs) != 0 {
		t.Fatalf("after delete: %+v", rs)
	}
}
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** `internal/store/userdata.go`:
```go
package store

import "touchline/internal/model"

func (s *Store) SetPref(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO prefs (key,value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

// GetPref returns the value, or "" if the key is absent.
func (s *Store) GetPref(key string) (string, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM prefs WHERE key = ?`, key).Scan(&v)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", nil
		}
		return "", err
	}
	return v, nil
}

func (s *Store) AddReminder(matchID, leadMinutes int) (int, error) {
	res, err := s.db.Exec(`INSERT INTO reminders (match_id,lead_minutes,fired) VALUES (?,?,0)`, matchID, leadMinutes)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) Reminders() ([]model.Reminder, error) {
	rows, err := s.db.Query(`SELECT id,match_id,lead_minutes,fired FROM reminders ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Reminder
	for rows.Next() {
		var r model.Reminder
		var fired int
		if err := rows.Scan(&r.ID, &r.MatchID, &r.LeadMinutes, &fired); err != nil {
			return nil, err
		}
		r.Fired = fired != 0
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DeleteReminder(id int) error {
	_, err := s.db.Exec(`DELETE FROM reminders WHERE id = ?`, id)
	return err
}
```
**Note:** prefer `errors.Is(err, sql.ErrNoRows)` over string-matching. Use:
```go
import (
	"database/sql"
	"errors"
	"touchline/internal/model"
)
// ...
	if errors.Is(err, sql.ErrNoRows) { return "", nil }
```
(Implementer: use the `errors.Is(err, sql.ErrNoRows)` form, not the string compare shown above.)

- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `git add internal/store/userdata.go internal/store/userdata_test.go && git commit -m "feat(schedule): prefs + reminders store methods"`

### Task B2: follows/reminders/prefs endpoints (TDD)

- [ ] **Step 1: Failing test** `internal/api/userdata_test.go`:
```go
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
	// follow team 7
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/follows/7", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST follow status = %d", rec.Code)
	}
	// list
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/follows", nil))
	if !strings.Contains(rec.Body.String(), "7") {
		t.Fatalf("follows list = %s", rec.Body.String())
	}
	// unfollow
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
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** `internal/api/userdata.go`:
```go
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func RegisterUserdata(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/follows", func(w http.ResponseWriter, r *http.Request) {
		ids, err := deps.Store.Follows()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ids)
	})
	mux.HandleFunc("POST /api/follows/{teamId}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("teamId"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := deps.Store.AddFollow(id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE /api/follows/{teamId}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("teamId"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := deps.Store.RemoveFollow(id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/prefs/{key}", func(w http.ResponseWriter, r *http.Request) {
		v, err := deps.Store.GetPref(r.PathValue("key"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"key": r.PathValue("key"), "value": v})
	})
	mux.HandleFunc("PUT /api/prefs/{key}", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if err := deps.Store.SetPref(r.PathValue("key"), body.Value); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/reminders", func(w http.ResponseWriter, r *http.Request) {
		rs, err := deps.Store.Reminders()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, rs)
	})
	mux.HandleFunc("POST /api/reminders", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			MatchID     int `json:"matchId"`
			LeadMinutes int `json:"leadMinutes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if body.LeadMinutes == 0 {
			body.LeadMinutes = 30
		}
		id, err := deps.Store.AddReminder(body.MatchID, body.LeadMinutes)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]int{"id": id})
	})
	mux.HandleFunc("DELETE /api/reminders/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := deps.Store.DeleteReminder(id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
```

- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `git add internal/api/userdata.go internal/api/userdata_test.go && git commit -m "feat(schedule): follows/reminders/prefs endpoints"`

### Task B3: Schedule page (frontend, TDD)

- [ ] **Step 1: Create** `web/src/features/schedule/api.ts`:
```ts
export async function follow(teamId: number): Promise<void> {
  await fetch(`/api/follows/${teamId}`, { method: 'POST' })
}
export async function unfollow(teamId: number): Promise<void> {
  await fetch(`/api/follows/${teamId}`, { method: 'DELETE' })
}
export async function getFollows(): Promise<number[]> {
  const r = await fetch('/api/follows', { headers: { Accept: 'application/json' } })
  return r.json()
}
```

- [ ] **Step 2: Failing test** `web/src/features/schedule/SchedulePage.test.tsx`:
```tsx
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import SchedulePage from './SchedulePage'
import type { Match } from '../../lib/types'

function wrap(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

test('groups fixtures by date', () => {
  const qc = new QueryClient()
  qc.setQueryData<Match[]>(['fixtures'], [
    { id: 1, stage: 'group', group: 'A', venueId: 0, homeId: 1, awayId: 2, kickoffUtc: '2026-06-11T19:00:00Z', status: 'scheduled', minute: 0, homeScore: 0, awayScore: 0 },
    { id: 2, stage: 'group', group: 'A', venueId: 0, homeId: 3, awayId: 4, kickoffUtc: '2026-06-12T19:00:00Z', status: 'scheduled', minute: 0, homeScore: 0, awayScore: 0 },
  ])
  qc.setQueryData<number[]>(['follows'], [])
  render(<SchedulePage />, { wrapper: wrap(qc) })
  // two distinct date headings
  expect(screen.getAllByRole('heading', { level: 3 }).length).toBeGreaterThanOrEqual(2)
})
```

- [ ] **Step 3: Run → FAIL.**

- [ ] **Step 4: Implement** `web/src/features/schedule/SchedulePage.tsx`:
```tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '../../lib/api'
import type { Match } from '../../lib/types'
import { getFollows } from './api'

function dayKey(iso: string): string {
  if (!iso) return 'TBD'
  return new Date(iso).toISOString().slice(0, 10)
}

export default function SchedulePage() {
  const { data: matches = [] } = useQuery({ queryKey: ['fixtures'], queryFn: api.fixtures })
  useQuery({ queryKey: ['follows'], queryFn: getFollows })
  const byDay = new Map<string, Match[]>()
  for (const m of matches) {
    const k = dayKey(m.kickoffUtc)
    if (!byDay.has(k)) byDay.set(k, [])
    byDay.get(k)!.push(m)
  }
  const days = [...byDay.keys()].sort()
  return (
    <div className="sched">
      {days.map((d) => (
        <section key={d}>
          <h3 className="sched__day">{d}</h3>
          <ul>
            {byDay.get(d)!.map((m) => (
              <li key={m.id} className="sched__row">
                <span>{m.group || m.stage}</span> <span>#{m.homeId} v #{m.awayId}</span>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  )
}
```

- [ ] **Step 5: Run → PASS.**
- [ ] **Step 6: Commit** — `git add web/src/features/schedule && git commit -m "feat(schedule): schedule page grouped by date"`

---

# SLICE C — Standings & Stats

**Files:** Create `internal/store/stats.go`, `internal/store/stats_test.go`, `internal/api/stats.go`, `internal/api/stats_test.go`, `web/src/features/standings/api.ts`, `web/src/features/standings/StandingsPage.tsx`, `web/src/features/standings/StandingsPage.test.tsx`.

### Task C1: standings recompute + top scorers (store, TDD)

- [ ] **Step 1: Failing test** `internal/store/stats_test.go`:
```go
package store_test

import (
	"testing"

	"touchline/internal/model"
)

func TestRecomputeStandingsFromFinishedMatches(t *testing.T) {
	s := newTestStore(t)
	_ = s.UpsertTeams([]model.Team{
		{ID: 1, Group: "A"}, {ID: 2, Group: "A"}, {ID: 3, Group: "A"}, {ID: 4, Group: "A"},
	})
	// team1 beat team2 2-0 (finished); other match scheduled (ignored)
	_ = s.UpsertMatches([]model.Match{
		{ID: 1, Stage: "group", Group: "A", HomeID: 1, AwayID: 2, Status: "finished", HomeScore: 2, AwayScore: 0},
		{ID: 2, Stage: "group", Group: "A", HomeID: 3, AwayID: 4, Status: "scheduled"},
	})
	if err := s.RecomputeStandings(); err != nil {
		t.Fatal(err)
	}
	rows, _ := s.Standings()
	byTeam := map[int]model.Standing{}
	for _, r := range rows {
		byTeam[r.TeamID] = r
	}
	if byTeam[1].Pts != 3 || byTeam[1].Won != 1 || byTeam[1].GF != 2 {
		t.Fatalf("team1 standing = %+v", byTeam[1])
	}
	if byTeam[2].Pts != 0 || byTeam[2].Lost != 1 || byTeam[2].GA != 2 {
		t.Fatalf("team2 standing = %+v", byTeam[2])
	}
	if byTeam[3].Played != 0 {
		t.Fatalf("team3 should have 0 played, got %+v", byTeam[3])
	}
}

func TestTopScorersCountsGoals(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 1, Type: "goal", PlayerID: 0, TeamID: 1})
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 1, Type: "goal", PlayerID: 0, TeamID: 1})
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 1, Type: "card", TeamID: 2})
	rows, err := s.TopScorersByTeam()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 || rows[0].TeamID != 1 || rows[0].Goals != 2 {
		t.Fatalf("top scorers = %+v", rows)
	}
}
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** `internal/store/stats.go`:
```go
package store

import "touchline/internal/model"

// RecomputeStandings derives the group standings table from all finished group
// matches and writes it back. Idempotent: recomputes from scratch each call.
func (s *Store) RecomputeStandings() error {
	teams, err := s.Teams()
	if err != nil {
		return err
	}
	matches, err := s.Matches()
	if err != nil {
		return err
	}
	tab := map[int]*model.Standing{}
	for _, t := range teams {
		tab[t.ID] = &model.Standing{Group: t.Group, TeamID: t.ID}
	}
	for _, m := range matches {
		if m.Stage != "group" || m.Status != "finished" {
			continue
		}
		h, a := tab[m.HomeID], tab[m.AwayID]
		if h == nil || a == nil {
			continue
		}
		h.Played++
		a.Played++
		h.GF += m.HomeScore
		h.GA += m.AwayScore
		a.GF += m.AwayScore
		a.GA += m.HomeScore
		switch {
		case m.HomeScore > m.AwayScore:
			h.Won++
			h.Pts += 3
			a.Lost++
		case m.HomeScore < m.AwayScore:
			a.Won++
			a.Pts += 3
			h.Lost++
		default:
			h.Drawn++
			a.Drawn++
			h.Pts++
			a.Pts++
		}
	}
	rows := make([]model.Standing, 0, len(tab))
	for _, st := range tab {
		rows = append(rows, *st)
	}
	return s.UpsertStandings(rows)
}

// ScorerRow aggregates goals; PlayerID 0 means team-attributed (no player data yet).
type ScorerRow struct {
	TeamID   int `json:"teamId"`
	PlayerID int `json:"playerId"`
	Goals    int `json:"goals"`
}

// TopScorersByTeam counts goal events grouped by team, ordered by goals desc.
func (s *Store) TopScorersByTeam() ([]ScorerRow, error) {
	rows, err := s.db.Query(`SELECT team_id, COUNT(*) AS goals FROM match_events WHERE type = 'goal' GROUP BY team_id ORDER BY goals DESC, team_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScorerRow
	for rows.Next() {
		var r ScorerRow
		if err := rows.Scan(&r.TeamID, &r.Goals); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
```
- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `git add internal/store/stats.go internal/store/stats_test.go && git commit -m "feat(standings): standings recompute + top scorers store methods"`

### Task C2: stats endpoint (TDD)

- [ ] **Step 1: Failing test** `internal/api/stats_test.go`:
```go
package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"touchline/internal/api"
	"touchline/internal/model"
	"touchline/internal/store"
)

func TestGetTopScorers(t *testing.T) {
	s, _ := store.Open(":memory:")
	t.Cleanup(func() { s.Close() })
	_, _ = s.AddEvent(model.MatchEvent{MatchID: 1, Type: "goal", TeamID: 5})
	mux := http.NewServeMux()
	api.RegisterStats(mux, api.Deps{Store: s})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stats/scorers", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"teamId":5`) {
		t.Fatalf("scorers: code=%d body=%s", rec.Code, rec.Body.String())
	}
}
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** `internal/api/stats.go`:
```go
package api

import "net/http"

func RegisterStats(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/stats/scorers", func(w http.ResponseWriter, r *http.Request) {
		rows, err := deps.Store.TopScorersByTeam()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, rows)
	})
}
```

- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `git add internal/api/stats.go internal/api/stats_test.go && git commit -m "feat(standings): top scorers endpoint"`

### Task C3: Standings page (frontend, TDD)

- [ ] **Step 1: Failing test** `web/src/features/standings/StandingsPage.test.tsx`:
```tsx
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import StandingsPage from './StandingsPage'
import type { Standing } from '../../lib/types'

function wrap(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

test('renders a group table with points', () => {
  const qc = new QueryClient()
  qc.setQueryData<Standing[]>(['standings'], [
    { group: 'A', teamId: 1, played: 1, won: 1, drawn: 0, lost: 0, gf: 2, ga: 0, pts: 3, form: '' },
    { group: 'A', teamId: 2, played: 1, won: 0, drawn: 0, lost: 1, gf: 0, ga: 2, pts: 0, form: '' },
  ])
  render(<StandingsPage />, { wrapper: wrap(qc) })
  expect(screen.getByText('Group A')).toBeInTheDocument()
  expect(screen.getAllByRole('row').length).toBeGreaterThanOrEqual(3) // header + 2 teams
})
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** `web/src/features/standings/api.ts`:
```ts
import type { Standing } from '../../lib/types'
export async function getStandings(): Promise<Standing[]> {
  const r = await fetch('/api/standings', { headers: { Accept: 'application/json' } })
  return r.json()
}
```
and `web/src/features/standings/StandingsPage.tsx`:
```tsx
import { useQuery } from '@tanstack/react-query'
import type { Standing } from '../../lib/types'
import { getStandings } from './api'

export default function StandingsPage() {
  const { data: standings = [] } = useQuery({ queryKey: ['standings'], queryFn: getStandings })
  const groups = [...new Set(standings.map((s) => s.group))].sort()
  return (
    <div className="standings">
      {groups.map((g) => (
        <section key={g}>
          <h3>Group {g}</h3>
          <table>
            <thead><tr><th>Team</th><th>P</th><th>W</th><th>D</th><th>L</th><th>GD</th><th>Pts</th></tr></thead>
            <tbody>
              {standings.filter((s) => s.group === g).map((s) => (
                <tr key={s.teamId}>
                  <td>#{s.teamId}</td><td>{s.played}</td><td>{s.won}</td><td>{s.drawn}</td><td>{s.lost}</td><td>{s.gf - s.ga}</td><td>{s.pts}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ))}
    </div>
  )
}
```

- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `git add web/src/features/standings && git commit -m "feat(standings): group tables page"`

---

# SLICE D — Explore + notes

**Files:** Create `internal/store/notes.go`, `internal/store/notes_test.go`, `internal/api/notes.go`, `internal/api/notes_test.go`, `web/src/features/explore/api.ts`, `web/src/features/explore/ExplorePage.tsx`, `web/src/features/explore/ExplorePage.test.tsx`.

### Task D1: notes store methods (TDD)

- [ ] **Step 1: Failing test** `internal/store/notes_test.go`:
```go
package store_test

import (
	"testing"

	"touchline/internal/model"
)

func TestNotesCRUD(t *testing.T) {
	s := newTestStore(t)
	id, err := s.AddNote(model.Note{SubjectType: "team", SubjectID: 21, Body: "Watch the press"})
	if err != nil || id <= 0 {
		t.Fatalf("AddNote id=%d err=%v", id, err)
	}
	notes, err := s.NotesBySubject("team", 21)
	if err != nil || len(notes) != 1 || notes[0].Body != "Watch the press" {
		t.Fatalf("NotesBySubject = %+v err %v", notes, err)
	}
	if notes[0].CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set")
	}
	if err := s.DeleteNote(id); err != nil {
		t.Fatal(err)
	}
	notes, _ = s.NotesBySubject("team", 21)
	if len(notes) != 0 {
		t.Fatalf("after delete: %+v", notes)
	}
}
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** `internal/store/notes.go`:
```go
package store

import (
	"time"

	"touchline/internal/model"
)

func (s *Store) AddNote(n model.Note) (int, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO notes (subject_type,subject_id,body,created_at,updated_at) VALUES (?,?,?,?,?)`,
		n.SubjectType, n.SubjectID, n.Body, now, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) NotesBySubject(subjectType string, subjectID int) ([]model.Note, error) {
	rows, err := s.db.Query(
		`SELECT id,subject_type,subject_id,body,created_at,updated_at FROM notes WHERE subject_type = ? AND subject_id = ? ORDER BY id DESC`,
		subjectType, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Note
	for rows.Next() {
		var n model.Note
		var created, updated string
		if err := rows.Scan(&n.ID, &n.SubjectType, &n.SubjectID, &n.Body, &created, &updated); err != nil {
			return nil, err
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339, created)
		n.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) DeleteNote(id int) error {
	_, err := s.db.Exec(`DELETE FROM notes WHERE id = ?`, id)
	return err
}
```

- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `git add internal/store/notes.go internal/store/notes_test.go && git commit -m "feat(explore): notes store methods"`

### Task D2: notes endpoints (TDD)

- [ ] **Step 1: Failing test** `internal/api/notes_test.go`:
```go
package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"touchline/internal/api"
	"touchline/internal/store"
)

func TestNotesEndpoint(t *testing.T) {
	s, _ := store.Open(":memory:")
	t.Cleanup(func() { s.Close() })
	mux := http.NewServeMux()
	api.RegisterNotes(mux, api.Deps{Store: s})

	// create
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/notes", strings.NewReader(`{"subjectType":"team","subjectId":21,"body":"hi"}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST note = %d body=%s", rec.Code, rec.Body.String())
	}
	// list
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/notes?subjectType=team&subjectId=21", nil))
	if !strings.Contains(rec.Body.String(), `"body":"hi"`) {
		t.Fatalf("GET notes = %s", rec.Body.String())
	}
}
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** `internal/api/notes.go`:
```go
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"touchline/internal/model"
)

func RegisterNotes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/notes", func(w http.ResponseWriter, r *http.Request) {
		st := r.URL.Query().Get("subjectType")
		sid, _ := strconv.Atoi(r.URL.Query().Get("subjectId"))
		notes, err := deps.Store.NotesBySubject(st, sid)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, notes)
	})
	mux.HandleFunc("POST /api/notes", func(w http.ResponseWriter, r *http.Request) {
		var n model.Note
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil || n.SubjectType == "" {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		id, err := deps.Store.AddNote(n)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]int{"id": id})
	})
	mux.HandleFunc("DELETE /api/notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := deps.Store.DeleteNote(id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
```

- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `git add internal/api/notes.go internal/api/notes_test.go && git commit -m "feat(explore): notes endpoints"`

### Task D3: Explore page (frontend, TDD)

- [ ] **Step 1: Failing test** `web/src/features/explore/ExplorePage.test.tsx`:
```tsx
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import ExplorePage from './ExplorePage'
import type { Team, Venue } from '../../lib/types'

function wrap(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

test('lists teams and venues', () => {
  const qc = new QueryClient()
  qc.setQueryData<Team[]>(['teams'], [{ id: 1, name: 'Brazil', country: 'Brazil', group: 'F', crestUrl: '' }])
  qc.setQueryData<Venue[]>(['venues'], [{ id: 1, name: 'Estadio Azteca', city: 'Mexico City', country: 'Mexico', capacity: 87000 }])
  render(<ExplorePage />, { wrapper: wrap(qc) })
  expect(screen.getByText('Brazil')).toBeInTheDocument()
  expect(screen.getByText(/Estadio Azteca/)).toBeInTheDocument()
})
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** `web/src/features/explore/ExplorePage.tsx`:
```tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '../../lib/api'

export default function ExplorePage() {
  const { data: teams = [] } = useQuery({ queryKey: ['teams'], queryFn: api.teams })
  const { data: venues = [] } = useQuery({ queryKey: ['venues'], queryFn: api.venues })
  return (
    <div className="explore">
      <section>
        <h3>Teams</h3>
        <ul className="explore__teams">
          {teams.map((t) => <li key={t.id}>{t.name} <span className="muted">({t.group})</span></li>)}
        </ul>
      </section>
      <section>
        <h3>Venues</h3>
        <ul className="explore__venues">
          {venues.map((v) => <li key={v.id}>{v.name} — {v.city}, {v.country} ({v.capacity.toLocaleString()})</li>)}
        </ul>
      </section>
    </div>
  )
}
```
(`web/src/features/explore/api.ts` is only needed for notes write/read; create it if used by the page later. For this page the shared `api.teams`/`api.venues` suffice — do NOT create an unused api.ts.)

- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `git add web/src/features/explore && git commit -m "feat(explore): teams + venues browse page"`

---

# TASK W — Wiring (SERIAL — run only after all slices land)

**Files:** Modify `internal/api/api.go`, `internal/server/server.go` (recompute standings on read? no — see below), `web/src/app/Shell.tsx`.

### W1: Register all slice routes (backend)

- [ ] **Step 1:** In `internal/api/api.go`, at the end of `Register`, add calls to the slice registrars (all are in `package api`, same file set):
```go
	RegisterMatches(mux, deps)
	RegisterUserdata(mux, deps)
	RegisterStats(mux, deps)
	RegisterNotes(mux, deps)
```
(Note: `RegisterUserdata`/`RegisterStats`/`RegisterNotes` use only `deps.Store`; `RegisterMatches` uses `deps.Store`+`deps.Hot`. `api.Deps` already has both.)

- [ ] **Step 2:** Standings should reflect finished sim matches. In `internal/sim/simulator.go` `Step()`, after a match's status becomes `"finished"`, recompute + broadcast standings. Add — right after the `s.bc.Broadcast(... match.update ...)` line, inside the loop, guarded by finish:
```go
		if m.Status == "finished" {
			if err := s.store.RecomputeStandings(); err == nil {
				if rows, err := s.store.Standings(); err == nil {
					s.hot.SetStandings(rows)
					s.bc.Broadcast(sse.Message{Type: "standings.update", Data: rows})
				}
			}
		}
```
This makes `/api/standings` reflect results as matches finish. (This is the one cross-slice edit to the simulator; it depends on Slice C's `RecomputeStandings`, hence wiring is serial and last.)

- [ ] **Step 3:** Run `go build ./... && go vet ./... && go test -race ./...` → all pass (every slice's handler is now reachable + simulator recompute compiles).
- [ ] **Step 4:** Commit — `git add internal/api/api.go internal/sim/simulator.go && git commit -m "feat: register slice routes + standings recompute on match finish"`

### W2: Mount pages in the shell (frontend)

- [ ] **Step 1:** Replace `web/src/app/Shell.tsx`'s body section to render the active view's page. Add imports and a `view → component` map:
```tsx
import { useState } from 'react'
import { useLiveStream } from '../lib/useLiveStream'
import MatchCenterPage from '../features/match-center/MatchCenterPage'
import SchedulePage from '../features/schedule/SchedulePage'
import StandingsPage from '../features/standings/StandingsPage'
import ExplorePage from '../features/explore/ExplorePage'

const VIEWS = ['Match Center', 'Schedule', 'Standings', 'Explore'] as const
type View = (typeof VIEWS)[number]

const PAGES: Record<View, () => JSX.Element> = {
  'Match Center': MatchCenterPage,
  Schedule: SchedulePage,
  Standings: StandingsPage,
  Explore: ExplorePage,
}

export default function Shell() {
  const [view, setView] = useState<View>('Match Center')
  const { connected } = useLiveStream()
  const Page = PAGES[view]
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
      <section className="shell__body shell__body--page">
        <Page />
      </section>
    </main>
  )
}
```
**Note on `JSX.Element` type:** if the project's TS config doesn't expose the global `JSX` namespace, type the map as `Record<View, React.ComponentType>` and `import type { ComponentType } from 'react'`. Implementer: use whichever type-checks under `tsc --noEmit`; prefer `import type { ComponentType } from 'react'` and `Record<View, ComponentType>`.

- [ ] **Step 2:** The existing `App.test.tsx` still asserts the wordmark — but now `Shell` renders a page that calls `useQuery` (needs the QueryClientProvider, already present in `App`). The render test should still pass (queries just have no data). Run `cd web && npm test` → all feature page tests + App test pass.
- [ ] **Step 3:** `cd web && npm run build` → succeeds.
- [ ] **Step 4:** Commit — `git add web/src/app/Shell.tsx && git commit -m "feat: mount 4 feature pages in the view shell"`

---

## Final Verification (whole-phase)

- [ ] `go build ./... && go vet ./... && go test -race ./...` → all pass (api: matches/userdata/stats/notes; store: userdata/stats/notes; sim recompute).
- [ ] `cd web && npm test` → all feature pages + hook + App tests pass.
- [ ] `cd web && npm run build` → succeeds, `.gitkeep` preserved.
- [ ] Tree clean, no `*.db`.

### Deferred to user runtime gate
```bash
docker compose up --build   # or local build+run
# Match Center shows live matches ticking; Schedule lists fixtures by day;
# Standings tables fill as sim matches finish; Explore lists 48 teams + 16 venues.
# follow/unfollow + notes persist across restart (named volume).
curl -s localhost:8080/api/matches/1 ; curl -s localhost:8080/api/stats/scorers
curl -s -X POST localhost:8080/api/follows/21 ; curl -s localhost:8080/api/follows
```

## Spec Coverage (Phase 2 slices)
| Spec item | Slice/Task |
|---|---|
| Match Center: live score + event timeline | A1, A2 |
| Schedule by date/group; follows; reminders; timezone pref | B1, B2, B3 |
| Standings group tables; top scorers; recompute from results | C1, C2, C3, W1 |
| Explore: countries + venues; notes | D1, D2, D3 |
| Personal layer (follows/reminders/notes persisted) | B, D + Phase 1 volume |

## Out of scope (Phase 3)
Real APIFootball provider + `LIVE_SOURCE=real`; quota budgeter + stale "data delayed" badges; team-name mapping polish in Match Center/Schedule (currently `#id`); knockout bracket view; lineups (no player data seeded); full dark-broadcast visual + motion pass; nav cohesion; Playwright smoke E2E.
