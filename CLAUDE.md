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
```

Server starts on `http://localhost:8080`. No test suite or linter is currently configured.

## Architecture

Click Trainer is a real-time multiplayer target-clicking game built with **Go + HTMX + SSE**. Players join, ready up in a lobby, then click SVG targets to score points. All state is in-memory (no database).

### Key Packages (`internal/`)

- **server** — HTTP handlers, route setup, and SSE broadcasting. Templates are loaded in `routes.go`; game logic (register, ready, target clicks) lives in `handlers.go`; real-time push via `sse.go`.
- **players** — Player model and thread-safe in-memory store (map + mutex).
- **targets** — Target model and thread-safe in-memory store. Targets have 4 concentric click zones worth different points.
- **gamedata** — Aggregates current scene, players, and targets into a single struct for template rendering.
- **events** — Channel-based event bus (e.g., scene changes) consumed by SSE broadcaster.

### Communication Flow

1. **Client → Server**: HTMX `hx-post` form submissions (register, ready, target click)
2. **Server → All Clients**: SSE broadcasts via `BroadcastOOB()` push HTML fragments with HTMX out-of-band swaps
3. Scene transitions (join → lobby → combat) are broadcast through the `events.SceneChanges` channel

### Frontend Stack

- **HTMX 2.0.4** with SSE extension for reactive updates without a SPA
- **Tailwind CSS 4** via CDN
- **Go `text/template`** for server-side HTML rendering (templates in `templates/`)
- SVG-based targets with CSS animations (styles in `static/styles.css`)

### Entry Point

`cmd/web/main.go` — seeds 3 initial targets, then starts the HTTP server.
