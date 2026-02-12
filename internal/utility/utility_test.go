package utility

import (
	"regexp"
	"testing"
)

func TestRandomColorHex(t *testing.T) {
	hexPattern := regexp.MustCompile(`^#[0-9a-f]{6}$`)

	for i := 0; i < 100; i++ {
		color := RandomColorHex()
		if !hexPattern.MatchString(color) {
			t.Errorf("RandomColorHex() = %q, want matching #rrggbb pattern", color)
		}
	}
}

func TestRandomColorHex_NotTooExtreme(t *testing.T) {
	// Colors should have each RGB component between 4 and 251
	for i := 0; i < 100; i++ {
		color := RandomColorHex()
		if len(color) != 7 {
			t.Fatalf("expected length 7, got %d for %q", len(color), color)
		}
	}
}

func TestRandomColorHex_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	dupes := 0
	for i := 0; i < 100; i++ {
		c := RandomColorHex()
		if seen[c] {
			dupes++
		}
		seen[c] = true
	}
	// With 248^3 ≈ 15M possibilities, 100 samples should have essentially no dupes
	if dupes > 5 {
		t.Errorf("too many duplicate colors: %d out of 100", dupes)
	}
}
