package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

// TelemetryBatch is the JSON body accepted by POST /telemetry.
type TelemetryBatch struct {
	SessionID string           `json:"session_id"`
	RoomCode  string           `json:"room_code"`
	PlayerID  string           `json:"player_id"`
	Events    []TelemetryEvent `json:"events"`
}

// TelemetryEvent represents a single frontend instrumentation event.
type TelemetryEvent struct {
	Type      string  `json:"type"`
	FromScene string  `json:"from_scene,omitempty"`
	ToScene   string  `json:"to_scene,omitempty"`
	Value     float64 `json:"value,omitempty"`
	Message   string  `json:"message,omitempty"`
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}

	var batch TelemetryBatch
	if err := json.Unmarshal(body, &batch); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Cap at 200 events per batch
	events := batch.Events
	if len(events) > 200 {
		events = events[:200]
	}

	m := s.Metrics
	for _, ev := range events {
		switch ev.Type {
		case "js_error":
			slog.Warn("frontend JS error", "room_code", batch.RoomCode, "player_id", batch.PlayerID, "message", ev.Message)
			if m != nil {
				m.FrontendJSErrorsTotal.Inc()
			}
		case "ws_connect":
			if m != nil {
				m.FrontendWSConnectsTotal.Inc()
			}
		case "ws_disconnect":
			if m != nil {
				m.FrontendWSDisconnectsTotal.Inc()
			}
		case "ws_reconnect":
			if m != nil {
				m.FrontendWSReconnectsTotal.Inc()
			}
		case "sse_connect":
			if m != nil {
				m.FrontendSSEConnectsTotal.Inc()
			}
		case "sse_error":
			slog.Warn("frontend SSE error", "room_code", batch.RoomCode, "player_id", batch.PlayerID)
			if m != nil {
				m.FrontendSSEErrorsTotal.Inc()
			}
		case "click_latency":
			if m != nil && ev.Value > 0 {
				m.FrontendClickLatencyMs.Observe(ev.Value)
			}
		case "scene_transition":
			slog.Info("scene transition", "room_code", batch.RoomCode, "from", ev.FromScene, "to", ev.ToScene)
			if m != nil && (ev.FromScene != "" || ev.ToScene != "") {
				m.FrontendSceneTransitionsTotal.WithLabelValues(ev.FromScene, ev.ToScene).Inc()
			}
		case "web_vital_lcp":
			if m != nil && ev.Value > 0 {
				m.FrontendLCPSeconds.Observe(ev.Value / 1000.0)
			}
		case "web_vital_cls":
			if m != nil && ev.Value >= 0 {
				m.FrontendCLSScore.Observe(ev.Value)
			}
		case "audio_error":
			slog.Warn("frontend audio error", "room_code", batch.RoomCode, "message", ev.Message)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
