package httpserver

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRouteRegistration(t *testing.T) {
	os.Setenv("APP_ENV", "development")
	s := Server{Version: "test", Templates: template.Must(template.New("x").Parse(`{{define "landing.html"}}ok{{end}}{{define "app.html"}}ok{{end}}`))}
	r := s.Routes()
	for _, p := range []string{"/healthz", "/readyz", "/version", "/", "/app"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Fatalf("route missing %s", p)
		}
	}
}
