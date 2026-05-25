package reminders

import (
	"context"
	"errors"
	"testing"
	"time"

	"evidencevault/internal/audit"
	"evidencevault/internal/evidence"
	"evidencevault/internal/persistence"
)

type MockEmailSender struct {
	SentEmails []Email
	ShouldFail bool
}

type Email struct {
	To      string
	Subject string
	Body    string
}

func (m *MockEmailSender) Send(to, subject, body string) error {
	if m.ShouldFail {
		return errors.New("failed to send")
	}
	m.SentEmails = append(m.SentEmails, Email{To: to, Subject: subject, Body: body})
	return nil
}

func setupTest(t *testing.T) (*Service, *MockEmailSender, persistence.Store, *evidence.Service, *audit.Service) {
	store := persistence.NewMemoryStore()
	mockEmail := &MockEmailSender{}
	auditSvc := audit.NewService(store)
	evSvc := evidence.NewService(store, 100)

	svc := NewService(store, mockEmail, auditSvc, evSvc)
	return svc, mockEmail, store, evSvc, auditSvc
}


func TestService_Run(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	t.Run("happy path - sends reminder for expiring evidence", func(t *testing.T) {
		svc, mockEmail, _, evSvc, auditSvc := setupTest(t)
		tenantID := "tenant-1"

		issueDate := now.Add(-30 * 24 * time.Hour)
		expiryDate := now.Add(5 * 24 * time.Hour) // expires in 5 days

		_, err := evSvc.Create(ctx, tenantID, evidence.Item{
			Title:              "SOC2 Report",
			Category:           "Compliance",
			Status:             "active",
			OwnerName:          "Alice",
			OwnerEmail:         "alice@example.com",
			IssueDate:          &issueDate,
			ExpiryDate:         &expiryDate,
			ReminderDaysBefore: 10, // within reminder window
		})
		if err != nil {
			t.Fatalf("failed to create evidence: %v", err)
		}

		sent, err := svc.Run(ctx)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		if sent != 1 {
			t.Errorf("expected 1 reminder sent, got %d", sent)
		}

		if len(mockEmail.SentEmails) != 1 {
			t.Errorf("expected 1 email sent, got %d", len(mockEmail.SentEmails))
		} else {
			email := mockEmail.SentEmails[0]
			if email.To != "alice@example.com" {
				t.Errorf("expected email to alice@example.com, got %s", email.To)
			}
			if email.Subject != "Evidence reminder: SOC2 Report" {
				t.Errorf("expected subject 'Evidence reminder: SOC2 Report', got %s", email.Subject)
			}
		}

		auditLogs := auditSvc.ListByTenant(tenantID, 10)
		if len(auditLogs) != 1 {
			t.Errorf("expected 1 audit log, got %d", len(auditLogs))
		} else {
			if auditLogs[0]["action"] != "reminder.sent" {
				t.Errorf("expected audit action 'reminder.sent', got %s", auditLogs[0]["action"])
			}
		}
	})

	t.Run("missing fields - ignores items without email or expiry", func(t *testing.T) {
		svc, mockEmail, _, evSvc, _ := setupTest(t)
		tenantID := "tenant-2"

		issueDate := now.Add(-30 * 24 * time.Hour)
		expiryDate := now.Add(5 * 24 * time.Hour)

		// missing email
		_, _ = evSvc.Create(ctx, tenantID, evidence.Item{
			Title:              "SOC2 Report",
			Category:           "Compliance",
			Status:             "active",
			OwnerName:          "Alice",
			IssueDate:          &issueDate,
			ExpiryDate:         &expiryDate,
			ReminderDaysBefore: 10,
		})

		// missing expiry
		_, _ = evSvc.Create(ctx, tenantID, evidence.Item{
			Title:              "ISO 27001",
			Category:           "Compliance",
			Status:             "active",
			OwnerName:          "Bob",
			OwnerEmail:         "bob@example.com",
			IssueDate:          &issueDate,
			ReminderDaysBefore: 10,
		})

		sent, err := svc.Run(ctx)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		if sent != 0 {
			t.Errorf("expected 0 reminders sent, got %d", sent)
		}
		if len(mockEmail.SentEmails) != 0 {
			t.Errorf("expected 0 emails sent, got %d", len(mockEmail.SentEmails))
		}
	})

	t.Run("timing conditions - ignores expired or not yet expiring items", func(t *testing.T) {
		svc, mockEmail, _, evSvc, _ := setupTest(t)
		tenantID := "tenant-3"

		issueDate := now.Add(-30 * 24 * time.Hour)
		expiredDate := now.Add(-1 * 24 * time.Hour) // already expired
		futureDate := now.Add(30 * 24 * time.Hour)  // expires in 30 days, well beyond 10-day reminder window

		// expired
		_, _ = evSvc.Create(ctx, tenantID, evidence.Item{
			Title:              "Expired Report",
			Category:           "Compliance",
			Status:             "active",
			OwnerEmail:         "alice@example.com",
			IssueDate:          &issueDate,
			ExpiryDate:         &expiredDate,
			ReminderDaysBefore: 10,
		})

		// not expiring soon
		_, _ = evSvc.Create(ctx, tenantID, evidence.Item{
			Title:              "Future Report",
			Category:           "Compliance",
			Status:             "active",
			OwnerEmail:         "bob@example.com",
			IssueDate:          &issueDate,
			ExpiryDate:         &futureDate,
			ReminderDaysBefore: 10,
		})

		sent, err := svc.Run(ctx)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		if sent != 0 {
			t.Errorf("expected 0 reminders sent, got %d", sent)
		}
		if len(mockEmail.SentEmails) != 0 {
			t.Errorf("expected 0 emails sent, got %d", len(mockEmail.SentEmails))
		}
	})

	t.Run("deduplication - sends only once per day", func(t *testing.T) {
		svc, mockEmail, _, evSvc, _ := setupTest(t)
		tenantID := "tenant-4"

		issueDate := now.Add(-30 * 24 * time.Hour)
		expiryDate := now.Add(5 * 24 * time.Hour)

		_, _ = evSvc.Create(ctx, tenantID, evidence.Item{
			Title:              "SOC2 Report",
			Category:           "Compliance",
			Status:             "active",
			OwnerEmail:         "alice@example.com",
			IssueDate:          &issueDate,
			ExpiryDate:         &expiryDate,
			ReminderDaysBefore: 10,
		})

		// First run should send email
		sent1, err := svc.Run(ctx)
		if err != nil {
			t.Fatalf("First Run failed: %v", err)
		}
		if sent1 != 1 {
			t.Errorf("expected 1 reminder sent, got %d", sent1)
		}

		// Second run on same day should not send
		sent2, err := svc.Run(ctx)
		if err != nil {
			t.Fatalf("Second Run failed: %v", err)
		}
		if sent2 != 0 {
			t.Errorf("expected 0 reminders sent on second run, got %d", sent2)
		}

		if len(mockEmail.SentEmails) != 1 {
			t.Errorf("expected 1 total email sent, got %d", len(mockEmail.SentEmails))
		}
	})

	t.Run("failed email sending - audit log reflects failure", func(t *testing.T) {
		svc, mockEmail, _, evSvc, auditSvc := setupTest(t)
		mockEmail.ShouldFail = true
		tenantID := "tenant-5"

		issueDate := now.Add(-30 * 24 * time.Hour)
		expiryDate := now.Add(5 * 24 * time.Hour)

		_, _ = evSvc.Create(ctx, tenantID, evidence.Item{
			Title:              "SOC2 Report",
			Category:           "Compliance",
			Status:             "active",
			OwnerEmail:         "alice@example.com",
			IssueDate:          &issueDate,
			ExpiryDate:         &expiryDate,
			ReminderDaysBefore: 10,
		})

		sent, err := svc.Run(ctx)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		if sent != 0 {
			t.Errorf("expected 0 successful reminders sent, got %d", sent)
		}

		auditLogs := auditSvc.ListByTenant(tenantID, 10)
		if len(auditLogs) != 1 {
			t.Errorf("expected 1 audit log, got %d", len(auditLogs))
		}
	})
}
