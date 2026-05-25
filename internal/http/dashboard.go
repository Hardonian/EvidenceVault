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

func calculateItemCounts(items []evidence.Item) (expiringSoon, expired, missingOwner int) {
	for _, it := range items {
		if it.Status == "expiring" {
			expiringSoon++
		}
		if it.Status == "expired" {
			expired++
		}
		if it.OwnerEmail == "" && it.OwnerName == "" {
			missingOwner++
		}
	}
	return
}

func findLatestProofpackTime(proofpacks []map[string]any) string {
	var latest time.Time
	for _, p := range proofpacks {
		if createdAt, ok := p["created_at"].(time.Time); ok && createdAt.After(latest) {
			latest = createdAt
		}
	}
	return formatTimeOrDefault(latest, "not generated yet")
}

func formatTimeOrDefault(t time.Time, defaultStr string) string {
	if t.IsZero() {
		return defaultStr
	}
	return t.Format(time.RFC3339)
}

func buildDegradedWarnings(degradedMode bool, persistenceMode string) []string {
	var warnings []string
	if degradedMode {
		warnings = append(warnings, "Running in memory mode: data is ephemeral and resets on restart.")
	}
	if persistenceMode == "file" {
		warnings = append(warnings, "File mode is durable for pilot use but is single-node storage.")
	}
	return warnings
}

func buildStarterTemplates() []TemplateCard {
	var templates []TemplateCard
	for _, t := range evidence.StarterTemplates(time.Now().UTC()) {
		templates = append(templates, TemplateCard{Key: t.Key, Name: t.Name, Description: t.Description})
	}
	return templates
}

func buildPriorityQueue(expired, missingOwner, expiringSoon, staleEvidence, recentActivityCount, totalProofpacks int) []PriorityItem {
	queue := []PriorityItem{
		{"Expired evidence", expired},
		{"Missing owners", missingOwner},
		{"Expiring soon", expiringSoon},
		{"Stale evidence", staleEvidence},
		{"Recent uploads/proofpacks", recentActivityCount + totalProofpacks},
	}

	sort.SliceStable(queue, func(i, j int) bool { return i < j })
	return queue
}

func buildDashboardViewModel(items []evidence.Item, proofpacks []map[string]any, freeTierLimit int, persistenceMode string, degradedMode bool, summary operations.Summary) DashboardViewModel {
	expiringSoon, expired, missingOwner := calculateItemCounts(items)

	vm := DashboardViewModel{
		TotalEvidence:          len(items),
		TotalProofpacks:        len(proofpacks),
		Plan:                   "free",
		FreeTierUsage:          usageText(len(items), freeTierLimit),
		PersistenceMode:        persistenceMode,
		HealthScore:            summary.HealthScore,
		UnresolvedIssues:       summary.Unresolved,
		StaleEvidence:          summary.StaleEvidence,
		OwnerSummaries:         summary.Owners,
		RecentActivity:         summary.RecentActivity,
		Cadence:                summary.Cadence,
		PreviousHealthScore:    summary.PreviousHealthScore,
		HealthDelta:            summary.HealthDelta,
		UnresolvedDelta:        summary.UnresolvedDelta,
		ReviewCompletionStreak: summary.ReviewCompletionStreak,
		DaysSinceLastReview:    summary.DaysSinceLastReview,
		ExpiringSoon:           expiringSoon,
		Expired:                expired,
		MissingOwner:           missingOwner,
		LatestProofpackTime:    findLatestProofpackTime(proofpacks),
		LastReviewedAt:         formatTimeOrDefault(summary.LastReviewedAt, "not reviewed yet"),
		NextRecommendedReview:  summary.NextRecommendedReview.Format(time.RFC3339),
		LastActivityAt:         formatTimeOrDefault(summary.LastActivityAt, "no activity yet"),
		DegradedWarnings:       buildDegradedWarnings(degradedMode, persistenceMode),
		Onboarding:             len(items) == 0,
		StarterTemplates:       buildStarterTemplates(),
		PriorityQueue:          buildPriorityQueue(expired, missingOwner, expiringSoon, summary.StaleEvidence, len(summary.RecentActivity), len(proofpacks)),
	}

	return vm
}

func usageText(total, limit int) string {
	if limit <= 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d/%d evidence records", total, limit)
}
