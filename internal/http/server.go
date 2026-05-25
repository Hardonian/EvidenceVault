package httpserver

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"

	"evidencevault/internal/auth"
	"evidencevault/internal/billing"
	"evidencevault/internal/evidence"
	"evidencevault/internal/proofpack"
	"evidencevault/internal/reminders"
	"evidencevault/internal/storage"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	Version    string
	Evidence   *evidence.Service
	Proofpack  *proofpack.Service
	Reminders  *reminders.Service
	Storage    storage.Client
	Billing    billing.Service
	Templates  *template.Template
	CronSecret string
}

func (s Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", s.landing)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	r.Get("/version", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(s.Version)) })
	r.Get("/app", s.app)
	r.Get("/api/evidence", s.listEvidence)
	r.Post("/api/evidence", s.createEvidence)
	r.Post("/api/evidence/upload", s.uploadEvidence)
	r.Get("/api/proofpack.json", s.exportProofpack)
	r.Post("/internal/reminders/run", s.runReminders)
	r.Post("/billing/checkout", s.checkout)
	r.Post("/billing/portal", s.portal)
	r.Post("/billing/webhook", s.webhook)
	return r
}

func (s Server) landing(w http.ResponseWriter, r *http.Request) {
	_ = s.Templates.ExecuteTemplate(w, "landing.html", nil)
}
func (s Server) app(w http.ResponseWriter, r *http.Request) {
	tenant := auth.TenantIDFromRequest(r)
	items, _ := s.Evidence.List(r.Context(), tenant)
	_ = s.Templates.ExecuteTemplate(w, "app.html", map[string]any{"Tenant": tenant, "Items": items})
}
func (s Server) listEvidence(w http.ResponseWriter, r *http.Request) {
	tenant := auth.TenantIDFromRequest(r)
	items, err := s.Evidence.List(r.Context(), tenant)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}
func (s Server) createEvidence(w http.ResponseWriter, r *http.Request) {
	tenant := auth.TenantIDFromRequest(r)
	var it evidence.Item
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	id, err := s.Evidence.Create(r.Context(), tenant, it)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}
func (s Server) uploadEvidence(w http.ResponseWriter, r *http.Request) {
	file, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	defer file.Close()
	loc, err := s.Storage.Upload(r.Context(), hdr.Filename, file)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"path": loc})
}
func (s Server) exportProofpack(w http.ResponseWriter, r *http.Request) {
	tenant := auth.TenantIDFromRequest(r)
	b, err := s.Proofpack.Export(r.Context(), tenant)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
func (s Server) runReminders(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+s.CronSecret {
		http.Error(w, "unauthorized", 401)
		return
	}
	n, err := s.Reminders.Run(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]int{"sent": n})
}
func (s Server) checkout(w http.ResponseWriter, r *http.Request) {
	url, err := s.Billing.CheckoutURL(r.URL.Query().Get("customer_id"))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, url, 302)
}
func (s Server) portal(w http.ResponseWriter, r *http.Request) {
	url, err := s.Billing.PortalURL(r.URL.Query().Get("customer_id"))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, url, 302)
}
func (s Server) webhook(w http.ResponseWriter, r *http.Request) {
	event, payload, err := s.Billing.VerifyWebhook(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	slog.Info("stripe_webhook", "type", event.Type, "id", event.ID, "bytes", len(payload))
	w.WriteHeader(200)
}
