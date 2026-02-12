package rooms

import (
	"clicktrainer/internal/broadcast"
	"clicktrainer/internal/events"
	"clicktrainer/internal/gamedata"
	"clicktrainer/internal/players"
	"clicktrainer/internal/targets"
	"fmt"
	"sync"
	"time"
)

const staleTTL = 1 * time.Hour

type Store struct {
	mu    sync.Mutex
	rooms map[string]*Room
	cfg   gamedata.Config
}

func NewStore(cfg gamedata.Config) *Store {
	s := &Store{
		rooms: make(map[string]*Room),
		cfg:   cfg,
	}
	go s.sweepStale()
	return s
}

func (s *Store) Create(hostID string) (*Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Try up to 10 times to generate a unique code
	for range 10 {
		code, err := GenerateCode()
		if err != nil {
			return nil, fmt.Errorf("generating room code: %w", err)
		}
		if _, exists := s.rooms[code]; exists {
			continue
		}

		ps := players.NewStore()
		ts := targets.NewStore()
		bus := events.NewBus()
		game := gamedata.NewGame(ps, ts, bus, s.cfg)
		b := broadcast.NewBroadcaster(bus)

		room := &Room{
			Code:        code,
			Game:        game,
			Broadcaster: b,
			CreatedAt:   time.Now(),
			HostID:      hostID,
		}
		s.rooms[code] = room
		return room, nil
	}
	return nil, fmt.Errorf("failed to generate unique room code after 10 attempts")
}

func (s *Store) Get(code string) *Room {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rooms[code]
}

func (s *Store) Delete(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rooms, code)
}

func (s *Store) List() []*Room {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := make([]*Room, 0, len(s.rooms))
	for _, r := range s.rooms {
		list = append(list, r)
	}
	return list
}

func (s *Store) sweepStale() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for code, room := range s.rooms {
			if now.Sub(room.CreatedAt) > staleTTL {
				delete(s.rooms, code)
			}
		}
		s.mu.Unlock()
	}
}
