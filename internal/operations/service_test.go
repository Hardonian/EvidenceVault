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
	sum, _ := svc.BuildSummary(ctx, "t2")
	if sum.ActivationCompletionPercent == 0 || sum.PilotMaturityStage == "exploring" {
		t.Fatal("expected activation progress")
	}
	if len(sum.ActivationChecklist) != 6 {
		t.Fatal("expected 6 milestones")
	}
}
