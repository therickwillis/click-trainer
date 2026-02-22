package rooms

import (
	"clicktrainer/internal/broadcast"
	"clicktrainer/internal/gamedata"
	"clicktrainer/internal/wshub"
	"time"
)

type Room struct {
	Code        string
	Game        *gamedata.Game
	Broadcaster *broadcast.Broadcaster
	Hub         *wshub.Hub
	CreatedAt   time.Time
	HostID      string
}
