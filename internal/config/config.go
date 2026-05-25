package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppEnv              string
	Addr                string
	DatabaseURL         string
	S3Endpoint          string
	S3Bucket            string
	S3AccessKeyID       string
	S3SecretAccessKey   string
	StripeSecretKey     string
	StripeWebhookSecret string
	StripePriceID       string
	BaseURL             string
	CronSecret          string
	Version             string
	FreeTierLimit       int
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:              getenv("APP_ENV", "development"),
		Addr:                getenv("ADDR", ":8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		S3Endpoint:          os.Getenv("S3_ENDPOINT"),
		S3Bucket:            os.Getenv("S3_BUCKET"),
		S3AccessKeyID:       os.Getenv("S3_ACCESS_KEY_ID"),
		S3SecretAccessKey:   os.Getenv("S3_SECRET_ACCESS_KEY"),
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripePriceID:       os.Getenv("STRIPE_PRICE_ID"),
		BaseURL:             getenv("BASE_URL", "http://localhost:8080"),
		CronSecret:          os.Getenv("CRON_SECRET"),
		Version:             getenv("VERSION", "dev"),
		FreeTierLimit:       getenvInt("FREE_TIER_LIMIT", 10),
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.CronSecret == "" {
		return cfg, fmt.Errorf("CRON_SECRET is required")
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
