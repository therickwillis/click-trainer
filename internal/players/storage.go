package players

import (
	utility "clicktrainer/internal"
	"sync"
)

var (
	players   = make(map[string]*Player)
	allReady  = false
	playersMu sync.Mutex
)

func Add(id string, name string) *Player {
	playersMu.Lock()
	defer playersMu.Unlock()
	player := &Player{ID: id, Name: name, Color: utility.RandomColorHex()}
	players[id] = player
	return player
}

func Get(id string) *Player {
	playersMu.Lock()
	defer playersMu.Unlock()
	return players[id]
}

func GetList() []*Player {
	playersMu.Lock()
	playerList := make([]*Player, 0, len(players))
	for _, p := range players {
		playerList = append(playerList, p)
	}
	playersMu.Unlock()
	return playerList
}

func UpdateScore(id string, points int) *Player {
	playersMu.Lock()
	defer playersMu.Unlock()
	if p, e := players[id]; e {
		p.Score += points
		return p
	}
	return nil
}

func SetReady(id string, isReady bool) *Player {
	playersMu.Lock()
	defer playersMu.Unlock()
	if p, e := players[id]; e {
		p.Ready = isReady
		return p
	}
	return nil
}

func AllReady() bool {
	playersMu.Lock()
	defer playersMu.Unlock()
	if len(players) == 0 {
		return false
	}

	for _, player := range players {
		if !player.Ready {
			return false
		}
	}
	return true
}

func ValidateSession(sessionId string) bool {
	playersMu.Lock()
	defer playersMu.Unlock()
	_, exists := players[sessionId]
	return exists
}
