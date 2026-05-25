package demo

import (
	"context"
	"errors"
	"testing"

	"evidencevault/internal/evidence"
	"evidencevault/internal/persistence"
)

func TestSeedBlockedInProduction(t *testing.T) {
	ev := evidence.NewService(persistence.NewMemoryStore(), 10)
	err := Seed(context.Background(), "production", true, ev, "tenant")
	if !errors.Is(err, ErrDemoSeedBlockedInProduction) {
		t.Fatalf("expected blocked error, got %v", err)
	}
}

func TestSeedCreatesExpectedEvidence(t *testing.T) {
	ev := evidence.NewService(persistence.NewMemoryStore(), 10)
	if err := Seed(context.Background(), "development", true, ev, "tenant"); err != nil {
		t.Fatal(err)
	}
	items, _ := ev.List(context.Background(), "tenant")
	if len(items) != 3 {
		t.Fatalf("expected 3 items got %d", len(items))
	}
}
