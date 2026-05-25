package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("CRON_SECRET", "secret")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FreeTierLimit != 10 {
		t.Fatal("expected default")
	}
}

func TestLoadMissingRequired(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("CRON_SECRET", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}
