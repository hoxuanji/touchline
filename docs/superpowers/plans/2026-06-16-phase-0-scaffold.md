# Touchline Phase 0 — Scaffold Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the single-binary skeleton — a Go server that embeds and serves a styled, empty React/Vite SPA, with a health endpoint and the full test + Docker toolchain working — verifiable with `docker compose up`.

**Architecture:** A Go module (`touchline`) with a testable `internal/server` HTTP handler (Go 1.22+ `net/http` pattern routing) that serves a `/healthz` JSON endpoint and serves the built SPA via `go:embed`. The frontend is a React + TypeScript Vite app under `web/` that builds into `web/dist`, which the `web` Go package embeds. A multi-stage Dockerfile builds the SPA (node) → compiles a static CGO-free binary (go) → ships it on `scratch`. `docker compose up` is the one-command run.

**Tech Stack:** Go 1.26 (`net/http` routing, `embed`), React 18 + Vite 5 + TypeScript 5, Vitest + Testing Library, Docker multi-stage + Compose.

---

## File Structure

**Created in this phase:**

| Path | Responsibility |
|------|----------------|
| `go.mod` | Go module definition (`module touchline`, `go 1.26`). |
| `cmd/touchline/main.go` | Binary entry point: read `PORT`, build handler, `ListenAndServe`. |
| `internal/server/server.go` | `Handler()` — builds the mux: `/healthz` + embedded SPA serving. The testable unit. |
| `internal/server/server_test.go` | Unit test for `/healthz` and SPA root serving. |
| `web/embed.go` | `package web`; `//go:embed all:dist` exposing `Assets fs.FS`. Co-located with the build output. |
| `web/package.json` | Frontend deps + scripts (`dev`, `build`, `test`). |
| `web/tsconfig.json` | TypeScript config for the SPA. |
| `web/vite.config.ts` | Vite + React plugin + Vitest (jsdom) config; `base: './'` for embed-relative assets. |
| `web/index.html` | SPA HTML entry. |
| `web/src/main.tsx` | React bootstrap. |
| `web/src/App.tsx` | The styled empty shell component. |
| `web/src/index.css` | Dark-broadcast base theme. |
| `web/src/test/setup.ts` | Vitest setup (jest-dom matchers). |
| `web/src/App.test.tsx` | Render test for the shell. |
| `web/dist/.gitkeep` | Placeholder so `go:embed all:dist` compiles before any frontend build. |
| `Dockerfile` | Multi-stage: node build → go build (CGO disabled) → scratch. |
| `.dockerignore` | Keep build context lean. |
| `compose.yaml` | One-command run; maps `PORT`. |

**Modified:**

| Path | Change |
|------|--------|
| `.gitignore` | Add negations so `web/dist/.gitkeep` is tracked while built assets stay ignored. |

**Boundary note:** `internal/server.Handler()` returns an `http.Handler` and takes no args in Phase 0. Later phases will extend its signature (store, hot store, SSE hub) — keep all wiring inside `Handler()` so `main.go` stays a thin caller.

---

## Task 1: Go module + testable health handler (TDD)

**Files:**
- Create: `go.mod`
- Create: `web/embed.go`
- Create: `web/dist/.gitkeep`
- Create: `internal/server/server.go`
- Test: `internal/server/server_test.go`
- Modify: `.gitignore`

- [ ] **Step 1: Create the Go module file**

`go.mod`:
```
module touchline

go 1.26
```

- [ ] **Step 2: Create the embed placeholder so the build compiles**

Create an empty file `web/dist/.gitkeep` (no content needed):
```bash
mkdir -p web/dist && touch web/dist/.gitkeep
```

- [ ] **Step 3: Create the embed package**

`web/embed.go`:
```go
// Package web embeds the built SPA so the binary serves it with no external files.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Assets is the built SPA rooted at the dist directory.
var Assets fs.FS = mustSub(distFS, "dist")

func mustSub(f fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
```

Note: `all:dist` (the `all:` prefix) is required so the leading-dot `.gitkeep` is matched — without it, a dist directory containing only `.gitkeep` matches no files and the build fails with "no matching files found".

- [ ] **Step 4: Write the failing test**

`internal/server/server_test.go`:
```go
package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"touchline/internal/server"
)

func TestHealthz(t *testing.T) {
	h, err := server.Handler()
	if err != nil {
		t.Fatalf("Handler() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("body = %q, want it to contain %q", body, `"status":"ok"`)
	}
}
```

- [ ] **Step 5: Run the test to verify it fails**

