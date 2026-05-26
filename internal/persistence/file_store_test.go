package persistence

import (
	"testing"
	"time"
)

func TestFileStoreRoundTrip(t *testing.T) {
	d := t.TempDir()
	fs, err := NewFileStore(d)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	err = fs.Write(func(st *State) error {
		st.StripeEvents["evt_1"] = struct{}{}
		st.Proofpacks["t"] = []ProofpackMeta{{ID: "p1", CreatedAt: now, EvidenceIDs: []string{"e1"}}}
		st.OperationalSnapshots["t"] = []OperationalSnapshot{{TenantID: "t", Date: "2026-05-26", CreatedAt: now, HealthScore: 80}}
		st.ReviewReports["t"] = []ReviewReport{{TenantID: "t", ID: "r1", GeneratedAt: now, Summary: "report"}}
		return nil
	})
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
		if len(st.Proofpacks["t"]) != 1 || len(st.Proofpacks["t"][0].EvidenceIDs) != 1 {
			t.Fatal("missing proofpack evidence manifest")
		}
		if len(st.OperationalSnapshots["t"]) != 1 || len(st.ReviewReports["t"]) != 1 {
			t.Fatal("missing operational history")
		}
		return nil
	})
}
