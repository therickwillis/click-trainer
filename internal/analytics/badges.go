package analytics

type BadgeID string

const (
	BadgeSharpshooter  BadgeID = "sharpshooter"
	BadgeSpeedDemon    BadgeID = "speed_demon"
	BadgeUnstoppable   BadgeID = "unstoppable"
	BadgeCenturion     BadgeID = "centurion"
	BadgeTriggerHappy  BadgeID = "trigger_happy"
	BadgeVeteran       BadgeID = "veteran"
	BadgePerfectionist BadgeID = "perfectionist"
)

type Badge struct {
	ID          BadgeID
	Name        string
	Description string
	Icon        string
}

var AllBadges = map[BadgeID]Badge{
	BadgeSharpshooter:  {ID: BadgeSharpshooter, Name: "Sharpshooter", Description: "10+ bullseyes in a single game", Icon: "🎯"},
	BadgeSpeedDemon:    {ID: BadgeSpeedDemon, Name: "Speed Demon", Description: "Average reaction time under 300ms", Icon: "⚡"},
	BadgeUnstoppable:   {ID: BadgeUnstoppable, Name: "Unstoppable", Description: "3-game win streak", Icon: "🔥"},
	BadgeCenturion:     {ID: BadgeCenturion, Name: "Centurion", Description: "100+ points in a single game", Icon: "💯"},
	BadgeTriggerHappy:  {ID: BadgeTriggerHappy, Name: "Trigger Happy", Description: "3+ clicks per second average", Icon: "🖱️"},
	BadgeVeteran:       {ID: BadgeVeteran, Name: "Veteran", Description: "Played 10+ games", Icon: "🏅"},
	BadgePerfectionist: {ID: BadgePerfectionist, Name: "Perfectionist", Description: "50%+ bullseye rate in a game", Icon: "✨"},
}

// EvaluateGameBadges checks which badges a player earned in a single game.
func EvaluateGameBadges(stats PlayerGameStats) []Badge {
	var earned []Badge

	// Sharpshooter: 10+ bullseyes in a game
	if stats.Bullseyes >= 10 {
		earned = append(earned, AllBadges[BadgeSharpshooter])
	}

	// Speed Demon: avg reaction < 300ms
	if stats.Clicks > 0 && stats.AvgReaction > 0 && stats.AvgReaction < 300 {
		earned = append(earned, AllBadges[BadgeSpeedDemon])
	}

	// Centurion: 100+ points in a game
	if stats.Score >= 100 {
		earned = append(earned, AllBadges[BadgeCenturion])
	}

	// Trigger Happy: 3+ CPS
	if stats.CPS >= 3.0 {
		earned = append(earned, AllBadges[BadgeTriggerHappy])
	}

	// Perfectionist: 50%+ bullseye rate
	if stats.Clicks > 0 && stats.BullseyeRate >= 50.0 {
		earned = append(earned, AllBadges[BadgePerfectionist])
	}

	return earned
}

// EvaluateLifetimeBadges checks which badges a player earned across their career.
func EvaluateLifetimeBadges(stats PlayerLifetimeStats) []Badge {
	var earned []Badge

	// Unstoppable: 3-game win streak
	if stats.WinStreak >= 3 {
		earned = append(earned, AllBadges[BadgeUnstoppable])
	}

	// Veteran: 10+ games
	if stats.GamesPlayed >= 10 {
		earned = append(earned, AllBadges[BadgeVeteran])
	}

	return earned
}
