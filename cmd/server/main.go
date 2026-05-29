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
	"evidencevault/internal/demo"
	"evidencevault/internal/email"
	"evidencevault/internal/evidence"
	"evidencevault/internal/evidencegraph"
	httpserver "evidencevault/internal/http"
	"evidencevault/internal/operations"
	"evidencevault/internal/persistence"
	"evidencevault/internal/proofpack"
	"evidencevault/internal/reminders"
	"evidencevault/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	var store persistence.Store = persistence.NewMemoryStore()
	if cfg.PersistenceMode == "file" {
		fs, err := persistence.NewFileStore(cfg.DataDir)
		if err != nil {
			log.Fatal(err)
		}
		store = fs
		log.Printf("persistence mode=file data_dir=%s", cfg.DataDir)
	} else {
		log.Printf("persistence mode=memory (ephemeral/degraded)")
	}
	ctx := context.Background()
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	auditSvc := audit.NewService(store)
	ev := evidence.NewService(store, cfg.FreeTierLimit)
	opsSvc := operations.NewService(store, ev)
	billingSvc := &billing.Service{PriceID: cfg.StripePriceID, BaseURL: cfg.BaseURL, WebhookSecret: cfg.StripeWebhookSecret, SecretKey: cfg.StripeSecretKey, Audit: auditSvc, Store: store}
	ppSvc := proofpack.NewService(store, auditSvc, ev)
	graphBuilder := evidencegraph.NewBuilder(store, ev, opsSvc)
	if err := demo.Seed(ctx, cfg.AppEnv, cfg.DemoMode, ev, opsSvc, ppSvc, "pilot-demo"); err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Addr: cfg.Addr, Handler: httpserver.Server{Version: cfg.Version, Evidence: ev, Proofpack: ppSvc, Reminders: reminders.NewService(store, email.LogSender{}, auditSvc, ev), Storage: storage.LocalClient{BasePath: "uploads"}, Billing: billingSvc, Templates: tmpl, CronSecret: cfg.CronSecret, FreeTierLimit: cfg.FreeTierLimit, PersistenceMode: cfg.PersistenceMode, DegradedMode: cfg.DegradedMode, Operations: opsSvc, EvidenceGraph: graphBuilder}.Routes()}
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
