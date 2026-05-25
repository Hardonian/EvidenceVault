package httpserver

import (
	"fmt"
	"sort"
	"time"

	"evidencevault/internal/evidence"
	"evidencevault/internal/operations"
)

type PriorityItem struct {
	Label string
	Count int
}
type TemplateCard struct{ Key, Name, Description string }

type DashboardViewModel struct {
	TotalEvidence, ExpiringSoon, Expired, MissingOwner, TotalProofpacks int
	LatestProofpackTime, Plan, FreeTierUsage, PersistenceMode           string
	DegradedWarnings                                                    []string
	HealthScore, UnresolvedIssues, StaleEvidence                        int
	PreviousHealthScore, HealthDelta, UnresolvedDelta                   int
	ReviewCompletionStreak, DaysSinceLastReview                         int
	LastReviewedAt, NextRecommendedReview, LastActivityAt               string
	OwnerSummaries                                                      []operations.OwnerSummary
	RecentActivity                                                      []operations.Activity
	Cadence                                                             []string
	Onboarding                                                          bool
	StarterTemplates                                                    []TemplateCard
	PriorityQueue                                                       []PriorityItem
}

func buildDashboardViewModel(items []evidence.Item, proofpacks []map[string]any, freeTierLimit int, persistenceMode string, degradedMode bool, summary operations.Summary) DashboardViewModel {
	vm := DashboardViewModel{TotalEvidence: len(items), TotalProofpacks: len(proofpacks), Plan: "free", FreeTierUsage: usageText(len(items), freeTierLimit), PersistenceMode: persistenceMode,
		HealthScore: summary.HealthScore, UnresolvedIssues: summary.Unresolved, StaleEvidence: summary.StaleEvidence, OwnerSummaries: summary.Owners, RecentActivity: summary.RecentActivity, Cadence: summary.Cadence,
		PreviousHealthScore: summary.PreviousHealthScore, HealthDelta: summary.HealthDelta, UnresolvedDelta: summary.UnresolvedDelta, ReviewCompletionStreak: summary.ReviewCompletionStreak, DaysSinceLastReview: summary.DaysSinceLastReview}
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
	if summary.LastActivityAt.IsZero() {
		vm.LastActivityAt = "no activity yet"
	} else {
		vm.LastActivityAt = summary.LastActivityAt.Format(time.RFC3339)
	}
	if degradedMode {
		vm.DegradedWarnings = append(vm.DegradedWarnings, "Running in memory mode: data is ephemeral and resets on restart.")
	}
	if persistenceMode == "file" {
		vm.DegradedWarnings = append(vm.DegradedWarnings, "File mode is durable for pilot use but is single-node storage.")
	}
	vm.Onboarding = len(items) == 0
	for _, t := range evidence.StarterTemplates(time.Now().UTC()) {
		vm.StarterTemplates = append(vm.StarterTemplates, TemplateCard{Key: t.Key, Name: t.Name, Description: t.Description})
	}
	vm.PriorityQueue = []PriorityItem{{"Expired evidence", vm.Expired}, {"Missing owners", vm.MissingOwner}, {"Expiring soon", vm.ExpiringSoon}, {"Stale evidence", vm.StaleEvidence}, {"Recent uploads/proofpacks", len(vm.RecentActivity) + vm.TotalProofpacks}}
	sort.SliceStable(vm.PriorityQueue, func(i, j int) bool { return i < j })
	return vm
}

func usageText(total, limit int) string {
	if limit <= 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d/%d evidence records", total, limit)
}
