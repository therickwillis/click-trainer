package gamedata

import (
	"clicktrainer/internal/events"
	"clicktrainer/internal/players"
	"clicktrainer/internal/targets"
	"testing"
	"time"
)

func newTestGame() *Game {
	ps := players.NewStore()
	ts := targets.NewStore()
	bus := events.NewBus()
	cfg := DefaultConfig()
	return NewGame(ps, ts, bus, cfg)
}

func TestNewGame_StartsInLobby(t *testing.T) {
	g := newTestGame()
	if g.Scene() != SceneLobby {
		t.Errorf("initial scene = %q, want %q", g.Scene(), SceneLobby)
	}
}

func TestGame_SetScene(t *testing.T) {
	g := newTestGame()

	// Drain the event in background
	go func() {
		<-g.Events.SceneChanges
	}()

	g.SetScene(SceneCombat)
	if g.Scene() != SceneCombat {
		t.Errorf("scene = %q, want %q", g.Scene(), SceneCombat)
	}
}

func TestGame_SetScene_SendsEvent(t *testing.T) {
	g := newTestGame()

	go g.SetScene(SceneCombat)

	select {
	case ev := <-g.Events.SceneChanges:
		if ev.Scene != string(SceneCombat) {
			t.Errorf("event scene = %q, want %q", ev.Scene, SceneCombat)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for scene change event")
	}
}

func TestGame_Get(t *testing.T) {
	g := newTestGame()
	g.Players.Add("p1", "Alice")
	g.Targets.Add()

	data := g.Get("p1")

	if data.Scene != SceneLobby {
		t.Errorf("Scene = %q, want %q", data.Scene, SceneLobby)
	}
	if data.Player == nil || data.Player.Name != "Alice" {
		t.Error("Player should be Alice")
	}
	if len(data.Players) != 1 {
		t.Errorf("Players count = %d, want 1", len(data.Players))
	}
	if len(data.Targets) != 1 {
		t.Errorf("Targets count = %d, want 1", len(data.Targets))
	}
}

func TestGame_StartRound(t *testing.T) {
	g := newTestGame()

	// Add some targets before starting (they should be cleared)
	g.Targets.Add()
	g.Targets.Add()

	g.StartRound()

	targets := g.Targets.GetList()
	if len(targets) != g.Config.InitialTargets {
		t.Errorf("targets after StartRound = %d, want %d", len(targets), g.Config.InitialTargets)
	}
	if g.TimeLeft() != g.Config.RoundDuration {
		t.Errorf("TimeLeft = %d, want %d", g.TimeLeft(), g.Config.RoundDuration)
	}
}

func TestGame_EndRound(t *testing.T) {
	g := newTestGame()
	g.Players.Add("p1", "Alice")
	g.Players.Add("p2", "Bob")
	g.Players.UpdateScore("p1", 50)
	g.Players.UpdateScore("p2", 100)

	// Drain the event
	go func() {
		<-g.Events.SceneChanges
	}()

	rankings := g.EndRound()

	if g.Scene() != SceneRecap {
		t.Errorf("scene after EndRound = %q, want %q", g.Scene(), SceneRecap)
	}
	if len(rankings) != 2 {
		t.Fatalf("rankings count = %d, want 2", len(rankings))
	}
	if rankings[0].Name != "Bob" {
		t.Errorf("first place = %q, want %q", rankings[0].Name, "Bob")
	}
	if rankings[1].Name != "Alice" {
		t.Errorf("second place = %q, want %q", rankings[1].Name, "Alice")
	}
}

func TestGame_ResetToLobby(t *testing.T) {
	g := newTestGame()
	g.Players.Add("p1", "Alice")
	g.Players.UpdateScore("p1", 100)
	g.Players.SetReady("p1", true)
	g.Targets.Add()

	// Drain the event
	go func() {
		<-g.Events.SceneChanges
	}()

	g.ResetToLobby()

	if g.Scene() != SceneLobby {
		t.Errorf("scene = %q, want %q", g.Scene(), SceneLobby)
	}
	if g.TimeLeft() != 0 {
		t.Errorf("TimeLeft = %d, want 0", g.TimeLeft())
	}

	p := g.Players.Get("p1")
	if p.Score != 0 {
		t.Errorf("player score = %d, want 0", p.Score)
	}
	if p.Ready {
		t.Error("player should not be ready")
	}

	targets := g.Targets.GetList()
	if len(targets) != 0 {
		t.Errorf("targets = %d, want 0", len(targets))
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.RoundDuration != 60 {
		t.Errorf("RoundDuration = %d, want 60", cfg.RoundDuration)
	}
	if cfg.InitialTargets != 3 {
		t.Errorf("InitialTargets = %d, want 3", cfg.InitialTargets)
	}
	if cfg.CountdownSecs != 3 {
		t.Errorf("CountdownSecs = %d, want 3", cfg.CountdownSecs)
	}
}
