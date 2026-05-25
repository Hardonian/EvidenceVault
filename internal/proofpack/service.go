package proofpack

import (
	"context"
	"encoding/json"
	"time"

	"evidencevault/internal/audit"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db    *pgxpool.Pool
	audit *audit.Service
}

func NewService(db *pgxpool.Pool, auditSvc *audit.Service) *Service {
	return &Service{db: db, audit: auditSvc}
}

func (s *Service) List(ctx context.Context, tenantID string) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `select id, created_at from proofpacks where tenant_id=$1 order by created_at desc limit 20`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id string
		var at time.Time
		if err := rows.Scan(&id, &at); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "created_at": at})
	}
	return out, rows.Err()
}

func (s *Service) Export(ctx context.Context, tenantID, version string) ([]byte, error) {
	payload := map[string]any{"generated_at": time.Now().UTC().Format(time.RFC3339), "app_version": version, "tenant": map[string]any{}, "evidence_records": []map[string]any{}, "linked_files": []map[string]any{}, "reminder_log_history": []map[string]any{}, "audit_log_summary": []map[string]any{}, "limitations": "EvidenceVault assists compliance operations and does not certify compliance, provide legal advice, or guarantee filing accuracy."}
	_ = s.db.QueryRow(ctx, `select id,name,plan,created_at from tenants where id=$1`, tenantID).Scan(&payload["tenant"].(map[string]any)["id"], &payload["tenant"].(map[string]any)["name"], &payload["tenant"].(map[string]any)["plan"], &payload["tenant"].(map[string]any)["created_at"])
	if rows, err := s.db.Query(ctx, `select id,title,category,status,owner_name,owner_email,issue_date,expiry_date,source_file_path,updated_at from evidence_items where tenant_id=$1`, tenantID); err == nil {
		defer rows.Close()
		ev := []map[string]any{}
		for rows.Next() {
			m := map[string]any{}
			_ = rows.Scan(&m["id"], &m["title"], &m["category"], &m["status"], &m["owner_name"], &m["owner_email"], &m["issue_date"], &m["expiry_date"], &m["source_file_path"], &m["updated_at"])
			ev = append(ev, m)
		}
		payload["evidence_records"] = ev
	}
	if rows, err := s.db.Query(ctx, `select evidence_id,file_path,content_type,size_bytes,created_at from evidence_files where tenant_id=$1 order by created_at desc`, tenantID); err == nil {
		defer rows.Close()
		fs := []map[string]any{}
		for rows.Next() {
			m := map[string]any{}
			_ = rows.Scan(&m["evidence_id"], &m["file_path"], &m["content_type"], &m["size_bytes"], &m["created_at"])
			fs = append(fs, m)
		}
		payload["linked_files"] = fs
	}
	if rows, err := s.db.Query(ctx, `select evidence_id,reminder_date,channel,status,created_at from reminder_logs where tenant_id=$1 order by created_at desc`, tenantID); err == nil {
		defer rows.Close()
		ls := []map[string]any{}
		for rows.Next() {
			m := map[string]any{}
			_ = rows.Scan(&m["evidence_id"], &m["reminder_date"], &m["channel"], &m["status"], &m["created_at"])
			ls = append(ls, m)
		}
		payload["reminder_log_history"] = ls
	}
	if rows, err := s.db.Query(ctx, `select action,entity_type,entity_id,created_at from audit_logs where tenant_id=$1 order by created_at desc limit 100`, tenantID); err == nil {
		defer rows.Close()
		as := []map[string]any{}
		for rows.Next() {
			m := map[string]any{}
			_ = rows.Scan(&m["action"], &m["entity_type"], &m["entity_id"], &m["created_at"])
			as = append(as, m)
		}
		payload["audit_log_summary"] = as
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	if _, err := s.db.Exec(ctx, `insert into proofpacks (id, tenant_id, payload) values ($1,$2,$3)`, id, tenantID, b); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, tenantID, "", "proofpack.generated", "proofpack", id, `{"size":`+time.Now().UTC().Format("20060102150405")+`}`)
	return b, nil
}
