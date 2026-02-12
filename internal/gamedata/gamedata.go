package gamedata

import (
	"clicktrainer/internal/events"
	"clicktrainer/internal/players"
	"clicktrainer/internal/targets"
	"sync"
)

type Scene string

const (
	SceneLobby  = Scene("lobby")
	SceneCombat = Scene("combat")
	SceneRecap  = Scene("recap")
)

type Config struct {
	RoundDuration  int // seconds
	InitialTargets int
	CountdownSecs  int
}

func DefaultConfig() Config {
	return Config{
		RoundDuration:  60,
		InitialTargets: 3,
		CountdownSecs:  3,
	}
}

type GameData struct {
	Scene    Scene
	Player   *players.Player
	Players  []*players.Player
	Targets  []*targets.Target
	TimeLeft int
	Rankings []*players.Player
	RoomCode string
}

type Game struct {
	mu            sync.Mutex
	scene         Scene
	timeLeft      int
	currentGameID string
	Players       *players.Store
	Targets       *targets.Store
	Events        *events.Bus
	Config        Config
}

func NewGame(ps *players.Store, ts *targets.Store, bus *events.Bus, cfg Config) *Game {
	return &Game{
		scene:   SceneLobby,
		Players: ps,
		Targets: ts,
		Events:  bus,
		Config:  cfg,
	}
}

func (g *Game) Get(id string) GameData {
	g.mu.Lock()
	defer g.mu.Unlock()
	return GameData{
		Scene:    g.scene,
		Player:   g.Players.Get(id),
		Players:  g.Players.GetList(),
		Targets:  g.Targets.GetList(),
		TimeLeft: g.timeLeft,
	}
}

func (g *Game) Scene() Scene {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.scene
}

func (g *Game) SetScene(s Scene) {
	g.mu.Lock()
	g.scene = s
	g.mu.Unlock()
	g.Events.SceneChanges <- events.SceneChangeEvent{Scene: string(s)}
}

func (g *Game) SetTimeLeft(t int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.timeLeft = t
}

func (g *Game) TimeLeft() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.timeLeft
}

func (g *Game) SetCurrentGameID(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.currentGameID = id
}

func (g *Game) CurrentGameID() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.currentGameID
}

func (g *Game) StartRound() {
	g.Targets.Clear()
	for i := 0; i < g.Config.InitialTargets; i++ {
		g.Targets.Add()
	}
	g.mu.Lock()
	g.timeLeft = g.Config.RoundDuration
	g.mu.Unlock()
}

func (g *Game) EndRound() []*players.Player {
	g.mu.Lock()
	g.scene = SceneRecap
	g.mu.Unlock()
	g.Events.SceneChanges <- events.SceneChangeEvent{Scene: string(SceneRecap)}

	ranked := g.Players.GetList()
	// Sort by score descending
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].Score > ranked[i].Score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	return ranked
}

func (g *Game) ResetToLobby() {
	g.Targets.Clear()
	g.Players.ResetAll()
	g.mu.Lock()
	g.scene = SceneLobby
	g.timeLeft = 0
	g.mu.Unlock()
	g.Events.SceneChanges <- events.SceneChangeEvent{Scene: string(SceneLobby)}
}
