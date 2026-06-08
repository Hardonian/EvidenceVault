package httpserver

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"evidencevault/internal/evidence"
	"evidencevault/internal/evidencegraph"
	"evidencevault/internal/operations"
	"evidencevault/internal/persistence"
)

type errorStore struct {
	*persistence.MemoryStore
}

func (e *errorStore) Read(fn func(*persistence.State) error) error {
	return errors.New("simulated builder error")
}

func TestBuildEvidenceGraph_BuilderError(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	store := &errorStore{persistence.NewMemoryStore()}
	ev := evidence.NewService(store, 10)
	ops := operations.NewService(store, ev)
	gb := evidencegraph.NewBuilder(store, ev, ops)

	s := Server{
		Version:       "test",
		Evidence:      ev,
		Operations:    ops,
		EvidenceGraph: gb,
		Templates:     template.Must(template.New("x").Parse(`{{define "evidence_graph.html"}}ok{{end}}`)),
	}

	r := s.Routes()

	req := httptest.NewRequest(http.MethodGet, "/app/evidence-graph", nil)
	req.Header.Set("X-Tenant-ID", "t")
	req.Header.Set("X-User-ID", "u")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "simulated builder error") {
		t.Fatalf("expected error message in body, got: %s", w.Body.String())
	}
}

func TestBuildEvidenceGraph_Unavailable(t *testing.T) {
	s, _ := testServer(t)
	s.EvidenceGraph = nil
	r := s.Routes()

	req := httptest.NewRequest(http.MethodGet, "/app/evidence-graph", nil)
	req.Header.Set("X-Tenant-ID", "t")
	req.Header.Set("X-User-ID", "u")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "evidence graph unavailable") {
		t.Fatalf("expected error message in body, got: %s", w.Body.String())
	}
}
