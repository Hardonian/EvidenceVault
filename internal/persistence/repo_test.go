package persistence

import (
	"testing"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	if store == nil {
		t.Fatal("expected NewMemoryStore to return a non-nil store")
	}

	err := store.WithLock(func(st *State) error {
		st.StripeEvents["evt_1"] = struct{}{}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error from WithLock: %v", err)
	}

	err = store.WithLock(func(st *State) error {
		if _, ok := st.StripeEvents["evt_1"]; !ok {
			t.Fatal("expected 'evt_1' to be present in StripeEvents, but it was missing")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error during verification: %v", err)
	}
}
