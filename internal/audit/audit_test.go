package audit

import (
	"context"
	"testing"

	"evidencevault/internal/persistence"
)

func TestService_ListByTenant(t *testing.T) {
	store := persistence.NewMemoryStore()
	svc := NewService(store)
	ctx := context.Background()

	// Log a few times for different tenants
	svc.Log(ctx, "tenant-A", "user-1", "login", "user", "user-1", `{"ip":"127.0.0.1"}`)
	svc.Log(ctx, "tenant-B", "user-2", "upload", "document", "doc-1", `{}`)
	svc.Log(ctx, "tenant-A", "user-1", "logout", "user", "user-1", `{}`)
	svc.Log(ctx, "tenant-C", "user-3", "login", "user", "user-3", `{}`)
	svc.Log(ctx, "tenant-A", "user-4", "view", "dashboard", "dash-1", `{}`)

	// Verify that ListByTenant returns only the targeted tenant's logs

	// Test tenant-A (should have 3 logs, returned in reverse chronological order)
	logsA := svc.ListByTenant("tenant-A", 0)
	if len(logsA) != 3 {
		t.Fatalf("Expected 3 logs for tenant-A, got %d", len(logsA))
	}
	if logsA[0]["action"] != "view" || logsA[1]["action"] != "logout" || logsA[2]["action"] != "login" {
		t.Errorf("Unexpected log order or actions for tenant-A: %v", logsA)
	}

	// Test tenant-B
	logsB := svc.ListByTenant("tenant-B", 0)
	if len(logsB) != 1 {
		t.Fatalf("Expected 1 log for tenant-B, got %d", len(logsB))
	}
	if logsB[0]["action"] != "upload" {
		t.Errorf("Unexpected action for tenant-B log: %v", logsB[0]["action"])
	}

	// Test tenant-D (no logs)
	logsD := svc.ListByTenant("tenant-D", 0)
	if len(logsD) != 0 {
		t.Fatalf("Expected 0 logs for tenant-D, got %d", len(logsD))
	}

	// Test limit
	logsALimited := svc.ListByTenant("tenant-A", 2)
	if len(logsALimited) != 2 {
		t.Fatalf("Expected 2 logs for tenant-A with limit 2, got %d", len(logsALimited))
	}
	if logsALimited[0]["action"] != "view" || logsALimited[1]["action"] != "logout" {
		t.Errorf("Unexpected limited logs for tenant-A: %v", logsALimited)
	}
}
