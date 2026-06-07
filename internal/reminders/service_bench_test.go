package reminders

import (
	"context"
	"testing"
	"time"
	"strconv"

	"evidencevault/internal/audit"
	"evidencevault/internal/evidence"
	"evidencevault/internal/persistence"
)

func BenchmarkServiceRun(b *testing.B) {
	ctx := context.Background()
	store := persistence.NewMemoryStore()
	mockEmail := &MockEmailSender{}
	auditSvc := audit.NewService(store)
	evSvc := evidence.NewService(store, 10000)
	svc := NewService(store, mockEmail, auditSvc, evSvc)

	tenantID := "tenant-bench"
	now := time.Now().UTC()
	expiryDate := now.Add(5 * 24 * time.Hour)
	issueDate := now.Add(-30 * 24 * time.Hour)

	for i := 0; i < 1000; i++ {
		_, _ = evSvc.Create(ctx, tenantID, evidence.Item{
			Title:              "Report " + strconv.Itoa(i),
			Category:           "Compliance",
			Status:             "active",
			OwnerName:          "Alice",
			OwnerEmail:         "alice@example.com",
			IssueDate:          &issueDate,
			ExpiryDate:         &expiryDate,
			ReminderDaysBefore: 10,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.Run(ctx)
		// Reset state between benchmark iterations
		_ = store.Write(func(st *persistence.State) error {
			st.ReminderSent = make(map[string]struct{})
			return nil
		})
		mockEmail.SentEmails = nil
	}
}
