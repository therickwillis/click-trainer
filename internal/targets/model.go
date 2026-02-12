package targets

import "time"

type Target struct {
	ID        int
	X         int
	Y         int
	Size      int
	Color     string
	Dead      bool
	SpawnedAt time.Time
}
