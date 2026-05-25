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
