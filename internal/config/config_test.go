package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("ROUND_DURATION", "")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "")
	}
	if cfg.RoundDuration != 60 {
		t.Errorf("RoundDuration = %d, want %d", cfg.RoundDuration, 60)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("PORT", "3000")
	t.Setenv("DATABASE_URL", "postgres://localhost/clicktrainer")
	t.Setenv("ROUND_DURATION", "30")

	cfg := Load()

	if cfg.Port != "3000" {
		t.Errorf("Port = %q, want %q", cfg.Port, "3000")
	}
	if cfg.DatabaseURL != "postgres://localhost/clicktrainer" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://localhost/clicktrainer")
	}
	if cfg.RoundDuration != 30 {
		t.Errorf("RoundDuration = %d, want %d", cfg.RoundDuration, 30)
	}
}

func TestLoad_InvalidRoundDuration(t *testing.T) {
	t.Setenv("ROUND_DURATION", "abc")

	cfg := Load()

	if cfg.RoundDuration != 60 {
		t.Errorf("RoundDuration = %d, want %d (fallback)", cfg.RoundDuration, 60)
	}
}
