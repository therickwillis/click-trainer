package rooms

import (
	"clicktrainer/internal/broadcast"
	"clicktrainer/internal/gamedata"
	"time"
)

type Room struct {
	Code        string
	Game        *gamedata.Game
	Broadcaster *broadcast.Broadcaster
	CreatedAt   time.Time
	HostID      string
}
