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
	_, _ = ev.Create(context.Background(), "t1", evidence.Item{Title: "act", Category: "IT", Status: "active", OwnerEmail: "o@x.com", ReminderDaysBefore: 30, UpdatedAt: now})
	svc := NewService(st, ev)
	sum, _ := svc.BuildSummary(context.Background(), "t1")
	if sum.HealthScore >= 100 || sum.Unresolved == 0 || len(sum.Owners) == 0 {
		t.Fatalf("unexpected summary: %+v", sum)
	}
	snap, _ := svc.GenerateReviewSnapshot(context.Background(), "t1")
	if snap.HealthScore != sum.HealthScore || snap.UnresolvedIssues == 0 {
		t.Fatalf("bad snapshot: %+v", snap)
	}
}
