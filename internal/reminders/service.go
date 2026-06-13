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
	day := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)

	sent := 0
	var toSend []evidence.Item

	// First, collect all items that need reminders
	_ = s.store.Read(func(st *persistence.State) error {
		for _, items := range st.Evidence {
			for _, item := range items {
				if item.OwnerEmail == "" || item.ExpiryDate == nil {
					continue
				}
				if item.ExpiryDate.UTC().Before(nowUTC) || item.ExpiryDate.UTC().After(nowUTC.AddDate(0, 0, item.ReminderDaysBefore)) {
					continue
				}
				k := item.ID + ":" + day.Format("2006-01-02")
				if _, dup := st.ReminderSent[k]; !dup {
					st.ReminderSent[k] = struct{}{}
					toSend = append(toSend, evidence.Item(item))
				}
			}
		}
		return nil
	})

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
