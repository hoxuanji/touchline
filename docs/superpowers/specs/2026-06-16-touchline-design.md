# Touchline — Design Spec

**Date:** 2026-06-16
**Status:** Draft for review
**Working name:** Touchline ("where the manager and commentator stand and make their notes")

---

## 1. Context & Problem

The FIFA World Cup 2026 (48 teams, 104 matches, 16 venues across USA / Canada / Mexico)
is underway. A fan/analyst wants **one** application that aggregates everything about the
tournament — live scores, schedules, standings, stats, and a reference encyclopedia of
countries, squads, players, and stadiums — and doubles as a **personal companion**: a
calendar, a tracker, a score-checker, and a place to keep "commentator notes" before and
during matches.

Existing score apps are generic and multi-sport; generic web-app scaffolds feel static and
CRUD-like. We want a **professional, minimal, football-themed UI with genuine real-time
behavior**, on a **light backend** that is easy to build, runs on any machine, and is
production-grade.

**Intended outcome:** a single small binary (frontend embedded) that runs with one command,
serves a live, broadcast-quality World Cup companion, and degrades gracefully when live data
is constrained.

## 2. Goals & Non-Goals

**Goals**
- Live match center: scores, minute-by-minute events, lineups, in-play stats.
- Schedule / calendar: full fixtures, group + knockout views, user timezone, follow teams.
- Standings & stats: group tables, top scorers, team/player stats, knockout bracket.
- Explore hub: country profiles, squads, player bios, venue info ("commentator notes" source).
- Personal layer: followed teams, reminders, and free-text notes attached to matches/teams/players.
- Light, fast, single-binary, `docker compose up`, runs anywhere.
- Works fully even on a **free API tier** (and offline) via simulation + bundled snapshot.

**Non-Goals (V1)**
- No multi-user accounts / auth (single-user personal companion; preferences are local to the deployment).
- No multi-device cloud sync.
- No betting/odds, no video, no social feed.
- No coverage of competitions other than WC 2026.

## 3. Users & Key Use Cases

Primary user: a football fan/analyst tracking the tournament for themselves.

- "What's live right now?" → open Match Center, see live scores + events updating in real time.
- "When does my team play, in my timezone?" → Schedule, follow team, get reminders.
- "Who's topping Group X / who are the top scorers?" → Standings & Stats.
- "Tell me about this country / player / stadium." → Explore hub.
- "Let me jot a note before the match." → personal notes attached to a match/team/player.

## 4. Scope — V1 Feature Areas

1. **Match Center (live):** list of today's/live matches; match detail with live score, event
   timeline (goals, cards, subs, VAR), lineups/formations, live team stats. Real-time updates.
2. **Schedule & Calendar:** all fixtures by date / group / stage; knockout view; timezone-aware
   times; follow-team filter; reminders for followed teams' upcoming matches.
3. **Standings & Stats:** group tables (pts, GD, form), knockout bracket progression, top scorers
   / assists, team and player stat leaders.
4. **Explore Hub:** countries (profile, squad list), players (bio, stats), venues (city, capacity,
   matches hosted). The "commentator notes" reference layer.
5. **Personal layer (cross-cutting):** follows, reminders, and free-text notes — persisted in the DB.

## 5. Architecture

Single Go binary (production) that embeds and serves the built React SPA, exposes a JSON REST
API for initial page loads, an SSE endpoint for live deltas, and runs a background ingest worker.

```
 React SPA  ──REST (initial load)──▶  Go server ──▶ SQLite (cache + user data)
    │                                    │  ▲              ▲
    └────────── SSE (live deltas) ───────┘  │              │
                                            │         ingest worker
                                  in-memory hot store    │
                                            ▲            ▼
                                            └──────  Provider
                                                     ├─ APIFootball (real; reference data now, live when paid)
                                                     ├─ Simulator   (live matches on free tier)
                                                     └─ Snapshot    (bundled WC2026 JSON: seed + fallback)
```

