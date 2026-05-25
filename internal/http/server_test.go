package httpserver

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"evidencevault/internal/audit"
	"evidencevault/internal/billing"
	"evidencevault/internal/email"
	"evidencevault/internal/evidence"
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
