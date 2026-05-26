package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"evidencevault/internal/evidencegraph"
)

func (s Server) buildEvidenceGraph(w http.ResponseWriter, r *http.Request) (evidencegraph.Graph, bool) {
	c, ok := s.authContext(w, r)
	if !ok {
		return evidencegraph.Graph{}, false
	}
	if s.EvidenceGraph == nil {
		http.Error(w, "evidence graph unavailable", http.StatusServiceUnavailable)
		return evidencegraph.Graph{}, false
	}
	g, err := s.EvidenceGraph.Build(r.Context(), c.TenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return evidencegraph.Graph{}, false
	}
	return g, true
}

func (s Server) evidenceGraphJSON(w http.ResponseWriter, r *http.Request) {
	g, ok := s.buildEvidenceGraph(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(g)
}

func (s Server) evidenceGraphPage(w http.ResponseWriter, r *http.Request) {
	g, ok := s.buildEvidenceGraph(w, r)
	if !ok {
		return
	}
	_ = s.Templates.ExecuteTemplate(w, "evidence_graph.html", map[string]any{"Graph": g})
}

func (s Server) exportEvidenceGraphMarkdown(w http.ResponseWriter, r *http.Request) {
	g, ok := s.buildEvidenceGraph(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/markdown")
	_, _ = w.Write([]byte(evidencegraph.Markdown(g)))
}

func (s Server) exportEvidenceGraphText(w http.ResponseWriter, r *http.Request) {
	g, ok := s.buildEvidenceGraph(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(evidencegraph.Text(g)))
}

func (s Server) exportEvidenceGraphJSON(w http.ResponseWriter, r *http.Request) {
	s.evidenceGraphJSON(w, r)
}

func (s Server) updateEvidenceMappings(w http.ResponseWriter, r *http.Request) {
	c, ok := s.authContext(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("evidence_id"))
	if id == "" {
		http.Error(w, "evidence_id is required", http.StatusBadRequest)
		return
	}
	item, err := s.Evidence.Get(r.Context(), c.TenantID, id)
	if err != nil {
		http.Error(w, "evidence not found", http.StatusNotFound)
		return
	}
	item.ControlRefs = splitRefs(r.FormValue("control_refs"))
	item.VendorRefs = splitRefs(r.FormValue("vendor_refs"))
	item.RiskRefs = splitRefs(r.FormValue("risk_refs"))
	if err := s.Evidence.Update(r.Context(), c.TenantID, id, item); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if s.Operations != nil {
		s.Operations.RecordEvent(c.TenantID, "evidence.mappings.updated", "Evidence mappings updated", id)
	}
	http.Redirect(w, r, "/app/evidence-graph", http.StatusSeeOther)
}

func splitRefs(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\n", ",")
	parts := strings.Split(raw, ",")
	out := []string{}
	seen := map[string]struct{}{}
	for _, p := range parts {
		clean := strings.Join(strings.Fields(p), " ")
		if clean == "" {
			continue
		}
		key := strings.ToLower(clean)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, clean)
	}
	return out
}
