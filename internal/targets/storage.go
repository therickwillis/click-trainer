package targets

import (
	utility "clicktrainer/internal"
	"math/rand"
	"sync"
)

var (
	targets   = make(map[int]*Target)
	nextID    = 1
	targetsMu sync.Mutex
)

const (
	gameHeight    = 400
	gameWidth     = 600
	minTargetSize = 50
	maxTargetSize = 100
)

func Add() *Target {
	targetsMu.Lock()
	id := nextID
	nextID++
	targetSize := rand.Intn(maxTargetSize-minTargetSize) + minTargetSize
	target := &Target{
		ID:    id,
		X:     rand.Intn(gameWidth - targetSize),
		Y:     rand.Intn(gameHeight - targetSize),
		Color: utility.RandomColorHex(),
		Size:  targetSize,
	}
	targets[id] = target
	targetsMu.Unlock()
	return target
}

func Kill(id int) {
	targetsMu.Lock()
	if t, e := targets[id]; e {
		t.Dead = true
	}
	targetsMu.Unlock()
}

func GetList() []*Target {
	targetsMu.Lock()
	targetList := make([]*Target, 0, len(targets))
	for _, t := range targets {
		if !t.Dead {
			targetList = append(targetList, t)
		}
	}
	targetsMu.Unlock()
	return targetList
}
