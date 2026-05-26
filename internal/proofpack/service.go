package proofpack

import (
	"context"
	"encoding/json"
	"time"

	"evidencevault/internal/audit"
	"evidencevault/internal/evidence"
	"evidencevault/internal/id"
	"evidencevault/internal/persistence"
)

type Service struct {
	evidence *evidence.Service
	audit    *audit.Service
	store    persistence.Store
}

func NewService(store persistence.Store, auditSvc *audit.Service, ev *evidence.Service) *Service {
	return &Service{store: store, evidence: ev, audit: auditSvc}
}
func (s *Service) List(_ context.Context, tenantID string) ([]map[string]any, error) {
	out := []map[string]any{}
	_ = s.store.Read(func(st *persistence.State) error {
		for _, p := range st.Proofpacks[tenantID] {
			out = append(out, map[string]any{"id": p.ID, "created_at": p.CreatedAt})
		}
		return nil
	})
	return out, nil
}
func (s *Service) Export(ctx context.Context, tenantID, version string) ([]byte, error) {
	items, _ := s.evidence.List(ctx, tenantID)
	payload := map[string]any{"generated_at": time.Now().UTC().Format(time.RFC3339), "app_version": version, "tenant": map[string]any{"id": tenantID, "plan": "free"}, "evidence_records": items, "linked_files": s.evidence.Files(tenantID), "reminder_log_history": []map[string]any{}, "audit_log_summary": s.audit.ListByTenant(tenantID, 100), "limitations": "EvidenceVault assists compliance operations and does not certify compliance, provide legal advice, or guarantee filing accuracy."}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	pid := id.New()
	_ = s.store.Write(func(st *persistence.State) error {
		evidenceIDs := make([]string, 0, len(items))
		for _, it := range items {
			evidenceIDs = append(evidenceIDs, it.ID)
		}
		st.Proofpacks[tenantID] = append([]persistence.ProofpackMeta{{ID: pid, CreatedAt: time.Now().UTC(), EvidenceIDs: evidenceIDs}}, st.Proofpacks[tenantID]...)
		return nil
	})
	s.audit.Log(ctx, tenantID, "", "proofpack.generated", "proofpack", pid, "{}")
	return b, nil
}
