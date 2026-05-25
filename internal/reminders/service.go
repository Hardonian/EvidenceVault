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
	toProcess := make([]evidence.Item, 0, len(items))
	nowUTC := time.Now().UTC()
	for _, it := range items {
		if it.OwnerEmail == "" || it.ExpiryDate == nil {
			continue
		}
		if it.ExpiryDate.UTC().Before(nowUTC) || it.ExpiryDate.UTC().After(nowUTC.AddDate(0, 0, it.ReminderDaysBefore)) {
			continue
		}
		k := it.ID + ":" + today
		dup := false
		_ = s.store.Write(func(st *persistence.State) error {
			_, dup = st.ReminderSent[k]
			if !dup {
				st.ReminderSent[k] = struct{}{}
			}
			return nil
		})
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
