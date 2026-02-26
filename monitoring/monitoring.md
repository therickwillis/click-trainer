# Click Trainer Monitoring

The monitoring stack runs automatically inside the dev container (activated via `COMPOSE_PROFILES=monitoring` in `.devcontainer/.env`). It is also available via `docker compose --profile monitoring up` from the project root.

## Services

| Service | URL | Credentials | Purpose |
|---------|-----|-------------|---------|
| **Grafana** | http://localhost:3000 | `admin` / `clicktrainer` | Dashboards, log viewer |
| **Prometheus** | http://localhost:9090 | — | Metrics store, PromQL REPL |
| **cAdvisor** | http://localhost:8081 | — | Raw container resource metrics |
| **Loki** | http://localhost:3100 | — | Log aggregation API (no UI) |

Prometheus and Loki are pre-wired as Grafana datasources — no configuration needed.

---

## Grafana Dashboards

Open Grafana → hamburger menu → **Dashboards** → folder **Click Trainer**.

### Game Overview

**Refresh:** every 10 s | **Default range:** last 1 hour

| Section | Panels |
|---------|--------|
| **Live Status** | Active Rooms, Active Players, SSE Connections, WS Connections |
| **HTTP Traffic** | Request Rate by Path (req/s), P99 Request Latency |
| **Game Lifecycle** | Games Started/hr, Games Completed/hr, Game Duration Distribution (p50/p95) |
| **Combat** | Click Rate by Points, Reaction Time P50/P75/P95, Targets Spawned vs Killed |
| **Buffer & DB** | Click Buffer Depth, Batch Flush Rate, DB Write Errors by Operation |
| **SSE / WebSocket** | SSE Message Rate by Event Type, WS/SSE Active Connections |
| **Frontend** | JS Error Rate, Click Latency P95, WS Connect/Disconnect/Reconnect, Scene Transitions, LCP/CLS Web Vitals |
| **Go Runtime** | Goroutines, GC Pause Duration, Heap In Use, Heap Alloc Rate |

### System

**Refresh:** every 30 s | **Default range:** last 1 hour

| Section | Panels |
|---------|--------|
| **Container Resources** | CPU Usage per Container, Memory Usage per Container, Network I/O per Container, Disk I/O per Container |
| **Application Logs** | App Logs (all levels), Error Logs only, DB component logs |

> Dashboards are provisioned from `monitoring/grafana/dashboards/` and are read-only in the UI (`disableDeletion: true`, `allowUiUpdates: false`). Edit the JSON files on disk and Grafana picks up changes within 30 seconds.

---

## Metrics Reference

Metrics are exposed at `GET /metrics` (Prometheus format) and scraped every 15 seconds.

### HTTP

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `http_requests_total` | Counter | `method`, `path`, `status_code` | Total HTTP requests |
| `http_request_duration_seconds` | Histogram | `method`, `path` | Request duration |

### Rooms & Players

| Metric | Type | Description |
|--------|------|-------------|
| `rooms_created_total` | Counter | Total rooms ever created |
| `rooms_active` | Gauge | Current open rooms |
| `players_registered_total` | Counter | Total player registrations |
| `players_active` | Gauge | Players currently in rooms |

### Game Lifecycle

| Metric | Type | Description |
|--------|------|-------------|
| `games_started_total` | Counter | Rounds that entered Combat |
| `games_completed_total` | Counter | Rounds that reached Recap |
| `game_duration_seconds` | Histogram | Round duration (buckets: 30s–300s) |

### Combat

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `clicks_processed_total` | Counter | `points`, `source` | Clicks handled by the server |
| `targets_spawned_total` | Counter | — | Targets created during rounds |
| `targets_killed_total` | Counter | — | Targets removed by player clicks |
| `reaction_time_milliseconds` | Histogram | — | Time from target spawn to click (buckets: 100ms–2000ms) |
| `click_buffer_depth` | Gauge | — | Pending clicks in the write buffer |
| `click_batch_flushes_total` | Counter | — | Batch writes to PostgreSQL |
| `db_write_errors_total` | Counter | `operation` | DB write failures by operation |

