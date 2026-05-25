package persistence

import (
	"errors"
	"sync"
	"time"
)

type EvidenceItem struct {
	ID, TenantID, Title, Category, Status, OwnerName, OwnerEmail, SourceFilePath, Notes string
	IssueDate, ExpiryDate                                                               *time.Time
	ReminderDaysBefore                                                                  int
	CreatedAt, UpdatedAt                                                                time.Time
}
type EvidenceFile struct {
	EvidenceID, FilePath, ContentType string
	SizeBytes                         int64
	CreatedAt                         time.Time
}
type AuditEntry struct {
	TenantID, UserID, Action, EntityType, EntityID, Metadata string
	CreatedAt                                                time.Time
}

type State struct {
	Tenants      map[string]string
	Evidence     map[string][]EvidenceItem
	EvidenceFile map[string][]EvidenceFile
	ReminderSent map[string]struct{}
	Proofpacks   map[string][]ProofpackMeta
	AuditLogs    []AuditEntry
	StripeEvents map[string]struct{}
}

// rest

type ProofpackMeta struct {
	ID        string
	CreatedAt time.Time
}
type Store interface {
	WithLock(func(*State) error) error
}
type MemoryStore struct {
	mu    sync.Mutex
	state State
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{state: emptyState()} }
func (m *MemoryStore) WithLock(fn func(*State) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fn(&m.state)
}
func emptyState() State {
	return State{Tenants: map[string]string{}, Evidence: map[string][]EvidenceItem{}, EvidenceFile: map[string][]EvidenceFile{}, ReminderSent: map[string]struct{}{}, Proofpacks: map[string][]ProofpackMeta{}, AuditLogs: []AuditEntry{}, StripeEvents: map[string]struct{}{}}
}

var ErrDataDirRequired = errors.New("DATA_DIR is required for file persistence mode")
