package httpserver

import (
	"fmt"
	"time"

	"evidencevault/internal/evidence"
	"evidencevault/internal/evidencegraph"
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
	ActivationCompletionPercent                                         int
	ActivationChecklist                                                 []operations.MilestoneState
	PilotMaturityStage                                                  string
	PilotRitual                                                         operations.PilotRitualState
	Friction, UpgradeSignals, FounderSignals                            []string
	Onboarding                                                          bool
	StarterTemplates                                                    []TemplateCard
	PriorityQueue                                                       []PriorityItem
	// Evidence Graph fields
	GraphReadinessScore int
	GraphHealth         string
	GraphDegradedReasons []string
	GraphNextActions    []evidencegraph.NextAction
	GraphNodeCount      int
	GraphEdgeCount      int
	GraphAvailable      bool
}

func calculateItemCounts(items []evidence.Item) (int, int, int) {
	e1, e2, e3 := 0, 0, 0
	for _, it := range items {
		if it.Status == "expiring" {
			e1++
		}
		if it.Status == "expired" {
			e2++
		}
		if it.OwnerEmail == "" && it.OwnerName == "" {
			e3++
		}
	}
	return e1, e2, e3
}
func buildDashboardViewModel(items []evidence.Item, proofpacks []map[string]any, freeTierLimit int, persistenceMode string, degradedMode bool, summary operations.Summary, graphData *evidencegraph.Graph) DashboardViewModel {
	expiringSoon, expired, missingOwner := calculateItemCounts(items)
	vm := DashboardViewModel{Onboarding: len(items) == 0, StarterTemplates: buildStarterTemplates(), PriorityQueue: buildPriorityQueue(expired, missingOwner, expiringSoon, summary.StaleEvidence, len(summary.RecentActivity), len(proofpacks)), TotalEvidence: len(items), TotalProofpacks: len(proofpacks), Plan: "free", FreeTierUsage: usageText(len(items), freeTierLimit), PersistenceMode: persistenceMode, HealthScore: summary.HealthScore, UnresolvedIssues: summary.Unresolved, StaleEvidence: summary.StaleEvidence, OwnerSummaries: summary.Owners, RecentActivity: summary.RecentActivity, Cadence: summary.Cadence, PreviousHealthScore: summary.PreviousHealthScore, HealthDelta: summary.HealthDelta, UnresolvedDelta: summary.UnresolvedDelta, ReviewCompletionStreak: summary.ReviewCompletionStreak, DaysSinceLastReview: summary.DaysSinceLastReview, ExpiringSoon: expiringSoon, Expired: expired, MissingOwner: missingOwner, LatestProofpackTime: findLatestProofpackTime(proofpacks), LastReviewedAt: formatTimeOrDefault(summary.LastReviewedAt, "not reviewed yet"), NextRecommendedReview: summary.NextRecommendedReview.Format(time.RFC3339), LastActivityAt: formatTimeOrDefault(summary.LastActivityAt, "no activity yet"), DegradedWarnings: buildDegradedWarnings(degradedMode, persistenceMode), ActivationCompletionPercent: summary.ActivationCompletionPercent, ActivationChecklist: summary.ActivationChecklist, PilotMaturityStage: summary.PilotMaturityStage, PilotRitual: summary.PilotRitual, Friction: summary.Friction, UpgradeSignals: summary.UpgradeSignals, FounderSignals: summary.FounderSignals}
	if graphData != nil {
		vm.GraphAvailable = true
		vm.GraphReadinessScore = graphData.Summary.GraphHealthScore
		vm.GraphHealth = graphData.Summary.PilotReadinessState
		vm.GraphDegradedReasons = graphData.DegradedReasons
		vm.GraphNextActions = graphData.NextActions
		vm.GraphNodeCount = len(graphData.Nodes)
		vm.GraphEdgeCount = len(graphData.Edges)
	}
	return vm
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
	var w []string
	if degradedMode {
		w = append(w, "Running in memory mode: data is ephemeral and resets on restart.")
	}
	if persistenceMode == "file" {
		w = append(w, "File mode is durable for pilot use but is single-node storage.")
	}
	return w
}
func usageText(total, limit int) string {
	if limit <= 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d/%d evidence records", total, limit)
}

func buildStarterTemplates() []TemplateCard {
	var templates []TemplateCard
	for _, t := range evidence.StarterTemplates(time.Now().UTC()) {
		templates = append(templates, TemplateCard{Key: t.Key, Name: t.Name, Description: t.Description})
	}
	return templates
}
func buildPriorityQueue(expired, missingOwner, expiringSoon, staleEvidence, recentActivityCount, totalProofpacks int) []PriorityItem {
	return []PriorityItem{{"Expired evidence", expired}, {"Missing owners", missingOwner}, {"Expiring soon", expiringSoon}, {"Stale evidence", staleEvidence}, {"Recent uploads/proofpacks", recentActivityCount + totalProofpacks}}
}
