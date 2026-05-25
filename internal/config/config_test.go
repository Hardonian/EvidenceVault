package config

import "testing"

func TestLoad(t *testing.T) {
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
	t.Setenv("CRON_SECRET", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}
func TestProductionFailsClosedMemory(t *testing.T) {
	t.Setenv("CRON_SECRET", "secret")
	t.Setenv("APP_ENV", "production")
	t.Setenv("PERSISTENCE_MODE", "memory")
	t.Setenv("ALLOW_EPHEMERAL_PRODUCTION", "false")
	_, err := Load()
	if err == nil {
		t.Fatal("expected fail closed")
	}
}
