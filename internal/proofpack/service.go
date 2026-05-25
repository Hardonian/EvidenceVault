package proofpack

import (
	"context"
	"encoding/json"
	"time"

	"evidencevault/internal/audit"
	"evidencevault/internal/evidence"
	"evidencevault/internal/id"
)

type Service struct {
	evidence *evidence.Service
	audit    *audit.Service
	packs    map[string][]map[string]any
}

func NewService(_ any, auditSvc *audit.Service, ev *evidence.Service) *Service {
	return &Service{evidence: ev, audit: auditSvc, packs: map[string][]map[string]any{}}
}
func (s *Service) List(_ context.Context, tenantID string) ([]map[string]any, error) {
	return append([]map[string]any{}, s.packs[tenantID]...), nil
}
func (s *Service) Export(ctx context.Context, tenantID, version string) ([]byte, error) {
	items, _ := s.evidence.List(ctx, tenantID)
	payload := map[string]any{"generated_at": time.Now().UTC().Format(time.RFC3339), "app_version": version, "tenant": map[string]any{"id": tenantID, "plan": "free"}, "evidence_records": items, "linked_files": s.evidence.Files(tenantID), "reminder_log_history": []map[string]any{}, "audit_log_summary": s.audit.ListByTenant(tenantID, 100), "limitations": "EvidenceVault assists compliance operations and does not certify compliance, provide legal advice, or guarantee filing accuracy."}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	pid := id.New()
	s.packs[tenantID] = append([]map[string]any{{"id": pid, "created_at": time.Now().UTC()}}, s.packs[tenantID]...)
	s.audit.Log(ctx, tenantID, "", "proofpack.generated", "proofpack", pid, "{}")
	return b, nil
}
