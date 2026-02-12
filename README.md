# Click Trainer

A real-time multiplayer target-clicking game built with Go, HTMX, and Server-Sent Events. Players create or join rooms using 4-character codes (Jackbox-style), ready up in a lobby, then race to click SVG targets during timed rounds. Scores, reaction times, and badges are tracked in an optional PostgreSQL database.

## How It Works

1. **Create or join a room** -- one player creates a room and shares the 4-character code with friends.
2. **Enter your name** and land in the lobby.
3. **Ready up** -- the round starts with a countdown once every player is ready.
4. **Click targets** -- colored circles appear on the game board for 60 seconds (configurable). Smaller targets are worth more points. Click fast to earn bonus points for quick reactions.
5. **See the recap** -- scores are ranked and badges are awarded. Hit "Play Again" to return to the lobby.

Game state is synchronized across all players in a room via SSE, so everyone sees targets appear, scores update, and scene transitions in real time.

## Running the Game

### Quick start with Docker

The fastest way to run the full stack (app + PostgreSQL) with no local toolchain required:

```bash
docker compose up -d
```

Open `http://localhost:8080` in your browser.

### Run from source

Requires [Go 1.24+](https://go.dev/dl/).

```bash
go build -o ./tmp/main ./cmd/web
./tmp/main
```

The server starts on `http://localhost:8080`. Without a `DATABASE_URL`, the app runs fully in-memory -- games work, but analytics and persistence are disabled.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | *(empty)* | PostgreSQL connection string. App degrades gracefully without it. |
| `ROUND_DURATION` | `60` | Round duration in seconds |

## Tech Stack

- **Go** standard library HTTP server (no framework)
- **HTMX 2.0** with the SSE extension for reactive updates without a SPA
- **Tailwind CSS 4** via CDN
- **PostgreSQL 16** for optional persistence (player stats, game history, badges, leaderboards)
- **Go `text/template`** for server-side HTML rendering
- SVG-based targets with CSS animations

---

## Development

### Dev Container (recommended)

The repo includes a full [Dev Container](https://containers.dev/) configuration. Open the project in VS Code and select **"Reopen in Container"** (or use GitHub Codespaces). The container provides:

- Go 1.24 with `air` (hot-reload) and `golangci-lint` pre-installed
- PostgreSQL 16 running as a sidecar with `DATABASE_URL` and `TEST_DATABASE_URL` pre-configured
- VS Code extensions for Go, HTMX, Tailwind, and Docker

Once inside the container:

```bash
# Start the server with hot-reload (rebuilds on .go/.html changes)
air

# Run all tests (DB tests use TEST_DATABASE_URL automatically)
go test ./...

# Lint
golangci-lint run ./...
```

### Local development

If you prefer working outside the container:

```bash
# Install dev tools
go install github.com/air-verse/air@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Hot-reload
air

# Tests
go test ./...

# Lint
golangci-lint run ./...
```

To run with a local PostgreSQL, set `DATABASE_URL`:

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/clicktrainer?sslmode=disable"
```

### Project structure

```
cmd/web/            Entry point (main.go)
internal/
  server/           HTTP handlers, routes, SSE, analytics endpoints
  broadcast/        Room-scoped SSE fan-out
  rooms/            Room model, store, code generation, stale room sweeper
  players/          Thread-safe player CRUD
  targets/          Target store with auto-incrementing IDs
  gamedata/         Game state, scene transitions, round lifecycle
  events/           Event bus for scene changes
  db/               PostgreSQL layer with embedded migrations
  analytics/        Badge evaluation, stats queries, leaderboards
  config/           Environment variable loading
  utility/          Shared helpers (color generation)
templates/          Go text/template HTML files
  analytics/        Dashboard, leaderboard, player/game detail pages
static/             CSS, SVGs, favicon
docs/               Design assets
```

### Running tests

```bash
# All tests (DB tests skip automatically when TEST_DATABASE_URL is unset)
go test ./...

# Verbose
go test -v ./...

# Single package
go test ./internal/players/
```
