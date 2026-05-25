package httpserver

import (
	"testing"
	"time"

	"evidencevault/internal/evidence"
	"evidencevault/internal/operations"
)

func TestBuildDashboardViewModel(t *testing.T) {
	now := time.Now().UTC()
	vm := buildDashboardViewModel([]evidence.Item{{Status: "expiring", OwnerEmail: "a@b.com"}, {Status: "expired"}, {Status: "active"}}, []map[string]any{{"id": "1", "created_at": now}}, 10, "memory", true, operations.Summary{HealthScore: 77, Unresolved: 2, NextRecommendedReview: now})
	if vm.TotalEvidence != 3 || vm.ExpiringSoon != 1 || vm.Expired != 1 || vm.MissingOwner != 2 {
		t.Fatalf("unexpected counters: %+v", vm)
	}
	if vm.TotalProofpacks != 1 || vm.LatestProofpackTime == "not generated yet" {
		t.Fatalf("expected proofpack data: %+v", vm)
	}
}
