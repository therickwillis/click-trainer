package targets

import (
	"clicktrainer/internal/utility"
	"math/rand"
	"sync"
	"time"
)

const (
	GameHeight    = 400
	GameWidth     = 600
	MinTargetSize = 50
	MaxTargetSize = 100
)

type Store struct {
	mu      sync.Mutex
	targets map[int]*Target
	nextID  int
}

func NewStore() *Store {
	return &Store{
		targets: make(map[int]*Target),
		nextID:  1,
	}
}

func (s *Store) Add() *Target {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	targetSize := rand.Intn(MaxTargetSize-MinTargetSize) + MinTargetSize
	target := &Target{
		ID:        id,
		X:         rand.Intn(GameWidth - targetSize),
		Y:         rand.Intn(GameHeight - targetSize),
		Color:     utility.RandomColorHex(),
		Size:      targetSize,
		SpawnedAt: time.Now(),
	}
	s.targets[id] = target
	return target
}

func (s *Store) Get(id int) *Target {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.targets[id]
}

func (s *Store) Kill(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, e := s.targets[id]; e {
		t.Dead = true
	}
}

func (s *Store) GetList() []*Target {
	s.mu.Lock()
	defer s.mu.Unlock()
	targetList := make([]*Target, 0, len(s.targets))
	for _, t := range s.targets {
		if !t.Dead {
			targetList = append(targetList, t)
		}
	}
	return targetList
}

func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.targets = make(map[int]*Target)
	s.nextID = 1
}
