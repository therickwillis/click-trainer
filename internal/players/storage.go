package players

import (
	"clicktrainer/internal/utility"
	"sync"
)

type Store struct {
	mu      sync.Mutex
	players map[string]*Player
}

func NewStore() *Store {
	return &Store{
		players: make(map[string]*Player),
	}
}

func (s *Store) Add(id string, name string) *Player {
	s.mu.Lock()
	defer s.mu.Unlock()
	player := &Player{ID: id, Name: name, Color: utility.RandomColorHex()}
	s.players[id] = player
	return player
}

func (s *Store) Get(id string) *Player {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.players[id]
}

func (s *Store) GetList() []*Player {
	s.mu.Lock()
	defer s.mu.Unlock()
	playerList := make([]*Player, 0, len(s.players))
	for _, p := range s.players {
		playerList = append(playerList, p)
	}
	return playerList
}

func (s *Store) UpdateScore(id string, points int) *Player {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, e := s.players[id]; e {
		p.Score += points
		return p
	}
	return nil
}

func (s *Store) SetReady(id string, isReady bool) *Player {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, e := s.players[id]; e {
		p.Ready = isReady
		return p
	}
	return nil
}

func (s *Store) AllReady() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.players) == 0 {
		return false
	}

	for _, player := range s.players {
		if !player.Ready {
			return false
		}
	}
	return true
}

func (s *Store) ValidateSession(sessionId string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.players[sessionId]
	return exists
}

func (s *Store) ResetAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, p := range s.players {
		p.Score = 0
		p.Ready = false
		s.players[id] = p
	}
}
