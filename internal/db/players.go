package db

import (
	"fmt"
	"time"
)

type PlayerRecord struct {
	ID        string
	Name      string
	Color     string
	CreatedAt time.Time
}

func (d *DB) UpsertPlayer(id, name, color string) error {
	_, err := d.conn.Exec(`
		INSERT INTO players (id, name, color)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET name = $2, color = $3
	`, id, name, color)
	if err != nil {
		return fmt.Errorf("upserting player: %w", err)
	}
	return nil
}

func (d *DB) GetPlayer(id string) (*PlayerRecord, error) {
	var p PlayerRecord
	err := d.conn.QueryRow(`
		SELECT id, name, color, created_at FROM players WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.Color, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting player: %w", err)
	}
	return &p, nil
}
