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
	longExpiry := now.AddDate(1, 0, 0)

	// Create diverse evidence across categories and risk states.
	insuranceID, _ := ev.Create(ctx, tenantID, evidence.Item{Title: "General Liability Insurance", Category: "Compliance", Status: "expiring", OwnerName: "Finance Owner", OwnerEmail: "finance@example.com", ReminderDaysBefore: 30, ExpiryDate: &soon, Notes: "Renewal in two weeks"})
	policyID, _ := ev.Create(ctx, tenantID, evidence.Item{Title: "Privacy Policy", Category: "Legal", Status: "active", OwnerName: "Legal Owner", OwnerEmail: "legal@example.com", ReminderDaysBefore: 30, Notes: "Published policy evidence"})
	certID, _ := ev.Create(ctx, tenantID, evidence.Item{Title: "Supplier Security Certificate", Category: "Compliance", Status: "expired", ReminderDaysBefore: 30, ExpiryDate: &expired, Notes: "Expired and unassigned — ownerless risk"})
	staleID, _ := ev.Create(ctx, tenantID, evidence.Item{Title: "Accessibility Statement", Category: "Compliance", Status: "active", OwnerName: "Product Owner", OwnerEmail: "product@example.com", ReminderDaysBefore: 30, UpdatedAt: stale, Notes: "Stale evidence — not updated in 220 days"})

	// Additional evidence for graph diversity.
	_, _ = ev.Create(ctx, tenantID, evidence.Item{Title: "SOC2 Type II Report", Category: "Security", Status: "active", OwnerName: "Security Owner", OwnerEmail: "security@example.com", ReminderDaysBefore: 30, ExpiryDate: &longExpiry, Notes: "Annual SOC2 audit report"})
	_, _ = ev.Create(ctx, tenantID, evidence.Item{Title: "Employee Handbook Acknowledgement", Category: "HR", Status: "active", OwnerName: "People Ops", OwnerEmail: "people@example.com", ReminderDaysBefore: 30, ExpiryDate: &soon, Notes: "Annual handbook acknowledgment"})
	_, _ = ev.Create(ctx, tenantID, evidence.Item{Title: "Vendor Agreement - Cloud Provider", Category: "Legal", Status: "active", OwnerName: "Legal Owner", OwnerEmail: "legal@example.com", ReminderDaysBefore: 45, ExpiryDate: &longExpiry, Notes: "Primary cloud vendor contract"})
	ownerlessID, _ := ev.Create(ctx, tenantID, evidence.Item{Title: "Data Retention Policy", Category: "IT", Status: "active", ReminderDaysBefore: 30, Notes: "Ownerless — needs accountability assignment"})

	if ops != nil {
		// Week 1: Initial state with problems.
		ops.RecordEvent(tenantID, "reminder.sent", "Renewal reminder sent", "")
		ops.RecordEvent(tenantID, "owner.missing", "Owner assignment missing", "")
		_, _ = ops.GenerateReviewSnapshot(ctx, tenantID)

		// Week 2: Fix expired cert — assign owner and renew.
		_ = ev.Update(ctx, tenantID, certID, evidence.Item{Title: "Supplier Security Certificate", Category: "Compliance", Status: "active", OwnerName: "Security Owner", OwnerEmail: "security@example.com", ReminderDaysBefore: 30, ExpiryDate: &longExpiry, Notes: "Week 2: owner assigned, certificate renewed"})
		ops.RecordEvent(tenantID, "evidence.updated", "Supplier certificate renewed", certID)
		_, _ = ops.GenerateReviewSnapshot(ctx, tenantID)

		// Week 3: Stale friction persists, policy updated.
		_ = ev.Update(ctx, tenantID, staleID, evidence.Item{Title: "Accessibility Statement", Category: "Compliance", Status: "active", OwnerName: "Product Owner", OwnerEmail: "product@example.com", ReminderDaysBefore: 30, UpdatedAt: stale, Notes: "Week 3: stale friction still present — 220 days old"})
		_ = ev.Update(ctx, tenantID, policyID, evidence.Item{Title: "Privacy Policy", Category: "Legal", Status: "active", OwnerName: "Legal Owner", OwnerEmail: "legal@example.com", ReminderDaysBefore: 30, Notes: "Week 3: continuity change — reviewed and updated"})
		ops.RecordEvent(tenantID, "evidence.updated", "Privacy policy reviewed", policyID)
		_, _ = ops.GenerateReviewSnapshot(ctx, tenantID)

		// Week 4: Insurance renewed, ownerless item assigned.
		_ = ev.Update(ctx, tenantID, insuranceID, evidence.Item{Title: "General Liability Insurance", Category: "Compliance", Status: "active", OwnerName: "Finance Owner", OwnerEmail: "finance@example.com", ReminderDaysBefore: 30, ExpiryDate: &longExpiry, Notes: "Week 4: insurance renewed for 1 year"})
		_ = ev.Update(ctx, tenantID, ownerlessID, evidence.Item{Title: "Data Retention Policy", Category: "IT", Status: "active", OwnerName: "IT Owner", OwnerEmail: "it@example.com", ReminderDaysBefore: 30, Notes: "Week 4: owner assigned — accountability gap closed"})
		ops.RecordEvent(tenantID, "evidence.updated", "Insurance renewed", insuranceID)
		ops.RecordEvent(tenantID, "evidence.updated", "Data retention owner assigned", ownerlessID)
		_, _ = ops.GenerateReviewSnapshot(ctx, tenantID)
		_, _ = ops.GenerateOperationalSnapshot(ctx, tenantID)
	}
	if pp != nil {
		_, _ = pp.Export(ctx, tenantID, "demo")
		_, _ = pp.Export(ctx, tenantID, "demo")
	}
	return nil
}
