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
	ControlRefs, VendorRefs, RiskRefs                                                   []string
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

type ProofpackMeta struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	EvidenceIDs []string  `json:"evidence_ids,omitempty"`
	Hash        string    `json:"hash,omitempty"`
}

type UnresolvedIssue struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	EvidenceID     string     `json:"evidence_id"`
	Type           string     `json:"type"`
	CreatedAt      time.Time  `json:"created_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	ResolvedReason string     `json:"resolved_reason,omitempty"`
}

type AdjudicationEvent struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	IssueID   string    `json:"issue_id"`
	Action    string    `json:"action"`
	Operator  string    `json:"operator"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type ReviewSnapshot struct {
	TenantID              string    `json:"tenant_id"`
	GeneratedAt           time.Time `json:"generated_at"`
	LastReviewedAt        time.Time `json:"last_reviewed_at"`
	NextRecommendedReview time.Time `json:"next_recommended_review"`
	HealthScore           int       `json:"health_score"`
	UnresolvedIssues      int       `json:"unresolved_issues"`
	ExpiredEvidence       int       `json:"expired_evidence"`
	ExpiringEvidence      int       `json:"expiring_evidence"`
	StaleEvidence         int       `json:"stale_evidence"`
	MissingOwners         int       `json:"missing_owners"`
	Disclaimer            string    `json:"disclaimer"`
}

type OperationalSnapshot struct {
	TenantID               string    `json:"tenant_id"`
	Date                   string    `json:"date"`
	CreatedAt              time.Time `json:"created_at"`
	UnresolvedCount        int       `json:"unresolved_count"`
	ExpiredEvidenceCount   int       `json:"expired_evidence_count"`
	OwnerlessEvidenceCount int       `json:"ownerless_evidence_count"`
	StaleEvidenceCount     int       `json:"stale_evidence_count"`
	HealthScore            int       `json:"health_score"`
	ProofpackCount         int       `json:"proofpack_count"`
	ReviewStreak           int       `json:"review_streak"`
	ActivationPercent      int       `json:"activation_percent"`
	MaturityStage          string    `json:"maturity_stage"`
	TotalEvidenceCount     int       `json:"total_evidence_count"`
	OwnersCount            int       `json:"owners_count"`
}

type ReviewReport struct {
	TenantID    string    `json:"tenant_id"`
	ID          string    `json:"id"`
	GeneratedAt time.Time `json:"generated_at"`
	Summary     string    `json:"summary"`
	Markdown    string    `json:"markdown"`
	PlainText   string    `json:"plain_text"`
	HTML        string    `json:"html"`
}

type OperationalEvent struct {
	TenantID  string    `json:"tenant_id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	EntityID  string    `json:"entity_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ActivationMilestones struct {
	FirstEvidenceCreatedAt    *time.Time `json:"first_evidence_created_at,omitempty"`
	FirstFileUploadedAt       *time.Time `json:"first_file_uploaded_at,omitempty"`
	FirstReminderRunAt        *time.Time `json:"first_reminder_run_at,omitempty"`
	FirstProofpackGeneratedAt *time.Time `json:"first_proofpack_generated_at,omitempty"`
	FirstOperationalReviewAt  *time.Time `json:"first_operational_review_at,omitempty"`
	SecondWeeklyReviewAt      *time.Time `json:"second_weekly_review_at,omitempty"`
}

