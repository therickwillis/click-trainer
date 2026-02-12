package analytics

import (
	"clicktrainer/internal/db"
	"fmt"
)

type Queries struct {
	DB *db.DB
}

func NewQueries(database *db.DB) *Queries {
	return &Queries{DB: database}
}

func (q *Queries) GetPlayerGameStats(gameID, playerID string) (*PlayerGameStats, error) {
	stats := &PlayerGameStats{
		GameID:   gameID,
		PlayerID: playerID,
	}

	err := q.DB.QueryRow(`
		SELECT p.name, p.color, gp.final_score
		FROM game_players gp
		JOIN players p ON p.id = gp.player_id
		WHERE gp.game_id = $1 AND gp.player_id = $2
	`, gameID, playerID).Scan(&stats.PlayerName, &stats.PlayerColor, &stats.Score)
	if err != nil {
		return nil, fmt.Errorf("getting game player: %w", err)
	}

	err = q.DB.QueryRow(`
		SELECT
			COUNT(*) as clicks,
			COALESCE(AVG(reaction_ms), 0) as avg_reaction,
			COALESCE(MIN(reaction_ms), 0) as best_reaction,
			COUNT(*) FILTER (WHERE points = 4) as bullseyes
		FROM click_events
		WHERE game_id = $1 AND player_id = $2
	`, gameID, playerID).Scan(&stats.Clicks, &stats.AvgReaction, &stats.BestReaction, &stats.Bullseyes)
	if err != nil {
		return nil, fmt.Errorf("getting click stats: %w", err)
	}

	// Calculate CPS from game duration
	var durationSecs float64
	q.DB.QueryRow(`
		SELECT EXTRACT(EPOCH FROM (ended_at - started_at))
		FROM games WHERE id = $1 AND ended_at IS NOT NULL AND started_at IS NOT NULL
	`, gameID).Scan(&durationSecs)
	if durationSecs > 0 {
		stats.CPS = float64(stats.Clicks) / durationSecs
	}

	if stats.Clicks > 0 {
		stats.BullseyeRate = float64(stats.Bullseyes) / float64(stats.Clicks) * 100
	}

	return stats, nil
}

func (q *Queries) GetPlayerLifetimeStats(playerID string) (*PlayerLifetimeStats, error) {
	stats := &PlayerLifetimeStats{
		PlayerID: playerID,
	}

	err := q.DB.QueryRow(`SELECT name, color FROM players WHERE id = $1`, playerID).
		Scan(&stats.PlayerName, &stats.PlayerColor)
	if err != nil {
		return nil, fmt.Errorf("getting player: %w", err)
	}

	err = q.DB.QueryRow(`
		SELECT
			COUNT(*) as games_played,
			COALESCE(SUM(final_score), 0) as total_score,
			COALESCE(MAX(final_score), 0) as best_game,
			COUNT(*) FILTER (WHERE rank = 1) as win_count
		FROM game_players
		WHERE player_id = $1
	`, playerID).Scan(&stats.GamesPlayed, &stats.TotalScore, &stats.BestGame, &stats.WinCount)
	if err != nil {
		return nil, fmt.Errorf("getting lifetime stats: %w", err)
	}

	// Calculate win streak (most recent consecutive wins)
	rows, err := q.DB.Query(`
		SELECT gp.rank
		FROM game_players gp
		JOIN games g ON g.id = gp.game_id
		WHERE gp.player_id = $1 AND g.ended_at IS NOT NULL
		ORDER BY g.ended_at DESC
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("getting win streak: %w", err)
	}
	defer rows.Close()

	streak := 0
	for rows.Next() {
		var rank int
		if err := rows.Scan(&rank); err != nil {
			return nil, err
		}
		if rank == 1 {
			streak++
		} else {
			break
		}
	}
	stats.WinStreak = streak

	stats.Badges = EvaluateLifetimeBadges(*stats)

	return stats, nil
}

func (q *Queries) GetLeaderboard(category string, limit int) ([]LeaderboardEntry, error) {
	var query string
	switch category {
	case "score":
		query = `
			SELECT p.id, p.name, p.color, COALESCE(SUM(gp.final_score), 0) as value
			FROM players p
			JOIN game_players gp ON gp.player_id = p.id
			GROUP BY p.id, p.name, p.color
			ORDER BY value DESC
			LIMIT $1`
	case "reaction":
		query = `
			SELECT p.id, p.name, p.color, COALESCE(MIN(ce.reaction_ms), 0) as value
			FROM players p
			JOIN click_events ce ON ce.player_id = p.id
			GROUP BY p.id, p.name, p.color
			ORDER BY value ASC
			LIMIT $1`
	case "wins":
		query = `
			SELECT p.id, p.name, p.color, COUNT(*) FILTER (WHERE gp.rank = 1) as value
			FROM players p
			JOIN game_players gp ON gp.player_id = p.id
			GROUP BY p.id, p.name, p.color
			ORDER BY value DESC
			LIMIT $1`
	case "bullseyes":
		query = `
			SELECT p.id, p.name, p.color, COUNT(*) FILTER (WHERE ce.points = 4) as value
			FROM players p
			JOIN click_events ce ON ce.player_id = p.id
			GROUP BY p.id, p.name, p.color
			ORDER BY value DESC
			LIMIT $1`
	case "cps":
		query = `
			SELECT p.id, p.name, p.color,
				COALESCE(ROUND(COUNT(ce.id)::numeric / NULLIF(SUM(EXTRACT(EPOCH FROM (g.ended_at - g.started_at))), 0), 2)::int, 0) as value
			FROM players p
			JOIN click_events ce ON ce.player_id = p.id
			JOIN games g ON g.id = ce.game_id AND g.ended_at IS NOT NULL AND g.started_at IS NOT NULL
			GROUP BY p.id, p.name, p.color
			ORDER BY value DESC
			LIMIT $1`
	default:
		return nil, fmt.Errorf("unknown leaderboard category: %s", category)
	}

	rows, err := q.DB.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("getting leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.PlayerID, &e.PlayerName, &e.PlayerColor, &e.Value); err != nil {
			return nil, err
		}
		e.Rank = rank
		rank++
		entries = append(entries, e)
	}
	return entries, nil
}

func (q *Queries) GetGameRecap(gameID string) (*GameRecap, error) {
	recap := &GameRecap{GameID: gameID}

	err := q.DB.QueryRow(`
		SELECT room_code, started_at, ended_at FROM games WHERE id = $1
	`, gameID).Scan(&recap.RoomCode, &recap.StartedAt, &recap.EndedAt)
	if err != nil {
		return nil, fmt.Errorf("getting game: %w", err)
	}

	rows, err := q.DB.Query(`
		SELECT gp.player_id FROM game_players gp WHERE gp.game_id = $1 ORDER BY gp.rank
	`, gameID)
	if err != nil {
		return nil, fmt.Errorf("getting game players: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var playerID string
		if err := rows.Scan(&playerID); err != nil {
			return nil, err
		}
		stats, err := q.GetPlayerGameStats(gameID, playerID)
		if err != nil {
			return nil, err
		}
		recap.Players = append(recap.Players, *stats)
	}

	return recap, nil
}
