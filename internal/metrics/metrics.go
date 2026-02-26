package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metric definitions for Click Trainer.
type Metrics struct {
	// HTTP
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// SSE / WebSocket
	SSEConnectionsActive    prometheus.Gauge
	SSEMessagesPublished    *prometheus.CounterVec
	WSConnectionsActive     prometheus.Gauge

	// Rooms & Players
	RoomsCreatedTotal    prometheus.Counter
	RoomsActive          prometheus.Gauge
	PlayersRegisteredTotal prometheus.Counter
	PlayersActive        prometheus.Gauge

	// Game lifecycle
	GamesStartedTotal    prometheus.Counter
	GamesCompletedTotal  prometheus.Counter
	GameDurationSeconds  prometheus.Histogram

	// Combat
	ClicksProcessedTotal  *prometheus.CounterVec
	TargetsKilledTotal    prometheus.Counter
	TargetsSpawnedTotal   prometheus.Counter
	ReactionTimeMs        prometheus.Histogram
	ClickBufferDepth      prometheus.Gauge
	ClickBatchFlushesTotal prometheus.Counter
	DBWriteErrorsTotal    *prometheus.CounterVec

	// Frontend (received via POST /telemetry)
	FrontendJSErrorsTotal        prometheus.Counter
	FrontendWSConnectsTotal      prometheus.Counter
	FrontendWSDisconnectsTotal   prometheus.Counter
	FrontendWSReconnectsTotal    prometheus.Counter
	FrontendSSEConnectsTotal     prometheus.Counter
	FrontendSSEErrorsTotal       prometheus.Counter
	FrontendClickLatencyMs       prometheus.Histogram
	FrontendSceneTransitionsTotal *prometheus.CounterVec
	FrontendLCPSeconds           prometheus.Histogram
	FrontendCLSScore             prometheus.Histogram
}

// New registers and returns all metrics.
func New() *Metrics {
	return &Metrics{
		HTTPRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests by method, path, and status code.",
		}, []string{"method", "path", "status_code"}),

		HTTPRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),

		SSEConnectionsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "sse_connections_active",
			Help: "Number of active SSE connections.",
		}),

		SSEMessagesPublished: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "sse_messages_published_total",
			Help: "Total SSE messages published by event type.",
		}, []string{"event_type"}),

		WSConnectionsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "ws_connections_active",
			Help: "Number of active WebSocket connections.",
		}),

		RoomsCreatedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rooms_created_total",
			Help: "Total rooms created.",
		}),

		RoomsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rooms_active",
			Help: "Current number of active rooms.",
		}),

		PlayersRegisteredTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "players_registered_total",
			Help: "Total players registered.",
		}),

		PlayersActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "players_active",
			Help: "Current number of active players across all rooms.",
		}),

		GamesStartedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "games_started_total",
			Help: "Total games started.",
		}),

		GamesCompletedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "games_completed_total",
			Help: "Total games completed.",
		}),

		GameDurationSeconds: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "game_duration_seconds",
			Help:    "Game round duration in seconds.",
			Buckets: []float64{30, 60, 90, 120, 180, 300},
		}),

		ClicksProcessedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "clicks_processed_total",
			Help: "Total clicks processed by points value and source.",
		}, []string{"points", "source"}),

		TargetsKilledTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "targets_killed_total",
			Help: "Total targets killed.",
		}),

		TargetsSpawnedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "targets_spawned_total",
			Help: "Total targets spawned.",
		}),

		ReactionTimeMs: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "reaction_time_milliseconds",
			Help:    "Player reaction time in milliseconds.",
			Buckets: []float64{100, 200, 300, 400, 500, 750, 1000, 1500, 2000},
		}),

		ClickBufferDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "click_buffer_depth",
			Help: "Current depth of the click event buffer.",
		}),

		ClickBatchFlushesTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "click_batch_flushes_total",
			Help: "Total click batch flushes to the database.",
		}),

		DBWriteErrorsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "db_write_errors_total",
			Help: "Total database write errors by operation.",
		}, []string{"operation"}),

		FrontendJSErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "frontend_js_errors_total",
			Help: "Total JavaScript errors reported by clients.",
		}),

		FrontendWSConnectsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "frontend_ws_connects_total",
			Help: "Total WebSocket connect events from clients.",
		}),

		FrontendWSDisconnectsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "frontend_ws_disconnects_total",
			Help: "Total WebSocket disconnect events from clients.",
		}),

		FrontendWSReconnectsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "frontend_ws_reconnects_total",
			Help: "Total WebSocket reconnect events from clients.",
		}),

		FrontendSSEConnectsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "frontend_sse_connects_total",
			Help: "Total SSE connect events from clients.",
		}),

		FrontendSSEErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "frontend_sse_errors_total",
			Help: "Total SSE error events from clients.",
		}),

		FrontendClickLatencyMs: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "frontend_click_latency_ms",
			Help:    "Frontend click-to-target-removal latency in milliseconds.",
			Buckets: []float64{16, 33, 50, 100, 200, 500},
		}),

		FrontendSceneTransitionsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "frontend_scene_transitions_total",
			Help: "Total scene transitions by from/to scene.",
		}, []string{"from_scene", "to_scene"}),

		FrontendLCPSeconds: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "frontend_lcp_seconds",
			Help:    "Largest Contentful Paint in seconds.",
			Buckets: []float64{0.5, 1, 1.5, 2, 2.5, 4, 6},
		}),

		FrontendCLSScore: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "frontend_cls_score",
			Help:    "Cumulative Layout Shift score.",
			Buckets: []float64{0.01, 0.05, 0.1, 0.15, 0.25, 0.5},
		}),
	}
}