Run: `go test ./internal/server/`
Expected: FAIL — compilation error, `undefined: server.Handler` (the package doesn't exist yet).

- [ ] **Step 6: Write the minimal implementation**

`internal/server/server.go`:
```go
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
```

- [ ] **Step 7: Run the test to verify it passes**

Run: `go test ./internal/server/`
Expected: PASS (`ok touchline/internal/server`).

- [ ] **Step 8: Make `web/dist/.gitkeep` trackable**

The global `dist/` rule in `.gitignore` hides `web/dist` entirely; the embed needs `.gitkeep` committed. Replace the `dist/` line in `.gitignore` with the block below (keeps generic `dist/` ignored, re-includes the embed target dir, ignores its built contents, but tracks `.gitkeep`):

`.gitignore` — change:
```
node_modules/
dist/
*.db
*.db-*
touchline
.env
.DS_Store
```
to:
```
node_modules/
dist/
!web/dist/
web/dist/*
!web/dist/.gitkeep
*.db
*.db-*
touchline
.env
.DS_Store
```

- [ ] **Step 9: Verify the gitkeep is tracked and tidy with `go vet`**

Run: `git check-ignore web/dist/.gitkeep; echo "exit=$?"`
Expected: prints nothing and `exit=1` (meaning the file is NOT ignored).

Run: `go vet ./...`
Expected: no output (clean).

- [ ] **Step 10: Commit**

```bash
git add go.mod web/embed.go web/dist/.gitkeep internal/server/server.go internal/server/server_test.go .gitignore
git add -f web/dist/.gitkeep
git commit -m "feat: Go module with embedded-SPA server and health endpoint"
```

---

## Task 2: Binary entry point

**Files:**
- Create: `cmd/touchline/main.go`

- [ ] **Step 1: Write `main.go`**

`cmd/touchline/main.go`:
```go
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
```

- [ ] **Step 2: Verify it builds**

Run: `go build ./...`
Expected: no output, exit 0. A `touchline` binary is NOT left in the repo (this builds without `-o`); confirm with `git status` showing no new tracked files.

- [ ] **Step 3: Verify it runs and serves health**

Run (in one line so the server is always stopped):
```bash
PORT=8099 go run ./cmd/touchline & SRV=$!; sleep 2; \
  curl -s -o /dev/null -w "%{http_code}" http://localhost:8099/healthz; echo; \
  curl -s http://localhost:8099/healthz; echo; \
  kill $SRV
```
Expected: prints `200` then `{"status":"ok"}`.

- [ ] **Step 4: Commit**

```bash
git add cmd/touchline/main.go
git commit -m "feat: binary entry point reading PORT and serving the handler"
```

---

## Task 3: Frontend scaffold — styled empty shell (TDD on render)

**Files:**
- Create: `web/package.json`
- Create: `web/tsconfig.json`
- Create: `web/vite.config.ts`
- Create: `web/index.html`
- Create: `web/src/main.tsx`
- Create: `web/src/App.tsx`
- Create: `web/src/index.css`
- Create: `web/src/test/setup.ts`
- Test: `web/src/App.test.tsx`

- [ ] **Step 1: Create `web/package.json`**

`web/package.json`:
```json
{
  "name": "touchline-web",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc --noEmit && vite build",
    "preview": "vite preview",
    "test": "vitest run"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1"
  },
  "devDependencies": {
    "@testing-library/jest-dom": "^6.4.8",
    "@testing-library/react": "^16.0.1",
    "@types/react": "^18.3.5",
    "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react": "^4.3.1",
    "jsdom": "^25.0.0",
    "typescript": "^5.5.4",
    "vite": "^5.4.2",
    "vitest": "^2.0.5"
  }
}
```

- [ ] **Step 2: Create `web/tsconfig.json`**

`web/tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "types": ["vitest/globals", "@testing-library/jest-dom"]
  },
  "include": ["src", "vite.config.ts"]
}
```

- [ ] **Step 3: Create `web/vite.config.ts`**

`web/vite.config.ts`:
```ts
/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// base: './' makes built asset URLs relative, so the SPA works when served
// from the embedded filesystem at the server root.
export default defineConfig({
  base: './',
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.ts',
  },
})
```

- [ ] **Step 4: Create `web/index.html`**

`web/index.html`:
```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Touchline · World Cup 2026</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 5: Create `web/src/main.tsx`**

`web/src/main.tsx`:
```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
```

- [ ] **Step 6: Create `web/src/App.tsx`**

`web/src/App.tsx`:
```tsx
export default function App() {
  return (
    <main className="shell">
      <header className="shell__bar">
        <span className="shell__mark" aria-hidden="true">▲</span>
        <h1 className="shell__title">TOUCHLINE</h1>
        <span className="shell__tag">WORLD CUP 2026</span>
      </header>
      <section className="shell__body">
        <p className="shell__note">Match center coming online…</p>
      </section>
    </main>
  )
}
```

- [ ] **Step 7: Create `web/src/index.css` (dark-broadcast base)**

`web/src/index.css`:
```css
:root {
  --bg: #0a0e0d;
  --panel: #111715;
  --line: #1e2825;
  --pitch: #00d26a;
  --text: #e8efe9;
  --muted: #6f827a;
  --font-display: "Arial Narrow", "Roboto Condensed", system-ui, sans-serif;
  --font-body: system-ui, -apple-system, "Segoe UI", sans-serif;
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  background: var(--bg);
  color: var(--text);
  font-family: var(--font-body);
  min-height: 100vh;
  -webkit-font-smoothing: antialiased;
}

.shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.shell__bar {
  display: flex;
  align-items: baseline;
  gap: 0.75rem;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--line);
  background: var(--panel);
}

