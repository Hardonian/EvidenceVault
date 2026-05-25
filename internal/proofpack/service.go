package proofpack

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ db *pgxpool.Pool }

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }

func (s *Service) Export(ctx context.Context, tenantID string) ([]byte, error) {
	rows, err := s.db.Query(ctx, `select id,title,category,status,owner_name,owner_email,expiry_date,updated_at from evidence_items where tenant_id=$1`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, title, category, status, ownerName, ownerEmail string
		var expiry, updated any
		if err := rows.Scan(&id, &title, &category, &status, &ownerName, &ownerEmail, &expiry, &updated); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "title": title, "category": category, "status": status, "owner_name": ownerName, "owner_email": ownerEmail, "expiry_date": expiry, "updated_at": updated})
	}
	return json.MarshalIndent(map[string]any{"tenant_id": tenantID, "evidence": out}, "", "  ")
}
