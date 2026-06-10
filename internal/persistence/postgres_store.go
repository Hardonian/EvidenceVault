package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type PostgresStore struct {
	db     *sqlx.DB
	mu     sync.RWMutex
	closed bool
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &PostgresStore{db: db}, nil
}

func (p *PostgresStore) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	return p.db.Close()
}

func (p *PostgresStore) Read(fn func(*State) error) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return fmt.Errorf("store closed")
	}

	tx, err := p.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("begin read tx: %w", err)
	}
	defer tx.Rollback()

	state, err := p.loadState(tx)
	if err != nil {
		return err
	}
	return fn(state)
}

func (p *PostgresStore) Write(fn func(*State) error) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return fmt.Errorf("store closed")
	}

	tx, err := p.db.BeginTxx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin write tx: %w", err)
	}

	state, err := p.loadState(tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := fn(state); err != nil {
		tx.Rollback()
		return err
	}

	return p.persistState(tx, state)
}

func (p *PostgresStore) loadState(tx *sqlx.Tx) (*State, error) {
	state := emptyState()

	rows, err := tx.Queryx("SELECT tenant_id, data FROM evidence_items")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var tenantID string
		var data []byte
		if err := rows.Scan(&tenantID, &data); err != nil {
			rows.Close()
			return nil, err
		}
		var items []EvidenceItem
		if err := json.Unmarshal(data, &items); err != nil {
			rows.Close()
			return nil, err
		}
		state.Evidence[tenantID] = items
	}
	rows.Close()

	rows, err = tx.Queryx("SELECT tenant_id, data FROM evidence_files")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var tenantID string
		var data []byte
		if err := rows.Scan(&tenantID, &data); err != nil {
			rows.Close()
			return nil, err
		}
		var files []EvidenceFile
		if err := json.Unmarshal(data, &files); err != nil {
			rows.Close()
			return nil, err
		}
		state.EvidenceFile[tenantID] = files
	}
	rows.Close()

	if err := p.loadSimpleTable(tx, "audit_logs", &state.AuditLogs); err != nil {
		return nil, err
	}
	if err := p.loadMapTable(tx, "proofpacks", &state.Proofpacks); err != nil {
		return nil, err
	}
	if err := p.loadMapTable(tx, "review_snapshots", &state.ReviewSnapshots); err != nil {
		return nil, err
	}
	if err := p.loadMapTable(tx, "operational_events", &state.OperationalEvents); err != nil {
		return nil, err
	}
	if err := p.loadMapTable(tx, "activation_milestones", &state.Activation); err != nil {
		return nil, err
	}
	if err := p.loadMapTable(tx, "operational_snapshots", &state.OperationalSnapshots); err != nil {
		return nil, err
	}
	if err := p.loadMapTable(tx, "review_reports", &state.ReviewReports); err != nil {
		return nil, err
	}
	if err := p.loadMapTable(tx, "unresolved_issues", &state.UnresolvedIssues); err != nil {
		return nil, err
	}
	if err := p.loadMapTable(tx, "adjudication_events", &state.AdjudicationEvents); err != nil {
		return nil, err
	}
	if err := p.loadTenants(tx); err != nil {
		return nil, err
	}
	if err := p.loadStripeEvents(tx, &state.StripeEvents); err != nil {
		return nil, err
	}
	if err := p.loadReminderSent(tx, &state.ReminderSent); err != nil {
		return nil, err
	}

	return &state, nil
}

func (p *PostgresStore) loadSimpleTable(tx *sqlx.Tx, table string, target any) error {
	rows, err := tx.Queryx("SELECT data FROM " + table)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return nil
		}
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return err
		}
		var item map[string]any
		if err := json.Unmarshal(data, &item); err != nil {
			return err
		}
		b, _ := json.Marshal(item)
		json.Unmarshal(b, target)
	}
	return nil
}

func (p *PostgresStore) loadMapTable(tx *sqlx.Tx, table string, target any) error {
	rows, err := tx.Queryx("SELECT tenant_id, data FROM " + table)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return nil
		}
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var tenantID string
		var data []byte
		if err := rows.Scan(&tenantID, &data); err != nil {
			return err
		}
		var items []map[string]any
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		b, _ := json.Marshal(map[string]any{tenantID: items})
		json.Unmarshal(b, target)
	}
	return nil
}

