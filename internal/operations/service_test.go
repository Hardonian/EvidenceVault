package operations

import (
	"context"
	"testing"
	"time"

	"evidencevault/internal/evidence"
	"evidencevault/internal/persistence"
)

func TestSummaryAndSnapshot(t *testing.T) {
	st := persistence.NewMemoryStore()
	ev := evidence.NewService(st, 100)
	now := time.Now().UTC()
	_, _ = ev.Create(context.Background(), "t1", evidence.Item{Title: "exp", Category: "IT", Status: "expired", ReminderDaysBefore: 30, UpdatedAt: now.AddDate(0, 0, -200)})
	svc := NewService(st, ev)
	sum, _ := svc.BuildSummary(context.Background(), "t1", nil)
	if sum.Unresolved == 0 {
		t.Fatal("expected unresolved")
	}
	_, _ = svc.GenerateReviewSnapshot(context.Background(), "t1")
	_, _ = ev.Create(context.Background(), "t1", evidence.Item{Title: "act", Category: "IT", Status: "active", OwnerEmail: "o@x.com", ReminderDaysBefore: 30, UpdatedAt: now})
	_, _ = svc.GenerateReviewSnapshot(context.Background(), "t1")
	sum2, _ := svc.BuildSummary(context.Background(), "t1", nil)
	if sum2.PreviousHealthScore == 0 && sum2.HealthDelta == 0 {
		t.Fatal("expected trend")
	}
}

func TestActivationAndMaturity(t *testing.T) {
	st := persistence.NewMemoryStore()
	ev := evidence.NewService(st, 100)
	svc := NewService(st, ev)
	ctx := context.Background()
	id, _ := ev.Create(ctx, "t2", evidence.Item{Title: "policy", Category: "Ops", Status: "active", ReminderDaysBefore: 7})
	svc.RecordEvent("t2", "evidence.created", "Evidence created", id)
	svc.RecordEvent("t2", "evidence.file.uploaded", "Evidence file uploaded", id)
	svc.RecordEvent("t2", "proofpack.generated", "Proofpack exported", "")
	_, _ = svc.GenerateReviewSnapshot(ctx, "t2")
	sum, _ := svc.BuildSummary(ctx, "t2", nil)
	if sum.ActivationCompletionPercent == 0 || sum.PilotMaturityStage == "exploring" {
		t.Fatal("expected activation progress")
	}
	if len(sum.ActivationChecklist) != 6 {
		t.Fatal("expected 6 milestones")
	}
}

func TestOperationalSnapshotAndReport(t *testing.T) {
	st := persistence.NewMemoryStore()
	ev := evidence.NewService(st, 100)
	svc := NewService(st, ev)
	ctx := context.Background()
	_, _ = ev.Create(ctx, "t3", evidence.Item{Title: "x", Category: "Ops", Status: "active", OwnerEmail: "a@b.com", ReminderDaysBefore: 7})
	snap, err := svc.GenerateOperationalSnapshot(ctx, "t3")
	if err != nil || snap.Date == "" {
		t.Fatal("expected snapshot")
	}
	rep, err := svc.GenerateReviewReport(ctx, "t3")
	if err != nil || rep.Markdown == "" || rep.HTML == "" {
		t.Fatal("expected review report exports")
	}
}

func TestNarrativesAndComparisons(t *testing.T) {
	st := persistence.NewMemoryStore()
	ev := evidence.NewService(st, 100)
	svc := NewService(st, ev)
	ctx := context.Background()
	_, _ = ev.Create(ctx, "t4", evidence.Item{Title: "old", Category: "Ops", Status: "expired", ReminderDaysBefore: 7, UpdatedAt: time.Now().UTC().AddDate(0, 0, -210)})
	_, _ = svc.GenerateReviewSnapshot(ctx, "t4")
	_, _ = ev.Create(ctx, "t4", evidence.Item{Title: "new", Category: "Ops", Status: "active", OwnerEmail: "o@x.com", ReminderDaysBefore: 7})
	_, _ = svc.GenerateReviewSnapshot(ctx, "t4")
	sum, _ := svc.BuildSummary(ctx, "t4", nil)
	if len(sum.Narratives) == 0 {
		t.Fatal("expected narratives")
	}
	cmp, err := svc.CompareReviews(ctx, "t4", 0, 1)
	if err != nil || cmp.Markdown() == "" || cmp.PlainText() == "" {
		t.Fatal("expected comparison export")
	}
}

func TestPilotRitualProgression(t *testing.T) {
	st := persistence.NewMemoryStore()
	ev := evidence.NewService(st, 100)
	svc := NewService(st, ev)
	ctx := context.Background()
	_, _ = ev.Create(ctx, "t5", evidence.Item{Title: "a", Category: "Ops", Status: "active", OwnerEmail: "o@x.com"})
	sum, _ := svc.BuildSummary(ctx, "t5", nil)
	if sum.PilotRitual.Week != 1 || sum.PilotRitual.NextAction != "generate first snapshot" {
		t.Fatal("expected week1 pre-review state")
	}
	for i := 0; i < 4; i++ {
		_, _ = svc.GenerateReviewSnapshot(ctx, "t5")
	}
	sum, _ = svc.BuildSummary(ctx, "t5", nil)
	if !sum.PilotRitual.Week4Ready || !sum.PilotRitual.ComparisonExportReady || sum.PilotRitual.Week != 4 {
		t.Fatal("expected week4 export readiness")
	}
}
