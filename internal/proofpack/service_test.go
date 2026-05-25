package proofpack_test

import (
	"context"
	"encoding/json"
	"testing"

	"evidencevault/internal/audit"
	"evidencevault/internal/evidence"
	"evidencevault/internal/persistence"
	"evidencevault/internal/proofpack"
)

func TestExport(t *testing.T) {
	store := persistence.NewMemoryStore()
	auditSvc := audit.NewService(store)
	evSvc := evidence.NewService(store, 10)
	svc := proofpack.NewService(store, auditSvc, evSvc)

	tenantID := "tenant-123"
	version := "1.0.0"

	// Create some evidence
	evID, err := evSvc.Create(context.Background(), tenantID, evidence.Item{
		Title:    "Test Evidence",
		Category: "Security",
		Status:   "active",
	})
	if err != nil {
		t.Fatalf("failed to create evidence: %v", err)
	}

	// Add an audit log manually or let Export do it later
	auditSvc.Log(context.Background(), tenantID, "user1", "test.action", "test", "123", "{}")

	// Call Export
	b, err := svc.Export(context.Background(), tenantID, version)
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Check payload structure
	if payload["app_version"] != version {
		t.Errorf("expected app_version %q, got %v", version, payload["app_version"])
	}

	tenantMap, ok := payload["tenant"].(map[string]any)
	if !ok || tenantMap["id"] != tenantID {
		t.Errorf("expected tenant id %q", tenantID)
	}

	records, ok := payload["evidence_records"].([]any)
	if !ok || len(records) != 1 {
		t.Errorf("expected 1 evidence record, got %v", payload["evidence_records"])
	} else {
		recordMap := records[0].(map[string]any)
		if recordMap["Title"] != "Test Evidence" {
			t.Errorf("expected evidence title 'Test Evidence', got %v", recordMap["Title"])
		}
		if recordMap["ID"] != evID {
			t.Errorf("expected evidence ID %q, got %v", evID, recordMap["ID"])
		}
	}

	auditLog, ok := payload["audit_log_summary"].([]any)
	if !ok || len(auditLog) != 1 {
		t.Errorf("expected 1 audit log (pre-export), got %v", payload["audit_log_summary"])
	} else {
		auditLogMap := auditLog[0].(map[string]any)
		if auditLogMap["action"] != "test.action" {
			t.Errorf("expected audit log action 'test.action', got %v", auditLogMap["action"])
		}
	}

	// Check that proofpack meta was saved
	packs, err := svc.List(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("failed to list proofpacks: %v", err)
	}

	if len(packs) != 1 {
		t.Errorf("expected 1 proofpack in store, got %d", len(packs))
	}
}
