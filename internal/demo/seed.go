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
	items, err := ev.List(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(items) > 0 {
		return nil
	}
	now := time.Now().UTC()
	soon := now.AddDate(0, 0, 14)
	expired := now.AddDate(0, 0, -7)
	stale := now.AddDate(0, 0, -220)
	longExpiry := now.AddDate(1, 0, 0)

	// Create diverse evidence across categories and risk states.
	insuranceID, err := ev.Create(ctx, tenantID, evidence.Item{Title: "General Liability Insurance", Category: "Compliance", Status: "expiring", OwnerName: "Finance Owner", OwnerEmail: "finance@example.com", ReminderDaysBefore: 30, ExpiryDate: &soon, Notes: "Renewal in two weeks"})
	if err != nil {
		return err
	}
	policyID, err := ev.Create(ctx, tenantID, evidence.Item{Title: "Privacy Policy", Category: "Legal", Status: "active", OwnerName: "Legal Owner", OwnerEmail: "legal@example.com", ReminderDaysBefore: 30, Notes: "Published policy evidence"})
	if err != nil {
		return err
	}
	certID, err := ev.Create(ctx, tenantID, evidence.Item{Title: "Supplier Security Certificate", Category: "Compliance", Status: "expired", ReminderDaysBefore: 30, ExpiryDate: &expired, Notes: "Expired and unassigned — ownerless risk"})
	if err != nil {
		return err
	}
	staleID, err := ev.Create(ctx, tenantID, evidence.Item{Title: "Accessibility Statement", Category: "Compliance", Status: "active", OwnerName: "Product Owner", OwnerEmail: "product@example.com", ReminderDaysBefore: 30, UpdatedAt: stale, Notes: "Stale evidence — not updated in 220 days"})
	if err != nil {
		return err
	}

	// Additional evidence for graph diversity.
	_, err = ev.Create(ctx, tenantID, evidence.Item{Title: "SOC2 Type II Report", Category: "Security", Status: "active", OwnerName: "Security Owner", OwnerEmail: "security@example.com", ReminderDaysBefore: 30, ExpiryDate: &longExpiry, Notes: "Annual SOC2 audit report"})
	if err != nil {
		return err
	}
	_, err = ev.Create(ctx, tenantID, evidence.Item{Title: "Employee Handbook Acknowledgement", Category: "HR", Status: "active", OwnerName: "People Ops", OwnerEmail: "people@example.com", ReminderDaysBefore: 30, ExpiryDate: &soon, Notes: "Annual handbook acknowledgment"})
	if err != nil {
		return err
	}
	_, err = ev.Create(ctx, tenantID, evidence.Item{Title: "Vendor Agreement - Cloud Provider", Category: "Legal", Status: "active", OwnerName: "Legal Owner", OwnerEmail: "legal@example.com", ReminderDaysBefore: 45, ExpiryDate: &longExpiry, Notes: "Primary cloud vendor contract"})
	if err != nil {
		return err
	}
	ownerlessID, err := ev.Create(ctx, tenantID, evidence.Item{Title: "Data Retention Policy", Category: "IT", Status: "active", ReminderDaysBefore: 30, Notes: "Ownerless — needs accountability assignment"})
	if err != nil {
		return err
	}

	if ops != nil {
		// Week 1: Initial state with problems.
		ops.RecordEvent(tenantID, "reminder.sent", "Renewal reminder sent", "")
		ops.RecordEvent(tenantID, "owner.missing", "Owner assignment missing", "")
		_, err = ops.GenerateReviewSnapshot(ctx, tenantID)
		if err != nil {
			return err
		}

		// Week 2: Fix expired cert — assign owner and renew.
		err = ev.Update(ctx, tenantID, certID, evidence.Item{Title: "Supplier Security Certificate", Category: "Compliance", Status: "active", OwnerName: "Security Owner", OwnerEmail: "security@example.com", ReminderDaysBefore: 30, ExpiryDate: &longExpiry, Notes: "Week 2: owner assigned, certificate renewed"})
		if err != nil {
			return err
		}
		ops.RecordEvent(tenantID, "evidence.updated", "Supplier certificate renewed", certID)
		_, err = ops.GenerateReviewSnapshot(ctx, tenantID)
		if err != nil {
			return err
		}

		// Week 3: Stale friction persists, policy updated.
		err = ev.Update(ctx, tenantID, staleID, evidence.Item{Title: "Accessibility Statement", Category: "Compliance", Status: "active", OwnerName: "Product Owner", OwnerEmail: "product@example.com", ReminderDaysBefore: 30, UpdatedAt: stale, Notes: "Week 3: stale friction still present — 220 days old"})
		if err != nil {
			return err
		}
		err = ev.Update(ctx, tenantID, policyID, evidence.Item{Title: "Privacy Policy", Category: "Legal", Status: "active", OwnerName: "Legal Owner", OwnerEmail: "legal@example.com", ReminderDaysBefore: 30, Notes: "Week 3: continuity change — reviewed and updated"})
		if err != nil {
			return err
		}
		ops.RecordEvent(tenantID, "evidence.updated", "Privacy policy reviewed", policyID)
		_, err = ops.GenerateReviewSnapshot(ctx, tenantID)
		if err != nil {
			return err
		}

		// Week 4: Insurance renewed, ownerless item assigned.
		err = ev.Update(ctx, tenantID, insuranceID, evidence.Item{Title: "General Liability Insurance", Category: "Compliance", Status: "active", OwnerName: "Finance Owner", OwnerEmail: "finance@example.com", ReminderDaysBefore: 30, ExpiryDate: &longExpiry, Notes: "Week 4: insurance renewed for 1 year"})
		if err != nil {
			return err
		}
		err = ev.Update(ctx, tenantID, ownerlessID, evidence.Item{Title: "Data Retention Policy", Category: "IT", Status: "active", OwnerName: "IT Owner", OwnerEmail: "it@example.com", ReminderDaysBefore: 30, Notes: "Week 4: owner assigned — accountability gap closed"})
		if err != nil {
			return err
		}
		ops.RecordEvent(tenantID, "evidence.updated", "Insurance renewed", insuranceID)
		ops.RecordEvent(tenantID, "evidence.updated", "Data retention owner assigned", ownerlessID)
		_, err = ops.GenerateReviewSnapshot(ctx, tenantID)
		if err != nil {
			return err
		}
		_, err = ops.GenerateOperationalSnapshot(ctx, tenantID)
		if err != nil {
			return err
		}
	}
	if pp != nil {
		_, err = pp.Export(ctx, tenantID, "demo")
		if err != nil {
			return err
		}
		_, err = pp.Export(ctx, tenantID, "demo")
		if err != nil {
			return err
		}
	}
	return nil
}
