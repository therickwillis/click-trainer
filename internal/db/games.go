package db

import (
	"fmt"
	"time"
)

type GameRecord struct {
	ID              string
	RoomCode        string
	HostID          string
	StartedAt       *time.Time
	EndedAt         *time.Time
	RoundDurationMs int
	CreatedAt       time.Time
}

func (d *DB) CreateGame(roomCode, hostID string, roundDurationMs int) (string, error) {
	var id string
	err := d.conn.QueryRow(`
		INSERT INTO games (room_code, host_id, round_duration_ms, started_at)
		VALUES ($1, $2, $3, now())
		RETURNING id
	`, roomCode, hostID, roundDurationMs).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("creating game: %w", err)
	}
	return id, nil
}

func (d *DB) EndGame(gameID string) error {
	_, err := d.conn.Exec(`
		UPDATE games SET ended_at = now() WHERE id = $1
	`, gameID)
	if err != nil {
		return fmt.Errorf("ending game: %w", err)
	}
	return nil
}

func (d *DB) AddGamePlayer(gameID, playerID string, finalScore, rank int) error {
	_, err := d.conn.Exec(`
		INSERT INTO game_players (game_id, player_id, final_score, rank)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (game_id, player_id) DO UPDATE SET final_score = $3, rank = $4
	`, gameID, playerID, finalScore, rank)
	if err != nil {
		return fmt.Errorf("adding game player: %w", err)
	}
	return nil
}
