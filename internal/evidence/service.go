package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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

type Service struct {
	db            *pgxpool.Pool
	freeTierLimit int
}

func NewService(db *pgxpool.Pool, freeTierLimit int) *Service {
	return &Service{db: db, freeTierLimit: freeTierLimit}
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

func (s *Service) List(ctx context.Context, tenantID string) ([]Item, error) { /* unchanged */
	rows, err := s.db.Query(ctx, `select id, tenant_id, title, category, status, owner_name, owner_email, source_file_path, notes, reminder_days_before, created_at, updated_at from evidence_items where tenant_id=$1 order by created_at desc`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.TenantID, &it.Title, &it.Category, &it.Status, &it.OwnerName, &it.OwnerEmail, &it.SourceFilePath, &it.Notes, &it.ReminderDaysBefore, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *Service) Create(ctx context.Context, tenantID string, it Item) (string, error) {
	if err := Validate(it); err != nil {
		return "", err
	}
	var count int
	if err := s.db.QueryRow(ctx, `select count(*) from evidence_items where tenant_id=$1`, tenantID).Scan(&count); err != nil {
		return "", err
	}
	if count >= s.freeTierLimit {
		return "", errors.New("free tier limit reached")
	}
	id := uuid.NewString()
	_, err := s.db.Exec(ctx, `insert into evidence_items (id, tenant_id, title, category, status, owner_name, owner_email, issue_date, expiry_date, reminder_days_before, source_file_path, notes) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, id, tenantID, it.Title, it.Category, it.Status, it.OwnerName, it.OwnerEmail, it.IssueDate, it.ExpiryDate, it.ReminderDaysBefore, it.SourceFilePath, it.Notes)
	return id, err
}
