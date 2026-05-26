package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"evidencevault/internal/id"
	"evidencevault/internal/persistence"
)

var (
	allowedStatus   = map[string]struct{}{"active": {}, "expiring": {}, "expired": {}, "missing": {}, "archived": {}}
	allowedCategory = map[string]struct{}{"Security": {}, "Compliance": {}, "Finance": {}, "HR": {}, "IT": {}, "Legal": {}, "Ops": {}, "Other": {}}
	maxReminderDays = 365
)

type Item struct {
	ID, TenantID, Title, Category, Status, OwnerName, OwnerEmail, SourceFilePath, Notes string
	IssueDate, ExpiryDate                                                               *time.Time
	ReminderDaysBefore                                                                  int
	ControlRefs, VendorRefs, RiskRefs                                                   []string
	CreatedAt, UpdatedAt                                                                time.Time
}
type File struct {
	EvidenceID, FilePath, ContentType string
	SizeBytes                         int64
	CreatedAt                         time.Time
}
type Service struct {
	store         persistence.Store
	freeTierLimit int
}

func NewService(store persistence.Store, freeTierLimit int) *Service {
	return &Service{store: store, freeTierLimit: freeTierLimit}
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
	if hasBlankRef(it.ControlRefs) || hasBlankRef(it.VendorRefs) || hasBlankRef(it.RiskRefs) {
		return errors.New("mapping references must not be blank")
	}
	return nil
}

func hasBlankRef(refs []string) bool {
	for _, r := range refs {
		if strings.TrimSpace(r) == "" {
			return true
		}
	}
	return false
}
func (s *Service) List(_ context.Context, tenantID string) ([]Item, error) {
	var out []Item
	_ = s.store.Read(func(st *persistence.State) error {
		out = append([]Item{}, toItems(st.Evidence[tenantID])...)
		return nil
	})
	return out, nil
}
func (s *Service) Get(_ context.Context, tenantID, idv string) (Item, error) {
	var out Item
	err := s.store.Read(func(st *persistence.State) error {
		for _, it := range st.Evidence[tenantID] {
			if it.ID == idv {
				out = Item(it)
				return nil
			}
		}
		return errors.New("not found")
	})
	return out, err
}
func (s *Service) Create(_ context.Context, tenantID string, it Item) (string, error) {
	var idv string
	err := s.store.Write(func(st *persistence.State) error {
		if len(st.Evidence[tenantID]) >= s.freeTierLimit {
			return errors.New("free tier limit reached")
		}
		it.Status = deriveStatus(it.Status, it.ExpiryDate, it.ReminderDaysBefore, time.Now())
		it.ControlRefs = normalizeRefs(it.ControlRefs)
		it.VendorRefs = normalizeRefs(it.VendorRefs)
		it.RiskRefs = normalizeRefs(it.RiskRefs)
		if err := Validate(it); err != nil {
			return err
		}
		it.ID = id.New()
		it.TenantID = tenantID
		it.CreatedAt = time.Now().UTC()
		it.UpdatedAt = it.CreatedAt
		st.Evidence[tenantID] = append([]persistence.EvidenceItem{toPersistItem(it)}, st.Evidence[tenantID]...)
		idv = it.ID
		return nil
	})
	return idv, err
}
func (s *Service) Update(_ context.Context, tenantID, idv string, it Item) error {
	return s.store.Write(func(st *persistence.State) error {
		it.Status = deriveStatus(it.Status, it.ExpiryDate, it.ReminderDaysBefore, time.Now())
		it.ControlRefs = normalizeRefs(it.ControlRefs)
		it.VendorRefs = normalizeRefs(it.VendorRefs)
		it.RiskRefs = normalizeRefs(it.RiskRefs)
		if err := Validate(it); err != nil {
			return err
		}
		arr := st.Evidence[tenantID]
		for i := range arr {
			if arr[i].ID == idv {
				it.ID = idv
				it.TenantID = tenantID
				it.CreatedAt = arr[i].CreatedAt
				it.UpdatedAt = time.Now().UTC()
				arr[i] = toPersistItem(it)
				st.Evidence[tenantID] = arr
				return nil
			}
		}
		return errors.New("not found")
	})
}

func normalizeRefs(refs []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, ref := range refs {
		clean := strings.Join(strings.Fields(ref), " ")
		if clean == "" {
			continue
		}
		key := strings.ToLower(clean)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, clean)
	}
	return out
}
func (s *Service) AttachFile(_ context.Context, tenantID, evidenceID, filePath, contentType string, sizeBytes int64) error {
	return s.store.Write(func(st *persistence.State) error {
		arr := st.Evidence[tenantID]
		for i := range arr {
			if arr[i].ID == evidenceID {
				arr[i].SourceFilePath = filePath
				arr[i].UpdatedAt = time.Now().UTC()
				st.Evidence[tenantID] = arr
				st.EvidenceFile[tenantID] = append(st.EvidenceFile[tenantID], persistence.EvidenceFile{EvidenceID: evidenceID, FilePath: filePath, ContentType: contentType, SizeBytes: sizeBytes, CreatedAt: time.Now().UTC()})
				return nil
			}
		}
		return errors.New("evidence not found")
	})
}
func (s *Service) Files(tenantID string) []File {
	var out []File
	_ = s.store.Read(func(st *persistence.State) error {
		out = append([]File{}, toFiles(st.EvidenceFile[tenantID])...)
		return nil
	})
	return out
}
func (s *Service) All() []Item {
	out := []Item{}
	_ = s.store.Read(func(st *persistence.State) error {
		for _, arr := range st.Evidence {
			out = append(out, toItems(arr)...)
		}
		return nil
	})
	return out
}

func toPersistItem(i Item) persistence.EvidenceItem { return persistence.EvidenceItem(i) }
func toItems(in []persistence.EvidenceItem) []Item {
	out := make([]Item, len(in))
	for i, v := range in {
		out[i] = Item(v)
	}
	return out
}
func toFiles(in []persistence.EvidenceFile) []File {
	out := make([]File, len(in))
	for i, v := range in {
		out[i] = File(v)
	}
	return out
}
