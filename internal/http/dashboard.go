package httpserver

import (
	"fmt"
	"time"

	"evidencevault/internal/evidence"
	"evidencevault/internal/operations"
)

type DashboardViewModel struct {
	TotalEvidence, ExpiringSoon, Expired, MissingOwner, TotalProofpacks int
	LatestProofpackTime, Plan, FreeTierUsage, PersistenceMode           string
	DegradedWarnings                                                    []string
	HealthScore, UnresolvedIssues, StaleEvidence                        int
	LastReviewedAt, NextRecommendedReview                               string
	OwnerSummaries                                                      []operations.OwnerSummary
	RecentActivity                                                      []operations.Activity
	Cadence                                                             []string
}

func buildDashboardViewModel(items []evidence.Item, proofpacks []map[string]any, freeTierLimit int, persistenceMode string, degradedMode bool, summary operations.Summary) DashboardViewModel {
	vm := DashboardViewModel{TotalEvidence: len(items), TotalProofpacks: len(proofpacks), Plan: "free", FreeTierUsage: usageText(len(items), freeTierLimit), PersistenceMode: persistenceMode,
		HealthScore: summary.HealthScore, UnresolvedIssues: summary.Unresolved, StaleEvidence: summary.StaleEvidence, OwnerSummaries: summary.Owners, RecentActivity: summary.RecentActivity, Cadence: summary.Cadence}
	var latest time.Time
	for _, it := range items {
		if it.Status == "expiring" {
			vm.ExpiringSoon++
		}
		if it.Status == "expired" {
			vm.Expired++
		}
		if it.OwnerEmail == "" && it.OwnerName == "" {
			vm.MissingOwner++
		}
	}
	for _, p := range proofpacks {
		if createdAt, ok := p["created_at"].(time.Time); ok && createdAt.After(latest) {
			latest = createdAt
		}
	}
	if latest.IsZero() {
		vm.LatestProofpackTime = "not generated yet"
	} else {
		vm.LatestProofpackTime = latest.Format(time.RFC3339)
	}
	if summary.LastReviewedAt.IsZero() {
		vm.LastReviewedAt = "not reviewed yet"
	} else {
		vm.LastReviewedAt = summary.LastReviewedAt.Format(time.RFC3339)
	}
	vm.NextRecommendedReview = summary.NextRecommendedReview.Format(time.RFC3339)
	if degradedMode {
		vm.DegradedWarnings = append(vm.DegradedWarnings, "Running in memory mode: data is ephemeral and resets on restart.")
	}
	if persistenceMode == "file" {
		vm.DegradedWarnings = append(vm.DegradedWarnings, "File mode is durable for pilot use but is single-node storage.")
	}
	return vm
}

func usageText(total, limit int) string {
	if limit <= 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d/%d evidence records", total, limit)
}
