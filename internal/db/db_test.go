package db

import (
	"os"
	"testing"
	"time"
)

func getTestDB(t *testing.T) *DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping database tests")
	}
	database, err := Connect(dsn)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	t.Cleanup(func() {
		// Clean up test data
		database.conn.Exec("DELETE FROM click_events")
		database.conn.Exec("DELETE FROM game_players")
		database.conn.Exec("DELETE FROM games")
		database.conn.Exec("DELETE FROM players")
		database.Close()
	})
	return database
}

func TestConnect(t *testing.T) {
	database := getTestDB(t)
	if err := database.Ping(); err != nil {
		t.Errorf("Ping() error: %v", err)
	}
}

func TestMigrate(t *testing.T) {
	database := getTestDB(t)

	// Verify tables exist by querying them
	tables := []string{"players", "games", "game_players", "click_events"}
	for _, table := range tables {
		var exists bool
		err := database.conn.QueryRow(`
			SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)
		`, table).Scan(&exists)
		if err != nil {
			t.Errorf("checking table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s does not exist", table)
		}
	}
}

func TestUpsertPlayer(t *testing.T) {
	database := getTestDB(t)

	id := "550e8400-e29b-41d4-a716-446655440000"
	err := database.UpsertPlayer(id, "Alice", "#ff0000")
	if err != nil {
		t.Fatalf("UpsertPlayer() error: %v", err)
	}

	// Upsert again with different data
	err = database.UpsertPlayer(id, "Alice Updated", "#00ff00")
	if err != nil {
		t.Fatalf("UpsertPlayer() update error: %v", err)
	}

	p, err := database.GetPlayer(id)
	if err != nil {
		t.Fatalf("GetPlayer() error: %v", err)
	}
	if p.Name != "Alice Updated" {
		t.Errorf("name = %q, want %q", p.Name, "Alice Updated")
	}
	if p.Color != "#00ff00" {
		t.Errorf("color = %q, want %q", p.Color, "#00ff00")
	}
}

func TestGetPlayer_NotFound(t *testing.T) {
	database := getTestDB(t)

	_, err := database.GetPlayer("00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Error("GetPlayer() should return error for nonexistent player")
	}
}

func TestCreateGame(t *testing.T) {
	database := getTestDB(t)

	// Create a host player first
	hostID := "550e8400-e29b-41d4-a716-446655440001"
	database.UpsertPlayer(hostID, "Host", "#aabbcc")

	gameID, err := database.CreateGame("ABCD", hostID, 60000)
	if err != nil {
		t.Fatalf("CreateGame() error: %v", err)
	}
	if gameID == "" {
		t.Error("CreateGame() returned empty ID")
	}
}

func TestEndGame(t *testing.T) {
	database := getTestDB(t)

	hostID := "550e8400-e29b-41d4-a716-446655440002"
	database.UpsertPlayer(hostID, "Host", "#aabbcc")

	gameID, _ := database.CreateGame("EFGH", hostID, 60000)

	err := database.EndGame(gameID)
	if err != nil {
		t.Fatalf("EndGame() error: %v", err)
	}

	// Verify ended_at is set
	var endedAt *time.Time
	database.conn.QueryRow("SELECT ended_at FROM games WHERE id = $1", gameID).Scan(&endedAt)
	if endedAt == nil {
		t.Error("ended_at should be set after EndGame()")
	}
}

func TestAddGamePlayer(t *testing.T) {
	database := getTestDB(t)

	hostID := "550e8400-e29b-41d4-a716-446655440003"
	playerID := "550e8400-e29b-41d4-a716-446655440004"
	database.UpsertPlayer(hostID, "Host", "#aabbcc")
	database.UpsertPlayer(playerID, "Player", "#ddeeff")

	gameID, _ := database.CreateGame("IJKL", hostID, 60000)

	err := database.AddGamePlayer(gameID, playerID, 150, 1)
	if err != nil {
		t.Fatalf("AddGamePlayer() error: %v", err)
	}

	// Upsert should work
	err = database.AddGamePlayer(gameID, playerID, 200, 1)
	if err != nil {
		t.Fatalf("AddGamePlayer() upsert error: %v", err)
	}
}

func TestRecordClick(t *testing.T) {
	database := getTestDB(t)

	hostID := "550e8400-e29b-41d4-a716-446655440005"
	database.UpsertPlayer(hostID, "Host", "#aabbcc")

	gameID, _ := database.CreateGame("MNOP", hostID, 60000)

	now := time.Now()
	err := database.RecordClick(ClickEvent{
		GameID:     gameID,
		PlayerID:   hostID,
		TargetID:   1,
		Points:     3,
		TargetSize: 75,
		TargetX:    100,
		TargetY:    200,
		SpawnedAt:  now.Add(-500 * time.Millisecond),
		ClickedAt:  now,
		ReactionMs: 500,
	})
	if err != nil {
		t.Fatalf("RecordClick() error: %v", err)
	}
}

func TestBatchRecordClicks(t *testing.T) {
	database := getTestDB(t)

	hostID := "550e8400-e29b-41d4-a716-446655440006"
	database.UpsertPlayer(hostID, "Host", "#aabbcc")

	gameID, _ := database.CreateGame("QRST", hostID, 60000)

	now := time.Now()
	events := []ClickEvent{
		{GameID: gameID, PlayerID: hostID, TargetID: 1, Points: 1, TargetSize: 50, TargetX: 10, TargetY: 20, SpawnedAt: now, ClickedAt: now, ReactionMs: 100},
		{GameID: gameID, PlayerID: hostID, TargetID: 2, Points: 4, TargetSize: 80, TargetX: 300, TargetY: 200, SpawnedAt: now, ClickedAt: now, ReactionMs: 200},
		{GameID: gameID, PlayerID: hostID, TargetID: 3, Points: 2, TargetSize: 60, TargetX: 500, TargetY: 350, SpawnedAt: now, ClickedAt: now, ReactionMs: 150},
	}

	err := database.BatchRecordClicks(events)
	if err != nil {
		t.Fatalf("BatchRecordClicks() error: %v", err)
	}

	var count int
	database.conn.QueryRow("SELECT COUNT(*) FROM click_events WHERE game_id = $1", gameID).Scan(&count)
	if count != 3 {
		t.Errorf("click count = %d, want 3", count)
	}
}
