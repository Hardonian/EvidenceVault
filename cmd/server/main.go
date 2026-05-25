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

	"evidencevault/internal/audit"
	"evidencevault/internal/billing"
	"evidencevault/internal/config"
	"evidencevault/internal/email"
	"evidencevault/internal/evidence"
	httpserver "evidencevault/internal/http"
	"evidencevault/internal/proofpack"
	"evidencevault/internal/reminders"
	"evidencevault/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.DegradedMode {
		log.Printf("DATABASE_URL not set: running in local in-memory degraded mode")
	}
	ctx := context.Background()
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	auditSvc := audit.NewService()
	ev := evidence.NewService(nil, cfg.FreeTierLimit)
	billingSvc := &billing.Service{PriceID: cfg.StripePriceID, BaseURL: cfg.BaseURL, WebhookSecret: cfg.StripeWebhookSecret, SecretKey: cfg.StripeSecretKey, Audit: auditSvc}
	srv := &http.Server{Addr: cfg.Addr, Handler: httpserver.Server{Version: cfg.Version, Evidence: ev, Proofpack: proofpack.NewService(nil, auditSvc, ev), Reminders: reminders.NewService(nil, email.LogSender{}, auditSvc, ev), Storage: storage.LocalClient{BasePath: "uploads"}, Billing: billingSvc, Templates: tmpl, CronSecret: cfg.CronSecret}.Routes()}
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
