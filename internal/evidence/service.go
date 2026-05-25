package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"evidencevault/internal/id"
)

var (
	allowedStatus   = map[string]struct{}{"active": {}, "expiring": {}, "expired": {}, "missing": {}, "archived": {}}
	allowedCategory = map[string]struct{}{"Security": {}, "Compliance": {}, "Finance": {}, "HR": {}, "IT": {}, "Legal": {}, "Other": {}}
	maxReminderDays = 365
)

type Item struct {
	ID, TenantID, Title, Category, Status, OwnerName, OwnerEmail, SourceFilePath, Notes string
	IssueDate, ExpiryDate                                                               *time.Time
	ReminderDaysBefore                                                                  int
	CreatedAt, UpdatedAt                                                                time.Time
}
type File struct {
	EvidenceID, FilePath, ContentType string
	SizeBytes                         int64
	CreatedAt                         time.Time
}
type Service struct {
	mu            sync.Mutex
	items         map[string][]Item
	files         map[string][]File
	freeTierLimit int
}

func NewService(_ any, freeTierLimit int) *Service {
	return &Service{items: map[string][]Item{}, files: map[string][]File{}, freeTierLimit: freeTierLimit}
}
func deriveStatus(existing string, expiry *time.Time, reminderDays int, now time.Time) string {
	if existing == "missing" || existing == "archived" {
		return existing
	}
	if expiry == nil {
		return "active"
	}
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	expiryDay := time.Date(expiry.UTC().Year(), expiry.UTC().Month(), expiry.UTC().Day(), 0, 0, 0, 0, time.UTC)
	if expiryDay.Before(today) {
		return "expired"
	}
	if !expiryDay.After(today.AddDate(0, 0, reminderDays)) {
		return "expiring"
	}
	return "active"
}
func Validate(it Item) error {
	if strings.TrimSpace(it.Title) == "" {
		return errors.New("title is required")
	}
	if _, ok := allowedStatus[it.Status]; !ok {
		return fmt.Errorf("invalid status: %s", it.Status)
	}
	if _, ok := allowedCategory[it.Category]; !ok {
		return fmt.Errorf("invalid category: %s", it.Category)
	}
	if it.ReminderDaysBefore < 0 || it.ReminderDaysBefore > maxReminderDays {
		return fmt.Errorf("reminder_days_before must be between 0 and %d", maxReminderDays)
	}
	if it.IssueDate != nil && it.ExpiryDate != nil && it.ExpiryDate.Before(*it.IssueDate) {
		return errors.New("expiry_date must be on or after issue_date")
	}
	return nil
}
func (s *Service) List(_ context.Context, tenantID string) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Item{}, s.items[tenantID]...)
	return out, nil
}
func (s *Service) Create(_ context.Context, tenantID string, it Item) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items[tenantID]) >= s.freeTierLimit {
		return "", errors.New("free tier limit reached")
	}
	it.Status = deriveStatus(it.Status, it.ExpiryDate, it.ReminderDaysBefore, time.Now())
	if err := Validate(it); err != nil {
		return "", err
	}
	it.ID = id.New()
	it.TenantID = tenantID
	it.CreatedAt = time.Now().UTC()
	it.UpdatedAt = it.CreatedAt
	s.items[tenantID] = append([]Item{it}, s.items[tenantID]...)
	return it.ID, nil
}
func (s *Service) Update(_ context.Context, tenantID, idv string, it Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	it.Status = deriveStatus(it.Status, it.ExpiryDate, it.ReminderDaysBefore, time.Now())
	if err := Validate(it); err != nil {
		return err
	}
	arr := s.items[tenantID]
	for i := range arr {
		if arr[i].ID == idv {
			it.ID = idv
			it.TenantID = tenantID
			it.CreatedAt = arr[i].CreatedAt
			it.UpdatedAt = time.Now().UTC()
			arr[i] = it
			s.items[tenantID] = arr
			return nil
		}
	}
	return errors.New("not found")
}
func (s *Service) AttachFile(_ context.Context, tenantID, evidenceID, filePath, contentType string, sizeBytes int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	arr := s.items[tenantID]
	for i := range arr {
		if arr[i].ID == evidenceID {
			arr[i].SourceFilePath = filePath
			arr[i].UpdatedAt = time.Now().UTC()
			s.items[tenantID] = arr
			s.files[tenantID] = append(s.files[tenantID], File{EvidenceID: evidenceID, FilePath: filePath, ContentType: contentType, SizeBytes: sizeBytes, CreatedAt: time.Now().UTC()})
			return nil
		}
	}
	return errors.New("evidence not found")
}
func (s *Service) Files(tenantID string) []File {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]File{}, s.files[tenantID]...)
}
func (s *Service) All() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Item{}
	for _, arr := range s.items {
		out = append(out, arr...)
	}
	return out
}
