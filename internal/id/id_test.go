package id

import (
	"testing"
)

func TestNew(t *testing.T) {
	// Test basic properties
	id1 := New()
	if id1 == "" {
		t.Fatal("expected non-empty string, got empty string")
	}

	if len(id1) != 24 {
		t.Fatalf("expected length of 24, got %d", len(id1))
	}

	// Test uniqueness
	n := 1000
	seen := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		genID := New()
		if seen[genID] {
			t.Fatalf("generated duplicate id: %s", genID)
		}
		seen[genID] = true
	}
}