func (p *PostgresStore) loadTenants(tx *sqlx.Tx) error {
	rows, err := tx.Queryx("SELECT id, data FROM tenants")
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return nil
		}
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return err
		}
		var tenant map[string]string
		if err := json.Unmarshal(data, &tenant); err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresStore) loadStripeEvents(tx *sqlx.Tx, target *map[string]struct{}) error {
	rows, err := tx.Queryx("SELECT event_id FROM stripe_events")
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return nil
		}
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var eventID string
		if err := rows.Scan(&eventID); err != nil {
			return err
		}
		(*target)[eventID] = struct{}{}
	}
	return nil
}

func (p *PostgresStore) loadReminderSent(tx *sqlx.Tx, target *map[string]struct{}) error {
	rows, err := tx.Queryx("SELECT key FROM reminder_sent")
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return nil
		}
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return err
		}
		(*target)[key] = struct{}{}
	}
	return nil
}

func (p *PostgresStore) persistState(tx *sqlx.Tx, state *State) error {
	if err := p.upsertJSON(tx, "evidence_items", "tenant_id", state.Evidence); err != nil {
		return err
	}
	if err := p.upsertJSON(tx, "evidence_files", "tenant_id", state.EvidenceFile); err != nil {
		return err
	}
	if err := p.upsertSimple(tx, "audit_logs", state.AuditLogs); err != nil {
		return err
	}
	if err := p.upsertMap(tx, "proofpacks", state.Proofpacks); err != nil {
		return err
	}
	if err := p.upsertMap(tx, "review_snapshots", state.ReviewSnapshots); err != nil {
		return err
	}
	if err := p.upsertMap(tx, "operational_events", state.OperationalEvents); err != nil {
		return err
	}
	if err := p.upsertMap(tx, "activation_milestones", state.Activation); err != nil {
		return err
	}
	if err := p.upsertMap(tx, "operational_snapshots", state.OperationalSnapshots); err != nil {
		return err
	}
	if err := p.upsertMap(tx, "review_reports", state.ReviewReports); err != nil {
		return err
	}
	if err := p.upsertMap(tx, "unresolved_issues", state.UnresolvedIssues); err != nil {
		return err
	}
	if err := p.upsertMap(tx, "adjudication_events", state.AdjudicationEvents); err != nil {
		return err
	}
	if err := p.upsertTenants(tx, state.Tenants); err != nil {
		return err
	}
	if err := p.upsertStripeEvents(tx, state.StripeEvents); err != nil {
		return err
	}
	if err := p.upsertReminderSent(tx, state.ReminderSent); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *PostgresStore) upsertJSON(tx *sqlx.Tx, table, key string, data map[string][]any) error {
	for tenantID, items := range data {
		b, err := json.Marshal(items)
		if err != nil {
			return err
		}
		_, err = tx.Exec(
			fmt.Sprintf("INSERT INTO %s (%s, data) VALUES ($1, $2) ON CONFLICT (%s) DO UPDATE SET data = $2, updated_at = NOW()", table, key, key),
			tenantID, b,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresStore) upsertSimple(tx *sqlx.Tx, table string, items []AuditEntry) error {
	for _, item := range items {
		b, err := json.Marshal(item)
		if err != nil {
			return err
		}
		_, err = tx.Exec(
			"INSERT INTO "+table+" (data, created_at) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			b, item.CreatedAt,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresStore) upsertMap(tx *sqlx.Tx, table string, data map[string][]map[string]any) error {
	for tenantID, items := range data {
		b, err := json.Marshal(items)
		if err != nil {
			return err
		}
		_, err = tx.Exec(
			fmt.Sprintf("INSERT INTO %s (tenant_id, data) VALUES ($1, $2) ON CONFLICT (tenant_id) DO UPDATE SET data = $2, updated_at = NOW()", table),
			tenantID, b,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresStore) upsertTenants(tx *sqlx.Tx, tenants map[string]string) error {
	for id, data := range tenants {
		b, _ := json.Marshal(map[string]string{"id": id, "name": data})
		_, err := tx.Exec(
			"INSERT INTO tenants (id, data) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET data = $2, updated_at = NOW()",
			id, b,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresStore) upsertStripeEvents(tx *sqlx.Tx, events map[string]struct{}) error {
	for eventID := range events {
		_, err := tx.Exec("INSERT INTO stripe_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING", eventID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresStore) upsertReminderSent(tx *sqlx.Tx, reminders map[string]struct{}) error {
	for key := range reminders {
		_, err := tx.Exec("INSERT INTO reminder_sent (key) VALUES ($1) ON CONFLICT DO NOTHING", key)
		if err != nil {
			return err
		}
	}
	return nil
}