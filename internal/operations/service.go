package operations

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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

type TrendWindow struct {
	WindowDays  int
	StartHealth int
	EndHealth   int
	HealthDelta int
	Status      string
}

type UsageSignals struct {
	Cadence            string
	ProofpackFrequency string
	RecoveryTrend      string
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
	Trend7, Trend30                                                             TrendWindow
	Usage                                                                       UsageSignals
	FounderSignals                                                              []string
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
	_ = s.store.Read(func(st *persistence.State) error {
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
	sum.Trend7, sum.Trend30 = s.buildTrends(tenantID, sum.HealthScore)
	sum.Usage = usageSignals(sum, len(sum.ProofpackHistory))
	sum.FounderSignals = founderSignals(sum)
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

func trendStatus(delta int) string {
	if delta > 2 {
		return "improving"
	}
	if delta < -2 {
		return "degrading"
	}
	return "stable"
}

func (s *Service) buildTrends(tenantID string, currentHealth int) (TrendWindow, TrendWindow) {
	var snaps []persistence.OperationalSnapshot
	_ = s.store.WithLock(func(st *persistence.State) error {
		snaps = append(snaps, st.OperationalSnapshots[tenantID]...)
		return nil
	})
	build := func(days int) TrendWindow {
		w := TrendWindow{WindowDays: days, EndHealth: currentHealth, StartHealth: currentHealth}
		cut := time.Now().UTC().AddDate(0, 0, -days)
		for i := len(snaps) - 1; i >= 0; i-- {
			if snaps[i].CreatedAt.After(cut) {
				w.StartHealth = snaps[i].HealthScore
			}
		}
		w.HealthDelta = w.EndHealth - w.StartHealth
		w.Status = trendStatus(w.HealthDelta)
		return w
	}
	return build(7), build(30)
}

func usageSignals(sum Summary, proofpacks int) UsageSignals {
	out := UsageSignals{Cadence: "stalled operational cadence", ProofpackFrequency: "infrequent", RecoveryTrend: "slowing"}
	if sum.ReviewCompletionStreak >= 2 {
		out.Cadence = "improving discipline"
	}
	if proofpacks >= 3 {
		out.ProofpackFrequency = "consistent"
	}
	if sum.UnresolvedDelta <= 0 {
		out.RecoveryTrend = "recovery improving"
	}
	return out
}

func founderSignals(sum Summary) []string {
	var out []string
	if sum.Trend30.HealthDelta < 0 {
		out = append(out, "pilot retention risk: health trend declined over 30 days")
	}
	if sum.DaysSinceLastReview > 9 {
		out = append(out, "cadence degradation warning")
	}
	if sum.UnresolvedDelta > 0 {
		out = append(out, "unresolved recovery trend worsening")
	}
	return out
}

func (s *Service) GenerateOperationalSnapshot(ctx context.Context, tenantID string) (persistence.OperationalSnapshot, error) {
	sum, _ := s.BuildSummary(ctx, tenantID, nil)
	now := time.Now().UTC()
	snap := persistence.OperationalSnapshot{TenantID: tenantID, Date: now.Format("2006-01-02"), CreatedAt: now, UnresolvedCount: sum.Unresolved, ExpiredEvidenceCount: sum.Expired, OwnerlessEvidenceCount: sum.MissingOwner, StaleEvidenceCount: sum.StaleEvidence, HealthScore: sum.HealthScore, ProofpackCount: len(sum.ProofpackHistory), ReviewStreak: sum.ReviewCompletionStreak, ActivationPercent: sum.ActivationCompletionPercent, MaturityStage: sum.PilotMaturityStage, TotalEvidenceCount: sum.Unresolved + (sum.HealthScore / 100), OwnersCount: len(sum.Owners)}
	_ = s.store.WithLock(func(st *persistence.State) error {
		arr := st.OperationalSnapshots[tenantID]
		if len(arr) > 0 && arr[0].Date == snap.Date {
			st.OperationalSnapshots[tenantID][0] = snap
			return nil
		}
		arr = append([]persistence.OperationalSnapshot{snap}, arr...)
		if len(arr) > 90 {
			arr = arr[:90]
		}
		st.OperationalSnapshots[tenantID] = arr
		return nil
	})
	s.RecordEvent(tenantID, "trend.snapshot.generated", "Operational trend snapshot generated", "")
	return snap, nil
}

func (s *Service) GenerateReviewReport(ctx context.Context, tenantID string) (persistence.ReviewReport, error) {
	sum, _ := s.BuildSummary(ctx, tenantID, nil)
	now := time.Now().UTC()
	md := fmt.Sprintf("# Operational Review\n\n- Health score: %d\n- Unresolved: %d (delta %d)\n- Stale evidence: %d\n- Missing owners: %d\n- Proofpack activity: %d\n- Activation: %d%%\n- Maturity stage: %s\n", sum.HealthScore, sum.Unresolved, sum.UnresolvedDelta, sum.StaleEvidence, sum.MissingOwner, len(sum.ProofpackHistory), sum.ActivationCompletionPercent, sum.PilotMaturityStage)
	rep := persistence.ReviewReport{TenantID: tenantID, ID: now.Format("20060102T150405"), GeneratedAt: now, Summary: "Deterministic operational review report", Markdown: md, PlainText: strings.ReplaceAll(md, "- ", "* "), HTML: "<pre>" + md + "</pre>"}
	_ = s.store.WithLock(func(st *persistence.State) error {
		st.ReviewReports[tenantID] = append([]persistence.ReviewReport{rep}, st.ReviewReports[tenantID]...)
		return nil
	})
	s.RecordEvent(tenantID, "review.report.generated", "Operational review report generated", rep.ID)
	return rep, nil
}

func itoa(n int) string { return strconv.Itoa(n) }

func (s *Service) GenerateReviewSnapshot(ctx context.Context, tenantID string) (persistence.ReviewSnapshot, error) {
	sum, _ := s.BuildSummary(ctx, tenantID, nil)
	now := time.Now().UTC()
	snap := persistence.ReviewSnapshot{TenantID: tenantID, GeneratedAt: now, LastReviewedAt: now, NextRecommendedReview: now.AddDate(0, 0, 7), HealthScore: sum.HealthScore, UnresolvedIssues: sum.Unresolved, ExpiredEvidence: sum.Expired, ExpiringEvidence: sum.ExpiringSoon, StaleEvidence: sum.StaleEvidence, MissingOwners: sum.MissingOwner, Disclaimer: disclaimer}
	_ = s.store.Write(func(st *persistence.State) error {
		st.ReviewSnapshots[tenantID] = append([]persistence.ReviewSnapshot{snap}, st.ReviewSnapshots[tenantID]...)
		return nil
	})
	s.RecordEvent(tenantID, "review.snapshot.generated", "Operational review snapshot generated", "")
	return snap, nil
}

func (s *Service) RecordEvent(tenantID, typ, message, entityID string) {
	_ = s.store.Write(func(st *persistence.State) error {
		st.OperationalEvents[tenantID] = append([]persistence.OperationalEvent{{TenantID: tenantID, Type: typ, Message: message, EntityID: entityID, CreatedAt: time.Now().UTC()}}, st.OperationalEvents[tenantID]...)
		return nil
	})
}
