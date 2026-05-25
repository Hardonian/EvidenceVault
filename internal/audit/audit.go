package audit

import (
	"context"
	"sync"
	"time"
)

type Entry struct {
	TenantID, UserID, Action, EntityType, EntityID, Metadata string
	CreatedAt                                                time.Time
}

type Service struct {
	mu      sync.Mutex
	entries []Entry
}

func NewService() *Service { return &Service{} }
func (s *Service) Log(_ context.Context, tenantID, userID, action, entity, entityID, data string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, Entry{TenantID: tenantID, UserID: userID, Action: action, EntityType: entity, EntityID: entityID, Metadata: data, CreatedAt: time.Now().UTC()})
}
func (s *Service) ListByTenant(tenantID string, limit int) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []map[string]any{}
	for i := len(s.entries) - 1; i >= 0; i-- {
		e := s.entries[i]
		if e.TenantID != tenantID {
			continue
		}
		out = append(out, map[string]any{"action": e.Action, "entity_type": e.EntityType, "entity_id": e.EntityID, "created_at": e.CreatedAt})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}
