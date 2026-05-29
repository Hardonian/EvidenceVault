package operations

import (
	"context"
	"fmt"
	"time"

	"evidencevault/internal/id"
	"evidencevault/internal/persistence"
)

type AdjudicationService struct {
	store persistence.Store
}

func NewAdjudicationService(store persistence.Store) *AdjudicationService {
	return &AdjudicationService{store: store}
}

func (s *AdjudicationService) EnsureIssue(ctx context.Context, tenantID, evidenceID, issueType string) (string, error) {
	var issueID string
	err := s.store.Write(func(st *persistence.State) error {
		now := time.Now().UTC()
		arr := st.UnresolvedIssues[tenantID]
		for _, issue := range arr {
			if issue.EvidenceID == evidenceID && issue.Type == issueType && issue.ResolvedAt == nil {
				issueID = issue.ID
				return nil
			}
		}
		issueID = id.New()
		newIssue := persistence.UnresolvedIssue{
			ID:         issueID,
			TenantID:   tenantID,
			EvidenceID: evidenceID,
			Type:       issueType,
			CreatedAt:  now,
		}
		st.UnresolvedIssues[tenantID] = append([]persistence.UnresolvedIssue{newIssue}, arr...)
		return nil
	})
	return issueID, err
}

func (s *AdjudicationService) ResolveIssue(ctx context.Context, tenantID, issueID, operator, reason string) error {
	return s.store.Write(func(st *persistence.State) error {
		arr := st.UnresolvedIssues[tenantID]
		found := false
		now := time.Now().UTC()
		for i, issue := range arr {
			if issue.ID == issueID {
				if issue.ResolvedAt != nil {
					return fmt.Errorf("issue %s is already resolved", issueID)
				}
				arr[i].ResolvedAt = &now
				arr[i].ResolvedReason = reason
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("issue %s not found", issueID)
		}
		st.UnresolvedIssues[tenantID] = arr

		event := persistence.AdjudicationEvent{
			ID:        id.New(),
			TenantID:  tenantID,
			IssueID:   issueID,
			Action:    "resolved",
			Operator:  operator,
			Reason:    reason,
			CreatedAt: now,
		}
		st.AdjudicationEvents[tenantID] = append([]persistence.AdjudicationEvent{event}, st.AdjudicationEvents[tenantID]...)
		return nil
	})
}
