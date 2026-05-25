package reminders

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ db *pgxpool.Pool }

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }

func (s *Service) Run(ctx context.Context) (int, error) {
	rows, err := s.db.Query(ctx, `select id, owner_email, title from evidence_items where expiry_date::date = (now()::date + reminder_days_before) and owner_email <> ''`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	sent := 0
	for rows.Next() {
		var id, email, title string
		if err := rows.Scan(&id, &email, &title); err != nil {
			return sent, err
		}
		res, err := s.db.Exec(ctx, `insert into reminder_logs (evidence_id, tenant_id, reminder_date, channel, status) select e.id,e.tenant_id,$2,'email','sent' from evidence_items e where e.id=$1 on conflict do nothing`, id, time.Now().UTC().Format("2006-01-02"))
		if err != nil {
			return sent, err
		}
		if res.RowsAffected() > 0 {
			sent++
		}
	}
	return sent, rows.Err()
}
