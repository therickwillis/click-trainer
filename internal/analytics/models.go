package analytics

import "time"

type PlayerGameStats struct {
	PlayerID     string
	PlayerName   string
	PlayerColor  string
	GameID       string
	Clicks       int
	Score        int
	AvgReaction  float64
	BestReaction int
	CPS          float64 // clicks per second
	BullseyeRate float64 // percentage of 4-point hits
	Bullseyes    int
}

type PlayerLifetimeStats struct {
	PlayerID    string
	PlayerName  string
	PlayerColor string
	GamesPlayed int
	TotalScore  int
	BestGame    int
	WinCount    int
	WinStreak   int
	Badges      []Badge
}

type LeaderboardEntry struct {
	PlayerID    string
	PlayerName  string
	PlayerColor string
	Value       int
	Rank        int
}

type GameRecap struct {
	GameID    string
	RoomCode  string
	StartedAt *time.Time
	EndedAt   *time.Time
	Players   []PlayerGameStats
}
