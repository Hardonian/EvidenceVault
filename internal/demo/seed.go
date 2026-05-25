package demo

import (
	"context"
	"errors"
	"time"

	"evidencevault/internal/evidence"
	"evidencevault/internal/operations"
	"evidencevault/internal/proofpack"
)

var ErrDemoSeedBlockedInProduction = errors.New("demo seed is blocked in production")

func Seed(ctx context.Context, appEnv string, enabled bool, ev *evidence.Service, ops *operations.Service, pp *proofpack.Service, tenantID string) error {
	if !enabled {
		return nil
	}
	if appEnv == "production" {
		return ErrDemoSeedBlockedInProduction
	}
	items, _ := ev.List(ctx, tenantID)
	if len(items) > 0 {
		return nil
	}
	now := time.Now().UTC()
	soon := now.AddDate(0, 0, 14)
	expired := now.AddDate(0, 0, -7)
	stale := now.AddDate(0, 0, -220)
	_, _ = ev.Create(ctx, tenantID, evidence.Item{Title: "General Liability Insurance", Category: "Compliance", Status: "expiring", OwnerName: "Finance Owner", OwnerEmail: "finance@example.com", ReminderDaysBefore: 30, ExpiryDate: &soon, Notes: "Renewal in two weeks"})
	_, _ = ev.Create(ctx, tenantID, evidence.Item{Title: "Privacy Policy", Category: "Legal", Status: "active", OwnerName: "Legal Owner", OwnerEmail: "legal@example.com", ReminderDaysBefore: 30, Notes: "Published policy evidence"})
	_, _ = ev.Create(ctx, tenantID, evidence.Item{Title: "Supplier Security Certificate", Category: "Compliance", Status: "expired", ReminderDaysBefore: 30, ExpiryDate: &expired, Notes: "Expired and unassigned"})
	_, _ = ev.Create(ctx, tenantID, evidence.Item{Title: "Accessibility Statement", Category: "Compliance", Status: "active", OwnerName: "Product Owner", OwnerEmail: "product@example.com", ReminderDaysBefore: 30, UpdatedAt: stale, Notes: "Stale evidence sample"})
	if ops != nil {
		ops.RecordEvent(tenantID, "reminder.sent", "Renewal reminder sent", "")
		ops.RecordEvent(tenantID, "owner.missing", "Owner assignment missing", "")
		_, _ = ops.GenerateReviewSnapshot(ctx, tenantID)
	}
	if pp != nil {
		_, _ = pp.Export(ctx, tenantID, "demo")
	}
	return nil
}
