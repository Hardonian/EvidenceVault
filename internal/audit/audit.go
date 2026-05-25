package audit

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ db *pgxpool.Pool }

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }
func (s *Service) Log(ctx context.Context, tenantID, userID, action, entity, entityID, data string) {
	_, _ = s.db.Exec(ctx, `insert into audit_logs (tenant_id, user_id, action, entity_type, entity_id, metadata) values ($1,$2,$3,$4,$5,$6::jsonb)`, tenantID, userID, action, entity, entityID, data)
}
