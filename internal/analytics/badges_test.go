package analytics

import "testing"

func TestEvaluateGameBadges_Sharpshooter(t *testing.T) {
	stats := PlayerGameStats{Bullseyes: 10, Clicks: 20}
	badges := EvaluateGameBadges(stats)
	if !hasBadge(badges, BadgeSharpshooter) {
		t.Error("should earn Sharpshooter with 10 bullseyes")
	}
}

func TestEvaluateGameBadges_NoSharpshooter(t *testing.T) {
	stats := PlayerGameStats{Bullseyes: 9, Clicks: 20}
	badges := EvaluateGameBadges(stats)
	if hasBadge(badges, BadgeSharpshooter) {
		t.Error("should not earn Sharpshooter with 9 bullseyes")
	}
}

func TestEvaluateGameBadges_SpeedDemon(t *testing.T) {
	stats := PlayerGameStats{Clicks: 10, AvgReaction: 250}
	badges := EvaluateGameBadges(stats)
	if !hasBadge(badges, BadgeSpeedDemon) {
		t.Error("should earn Speed Demon with 250ms avg reaction")
	}
}

func TestEvaluateGameBadges_NoSpeedDemon(t *testing.T) {
	stats := PlayerGameStats{Clicks: 10, AvgReaction: 350}
	badges := EvaluateGameBadges(stats)
	if hasBadge(badges, BadgeSpeedDemon) {
		t.Error("should not earn Speed Demon with 350ms avg reaction")
	}
}

func TestEvaluateGameBadges_Centurion(t *testing.T) {
	stats := PlayerGameStats{Score: 100}
	badges := EvaluateGameBadges(stats)
	if !hasBadge(badges, BadgeCenturion) {
		t.Error("should earn Centurion with 100 points")
	}
}

func TestEvaluateGameBadges_NoCenturion(t *testing.T) {
	stats := PlayerGameStats{Score: 99}
	badges := EvaluateGameBadges(stats)
	if hasBadge(badges, BadgeCenturion) {
		t.Error("should not earn Centurion with 99 points")
	}
}

func TestEvaluateGameBadges_TriggerHappy(t *testing.T) {
	stats := PlayerGameStats{CPS: 3.0}
	badges := EvaluateGameBadges(stats)
	if !hasBadge(badges, BadgeTriggerHappy) {
		t.Error("should earn Trigger Happy with 3.0 CPS")
	}
}

func TestEvaluateGameBadges_NoTriggerHappy(t *testing.T) {
	stats := PlayerGameStats{CPS: 2.9}
	badges := EvaluateGameBadges(stats)
	if hasBadge(badges, BadgeTriggerHappy) {
		t.Error("should not earn Trigger Happy with 2.9 CPS")
	}
}

func TestEvaluateGameBadges_Perfectionist(t *testing.T) {
	stats := PlayerGameStats{Clicks: 10, Bullseyes: 5, BullseyeRate: 50.0}
	badges := EvaluateGameBadges(stats)
	if !hasBadge(badges, BadgePerfectionist) {
		t.Error("should earn Perfectionist with 50% bullseye rate")
	}
}

func TestEvaluateGameBadges_NoPerfectionist(t *testing.T) {
	stats := PlayerGameStats{Clicks: 10, Bullseyes: 4, BullseyeRate: 40.0}
	badges := EvaluateGameBadges(stats)
	if hasBadge(badges, BadgePerfectionist) {
		t.Error("should not earn Perfectionist with 40% bullseye rate")
	}
}

func TestEvaluateGameBadges_NoBadges(t *testing.T) {
	stats := PlayerGameStats{
		Clicks:       5,
		Score:        10,
		AvgReaction:  500,
		CPS:          1.0,
		BullseyeRate: 10.0,
		Bullseyes:    1,
	}
	badges := EvaluateGameBadges(stats)
	if len(badges) != 0 {
		t.Errorf("should earn no badges, got %d", len(badges))
	}
}

func TestEvaluateGameBadges_MultipleBadges(t *testing.T) {
	stats := PlayerGameStats{
		Clicks:       30,
		Score:        120,
		AvgReaction:  200,
		CPS:          3.5,
		BullseyeRate: 60.0,
		Bullseyes:    18,
	}
	badges := EvaluateGameBadges(stats)
	// Should earn: Sharpshooter, SpeedDemon, Centurion, TriggerHappy, Perfectionist
	if len(badges) != 5 {
		t.Errorf("should earn 5 badges, got %d", len(badges))
	}
}

func TestEvaluateLifetimeBadges_Unstoppable(t *testing.T) {
	stats := PlayerLifetimeStats{WinStreak: 3}
	badges := EvaluateLifetimeBadges(stats)
	if !hasBadge(badges, BadgeUnstoppable) {
		t.Error("should earn Unstoppable with 3-game win streak")
	}
}

func TestEvaluateLifetimeBadges_NoUnstoppable(t *testing.T) {
	stats := PlayerLifetimeStats{WinStreak: 2}
	badges := EvaluateLifetimeBadges(stats)
	if hasBadge(badges, BadgeUnstoppable) {
		t.Error("should not earn Unstoppable with 2-game win streak")
	}
}

func TestEvaluateLifetimeBadges_Veteran(t *testing.T) {
	stats := PlayerLifetimeStats{GamesPlayed: 10}
	badges := EvaluateLifetimeBadges(stats)
	if !hasBadge(badges, BadgeVeteran) {
		t.Error("should earn Veteran with 10 games")
	}
}

func TestEvaluateLifetimeBadges_NoVeteran(t *testing.T) {
	stats := PlayerLifetimeStats{GamesPlayed: 9}
	badges := EvaluateLifetimeBadges(stats)
	if hasBadge(badges, BadgeVeteran) {
		t.Error("should not earn Veteran with 9 games")
	}
}

func hasBadge(badges []Badge, id BadgeID) bool {
	for _, b := range badges {
		if b.ID == id {
			return true
		}
	}
	return false
}
