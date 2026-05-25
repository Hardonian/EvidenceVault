package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppEnv, Addr, PersistenceMode, DataDir                                            string
	AllowEphemeralProduction                                                          bool
	DegradedMode, DemoSeed                                                            bool
	StripeSecretKey, StripeWebhookSecret, StripePriceID, BaseURL, CronSecret, Version string
	FreeTierLimit                                                                     int
}

func Load() (Config, error) {
	cfg := Config{AppEnv: getenv("APP_ENV", "development"), Addr: getenv("ADDR", ":8080"), PersistenceMode: getenv("PERSISTENCE_MODE", "memory"), DataDir: getenv("DATA_DIR", "./data"), AllowEphemeralProduction: getenv("ALLOW_EPHEMERAL_PRODUCTION", "") == "true", StripeSecretKey: os.Getenv("STRIPE_SECRET_KEY"), StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"), StripePriceID: os.Getenv("STRIPE_PRICE_ID"), BaseURL: getenv("BASE_URL", "http://localhost:8080"), CronSecret: os.Getenv("CRON_SECRET"), Version: getenv("VERSION", "dev"), FreeTierLimit: getenvInt("FREE_TIER_LIMIT", 10), DemoSeed: getenv("DEMO_SEED", "") == "true"}
	if cfg.CronSecret == "" {
		return cfg, fmt.Errorf("CRON_SECRET is required")
	}
	if cfg.PersistenceMode != "memory" && cfg.PersistenceMode != "file" {
		return cfg, fmt.Errorf("PERSISTENCE_MODE must be memory or file")
	}
	if cfg.PersistenceMode == "file" && cfg.DataDir == "" {
		return cfg, fmt.Errorf("DATA_DIR is required when PERSISTENCE_MODE=file")
	}
	cfg.DegradedMode = cfg.PersistenceMode == "memory"
	if cfg.AppEnv == "production" && cfg.PersistenceMode == "memory" && !cfg.AllowEphemeralProduction {
		return cfg, fmt.Errorf("production requires durable persistence: set PERSISTENCE_MODE=file or ALLOW_EPHEMERAL_PRODUCTION=true")
	}
	return cfg, nil
}
func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func getenvInt(k string, d int) int {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return d
	}
	return n
}
