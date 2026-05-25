package operations

import (
	"context"
	"sort"
	"strings"
	"time"

	"evidencevault/internal/evidence"
	"evidencevault/internal/persistence"
)

const disclaimer = "Operational review supports evidence hygiene workflows and does not certify compliance or replace legal review."

type OwnerSummary struct {
	OwnerName, OwnerEmail string
	Total, Unresolved     int
}
type Activity struct {
	Type, Message, EntityID string
	CreatedAt               time.Time
}

type Summary struct {
	HealthScore, Expired, ExpiringSoon, MissingOwner, StaleEvidence, Unresolved int
	PreviousHealthScore, HealthDelta, UnresolvedDelta                           int
	ReviewCompletionStreak, DaysSinceLastReview                                 int
	LastReviewedAt, NextRecommendedReview, LastActivityAt                       time.Time
	Owners                                                                      []OwnerSummary
	RecentActivity                                                              []Activity
	ProofpackHistory                                                            []persistence.ProofpackMeta
	Cadence                                                                     []string
	ActivationCompletionPercent                                                 int
	ActivationChecklist                                                         []MilestoneState
	PilotMaturityStage                                                          string
	DaysSinceLastUpload, DaysSinceLastProofpack                                 int
	Friction                                                                    []string
	UpgradeSignals                                                              []string
}
type MilestoneState struct {
	Key, Label string
	ReachedAt  *time.Time
}

type Service struct {
	store    persistence.Store
	evidence *evidence.Service
}

func NewService(store persistence.Store, ev *evidence.Service) *Service {
	return &Service{store: store, evidence: ev}
}

func (s *Service) BuildSummary(ctx context.Context, tenantID string, items []evidence.Item) (Summary, error) {
	if items == nil {
		items, _ = s.evidence.List(ctx, tenantID)
	}
	now := time.Now().UTC()
	sum := evaluate(items, now)
	sum.NextRecommendedReview = now.AddDate(0, 0, 7)
	sum.Cadence = []string{"Weekly operational review", "Monthly proofpack export", "Quarterly evidence audit"}
	_ = s.store.WithLock(func(st *persistence.State) error {
		m := st.Activation[tenantID]
		sum.ActivationChecklist = []MilestoneState{
			{Key: "first_evidence_created", Label: "First evidence created", ReachedAt: m.FirstEvidenceCreatedAt},
			{Key: "first_file_uploaded", Label: "First file uploaded", ReachedAt: m.FirstFileUploadedAt},
			{Key: "first_reminder_run", Label: "First reminder run", ReachedAt: m.FirstReminderRunAt},
			{Key: "first_proofpack_generated", Label: "First proofpack generated", ReachedAt: m.FirstProofpackGeneratedAt},
			{Key: "first_operational_review", Label: "First operational review", ReachedAt: m.FirstOperationalReviewAt},
			{Key: "second_weekly_review", Label: "Second weekly review", ReachedAt: m.SecondWeeklyReviewAt},
		}
		if arr := st.ReviewSnapshots[tenantID]; len(arr) > 0 {
			sum.LastReviewedAt = arr[0].LastReviewedAt
			sum.NextRecommendedReview = arr[0].NextRecommendedReview
		}
		sum.ProofpackHistory = append([]persistence.ProofpackMeta{}, st.Proofpacks[tenantID]...)
		for i, ev := range st.OperationalEvents[tenantID] {
			if i == 0 {
				sum.LastActivityAt = ev.CreatedAt
			}
			if i >= 8 {
				break
			}
			sum.RecentActivity = append(sum.RecentActivity, Activity{Type: ev.Type, Message: ev.Message, EntityID: ev.EntityID, CreatedAt: ev.CreatedAt})
		}
		if arr := st.ReviewSnapshots[tenantID]; len(arr) > 1 {
			sum.PreviousHealthScore = arr[1].HealthScore
			sum.HealthDelta = sum.HealthScore - arr[1].HealthScore
			sum.UnresolvedDelta = sum.Unresolved - arr[1].UnresolvedIssues
		}
		if arr := st.ReviewSnapshots[tenantID]; len(arr) > 0 {
			sum.ReviewCompletionStreak = reviewStreak(arr)
		}
		return nil
	})
	done := 0
	for _, c := range sum.ActivationChecklist {
		if c.ReachedAt != nil {
			done++
		}
	}
	sum.ActivationCompletionPercent = done * 100 / len(sum.ActivationChecklist)
	sum.PilotMaturityStage = deriveStage(sum, len(items))
	sum.Friction = frictionIndicators(sum, now)
	sum.UpgradeSignals = upgradeSignals(sum, len(items))
	if !sum.LastReviewedAt.IsZero() {
		sum.DaysSinceLastReview = int(now.Sub(sum.LastReviewedAt).Hours() / 24)
	}
	return sum, nil
}

