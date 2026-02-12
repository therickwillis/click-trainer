CREATE TABLE IF NOT EXISTS player_badges (
    player_id UUID NOT NULL REFERENCES players(id),
    badge_id TEXT NOT NULL,
    awarded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    game_id UUID REFERENCES games(id),
    PRIMARY KEY (player_id, badge_id)
);
