package httpserver

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

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
	Billing    *billing.Service
	Templates  *template.Template
	CronSecret string
}

func (s Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ok")) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ready")) })
	r.Get("/version", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(s.Version)) })
	r.Get("/", s.landing)
	r.Get("/app", s.app)
	r.Get("/app/evidence", s.listEvidence)
	r.Post("/app/evidence", s.createEvidence)
	r.Put("/app/evidence/{id}", s.updateEvidence)
	r.Post("/app/evidence/upload", s.uploadEvidence)
	r.Get("/app/proofpacks", s.listProofpacks)
	r.Post("/app/proofpacks", s.createProofpack)
	r.Post("/api/cron/reminders", s.runReminders)
	r.Post("/billing/checkout", s.checkout)
	r.Post("/billing/portal", s.portal)
	r.Post("/webhooks/stripe", s.webhook)
	return r
}
func (s Server) authContext(w http.ResponseWriter, r *http.Request) (auth.Context, bool) {
	c, err := auth.FromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), 401)
		return auth.Context{}, false
	}
	return c, true
}
func (s Server) landing(w http.ResponseWriter, r *http.Request) {
	_ = s.Templates.ExecuteTemplate(w, "landing.html", nil)
}
func (s Server) app(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	items, _ := s.Evidence.List(r.Context(), c.TenantID)
	packs, _ := s.Proofpack.List(r.Context(), c.TenantID)
	_ = s.Templates.ExecuteTemplate(w, "app.html", map[string]any{"Tenant": c.TenantID, "Items": items, "Proofpacks": packs})
}
func (s Server) listEvidence(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	items, err := s.Evidence.List(r.Context(), c.TenantID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}
func (s Server) createEvidence(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	var it evidence.Item
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	id, err := s.Evidence.Create(r.Context(), c.TenantID, it)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}
func (s Server) updateEvidence(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	var it evidence.Item
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := s.Evidence.Update(r.Context(), c.TenantID, chi.URLParam(r, "id"), it); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.WriteHeader(204)
}
func (s Server) uploadEvidence(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	evidenceID := r.FormValue("evidence_id")
	if evidenceID == "" {
		http.Error(w, "evidence_id is required", 400)
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	defer file.Close()
	ctype := hdr.Header.Get("Content-Type")
	if !strings.HasPrefix(ctype, "application/pdf") && !strings.HasPrefix(ctype, "image/") && ctype != "text/plain" {
		http.Error(w, "unsupported MIME type", 400)
		return
	}
	if hdr.Size > 10<<20 {
		http.Error(w, "file too large", 400)
		return
	}
	loc, err := s.Storage.Upload(r.Context(), hdr.Filename, file)
	if err != nil {
		http.Error(w, "storage unavailable: "+err.Error(), 503)
		return
	}
	if err := s.Evidence.AttachFile(r.Context(), c.TenantID, evidenceID, loc, ctype, hdr.Size); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"path": loc})
}
func (s Server) listProofpacks(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	packs, err := s.Proofpack.List(r.Context(), c.TenantID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(packs)
}
func (s Server) createProofpack(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	b, err := s.Proofpack.Export(r.Context(), c.TenantID, s.Version)
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
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	url, err := s.Billing.CheckoutURL(r.Context(), c.TenantID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"url": url})
}
func (s Server) portal(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	url, err := s.Billing.PortalURL(r.Context(), c.TenantID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"url": url})
}
func (s Server) webhook(w http.ResponseWriter, r *http.Request) {
	event, payload, err := s.Billing.VerifyWebhook(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := s.Billing.RecordAndProcessEvent(r.Context(), *event, payload); err != nil {
		if errors.Is(err, billing.ErrDuplicateEvent) {
			w.WriteHeader(200)
			return
		}
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(200)
}

var _ = strconv.IntSize
