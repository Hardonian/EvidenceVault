package evidencegraph

import (
	"context"
	"testing"

	"evidencevault/internal/persistence"
)

func TestStoreTenantSource_LoadTenantGraphData(t *testing.T) {
	t.Run("nil store", func(t *testing.T) {
		source := StoreTenantSource{Store: nil}
		data, err := source.LoadTenantGraphData(context.Background(), "tenant-1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(data.Proofpacks) != 0 || len(data.ReviewSnapshots) != 0 || len(data.OperationalSnapshots) != 0 || len(data.OperationalEvents) != 0 || len(data.ReviewReports) != 0 {
			t.Errorf("expected empty data, got %+v", data)
		}
	})

	t.Run("with populated store", func(t *testing.T) {
		store := persistence.NewMemoryStore()
		err := store.Write(func(s *persistence.State) error {
			s.Proofpacks["tenant-1"] = []persistence.ProofpackMeta{{ID: "pp1"}}
			s.Proofpacks["tenant-2"] = []persistence.ProofpackMeta{{ID: "pp2"}}

			s.ReviewSnapshots["tenant-1"] = []persistence.ReviewSnapshot{{TenantID: "tenant-1", HealthScore: 90}}
			s.ReviewSnapshots["tenant-2"] = []persistence.ReviewSnapshot{{TenantID: "tenant-2", HealthScore: 80}}

			s.OperationalSnapshots["tenant-1"] = []persistence.OperationalSnapshot{{TenantID: "tenant-1", Date: "2023-01-01"}}
			s.OperationalSnapshots["tenant-2"] = []persistence.OperationalSnapshot{{TenantID: "tenant-2", Date: "2023-01-02"}}

			s.OperationalEvents["tenant-1"] = []persistence.OperationalEvent{{TenantID: "tenant-1", Type: "event1"}}
			s.OperationalEvents["tenant-2"] = []persistence.OperationalEvent{{TenantID: "tenant-2", Type: "event2"}}

			s.ReviewReports["tenant-1"] = []persistence.ReviewReport{{TenantID: "tenant-1", ID: "rr1"}}
			s.ReviewReports["tenant-2"] = []persistence.ReviewReport{{TenantID: "tenant-2", ID: "rr2"}}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to seed store: %v", err)
		}

		source := StoreTenantSource{Store: store}
		data, err := source.LoadTenantGraphData(context.Background(), "tenant-1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if len(data.Proofpacks) != 1 || data.Proofpacks[0].ID != "pp1" {
			t.Errorf("expected 1 proofpack for tenant-1, got %v", data.Proofpacks)
		}
		if len(data.ReviewSnapshots) != 1 || data.ReviewSnapshots[0].HealthScore != 90 {
			t.Errorf("expected 1 review snapshot for tenant-1, got %v", data.ReviewSnapshots)
		}
		if len(data.OperationalSnapshots) != 1 || data.OperationalSnapshots[0].Date != "2023-01-01" {
			t.Errorf("expected 1 operational snapshot for tenant-1, got %v", data.OperationalSnapshots)
		}
		if len(data.OperationalEvents) != 1 || data.OperationalEvents[0].Type != "event1" {
			t.Errorf("expected 1 operational event for tenant-1, got %v", data.OperationalEvents)
		}
		if len(data.ReviewReports) != 1 || data.ReviewReports[0].ID != "rr1" {
			t.Errorf("expected 1 review report for tenant-1, got %v", data.ReviewReports)
		}
	})

	t.Run("tenant with no data", func(t *testing.T) {
		store := persistence.NewMemoryStore()
		source := StoreTenantSource{Store: store}
		data, err := source.LoadTenantGraphData(context.Background(), "tenant-empty")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if len(data.Proofpacks) != 0 || len(data.ReviewSnapshots) != 0 || len(data.OperationalSnapshots) != 0 || len(data.OperationalEvents) != 0 || len(data.ReviewReports) != 0 {
			t.Errorf("expected empty data, got %+v", data)
		}
	})
}