func deriveStage(sum Summary, totalEvidence int) string {
	has := func(key string) bool {
		for _, m := range sum.ActivationChecklist {
			if m.Key == key {
				return m.ReachedAt != nil
			}
		}
		return false
	}
	if sum.HealthScore >= 75 && totalEvidence >= 12 && len(sum.Owners) > 1 && len(sum.ProofpackHistory) >= 3 && sum.ReviewCompletionStreak >= 3 {
		return "expansion-ready"
	}
	if has("second_weekly_review") {
		return "recurring"
	}
	if has("first_proofpack_generated") && has("first_operational_review") {
		return "operational"
	}
	if has("first_evidence_created") && has("first_file_uploaded") {
		return "activated"
	}
	return "exploring"
}

func frictionIndicators(sum Summary, now time.Time) []string {
	var out []string
	if sum.UnresolvedDelta > 0 {
		out = append(out, "Unresolved issues are trending upward")
	}
	if sum.ExpiringSoon+sum.Expired > 0 {
		out = append(out, "Overdue or expiring evidence needs action")
	}
	if sum.MissingOwner > 0 {
		out = append(out, "Ownerless evidence is creating accountability gaps")
	}
	if !sum.LastActivityAt.IsZero() && int(now.Sub(sum.LastActivityAt).Hours()/24) > 10 {
		out = append(out, "Operational drift warning: activity gap exceeds 10 days")
	}
	if sum.ReviewCompletionStreak < 2 && sum.DaysSinceLastReview > 9 {
		out = append(out, "Review streak break detected")
	}
	return out
}

func upgradeSignals(sum Summary, totalEvidence int) []string {
	var out []string
	if totalEvidence >= 20 {
		out = append(out, "You may outgrow file mode as evidence volume grows")
	}
	if len(sum.Owners) >= 2 {
		out = append(out, "Multi-user workflow emerging")
	}
	if sum.ReviewCompletionStreak >= 2 {
		out = append(out, "Recurring reviews established")
	}
	if len(sum.ProofpackHistory) >= 4 {
		out = append(out, "History depth increasing")
	}
	return out
}

func reviewStreak(arr []persistence.ReviewSnapshot) int {
	if len(arr) == 0 {
		return 0
	}
	streak := 1
	for i := 1; i < len(arr); i++ {
		d := arr[i-1].LastReviewedAt.Sub(arr[i].LastReviewedAt)
		if d.Hours() <= 24*9 {
			streak++
			continue
		}
		break
	}
	return streak
}

func evaluateItem(it evidence.Item, now time.Time, s *Summary, o *OwnerSummary) {
	o.Total++
	if it.Status == "expired" {
		s.Expired++
		s.Unresolved++
		o.Unresolved++
		s.HealthScore -= 25
	}
	if it.Status == "expiring" {
		s.ExpiringSoon++
		s.Unresolved++
		o.Unresolved++
		s.HealthScore -= 10
	}
	if strings.TrimSpace(it.OwnerEmail) == "" && strings.TrimSpace(it.OwnerName) == "" {
		s.MissingOwner++
		s.Unresolved++
		s.HealthScore -= 15
	}
	if it.Status == "active" {
		s.HealthScore += 1
	}
	if it.UpdatedAt.Before(now.AddDate(0, 0, -180)) {
		s.StaleEvidence++
		s.Unresolved++
		o.Unresolved++
		s.HealthScore -= 8
	}
}

