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
	nowUTC := time.Now().UTC()
	items := s.evidence.All(func(it evidence.Item) bool {
		if it.OwnerEmail == "" || it.ExpiryDate == nil {
			return false
		}
		if it.ExpiryDate.UTC().Before(nowUTC) || it.ExpiryDate.UTC().After(nowUTC.AddDate(0, 0, it.ReminderDaysBefore)) {
			return false
		}
		return true
	})

	sent := 0
	today := nowUTC.Format("2006-01-02")
	toSend := make([]evidence.Item, 0, len(items))

	for _, it := range items {
		k := it.ID + ":" + today
		dup := false
		_ = s.store.Write(func(st *persistence.State) error {
			_, dup = st.ReminderSent[k]
			if !dup {
				st.ReminderSent[k] = struct{}{}
			}
			return nil
		})
		if !dup {
			toSend = append(toSend, it)
		}
	}

	for _, it := range toSend {
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
