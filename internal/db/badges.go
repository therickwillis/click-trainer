package db

import "fmt"

func (d *DB) AwardBadge(playerID, badgeID string, gameID *string) error {
	_, err := d.conn.Exec(`
		INSERT INTO player_badges (player_id, badge_id, game_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (player_id, badge_id) DO NOTHING
	`, playerID, badgeID, gameID)
	if err != nil {
		return fmt.Errorf("awarding badge: %w", err)
	}
	return nil
}

func (d *DB) GetPlayerBadges(playerID string) ([]string, error) {
	rows, err := d.conn.Query(`
		SELECT badge_id FROM player_badges WHERE player_id = $1 ORDER BY awarded_at
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("getting badges: %w", err)
	}
	defer rows.Close()

	var badges []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		badges = append(badges, id)
	}
	return badges, nil
}
