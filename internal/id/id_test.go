package id

import (
	"testing"
)

func TestNew(t *testing.T) {
	// Verify it returns a non-empty string and length is 24
	id1 := New()
	if id1 == "" {
		t.Error("expected non-empty string, got empty string")
	}
	if len(id1) != 24 {
		t.Errorf("expected string of length 24, got length %d: %q", len(id1), id1)
	}

	// Verify multiple calls return unique values
	id2 := New()
	if id1 == id2 {
		t.Errorf("expected unique IDs, got identical values: %q", id1)
	}
}