### SSE / WebSocket

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `sse_connections_active` | Gauge | — | Open SSE streams |
| `sse_messages_published_total` | Counter | `event_type` | SSE events sent to clients |
| `ws_connections_active` | Gauge | — | Open WebSocket connections |

### Frontend (via `POST /telemetry`)

The browser batches and POSTs these events to the server, which records them as Prometheus metrics and structured log entries.

| Metric | Type | Description |
|--------|------|-------------|
| `frontend_js_errors_total` | Counter | Uncaught JS exceptions |
| `frontend_ws_connects_total` | Counter | WS connect events |
| `frontend_ws_disconnects_total` | Counter | WS disconnect events |
| `frontend_ws_reconnects_total` | Counter | WS reconnect attempts |
| `frontend_sse_connects_total` | Counter | SSE connect events |
| `frontend_sse_errors_total` | Counter | SSE error events |
| `frontend_click_latency_ms` | Histogram | Click → target-removal RTT (buckets: 16ms–500ms) |
| `frontend_scene_transitions_total` | Counter | Scene changes, labelled `from_scene`/`to_scene` |
| `frontend_lcp_seconds` | Histogram | Largest Contentful Paint (buckets: 0.5s–6s) |
| `frontend_cls_score` | Histogram | Cumulative Layout Shift (buckets: 0.01–0.5) |

---

## Logs

Promtail ships logs from all Docker containers to Loki by tailing the Docker daemon socket. It automatically applies these labels to every log line:

| Label | Source |
|-------|--------|
| `container` | Docker container name |
| `service` | `com.docker.compose.service` label |
| `level` | Parsed from JSON log field `level` |

The app writes structured JSON logs via `log/slog`. Additional fields parsed and indexed as labels: `level`, `service`, `container`. Fields available for filtering (not indexed): `room_code`, `player_id`, `game_id`, `handler`, `error`.

### Querying Logs in Grafana

Go to **Explore** → select the **Loki** datasource.

```logql
# All app logs
{service="app"}

# Errors only
{service="app", level="error"}

# Logs for a specific room
{service="app"} | json | room_code = "ABCD"

# DB component logs
{service="app"} | json | component = "db"

# Slow requests (parse json and filter by field)
{service="app"} | json | handler = "handleTargetClick"

# Log rate over time (used in dashboards)
sum(rate({service="app"}[1m]))
```

---

## Prometheus: Ad-hoc PromQL

Go to http://localhost:9090 → **Graph** tab to run raw queries.

```promql
# Current click throughput
sum(rate(clicks_processed_total[1m]))

# P95 reaction time over last 5 minutes
histogram_quantile(0.95, rate(reaction_time_milliseconds_bucket[5m]))

# HTTP error rate
sum(rate(http_requests_total{status_code=~"5.."}[1m]))

# Click buffer saturation (0–1 relative to cap of 1000)
click_buffer_depth / 1000

# Targets remaining alive (spawned but not killed)
targets_spawned_total - targets_killed_total

# DB write error rate by operation
sum by(operation) (rate(db_write_errors_total[5m]))
```

Use `Prometheus` → **Status** → **Targets** to confirm both scrape jobs (`clicktrainer` and `cadvisor`) are UP.

---

## cAdvisor

http://localhost:8081 shows a per-container breakdown of CPU, memory, network, and disk in its own minimal UI. It is also the source for all `container_*` metrics in the System Grafana dashboard.

---

## Adding a New Dashboard

1. Build it interactively in Grafana Explore or as a new dashboard panel.
2. Export the JSON: dashboard menu → **Share** → **Export** → **Save to file**.
3. Drop the file in `monitoring/grafana/dashboards/`. Grafana reloads it within 30 seconds.
4. Commit the JSON — dashboards are version-controlled alongside the code.

## Adding a New Metric

1. Add the field to the `Metrics` struct in `internal/metrics/metrics.go`.
2. Register it with `promauto.New*` in `New()`.
3. Inject `s.Metrics` into the handler or package where you want to record it and call `.Inc()` / `.Observe()` / `.Set()`.
4. Add a panel to `monitoring/grafana/dashboards/game-overview.json` (or create a new dashboard).
