# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Development with hot-reload (watches .go, .tpl, .html files)
air

# Manual build
go build -o ./tmp/main.exe ./cmd/web

# Run
./tmp/main.exe

# Run tests
go test ./...

# Lint
golangci-lint run ./...

# Docker (full stack with PostgreSQL)
docker compose up
```

Server starts on `http://localhost:8080` (configurable via `PORT` env var).

### Cross-Platform Notes

**Apple Silicon (M1/M2/M3 Macs)**:
- Dev container auto-detects arm64 and installs the correct Go binary
- `docker compose up` builds natively for arm64 — no emulation needed
- Playwright Chromium works on arm64 Linux within the dev container

**Windows**:
- Use WSL2 backend in Docker Desktop (default since Docker Desktop 4.x)
- Clone the repo inside WSL2 filesystem (`/home/...`) for best I/O performance
- Avoid cloning to `/mnt/c/...` (Windows filesystem) — volume mounts are significantly slower

**Multi-arch production images**:
```bash
# Build for amd64 + arm64 and push to a registry
REGISTRY=ghcr.io/yourorg/ TAG=v1.0.0 docker buildx bake app --push

# Build for local use only (native architecture)
docker buildx bake app-local
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | (empty) | PostgreSQL connection string. App runs without DB if unset. |
| `ROUND_DURATION` | `60` | Round duration in seconds |

## Architecture

Click Trainer is a real-time multiplayer target-clicking game built with **Go + HTMX + SSE**. Players create/join rooms with 4-character codes (Jackbox-style), ready up in a lobby, then click SVG targets in timed rounds to score points. Game data is optionally persisted in PostgreSQL.

### Key Packages (`internal/`)

- **server** — HTTP handlers (`handlers.go`), route setup (`routes.go`), analytics endpoints (`analytics_handlers.go`). Server struct holds rooms store, templates, optional DB, and click buffer.
- **broadcast** — `Broadcaster` struct for SSE fan-out. Each room gets its own broadcaster. Listens on events bus for scene changes.
- **rooms** — Room model and store. `GenerateCode()` produces 4-char codes using `crypto/rand`. Background goroutine sweeps stale rooms (1hr TTL).
- **players** — `Store` struct with thread-safe CRUD (map + mutex). Methods: Add, Get, GetList, UpdateScore, SetReady, AllReady, ValidateSession, ResetAll.
- **targets** — `Store` struct with auto-incrementing IDs. Targets have SpawnedAt timestamp for reaction time calculation.
- **gamedata** — `Game` struct holding player/target stores, event bus, config. Manages scene transitions and round lifecycle (StartRound, EndRound, ResetToLobby).
- **events** — `Bus` struct with buffered SceneChanges channel.
- **db** — PostgreSQL layer with embedded migrations. Connect, Migrate, UpsertPlayer, CreateGame, EndGame, RecordClick, BatchRecordClicks, AwardBadge.
- **analytics** — Badge definitions and evaluation, SQL queries for player stats, leaderboards, game recaps.
- **config** — Reads PORT, DATABASE_URL, ROUND_DURATION from environment.

### Game Flow

**Home → Create/Join Room → Join (enter name) → Lobby → Countdown → Combat (timed) → Recap → Play Again → Lobby**

### Route Structure

| Route | Purpose |
|-------|---------|
| `GET /` | Home: create or join room |
| `POST /rooms/create` | Create room |
| `POST /rooms/join` | Join room by code |
| `GET /room` | Game view |
| `POST /room/register` | Register player in room |
| `POST /room/ready` | Ready up |
| `POST /room/target/{id}/{points}` | Click target |
| `GET /room/events` | SSE stream |
| `POST /room/play-again` | Reset to lobby |
| `GET /health` | Health check |
| `GET /analytics` | Analytics dashboard |
| `GET /analytics/player/{id}` | Player profile |
| `GET /analytics/game/{id}` | Game recap |
| `GET /analytics/leaderboard` | Leaderboard (HTMX partial) |

### Communication Flow

1. **Client → Server**: HTMX `hx-post` form submissions
2. **Server → All Clients in Room**: Room-scoped SSE broadcasts via `Broadcaster.BroadcastOOB()`
3. Scene transitions broadcast through `events.Bus.SceneChanges` channel

### Frontend Stack

- **HTMX 2.0.4** with SSE extension for reactive updates without a SPA
- **Tailwind CSS 4** via CDN
- **Go `text/template`** for server-side HTML rendering
- SVG-based targets with CSS animations

### Entry Point

`cmd/web/main.go` — calls `server.Run()` which creates stores, optional DB, and starts HTTP server.

## UI Interaction with Playwright CLI

`@playwright/cli` is installed in the dev container for browser-based UI testing. It provides persistent browser sessions controlled via bash commands — ideal for verifying the HTMX/SSE-driven UI.

### Basic Commands

```bash
# Open the app (starts a persistent browser session)
playwright-cli open http://localhost:8080

# Take an accessibility snapshot (shows element refs for interaction)
playwright-cli snapshot

# Click an element by ref
playwright-cli click ref123

# Fill a text input by ref
playwright-cli fill ref456 "PlayerName"

# Take a screenshot (Claude can read the PNG)
playwright-cli screenshot

# Screenshot to a specific file — ALWAYS use screenshots/ prefix
playwright-cli screenshot --filename=screenshots/lobby.png
```

**IMPORTANT**: When using `--filename`, always prefix with `screenshots/` (e.g. `--filename=screenshots/combat.png`). Without the prefix, files land in the repo root and get committed accidentally. The `screenshots/` directory is gitignored.

### Multi-Player Testing with Sessions

Use named sessions (`-s`) to simulate multiple players in the same room:

```bash
# Player 1 creates a room
playwright-cli -s=player1 open http://localhost:8080
playwright-cli -s=player1 snapshot
playwright-cli -s=player1 click <create-room-ref>

# Player 2 joins the same room
playwright-cli -s=player2 open http://localhost:8080
playwright-cli -s=player2 fill <code-input-ref> "ABCD"
playwright-cli -s=player2 click <join-room-ref>
```

### Common Game Flow

1. `open http://localhost:8080` — land on home page
2. `snapshot` + `click` — create or join a room
3. `fill` + `click` — enter player name and register
4. `snapshot` — verify lobby state, check player list
5. `click` — ready up
6. `screenshot` — capture combat/recap scenes

Screenshots are saved to `screenshots/` (gitignored). Config is in `playwright-cli.json`.