- **No external services.** SQLite (pure-Go `modernc.org/sqlite`, no cgo) keeps the single
  static binary and "runs anywhere" promise. Used for: persistent cache of reference data
  (survives restarts → saves API quota), user data (follows, timezone, reminders, notes),
  and the daily request-budget counter.
- **In-memory hot store** holds the current normalized state for fast reads + SSE fan-out;
  hydrated from SQLite on boot, persisted on change.
- **Provider interface** with two real source implementations (APIFootball, Simulator) and a
  Snapshot loader used as seed + fallback. `LIVE_SOURCE=sim|real` selects the live source.
- **SSE hub** fans out one upstream poll to all connected browsers — API cost is independent
  of the number of users/tabs.

### Backend layout (Go)
- `cmd/touchline/main.go` — entry, wiring, `go:embed` of the built SPA.
- `internal/model/` — domain types: Match, Team, Player, Venue, Standing, Event, Note, Follow.
- `internal/provider/` — `Provider` interface; `apifootball.go`, `simulated.go`; `snapshot.go` loader.
- `internal/store/` — SQLite access (migrations, queries) + in-memory hot store.
- `internal/ingest/` — budgeted poller (reference data) + match simulator clock/event engine.
- `internal/sse/` — subscriber hub, event encoding.
- `internal/api/` — handlers + router (Go 1.22+ `net/http` pattern routing; no router dep).
- `data/snapshot/*.json` — bundled WC2026 structure (teams, venues, group/fixture skeleton).

### Frontend (React + Vite + TypeScript)
- Pages: **Match Center**, **Schedule/Calendar**, **Standings & Stats**, **Explore**, plus a
  cross-cutting **Notes/Follows** layer.
- Data: TanStack Query for REST; a small `useLiveStream` EventSource hook that patches the
  query cache / store on SSE messages; auto-reconnect with backoff.
