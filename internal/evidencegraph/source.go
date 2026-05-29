package evidencegraph

import (
	"context"

	"evidencevault/internal/persistence"
)

type TenantData struct {
	Proofpacks           []persistence.ProofpackMeta
	ReviewSnapshots      []persistence.ReviewSnapshot
	OperationalSnapshots []persistence.OperationalSnapshot
	OperationalEvents    []persistence.OperationalEvent
	ReviewReports        []persistence.ReviewReport
}

type TenantSource interface {
	LoadTenantGraphData(ctx context.Context, tenantID string) (TenantData, error)
}

type StoreTenantSource struct {
	Store persistence.Store
}

func (s StoreTenantSource) LoadTenantGraphData(_ context.Context, tenantID string) (TenantData, error) {
	data := TenantData{}
	if s.Store == nil {
		return data, nil
	}
	err := s.Store.Read(func(st *persistence.State) error {
		data.Proofpacks = append([]persistence.ProofpackMeta{}, st.Proofpacks[tenantID]...)
		data.ReviewSnapshots = append([]persistence.ReviewSnapshot{}, st.ReviewSnapshots[tenantID]...)
		data.OperationalSnapshots = append([]persistence.OperationalSnapshot{}, st.OperationalSnapshots[tenantID]...)
		data.OperationalEvents = append([]persistence.OperationalEvent{}, st.OperationalEvents[tenantID]...)
		data.ReviewReports = append([]persistence.ReviewReport{}, st.ReviewReports[tenantID]...)
		return nil
	})
	return data, err
}
