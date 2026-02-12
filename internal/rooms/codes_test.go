package rooms

import (
	"regexp"
	"testing"
)

func TestGenerateCode_Format(t *testing.T) {
	pattern := regexp.MustCompile(`^[ABCDEFGHJKMNPQRSTUVWXYZ23456789]{4}$`)

	for i := 0; i < 100; i++ {
		code, err := GenerateCode()
		if err != nil {
			t.Fatalf("GenerateCode() error: %v", err)
		}
		if !pattern.MatchString(code) {
			t.Errorf("GenerateCode() = %q, doesn't match expected pattern", code)
		}
	}
}

func TestGenerateCode_Length(t *testing.T) {
	code, err := GenerateCode()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != codeLength {
		t.Errorf("code length = %d, want %d", len(code), codeLength)
	}
}

func TestGenerateCode_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	dupes := 0
	for i := 0; i < 1000; i++ {
		code, err := GenerateCode()
		if err != nil {
			t.Fatal(err)
		}
		if seen[code] {
			dupes++
		}
		seen[code] = true
	}
	// With 30^4 = 810k combinations, 1000 samples should have essentially no dupes
	if dupes > 5 {
		t.Errorf("too many duplicate codes: %d out of 1000", dupes)
	}
}

func TestGenerateCode_NoAmbiguousChars(t *testing.T) {
	ambiguous := "0OIL1"
	for i := 0; i < 100; i++ {
		code, err := GenerateCode()
		if err != nil {
			t.Fatal(err)
		}
		for _, ch := range code {
			for _, a := range ambiguous {
				if ch == a {
					t.Errorf("code %q contains ambiguous character %c", code, ch)
				}
			}
		}
	}
}
