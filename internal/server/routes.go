package server

import (
	"bufio"
	"clicktrainer/internal/config"
	"clicktrainer/internal/db"
	"clicktrainer/internal/gamedata"
	"clicktrainer/internal/metrics"
	"clicktrainer/internal/rooms"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Run() error {
	appCfg := config.Load()

	gameCfg := gamedata.Config{
		RoundDuration:  appCfg.RoundDuration,
		InitialTargets: 3,
		CountdownSecs:  3,
	}
	roomStore := rooms.NewStore(gameCfg)

	m := metrics.New()

	funcMap := template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseFiles(
		"templates/home.html",
		"templates/game.html",
		"templates/join.html",
		"templates/target.html",
		"templates/lobby.html",
		"templates/recap.html",
		"templates/analytics/dashboard.html",
		"templates/analytics/leaderboard.html",
		"templates/analytics/player.html",
		"templates/analytics/game.html",
	))

	srv := &Server{
		Rooms:   roomStore,
		Tmpl:    tmpl,
		Metrics: m,
	}

	// Optional database connection
	if appCfg.DatabaseURL != "" {
		database, err := db.Connect(appCfg.DatabaseURL)
		if err != nil {
			slog.Error("failed to connect to database, running without", "error", err)
		} else {
			if err := database.Migrate(); err != nil {
				slog.Error("database migration failed", "error", err)
			}
			srv.DB = database
			srv.ClickBuffer = make(chan db.ClickEvent, 1000)
			srv.FlushSignal = make(chan chan struct{})
			go clickBatchWriter(database, srv.ClickBuffer, srv.FlushSignal, m)
			slog.Info("database connected and migrations applied", "component", "db")
		}
	} else {
		slog.Info("DATABASE_URL not set, running without database", "component", "db")
	}

	// Background goroutine: poll room/player counts for gauges every 15s
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			rooms := roomStore.List()
			m.RoomsActive.Set(float64(len(rooms)))
			total := 0
			for _, room := range rooms {
				total += room.Game.Players.Count()
			}
			m.PlayersActive.Set(float64(total))
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleHome)
	mux.HandleFunc("/rooms/create", srv.handleCreateRoom)
	mux.HandleFunc("/rooms/join", srv.handleJoinRoom)
	mux.HandleFunc("GET /room/{code}", srv.handleRoomWithCode)
	mux.HandleFunc("GET /room", srv.handleRoom)
	mux.HandleFunc("POST /room/register", srv.handleRegister)
	mux.HandleFunc("POST /room/ready", srv.handleReady)
	mux.HandleFunc("POST /room/target/", srv.handleTarget)
	mux.HandleFunc("GET /room/ws", srv.handleWebSocket)
	mux.HandleFunc("POST /room/leave", srv.handleLeaveRoom)
	mux.HandleFunc("GET /room/events", srv.handleEvents)
	mux.HandleFunc("GET /room/poll", srv.handlePoll)
	mux.HandleFunc("POST /room/play-again", srv.handlePlayAgain)
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("POST /telemetry", srv.handleTelemetry)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/analytics", srv.handleAnalyticsDashboard)
	mux.HandleFunc("/analytics/leaderboard", srv.handleAnalyticsLeaderboard)
	mux.HandleFunc("/analytics/player/", srv.handleAnalyticsPlayer)
	mux.HandleFunc("/analytics/game/", srv.handleAnalyticsGame)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	addr := "0.0.0.0:" + appCfg.Port
	slog.Info("server listening", "addr", fmt.Sprintf("http://localhost:%s", appCfg.Port))
	return http.ListenAndServe(addr, metricsMiddleware(m, mux))
}

// metricsMiddleware wraps an http.Handler to record request count and duration.
func metricsMiddleware(m *metrics.Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := normalizePath(r.URL.Path)
		start := time.Now()
		rw := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start).Seconds()
		statusStr := strconv.Itoa(rw.status)
		m.HTTPRequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
		m.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// statusResponseWriter captures the HTTP status code written by a handler.
// It forwards http.Flusher and http.Hijacker to the underlying ResponseWriter
// so SSE and WebSocket handlers continue to work through the middleware.
type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *statusResponseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *statusResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *statusResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
	}
	return h.Hijack()
}

func (rw *statusResponseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// normalizePath replaces dynamic path segments with placeholders to prevent
// label cardinality explosion in Prometheus.
func normalizePath(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		if part == "" {
			continue
		}
		// /room/target/{id}/{points}
		if i >= 3 && len(parts) > 3 && parts[2] == "target" {
			if i == 3 {
				parts[i] = "{id}"
			} else if i == 4 {
				parts[i] = "{points}"
			}
			continue
		}
		// /analytics/player/{id} and /analytics/game/{id}
		if i == 3 && len(parts) > 3 && (parts[2] == "player" || parts[2] == "game") {
			parts[i] = "{id}"
			continue
		}
		// /room/{code}
		if i == 2 && len(parts) == 3 && parts[1] == "room" {
			parts[i] = "{code}"
			continue
		}
	}
	return strings.Join(parts, "/")
}

func clickBatchWriter(database *db.DB, buffer chan db.ClickEvent, flushSignal chan chan struct{}, m *metrics.Metrics) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]db.ClickEvent, 0, 50)

	writeBatch := func() {
		if m != nil {
			m.ClickBufferDepth.Set(float64(len(buffer)))
		}
		if len(batch) > 0 {
			if err := database.BatchRecordClicks(batch); err != nil {
				slog.Error("BatchRecordClicks failed", "component", "batch_writer", "error", err)
				if m != nil {
					m.DBWriteErrorsTotal.WithLabelValues("batch_record_clicks").Inc()
				}
			} else if m != nil {
				m.ClickBatchFlushesTotal.Inc()
			}
			batch = batch[:0]
		}
	}

	for {
		select {
		case ev := <-buffer:
			batch = append(batch, ev)
			if len(batch) >= 50 {
				writeBatch()
			}
		case <-ticker.C:
			writeBatch()
		case done := <-flushSignal:
			// Drain any remaining events from the buffer channel
		drain:
			for {
				select {
				case ev := <-buffer:
					batch = append(batch, ev)
				default:
					break drain
				}
			}
			writeBatch()
			close(done)
		}
	}
}
