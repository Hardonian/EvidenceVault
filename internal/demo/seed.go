package demo

import (
	"context"
	"errors"
	"time"

	"evidencevault/internal/evidence"
)

var ErrDemoSeedBlockedInProduction = errors.New("demo seed is blocked in production")

func Seed(ctx context.Context, appEnv string, enabled bool, ev *evidence.Service, tenantID string) error {
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
	_, err := ev.Create(ctx, tenantID, evidence.Item{Title: "General Liability Insurance", Category: "Compliance", Status: "active", OwnerName: "Finance Owner", OwnerEmail: "finance@example.com", ReminderDaysBefore: 30, ExpiryDate: &soon, Notes: "Expiring soon demo record"})
	if err != nil {
		return err
	}
	_, err = ev.Create(ctx, tenantID, evidence.Item{Title: "Privacy Policy", Category: "Legal", Status: "active", OwnerName: "Legal Owner", OwnerEmail: "legal@example.com", ReminderDaysBefore: 30, Notes: "Active policy evidence"})
	if err != nil {
		return err
	}
	_, err = ev.Create(ctx, tenantID, evidence.Item{Title: "Supplier Security Certificate", Category: "Compliance", Status: "active", ReminderDaysBefore: 30, ExpiryDate: &expired, Notes: "Expired demo record"})
	return err
}
