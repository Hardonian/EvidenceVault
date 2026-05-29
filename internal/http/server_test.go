package httpserver

import (
	"context"
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
	"evidencevault/internal/evidencegraph"
	"evidencevault/internal/operations"
	"evidencevault/internal/persistence"
	"evidencevault/internal/proofpack"
	"evidencevault/internal/reminders"
)

func testServer(t *testing.T) (Server, http.Handler) {
	t.Helper()
	t.Setenv("APP_ENV", "development")
	store := persistence.NewMemoryStore()
	ev := evidence.NewService(store, 10)
	a := audit.NewService(store)
	ops := operations.NewService(store, ev)
	gb := evidencegraph.NewBuilder(store, ev, ops)
	s := Server{Version: "test", Evidence: ev, Proofpack: proofpack.NewService(store, a, ev), Reminders: reminders.NewService(store, email.LogSender{}, a, ev), Billing: &billing.Service{}, Operations: ops, EvidenceGraph: gb, Templates: template.Must(template.New("x").Parse(`{{define "landing.html"}}ok{{end}}{{define "app.html"}}ok{{end}}{{define "evidence_graph.html"}}ok{{end}}`))}
	return s, s.Routes()
}

func TestRouteRegistration(t *testing.T) {
	_, r := testServer(t)
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
	_, r := testServer(t)
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
	s, r := testServer(t)
	_, _ = s.Evidence.Create(context.Background(), "t", evidence.Item{Title: "Policy", Category: "Legal", Status: "active", OwnerName: "A", OwnerEmail: "a@example.com", ReminderDaysBefore: 30})
	_, _ = s.Operations.GenerateReviewSnapshot(context.Background(), "t")
	_, _ = s.Operations.GenerateReviewSnapshot(context.Background(), "t")
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
	_, r := testServer(t)
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

// --- Evidence Graph route tests ---

func TestGraphPageRenders(t *testing.T) {
	_, r := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/app/evidence-graph", nil)
	req.Header.Set("X-Tenant-ID", "t")
	req.Header.Set("X-User-ID", "u")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("graph page status %d", w.Code)
	}
}

func TestGraphExportRoutes(t *testing.T) {
	s, r := testServer(t)
	_, _ = s.Evidence.Create(context.Background(), "t", evidence.Item{Title: "X", Category: "IT", Status: "active", OwnerEmail: "o@t.com", ReminderDaysBefore: 30})

	tests := []struct {
		path        string
		contentType string
	}{
		{"/app/export/evidence-graph.md", "text/markdown"},
		{"/app/export/evidence-graph.txt", "text/plain"},
		{"/app/export/evidence-graph.json", "application/json"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		req.Header.Set("X-Tenant-ID", "t")
		req.Header.Set("X-User-ID", "u")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status %d", tt.path, w.Code)
		}
		if got := w.Header().Get("Content-Type"); got != tt.contentType {
			t.Fatalf("%s content type: got %s want %s", tt.path, got, tt.contentType)
		}
		body, _ := io.ReadAll(w.Body)
		if len(strings.TrimSpace(string(body))) == 0 {
			t.Fatalf("%s empty body", tt.path)
		}
	}
}

func TestGraphAPIRoutes(t *testing.T) {
	s, r := testServer(t)
	_, _ = s.Evidence.Create(context.Background(), "t", evidence.Item{Title: "X", Category: "IT", Status: "active", OwnerEmail: "o@t.com", ReminderDaysBefore: 30})

	for _, path := range []string{"/app/api/evidence-graph"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Tenant-ID", "t")
		req.Header.Set("X-User-ID", "u")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status %d", path, w.Code)
		}
		if got := w.Header().Get("Content-Type"); got != "application/json" {
			t.Fatalf("%s content type: %s", path, got)
		}
	}
}

func TestGraphRoutesRequireAuth(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	store := persistence.NewMemoryStore()
	ev := evidence.NewService(store, 10)
	a := audit.NewService(store)
	ops := operations.NewService(store, ev)
	gb := evidencegraph.NewBuilder(store, ev, ops)
	s := Server{Version: "test", Evidence: ev, Proofpack: proofpack.NewService(store, a, ev), Reminders: reminders.NewService(store, email.LogSender{}, a, ev), Billing: &billing.Service{}, Operations: ops, EvidenceGraph: gb, Templates: template.Must(template.New("x").Parse(`{{define "landing.html"}}ok{{end}}{{define "app.html"}}ok{{end}}{{define "evidence_graph.html"}}ok{{end}}`))}
	r := s.Routes()
	for _, path := range []string{"/app/evidence-graph", "/app/api/evidence-graph", "/app/export/evidence-graph.md", "/app/export/evidence-graph.json"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusOK {
			t.Fatalf("%s should require auth but returned 200", path)
		}
	}
}

func TestEmptyTenantGraphDegrades(t *testing.T) {
	_, r := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/app/export/evidence-graph.md", nil)
	req.Header.Set("X-Tenant-ID", "empty")
	req.Header.Set("X-User-ID", "u")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Degraded State") {
		t.Fatal("expected degraded state in empty tenant graph")
	}
}
