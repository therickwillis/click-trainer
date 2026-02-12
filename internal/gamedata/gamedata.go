package gamedata

import (
	"clicktrainer/internal/events"
	"clicktrainer/internal/players"
	"clicktrainer/internal/targets"
)

type Scene string

type GameData struct {
	Scene   Scene
	Player  *players.Player
	Players []*players.Player
	Targets []*targets.Target
}

const (
	SceneLobby  = Scene("lobby")
	SceneCombat = Scene("combat")
	SceneRecap  = Scene("recap")
)

var (
	scene = SceneLobby
)

func Get(id string) GameData {
	return GameData{
		Scene:   scene,
		Player:  players.Get(id),
		Players: players.GetList(),
		Targets: targets.GetList(),
	}
}

func SetScene(s Scene) {
	scene = s
	events.SceneChanges <- events.SceneChangeEvent{Scene: string(s)}
}
