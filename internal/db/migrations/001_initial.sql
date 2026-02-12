CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_code TEXT NOT NULL,
    host_id UUID REFERENCES players(id),
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    round_duration_ms INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game_players (
    game_id UUID NOT NULL REFERENCES games(id),
    player_id UUID NOT NULL REFERENCES players(id),
    final_score INT NOT NULL DEFAULT 0,
    rank INT NOT NULL DEFAULT 0,
    PRIMARY KEY (game_id, player_id)
);

CREATE TABLE IF NOT EXISTS click_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id),
    player_id UUID NOT NULL REFERENCES players(id),
    target_id INT NOT NULL,
    points INT NOT NULL,
    target_size INT NOT NULL,
    target_x INT NOT NULL,
    target_y INT NOT NULL,
    spawned_at TIMESTAMPTZ NOT NULL,
    clicked_at TIMESTAMPTZ NOT NULL,
    reaction_ms INT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_click_events_game_id ON click_events(game_id);
CREATE INDEX IF NOT EXISTS idx_click_events_player_id ON click_events(player_id);
CREATE INDEX IF NOT EXISTS idx_games_ended_at ON games(ended_at);
