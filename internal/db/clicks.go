package db

import (
	"fmt"
	"time"
)

type ClickEvent struct {
	GameID     string
	PlayerID   string
	TargetID   int
	Points     int
	TargetSize int
	TargetX    int
	TargetY    int
	SpawnedAt  time.Time
	ClickedAt  time.Time
	ReactionMs int
}

func (d *DB) RecordClick(ev ClickEvent) error {
	_, err := d.conn.Exec(`
		INSERT INTO click_events (game_id, player_id, target_id, points, target_size, target_x, target_y, spawned_at, clicked_at, reaction_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, ev.GameID, ev.PlayerID, ev.TargetID, ev.Points, ev.TargetSize, ev.TargetX, ev.TargetY, ev.SpawnedAt, ev.ClickedAt, ev.ReactionMs)
	if err != nil {
		return fmt.Errorf("recording click: %w", err)
	}
	return nil
}

func (d *DB) BatchRecordClicks(events []ClickEvent) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO click_events (game_id, player_id, target_id, points, target_size, target_x, target_y, spawned_at, clicked_at, reaction_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	for _, ev := range events {
		if _, err := stmt.Exec(ev.GameID, ev.PlayerID, ev.TargetID, ev.Points, ev.TargetSize, ev.TargetX, ev.TargetY, ev.SpawnedAt, ev.ClickedAt, ev.ReactionMs); err != nil {
			return fmt.Errorf("recording click in batch: %w", err)
		}
	}

	return tx.Commit()
}
