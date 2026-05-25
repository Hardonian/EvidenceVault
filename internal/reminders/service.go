package reminders

import (
	"context"
	"time"

	"evidencevault/internal/audit"
	"evidencevault/internal/email"
	"evidencevault/internal/evidence"
	"evidencevault/internal/persistence"
)

type Service struct {
	evidence *evidence.Service
	email    email.Sender
	audit    *audit.Service
	store    persistence.Store
}

func NewService(store persistence.Store, sender email.Sender, auditSvc *audit.Service, ev *evidence.Service) *Service {
	return &Service{store: store, evidence: ev, email: sender, audit: auditSvc}
}
func (s *Service) Run(ctx context.Context) (int, error) {
	items := s.evidence.All()
	sent := 0
	today := time.Now().UTC().Format("2006-01-02")
	for _, it := range items {
		if it.OwnerEmail == "" || it.ExpiryDate == nil {
			continue
		}
		if it.ExpiryDate.UTC().Before(time.Now().UTC()) || it.ExpiryDate.UTC().After(time.Now().UTC().AddDate(0, 0, it.ReminderDaysBefore)) {
			continue
		}
		k := it.ID + ":" + today
		dup := false
		_ = s.store.WithLock(func(st *persistence.State) error {
			_, dup = st.ReminderSent[k]
			if !dup {
				st.ReminderSent[k] = struct{}{}
			}
			return nil
		})
		if dup {
			continue
		}
		status := "sent"
		if err := s.email.Send(it.OwnerEmail, "Evidence reminder: "+it.Title, "This evidence is expiring soon."); err != nil {
			status = "failed"
		} else {
			sent++
		}
		s.audit.Log(ctx, it.TenantID, "", "reminder.sent", "evidence_item", it.ID, "{\"channel\":\"email\",\"status\":\""+status+"\"}")
	}
	return sent, nil
}
