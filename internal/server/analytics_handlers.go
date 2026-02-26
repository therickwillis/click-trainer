package server

import (
	"clicktrainer/internal/analytics"
	"log/slog"
	"net/http"
	"strings"
)

func (s *Server) handleAnalyticsDashboard(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "Analytics requires a database connection", http.StatusServiceUnavailable)
		return
	}

	q := analytics.NewQueries(s.DB)

	data := struct {
		PlayerStats *analytics.PlayerLifetimeStats
		Leaderboard []analytics.LeaderboardEntry
	}{}

	// Get player stats if logged in
	if idCookie, err := r.Cookie("player_id"); err == nil {
		stats, err := q.GetPlayerLifetimeStats(idCookie.Value)
		if err == nil {
			data.PlayerStats = stats
		}
	}

	// Default leaderboard: score
	leaderboard, err := q.GetLeaderboard("score", 10)
	if err != nil {
		slog.Error("leaderboard query failed", "handler", "analytics_dashboard", "error", err)
	}
	data.Leaderboard = leaderboard

	if err := s.Tmpl.ExecuteTemplate(w, "analytics-dashboard", data); err != nil {
		slog.Error("template error", "handler", "analytics_dashboard", "error", err)
		http.Error(w, "Error rendering analytics", http.StatusInternalServerError)
	}
}

func (s *Server) handleAnalyticsLeaderboard(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "Analytics requires a database connection", http.StatusServiceUnavailable)
		return
	}

	q := analytics.NewQueries(s.DB)
	category := r.URL.Query().Get("cat")
	if category == "" {
		category = "score"
	}

	entries, err := q.GetLeaderboard(category, 10)
	if err != nil {
		slog.Error("leaderboard query failed", "handler", "analytics_leaderboard", "category", category, "error", err)
		http.Error(w, "Error loading leaderboard", http.StatusInternalServerError)
		return
	}

	if err := s.Tmpl.ExecuteTemplate(w, "leaderboard-entries", entries); err != nil {
		slog.Error("template error", "handler", "analytics_leaderboard", "error", err)
	}
}

func (s *Server) handleAnalyticsPlayer(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "Analytics requires a database connection", http.StatusServiceUnavailable)
		return
	}

	// /analytics/player/{id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Player ID required", http.StatusBadRequest)
		return
	}
	playerID := parts[3]

	q := analytics.NewQueries(s.DB)
	stats, err := q.GetPlayerLifetimeStats(playerID)
	if err != nil {
		slog.Error("player stats query failed", "handler", "analytics_player", "player_id", playerID, "error", err)
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	if err := s.Tmpl.ExecuteTemplate(w, "analytics-player", stats); err != nil {
		slog.Error("template error", "handler", "analytics_player", "error", err)
		http.Error(w, "Error rendering player stats", http.StatusInternalServerError)
	}
}

func (s *Server) handleAnalyticsGame(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "Analytics requires a database connection", http.StatusServiceUnavailable)
		return
	}

	// /analytics/game/{id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Game ID required", http.StatusBadRequest)
		return
	}
	gameID := parts[3]

	q := analytics.NewQueries(s.DB)
	recap, err := q.GetGameRecap(gameID)
	if err != nil {
		slog.Error("game recap query failed", "handler", "analytics_game", "game_id", gameID, "error", err)
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	if err := s.Tmpl.ExecuteTemplate(w, "analytics-game", recap); err != nil {
		slog.Error("template error", "handler", "analytics_game", "error", err)
		http.Error(w, "Error rendering game recap", http.StatusInternalServerError)
	}
}
