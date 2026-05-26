package httpserver

import (
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"evidencevault/internal/audit"
	"evidencevault/internal/billing"
	"evidencevault/internal/email"
	"evidencevault/internal/evidence"
	"evidencevault/internal/operations"
	"evidencevault/internal/persistence"
	"evidencevault/internal/proofpack"
	"evidencevault/internal/reminders"
)

func TestRouteRegistration(t *testing.T) {
	ev := evidence.NewService(persistence.NewMemoryStore(), 10)
	a := audit.NewService(persistence.NewMemoryStore())
	s := Server{Version: "test", Evidence: ev, Proofpack: proofpack.NewService(persistence.NewMemoryStore(), a, ev), Reminders: reminders.NewService(persistence.NewMemoryStore(), email.LogSender{}, a, ev), Billing: &billing.Service{}, Templates: template.Must(template.New("x").Parse(`{{define "landing.html"}}ok{{end}}{{define "app.html"}}ok{{end}}`))}
	r := s.Routes()
	for _, p := range []string{"/healthz", "/readyz", "/version", "/", "/app", "/app/evidence", "/app/proofpacks"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Fatalf("route missing %s", p)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/billing/checkout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 got %d", w.Code)
	}
}

func TestExportRoutesDegradeGracefully(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	store := persistence.NewMemoryStore()
	ev := evidence.NewService(store, 10)
	a := audit.NewService(store)
	ops := operations.NewService(store, ev)
	s := Server{Version: "test", Evidence: ev, Proofpack: proofpack.NewService(store, a, ev), Reminders: reminders.NewService(store, email.LogSender{}, a, ev), Billing: &billing.Service{}, Operations: ops, Templates: template.Must(template.New("x").Parse(`{{define "landing.html"}}ok{{end}}{{define "app.html"}}ok{{end}}`))}
	r := s.Routes()
	for _, path := range []string{"/app/export/narratives.md", "/app/export/review-comparison.md", "/app/export/review-comparison.txt", "/app/export/pilot-proof.md", "/app/export/pilot-proof.txt"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Tenant-ID", "t")
		req.Header.Set("X-User-ID", "u")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status %d", path, w.Code)
		}
		body, _ := io.ReadAll(w.Body)
		if len(strings.TrimSpace(string(body))) == 0 {
			t.Fatalf("%s empty body", path)
		}
	}
}

func TestPilotProofExportWithHistoryIncludesNarrativeAndComparisonReadiness(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	store := persistence.NewMemoryStore()
	ev := evidence.NewService(store, 10)
	a := audit.NewService(store)
	ops := operations.NewService(store, ev)
	s := Server{Version: "test", Evidence: ev, Proofpack: proofpack.NewService(store, a, ev), Reminders: reminders.NewService(store, email.LogSender{}, a, ev), Billing: &billing.Service{}, Operations: ops, Templates: template.Must(template.New("x").Parse(`{{define "landing.html"}}ok{{end}}{{define "app.html"}}ok{{end}}`))}
	r := s.Routes()
	_, _ = ev.Create(t.Context(), "t", evidence.Item{Title: "Policy", Category: "Legal", Status: "active", OwnerName: "A", OwnerEmail: "a@example.com", ReminderDaysBefore: 30})
	_, _ = ops.GenerateReviewSnapshot(t.Context(), "t")
	_, _ = ops.GenerateReviewSnapshot(t.Context(), "t")
	req := httptest.NewRequest(http.MethodGet, "/app/export/pilot-proof.md", nil)
	req.Header.Set("X-Tenant-ID", "t")
	req.Header.Set("X-User-ID", "u")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Content-Type"); got != "text/markdown" {
		t.Fatalf("content type %s", got)
	}
	body := w.Body.String()
	if !strings.Contains(body, "## Narrative") || !strings.Contains(body, "## Comparison") || !strings.Contains(body, "Comparison readiness: true") {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestTemplatesRender(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	ev := evidence.NewService(persistence.NewMemoryStore(), 10)
	a := audit.NewService(persistence.NewMemoryStore())
	s := Server{Version: "test", Evidence: ev, Proofpack: proofpack.NewService(persistence.NewMemoryStore(), a, ev), Reminders: reminders.NewService(persistence.NewMemoryStore(), email.LogSender{}, a, ev), Billing: &billing.Service{}, Templates: template.Must(template.New("x").Parse(`{{define "landing.html"}}landing{{end}}{{define "app.html"}}app{{end}}`)), FreeTierLimit: 10, PersistenceMode: "memory", DegradedMode: true}
	r := s.Routes()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("landing status %d", w.Code)
	}
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	req.Header.Set("X-Tenant-ID", "t")
	req.Header.Set("X-User-ID", "u")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("app status %d", w.Code)
	}
}
