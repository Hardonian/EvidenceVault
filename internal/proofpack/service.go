package proofpack

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ db *pgxpool.Pool }

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }

func (s *Service) Export(ctx context.Context, tenantID, version string) ([]byte, error) {
	eRows, err := s.db.Query(ctx, `select id,title,category,status,owner_name,owner_email,expiry_date,updated_at from evidence_items where tenant_id=$1`, tenantID)
	if err != nil {
		return nil, err
	}
	defer eRows.Close()
	var evidence []map[string]any
	for eRows.Next() {
		var id, title, category, status, ownerName, ownerEmail string
		var expiry, updated any
		if err := eRows.Scan(&id, &title, &category, &status, &ownerName, &ownerEmail, &expiry, &updated); err != nil {
			return nil, err
		}
		evidence = append(evidence, map[string]any{"id": id, "title": title, "category": category, "status": status, "owner_name": ownerName, "owner_email": ownerEmail, "expiry_date": expiry, "updated_at": updated})
	}

	rRows, err := s.db.Query(ctx, `select evidence_id, reminder_date, channel, status, created_at from reminder_logs where tenant_id=$1 order by created_at desc`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rRows.Close()
	var logs []map[string]any
	for rRows.Next() {
		var evid, channel, status string
		var reminderDate, createdAt any
		if err := rRows.Scan(&evid, &reminderDate, &channel, &status, &createdAt); err != nil {
			return nil, err
		}
		logs = append(logs, map[string]any{"evidence_id": evid, "reminder_date": reminderDate, "channel": channel, "status": status, "created_at": createdAt})
	}

	payload := map[string]any{"tenant_id": tenantID, "generated_at": time.Now().UTC().Format(time.RFC3339), "source_version": version, "evidence_items": evidence, "reminder_logs": logs}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	_, err = s.db.Exec(ctx, `insert into proofpacks (id, tenant_id, payload) values ($1,$2,$3)`, uuid.NewString(), tenantID, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
