package demo

import (
	"context"
	"errors"
	"testing"

	"evidencevault/internal/audit"
	"evidencevault/internal/evidence"
	"evidencevault/internal/operations"
	"evidencevault/internal/persistence"
	"evidencevault/internal/proofpack"
)

func TestSeedBlockedInProduction(t *testing.T) {
	st := persistence.NewMemoryStore()
	ev := evidence.NewService(st, 10)
	err := Seed(context.Background(), "production", true, ev, nil, nil, "tenant")
	if !errors.Is(err, ErrDemoSeedBlockedInProduction) {
		t.Fatalf("expected blocked error, got %v", err)
	}
}

func TestSeedCreatesExpectedEvidence(t *testing.T) {
	st := persistence.NewMemoryStore()
	ev := evidence.NewService(st, 10)
	ops := operations.NewService(st, ev)
	pp := proofpack.NewService(st, audit.NewService(st), ev)
	if err := Seed(context.Background(), "development", true, ev, ops, pp, "tenant"); err != nil {
		t.Fatal(err)
	}
	items, _ := ev.List(context.Background(), "tenant")
	if len(items) < 4 {
		t.Fatalf("expected richer seed")
	}
	sum, _ := ops.BuildSummary(context.Background(), "tenant", nil)
	if len(sum.RecentActivity) == 0 || len(sum.ProofpackHistory) == 0 {
		t.Fatal("expected demo activity/history")
	}
}