.shell__mark {
  color: var(--pitch);
  font-size: 1.1rem;
}

.shell__title {
  font-family: var(--font-display);
  letter-spacing: 0.15em;
  font-size: 1.25rem;
  font-weight: 700;
}

.shell__tag {
  color: var(--muted);
  font-size: 0.7rem;
  letter-spacing: 0.2em;
}

.shell__body {
  flex: 1;
  display: grid;
  place-items: center;
}

.shell__note {
  color: var(--muted);
  letter-spacing: 0.05em;
}
```

- [ ] **Step 8: Create the Vitest setup file**

`web/src/test/setup.ts`:
```ts
import '@testing-library/jest-dom'
```

- [ ] **Step 9: Write the failing render test**

`web/src/App.test.tsx`:
```tsx
import { render, screen } from '@testing-library/react'
import App from './App'

test('renders the Touchline shell title and tournament tag', () => {
  render(<App />)
  expect(screen.getByText('TOUCHLINE')).toBeInTheDocument()
  expect(screen.getByText('WORLD CUP 2026')).toBeInTheDocument()
})
```

- [ ] **Step 10: Install dependencies**

Run: `cd web && npm install`
Expected: creates `web/node_modules` and `web/package-lock.json`; exit 0.

- [ ] **Step 11: Run the frontend test to verify it passes**

Run: `cd web && npm test`
Expected: Vitest reports `1 passed` (the App render test).

- [ ] **Step 12: Verify type-check + production build succeed**

Run: `cd web && npm run build`
Expected: exit 0; `web/dist/index.html` and hashed assets under `web/dist/assets/` are produced. (These are gitignored except `.gitkeep`.)

- [ ] **Step 13: Commit**

```bash
git add web/package.json web/package-lock.json web/tsconfig.json web/vite.config.ts web/index.html web/src
git commit -m "feat: React+Vite+TS dark-broadcast empty shell with Vitest"
```

---

## Task 4: Verify the binary serves the real built SPA

This confirms the `go:embed` → server → SPA path works end-to-end outside Docker before adding container complexity.

**Files:** none created (verification only).

- [ ] **Step 1: Build the frontend so `web/dist` holds real assets**

Run: `cd web && npm run build`
Expected: `web/dist/index.html` exists.

- [ ] **Step 2: Run the binary and confirm it serves the SPA HTML**

Run:
```bash
PORT=8099 go run ./cmd/touchline & SRV=$!; sleep 2; \
  curl -s http://localhost:8099/ | grep -o '<div id="root">'; \
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8099/healthz; \
  kill $SRV
```
Expected: prints `<div id="root">` then `200`.

- [ ] **Step 3: No commit**

Verification only — `web/dist` contents are gitignored. Nothing to commit.

---

## Task 5: Dockerfile + .dockerignore (multi-stage static binary)

**Files:**
- Create: `Dockerfile`
- Create: `.dockerignore`

- [ ] **Step 1: Create `.dockerignore`**

`.dockerignore`:
```
.git
**/node_modules
web/dist
*.db
*.db-*
.env
.DS_Store
docs
```
(`web/dist` is rebuilt inside the image; excluding it keeps the build context small and avoids stale assets.)

- [ ] **Step 2: Create the multi-stage `Dockerfile`**

`Dockerfile`:
```dockerfile
# syntax=docker/dockerfile:1

# 1) Build the SPA.
FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 2) Build the static Go binary with the freshly built SPA embedded.
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /touchline ./cmd/touchline

