package httpserver

import "net/http"

func (s Server) registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/reminders/run", method(http.MethodPost, s.runReminders))
}

func (s Server) registerBillingRoutes(_ *http.ServeMux) {}