type State struct {
	Tenants              map[string]string
	Evidence             map[string][]EvidenceItem
	EvidenceFile         map[string][]EvidenceFile
	ReminderSent         map[string]struct{}

type ReviewSnapshot struct {
	TenantID              string    `json:"tenant_id"`
	GeneratedAt           time.Time `json:"generated_at"`
	LastReviewedAt        time.Time `json:"last_reviewed_at"`
	NextRecommendedReview time.Time `json:"next_recommended_review"`
	HealthScore           int       `json:"health_score"`
	UnresolvedIssues      int       `json:"unresolved_issues"`
	ExpiredEvidence       int       `json:"expired_evidence"`
	ExpiringEvidence      int       `json:"expiring_evidence"`
	StaleEvidence         int       `json:"stale_evidence"`
	MissingOwners         int       `json:"missing_owners"`
	Disclaimer            string    `json:"disclaimer"`
}

type OperationalSnapshot struct {
	TenantID               string    `json:"tenant_id"`
	Date                   string    `json:"date"`
	CreatedAt              time.Time `json:"created_at"`
	UnresolvedCount        int       `json:"unresolved_count"`
	ExpiredEvidenceCount   int       `json:"expired_evidence_count"`
	OwnerlessEvidenceCount int       `json:"ownerless_evidence_count"`
	StaleEvidenceCount     int       `json:"stale_evidence_count"`
	HealthScore            int       `json:"health_score"`
	ProofpackCount         int       `json:"proofpack_count"`
	ReviewStreak           int       `json:"review_streak"`
	ActivationPercent      int       `json:"activation_percent"`
	MaturityStage          string    `json:"maturity_stage"`
	TotalEvidenceCount     int       `json:"total_evidence_count"`
	OwnersCount            int       `json:"owners_count"`
}

type ReviewReport struct {
	TenantID    string    `json:"tenant_id"`
	ID          string    `json:"id"`
	GeneratedAt time.Time `json:"generated_at"`
	Summary     string    `json:"summary"`
	Markdown    string    `json:"markdown"`
	PlainText   string    `json:"plain_text"`
	HTML        string    `json:"html"`
}

type OperationalEvent struct {
	TenantID  string    `json:"tenant_id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	EntityID  string    `json:"entity_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ActivationMilestones struct {
	FirstEvidenceCreatedAt    *time.Time `json:"first_evidence_created_at,omitempty"`
	FirstFileUploadedAt       *time.Time `json:"first_file_uploaded_at,omitempty"`
	FirstReminderRunAt        *time.Time `json:"first_reminder_run_at,omitempty"`
	FirstProofpackGeneratedAt *time.Time `json:"first_proofpack_generated_at,omitempty"`
	FirstOperationalReviewAt  *time.Time `json:"first_operational_review_at,omitempty"`
	SecondWeeklyReviewAt      *time.Time `json:"second_weekly_review_at,omitempty"`
}

type State struct {
	Tenants              map[string]string
	Evidence             map[string][]EvidenceItem
	EvidenceFile         map[string][]EvidenceFile
	ReminderSent         map[string]struct{}
	Proofpacks           map[string][]ProofpackMeta
	AuditLogs            []AuditEntry
	StripeEvents         map[string]struct{}
	ReviewSnapshots      map[string][]ReviewSnapshot
	OperationalEvents    map[string][]OperationalEvent
	Activation           map[string]ActivationMilestones
	OperationalSnapshots map[string][]OperationalSnapshot
	ReviewReports        map[string][]ReviewReport
	UnresolvedIssues     map[string][]UnresolvedIssue
	AdjudicationEvents   map[string][]AdjudicationEvent
}

type Store interface {
	Read(func(*State) error) error
	Write(func(*State) error) error
}
type MemoryStore struct {
	mu    sync.RWMutex
	state State
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{state: emptyState()} }
func (m *MemoryStore) Read(fn func(*State) error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return fn(&m.state)
}
func (m *MemoryStore) Write(fn func(*State) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fn(&m.state)
}
func (m *MemoryStore) WithLock(fn func(*State) error) error { return m.Write(fn) }
func emptyState() State {
	return State{Tenants: map[string]string{}, Evidence: map[string][]EvidenceItem{}, EvidenceFile: map[string][]EvidenceFile{}, ReminderSent: map[string]struct{}{}, Proofpacks: map[string][]ProofpackMeta{}, AuditLogs: []AuditEntry{}, StripeEvents: map[string]struct{}{}, ReviewSnapshots: map[string][]ReviewSnapshot{}, OperationalEvents: map[string][]OperationalEvent{}, Activation: map[string]ActivationMilestones{}, OperationalSnapshots: map[string][]OperationalSnapshot{}, ReviewReports: map[string][]ReviewReport{}, UnresolvedIssues: map[string][]UnresolvedIssue{}, AdjudicationEvents: map[string][]AdjudicationEvent{}}
}

var ErrDataDirRequired = errors.New("DATA_DIR is required for file persistence mode")
