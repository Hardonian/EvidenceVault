package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"evidencevault/internal/billing"
	"evidencevault/internal/config"
	"evidencevault/internal/db"
	"evidencevault/internal/evidence"
	httpserver "evidencevault/internal/http"
	"evidencevault/internal/proofpack"
	"evidencevault/internal/reminders"
	"evidencevault/internal/storage"
	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v78"
)

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	stripe.Key = cfg.StripeSecretKey
	ctx := context.Background()
	database, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Pool.Close()
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	srv := &http.Server{Addr: cfg.Addr, Handler: httpserver.Server{Version: cfg.Version, Evidence: evidence.NewService(database.Pool, cfg.FreeTierLimit), Proofpack: proofpack.NewService(database.Pool), Reminders: reminders.NewService(database.Pool), Storage: storage.LocalClient{BasePath: "uploads"}, Billing: billing.Service{PriceID: cfg.StripePriceID, BaseURL: cfg.BaseURL, WebhookSecret: cfg.StripeWebhookSecret}, Templates: tmpl, CronSecret: cfg.CronSecret}.Routes()}
	go func() {
		log.Printf("listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(c)
}