func evaluate(items []evidence.Item, now time.Time) Summary {
	s := Summary{HealthScore: 100}
	owners := map[string]*OwnerSummary{}
	for _, it := range items {
		key := strings.TrimSpace(it.OwnerEmail) + "|" + strings.TrimSpace(it.OwnerName)
		if _, ok := owners[key]; !ok {
			owners[key] = &OwnerSummary{OwnerName: it.OwnerName, OwnerEmail: it.OwnerEmail}
		}
		evaluateItem(it, now, &s, owners[key])
	}
	if s.HealthScore < 0 {
		s.HealthScore = 0
	}
	if s.HealthScore > 100 {
		s.HealthScore = 100
	}
	for _, o := range owners {
		s.Owners = append(s.Owners, *o)
	}
	sort.Slice(s.Owners, func(i, j int) bool { return s.Owners[i].Unresolved > s.Owners[j].Unresolved })
	return s
}

func (s *Service) GenerateReviewSnapshot(ctx context.Context, tenantID string) (persistence.ReviewSnapshot, error) {
	sum, _ := s.BuildSummary(ctx, tenantID, nil)
	now := time.Now().UTC()
	snap := persistence.ReviewSnapshot{TenantID: tenantID, GeneratedAt: now, LastReviewedAt: now, NextRecommendedReview: now.AddDate(0, 0, 7), HealthScore: sum.HealthScore, UnresolvedIssues: sum.Unresolved, ExpiredEvidence: sum.Expired, ExpiringEvidence: sum.ExpiringSoon, StaleEvidence: sum.StaleEvidence, MissingOwners: sum.MissingOwner, Disclaimer: disclaimer}
	_ = s.store.WithLock(func(st *persistence.State) error {
		st.ReviewSnapshots[tenantID] = append([]persistence.ReviewSnapshot{snap}, st.ReviewSnapshots[tenantID]...)
		return nil
	})
	s.RecordEvent(tenantID, "review.snapshot.generated", "Operational review snapshot generated", "")
	return snap, nil
}

func (s *Service) RecordEvent(tenantID, typ, message, entityID string) {
	now := time.Now().UTC()
	_ = s.store.WithLock(func(st *persistence.State) error {
		st.OperationalEvents[tenantID] = append([]persistence.OperationalEvent{{TenantID: tenantID, Type: typ, Message: message, EntityID: entityID, CreatedAt: now}}, st.OperationalEvents[tenantID]...)
		m := st.Activation[tenantID]
		switch typ {
		case "evidence.created":
			if m.FirstEvidenceCreatedAt == nil {
				m.FirstEvidenceCreatedAt = &now
				st.OperationalEvents[tenantID] = append([]persistence.OperationalEvent{{TenantID: tenantID, Type: "milestone.reached", Message: "Milestone reached: first evidence created", EntityID: entityID, CreatedAt: now}}, st.OperationalEvents[tenantID]...)
			}
		case "evidence.file.uploaded":
			if m.FirstFileUploadedAt == nil {
				m.FirstFileUploadedAt = &now
			}
		case "reminders.run":
			if m.FirstReminderRunAt == nil {
				m.FirstReminderRunAt = &now
			}
		case "proofpack.generated":
			if m.FirstProofpackGeneratedAt == nil {
				m.FirstProofpackGeneratedAt = &now
			}
		case "review.snapshot.generated":
			if m.FirstOperationalReviewAt == nil {
				m.FirstOperationalReviewAt = &now
			} else if m.SecondWeeklyReviewAt == nil && now.Sub(*m.FirstOperationalReviewAt).Hours() >= 24*6 {
				m.SecondWeeklyReviewAt = &now
			}
		}
		st.Activation[tenantID] = m
		return nil
	})
}
