package persistence

import "testing"

func TestFileStoreRoundTrip(t *testing.T) {
	d := t.TempDir()
	fs, err := NewFileStore(d)
	if err != nil {
		t.Fatal(err)
	}
	err = fs.Write(func(st *State) error { st.StripeEvents["evt_1"] = struct{}{}; return nil })
	if err != nil {
		t.Fatal(err)
	}
	fs2, err := NewFileStore(d)
	if err != nil {
		t.Fatal(err)
	}
	_ = fs2.Read(func(st *State) error {
		if _, ok := st.StripeEvents["evt_1"]; !ok {
			t.Fatal("missing persisted event")
		}
		return nil
	})
}
