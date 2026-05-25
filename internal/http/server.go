package httpserver

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"evidencevault/internal/auth"
	"evidencevault/internal/billing"
	"evidencevault/internal/evidence"
	"evidencevault/internal/operations"
	"evidencevault/internal/proofpack"
	"evidencevault/internal/reminders"
	"evidencevault/internal/storage"
)

type Server struct {
	Version         string
	Evidence        *evidence.Service
	Proofpack       *proofpack.Service
	Reminders       *reminders.Service
	Storage         storage.Client
	Billing         *billing.Service
	Operations      *operations.Service
	Templates       *template.Template
	CronSecret      string
	FreeTierLimit   int
	PersistenceMode string
	DegradedMode    bool
}

func method(m string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != m {
			w.Header().Set("Allow", m)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}
func (s Server) Routes() http.Handler {
	mux := http.NewServeMux()
	s.registerSystemRoutes(mux)
	s.registerAppRoutes(mux)
	s.registerAPIRoutes(mux)
	s.registerBillingRoutes(mux)
	return mux
}

func (s Server) registerSystemRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", method(http.MethodGet, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ok")) }))
	mux.HandleFunc("/readyz", method(http.MethodGet, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ready")) }))
	mux.HandleFunc("/version", method(http.MethodGet, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(s.Version)) }))
	mux.HandleFunc("/", method(http.MethodGet, s.landing))
}

func (s Server) registerAppRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/app", method(http.MethodGet, s.app))
	mux.HandleFunc("/app/evidence", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.listEvidence(w, r)
			return
		}
		if r.Method == http.MethodPost {
			s.createEvidence(w, r)
			return
		}
		http.Error(w, "method not allowed", 405)
	})
	mux.HandleFunc("/app/evidence/upload", method(http.MethodPost, s.uploadEvidence))
	mux.HandleFunc("/app/evidence/template", method(http.MethodPost, s.createEvidenceFromTemplate))
	mux.HandleFunc("/app/proofpacks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.listProofpacks(w, r)
			return
		}
		if r.Method == http.MethodPost {
			s.createProofpack(w, r)
			return
		}
		http.Error(w, "method not allowed", 405)
	})
	mux.HandleFunc("/app/reviews", method(http.MethodPost, s.generateReview))
	mux.HandleFunc("/app/export/evidence.csv", method(http.MethodGet, s.exportEvidenceCSV))
	mux.HandleFunc("/app/export/reviews.csv", method(http.MethodGet, s.exportReviewCSV))
	mux.HandleFunc("/billing/checkout", method(http.MethodPost, s.checkout))
	mux.HandleFunc("/billing/portal", method(http.MethodPost, s.portal))
	mux.HandleFunc("/webhooks/stripe", method(http.MethodPost, s.webhook))
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
	sum := operations.Summary{}
	if s.Operations != nil {
		sum, _ = s.Operations.BuildSummary(r.Context(), c.TenantID, items)
	}
	_ = s.Templates.ExecuteTemplate(w, "app.html", map[string]any{"Tenant": c.TenantID, "Items": items, "Proofpacks": packs, "Dashboard": buildDashboardViewModel(items, packs, s.FreeTierLimit, s.PersistenceMode, s.DegradedMode, sum)})
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
	if s.Operations != nil {
		s.Operations.RecordEvent(c.TenantID, "evidence.created", "Evidence created", id)
	}
}

func (s Server) createEvidenceFromTemplate(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	key := r.FormValue("template")
	t, found := evidence.TemplateByKey(time.Now().UTC(), key)
	if !found {
		http.Error(w, "unknown template", 400)
		return
	}
	id, err := s.Evidence.Create(r.Context(), c.TenantID, t.Item)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if s.Operations != nil {
		s.Operations.RecordEvent(c.TenantID, "evidence.template.created", "Evidence created from template", id)
	}
	http.Redirect(w, r, "/app", http.StatusSeeOther)
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
	loc, err := s.Storage.Upload(r.Context(), hdr.Filename, file)
	if err != nil {
		http.Error(w, "storage unavailable: "+err.Error(), 503)
		return
	}
	if err := s.Evidence.AttachFile(r.Context(), c.TenantID, evidenceID, loc, ctype, hdr.Size); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if s.Operations != nil {
		s.Operations.RecordEvent(c.TenantID, "evidence.file.uploaded", "Evidence file uploaded", evidenceID)
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
	if s.Operations != nil {
		s.Operations.RecordEvent(c.TenantID, "proofpack.generated", "Proofpack exported", "")
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
	if s.Operations != nil {
		s.Operations.RecordEvent("global", "reminders.run", "Reminder run completed", "")
	}
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

func (s Server) generateReview(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	snap, err := s.Operations.GenerateReviewSnapshot(r.Context(), c.TenantID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = json.NewEncoder(w).Encode(snap)
}

func (s Server) exportEvidenceCSV(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	items, err := s.Evidence.List(r.Context(), c.TenantID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var b bytes.Buffer
	cw := csv.NewWriter(&b)
	_ = cw.Write([]string{"id", "title", "category", "status", "owner_name", "owner_email", "updated_at"})
	for _, it := range items {
		_ = cw.Write([]string{it.ID, it.Title, it.Category, it.Status, it.OwnerName, it.OwnerEmail, it.UpdatedAt.Format(time.RFC3339)})
	}
	cw.Flush()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="evidence.csv"`)
	_, _ = w.Write(b.Bytes())
}

func (s Server) exportReviewCSV(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	var snaps []operations.Summary
	if s.Operations != nil {
		sum, _ := s.Operations.BuildSummary(r.Context(), c.TenantID)
		snaps = append(snaps, sum)
	}
	var b bytes.Buffer
	cw := csv.NewWriter(&b)
	_ = cw.Write([]string{"generated_at", "health_score", "unresolved", "expired", "expiring_soon", "missing_owner", "stale_evidence", "maturity_stage"})
	for _, s := range snaps {
		_ = cw.Write([]string{time.Now().UTC().Format(time.RFC3339), intToStr(s.HealthScore), intToStr(s.Unresolved), intToStr(s.Expired), intToStr(s.ExpiringSoon), intToStr(s.MissingOwner), intToStr(s.StaleEvidence), s.PilotMaturityStage})
	}
	cw.Flush()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="review_snapshots.csv"`)
	_, _ = w.Write(b.Bytes())
}
func intToStr(n int) string { return strconv.Itoa(n) }
