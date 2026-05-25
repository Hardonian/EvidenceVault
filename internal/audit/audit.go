package audit

import (
	"context"
	"time"

	"evidencevault/internal/persistence"
)

type Entry struct {
	TenantID, UserID, Action, EntityType, EntityID, Metadata string
	CreatedAt                                                time.Time
}
type Service struct{ store persistence.Store }

func NewService(store persistence.Store) *Service { return &Service{store: store} }
func (s *Service) Log(_ context.Context, tenantID, userID, action, entity, entityID, data string) {
	_ = s.store.WithLock(func(st *persistence.State) error {
		st.AuditLogs = append(st.AuditLogs, persistence.AuditEntry{TenantID: tenantID, UserID: userID, Action: action, EntityType: entity, EntityID: entityID, Metadata: data, CreatedAt: time.Now().UTC()})
		return nil
	})
}
func (s *Service) ListByTenant(tenantID string, limit int) []map[string]any {
	out := []map[string]any{}
	_ = s.store.WithLock(func(st *persistence.State) error {
		for i := len(st.AuditLogs) - 1; i >= 0; i-- {
			e := st.AuditLogs[i]
			if e.TenantID != tenantID {
				continue
			}
			out = append(out, map[string]any{"action": e.Action, "entity_type": e.EntityType, "entity_id": e.EntityID, "created_at": e.CreatedAt})
			if limit > 0 && len(out) >= limit {
				break
			}
		}
		return nil
	})
	return out
}
