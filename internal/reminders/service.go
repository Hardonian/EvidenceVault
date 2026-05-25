package reminders

import (
	"context"
	"time"

	"evidencevault/internal/audit"
	"evidencevault/internal/email"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db    *pgxpool.Pool
	email email.Sender
	audit *audit.Service
}

func NewService(db *pgxpool.Pool, sender email.Sender, auditSvc *audit.Service) *Service {
	return &Service{db: db, email: sender, audit: auditSvc}
}

func (s *Service) Run(ctx context.Context) (int, error) {
	rows, err := s.db.Query(ctx, `select id, tenant_id, owner_email, title from evidence_items where owner_email <> '' and expiry_date is not null and expiry_date::date between now()::date and (now()::date + reminder_days_before)`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	sent := 0
	today := time.Now().UTC().Format("2006-01-02")
	for rows.Next() {
		var id, tenantID, to, title string
		if err := rows.Scan(&id, &tenantID, &to, &title); err != nil {
			return sent, err
		}
		var existing int
		if err := s.db.QueryRow(ctx, `select count(*) from reminder_logs where evidence_id=$1 and reminder_date=$2 and channel='email'`, id, today).Scan(&existing); err != nil {
			return sent, err
		}
		if existing > 0 {
			continue
		}
		status := "sent"
		if err := s.email.Send(to, "Evidence reminder: "+title, "This evidence is expiring soon."); err != nil {
			status = "failed"
		} else {
			sent++
		}
		_, err = s.db.Exec(ctx, `insert into reminder_logs (evidence_id, tenant_id, reminder_date, channel, status) values ($1,$2,$3,'email',$4) on conflict do nothing`, id, tenantID, today, status)
		if err != nil {
			return sent, err
		}
		s.audit.Log(ctx, tenantID, "", "reminder.sent", "evidence_item", id, `{"channel":"email","status":"`+status+`"}`)
	}
	return sent, rows.Err()
}