# 3) Ship just the binary.
FROM scratch
COPY --from=build /touchline /touchline
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/touchline"]
```

Notes for the engineer:
- `go mod download` after copying only `go.mod` (no `go.sum` exists yet — Phase 0 has no third-party deps) caches modules across rebuilds. When dependencies are added in Phase 1, also `COPY go.sum ./` before download.
- `scratch` is fine for Phase 0 (no outbound TLS). When real APIFootball HTTPS calls are added later, switch the final stage to `gcr.io/distroless/static:nonroot` (it includes CA certs) or `COPY` certs in.

- [ ] **Step 3: Build the image**

Run: `docker build -t touchline:dev .`
Expected: build succeeds; final line shows the image tagged `touchline:dev`.

- [ ] **Step 4: Run the container and verify both routes**

Run:
```bash
docker run -d --rm -p 8099:8080 --name touchline-dev touchline:dev; sleep 2; \
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8099/healthz; \
  curl -s http://localhost:8099/ | grep -o '<div id="root">'; \
  docker stop touchline-dev
```
Expected: prints `200` then `<div id="root">`.

- [ ] **Step 5: Commit**

```bash
git add Dockerfile .dockerignore
git commit -m "feat: multi-stage Dockerfile building static single binary"
```

---

## Task 6: Compose — the one-command run (Phase 0 verify gate)

**Files:**
- Create: `compose.yaml`

- [ ] **Step 1: Create `compose.yaml`**

`compose.yaml`:
```yaml
services:
  touchline:
    build: .
    ports:
      - "${PORT:-8080}:8080"
    environment:
      - PORT=8080
```
(No DB volume yet — SQLite lands in Phase 1, which will add a named volume for `DB_PATH`.)

- [ ] **Step 2: Bring the stack up**

Run: `docker compose up --build -d`
Expected: builds and starts the `touchline` service; exit 0.

- [ ] **Step 3: PHASE 0 SUCCESS GATE — the app serves a styled empty shell**

Run:
```bash
sleep 2
curl -s -o /dev/null -w "healthz=%{http_code}\n" http://localhost:8080/healthz
curl -s http://localhost:8080/ | grep -o '<div id="root">'
curl -s http://localhost:8080/healthz; echo
```
Expected:
```
healthz=200
<div id="root">
{"status":"ok"}
```
Then open `http://localhost:8080/` in a browser: a dark page with a green `▲`, the condensed `TOUCHLINE` wordmark, the `WORLD CUP 2026` tag, and the muted "Match center coming online…" line. This is the Phase 0 acceptance per the spec ("`docker compose up` serves an empty styled shell").

- [ ] **Step 4: Tear down**

Run: `docker compose down`
Expected: stops and removes the container.

- [ ] **Step 5: Commit**

```bash
git add compose.yaml
git commit -m "feat: docker compose one-command run (Phase 0 gate)"
```

---

## Final Verification (whole-phase)

- [ ] **All Go tests pass:** `go test ./...` → `ok` for `touchline/internal/server` (no failures).
- [ ] **Go vet clean:** `go vet ./...` → no output.
- [ ] **Frontend tests pass:** `cd web && npm test` → `1 passed`.
- [ ] **Frontend builds:** `cd web && npm run build` → exit 0.
- [ ] **One-command run works:** `docker compose up --build` → `http://localhost:8080/healthz` returns `{"status":"ok"}` and `/` serves the styled shell.
- [ ] **Repo is clean:** `git status` shows no untracked build artifacts (only `.gitkeep` tracked under `web/dist`; `node_modules`, built `dist` assets, and any binary are ignored).

---

## Spec Coverage Check (Phase 0 scope only)

| Spec Phase 0 requirement | Task |
|---|---|
| Go module | Task 1 |
| Vite app (React + TS) | Task 3 |
| `go:embed` of built SPA | Task 1 (embed) + Task 4 (verify) |
| Dockerfile + compose | Tasks 5, 6 |
| Health check | Task 1 |
| Base test setup (Go + frontend) | Task 1 (Go), Task 3 (Vitest) |
| Verify: `docker compose up` serves a styled empty shell | Task 6 Step 3 |

Playwright smoke E2E is intentionally deferred — per the spec's testing strategy it drives a *live simulated match*, which doesn't exist until Phase 2; it belongs in Phase 3.

---

## Out of Scope (later phases — do NOT build here)

- Domain model, SQLite migrations/store, snapshot seed loader, Provider interface, REST API, SSE hub → **Phase 1** (`phase-1-core.md`).
- **WC2026 snapshot data assembly** is an explicit build task in the Phase 1 plan (teams, venues, group draw, 104-match fixture skeleton), per the agreed decision.
- Match Center, Schedule/Calendar, Standings, Explore, follows/reminders/notes → **Phase 2** (four parallel slice plans).
- Navigation/theme cohesion, motion pass, stale badges, quota guardrails, real-API flip → **Phase 3**.