- Preferences/notes: written through the REST API to SQLite (so they persist + could later sync).
- **Theme: Dark broadcast** — stadium-night palette, pitch-green accents, condensed scoreboard
  display type + clean body font, generous whitespace, smooth score/event transitions (CSS
  transitions first; a motion lib only if CSS can't do it).

## 6. Data Model (SQLite tables, normalized from provider)

- `teams` (id, name, country, group, crest_url, …)
- `venues` (id, name, city, country, capacity, …)
- `players` (id, team_id, name, position, number, bio fields, …)
- `matches` (id, stage, group, venue_id, home_id, away_id, kickoff_utc, status, minute,
  home_score, away_score, …)
- `match_events` (id, match_id, minute, type, team_id, player_id, detail)
- `lineups` (match_id, team_id, player_id, role, position)
- `standings` (group, team_id, played, won, drawn, lost, gf, ga, pts, form)
- `player_stats` (player_id, goals, assists, minutes, cards, …)
- `meta_cache` (resource, fetched_at, ttl) — drives cache freshness + budget.
- **User layer:** `follows` (team_id), `reminders` (match_id, lead_minutes, fired),
  `notes` (id, subject_type, subject_id, body, created_at, updated_at), `prefs` (key, value: timezone, etc.)

## 7. Data Sourcing & Quota Strategy (free tier reality)

The free API tier (~100 req/day) cannot sustain live polling, so data is split:

- **Reference data** (fixtures, standings, teams, venues, players): fetched from APIFootball,
  **persisted to SQLite** and cached with long TTLs (fixtures 6–12h, standings 1–3h, teams/
  venues ~permanent w/ manual refresh). A **daily request budget** (`DAILY_REQUEST_BUDGET`,
  persisted counter) hard-caps calls so quota can't be exceeded. The bundled **snapshot**
  seeds the DB on first run so the app is fully usable at zero quota / offline.
- **Live matches:** driven by the **simulator** on the free tier (realistic clock, goals,
  cards, subs, momentum) so the entire live UX works end-to-end today. Setting
  `LIVE_SOURCE=real` flips to real live polling with no code change when a paid tier is added.

> Honest limitation: without paid API access, real fixture dates/results are not guaranteed;
> the snapshot is structural/seed data and real values flow in from the API as quota allows.

## 8. Real-Time Design

- SSE endpoint `/api/stream`; the hub maintains subscribers and broadcasts typed messages:
  `match.update` (score/minute/status), `match.event` (new timeline event), `standings.update`.
- Live source (simulator or real poller) updates the hot store + SQLite, then broadcasts deltas.
- Client subscribes per relevant view, patches local state, animates changes; reconnects with backoff.

## 9. Resilience & Error Handling

- Provider failure → serve cache/snapshot; show a subtle "data delayed · <timestamp>" badge.
- Quota exhausted → stop real calls, keep serving cache; log; badge stays until window resets.
- SSE disconnect → client backoff reconnect; REST refetch on reconnect to resync.
- The app is **never broken** — worst case it is slightly stale.

## 10. Packaging & Deployment

- Multi-stage Dockerfile: build SPA (node) → embed in Go build → small final image (distroless/scratch).
- `docker compose up` = one command. The same binary also runs directly (`./touchline`).
- Config via env: `API_FOOTBALL_KEY`, `LIVE_SOURCE` (sim|real), `DAILY_REQUEST_BUDGET`, `PORT`,
  `DB_PATH`.

## 11. Testing Strategy

- **Go unit tests:** provider normalization, quota budgeter, SSE hub fan-out, simulator
  clock/event generation, SQLite migrations + queries (table-driven).
- **Frontend tests (Vitest + Testing Library):** score/timeline rendering, SSE reducer/patching.
- **Smoke E2E (Playwright):** drive a live match through the simulator and assert the score
  visibly updates in the UI via SSE.
- **Manual E2E:** `docker compose up` → open app → simulator drives a live score → observe update.

## 12. Build Phases & Parallelization Strategy

Dependencies force a serial core; independent feature areas then fan out in parallel via a
Workflow of subagents (multi-agent), integrated and polished at the end.

- **Phase 0 — Scaffold (serial):** Go module, Vite app, `go:embed`, Dockerfile + compose,
  health check, base test setup. Verify: `docker compose up` serves an empty styled shell.
- **Phase 1 — Core (serial):** domain model, SQLite migrations + store, snapshot seed loader,
  Provider interface, REST skeleton, SSE hub. Verify: REST returns seeded fixtures; SSE connects.
- **Phase 2 — Feature fan-out (parallel subagents):** four independent slices built concurrently:
  (a) Match Center + simulator, (b) Schedule/Calendar + follows/reminders, (c) Standings & Stats,
  (d) Explore hub + notes. Each consumes the Phase-1 contracts; each has its own verify step.
- **Phase 3 — Integrate & polish (serial):** wire navigation/theme cohesively, dark-broadcast
  visual pass + motion, stale badges, quota guardrails, real-API live flip path, full test pass.
  Verify: all tests green; manual E2E across all four areas + live simulator.

## 13. Success Criteria

- `docker compose up` on a clean machine serves the full app at `localhost:<PORT>`.
- All four areas usable with bundled snapshot at zero API quota.
- A simulated live match updates score + timeline in the browser in real time via SSE.
- Following a team and adding a note/reminder persists across restarts (SQLite).
- Setting `LIVE_SOURCE=real` with a paid key polls real live data with no code change.
- Unit + frontend + smoke tests pass.

## 14. Risks & Open Questions

- **Real WC2026 data accuracy** depends on API tier; snapshot is structural until then.
- **Simulator realism** is "believable", not real results — clearly labeled as simulated in dev.
- APIFootball endpoint/field shapes must be confirmed against live docs during Phase 1.
- Snapshot must be assembled from public WC2026 structure (teams/venues/schedule skeleton).
