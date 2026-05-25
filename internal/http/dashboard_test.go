package httpserver

import (
	"testing"
	"time"

	"evidencevault/internal/evidence"
	"evidencevault/internal/operations"
)

func TestBuildDashboardViewModel(t *testing.T) {
	now := time.Now().UTC()
	vm := buildDashboardViewModel([]evidence.Item{{Status: "expiring", OwnerEmail: "a@b.com"}, {Status: "expired"}, {Status: "active"}}, []map[string]any{{"id": "1", "created_at": now}}, 10, "memory", true, operations.Summary{HealthScore: 77, Unresolved: 2, NextRecommendedReview: now, PreviousHealthScore: 70, HealthDelta: 7})
	if vm.TotalEvidence != 3 || vm.ExpiringSoon != 1 || vm.Expired != 1 || vm.MissingOwner != 2 {
		t.Fatalf("unexpected counters")
	}
	if vm.Onboarding {
		t.Fatal("should not be onboarding")
	}
	if vm.PriorityQueue[0].Label != "Expired evidence" {
		t.Fatal("priority order broken")
	}
}

func TestBuildDashboardOnboarding(t *testing.T) {
	now := time.Now().UTC()
	vm := buildDashboardViewModel(nil, nil, 10, "file", false, operations.Summary{NextRecommendedReview: now})
	if !vm.Onboarding || len(vm.StarterTemplates) != 6 {
		t.Fatal("expected onboarding templates")
	}
}
