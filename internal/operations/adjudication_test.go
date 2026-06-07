package operations

import (
	"context"
	"testing"

	"evidencevault/internal/persistence"
)

func TestEnsureIssue(t *testing.T) {
	st := persistence.NewMemoryStore()
	svc := NewAdjudicationService(st)
	ctx := context.Background()

	tenantID := "t1"
	evidenceID := "e1"
	issueType := "missing_owner"

	// Case 1: Create a new issue when none exists
	issueID1, err := svc.EnsureIssue(ctx, tenantID, evidenceID, issueType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issueID1 == "" {
		t.Fatal("expected a non-empty issue ID")
	}

	// Verify the issue is stored and unresolved
	err = st.Read(func(state *persistence.State) error {
		arr := state.UnresolvedIssues[tenantID]
		if len(arr) != 1 {
			t.Fatalf("expected 1 unresolved issue, got %d", len(arr))
		}
		if arr[0].ID != issueID1 {
			t.Errorf("expected issue ID %s, got %s", issueID1, arr[0].ID)
		}
		if arr[0].ResolvedAt != nil {
			t.Error("expected issue to be unresolved")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Case 2: Return existing issue ID when an unresolved issue already exists
	issueID2, err := svc.EnsureIssue(ctx, tenantID, evidenceID, issueType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issueID2 != issueID1 {
		t.Errorf("expected to get the same issue ID %s, but got %s", issueID1, issueID2)
	}

	err = st.Read(func(state *persistence.State) error {
		arr := state.UnresolvedIssues[tenantID]
		if len(arr) != 1 {
			t.Fatalf("expected still 1 unresolved issue, got %d", len(arr))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Case 3: Create a new issue when the previous issue is already resolved
	err = svc.ResolveIssue(ctx, tenantID, issueID1, "admin", "found owner")
	if err != nil {
		t.Fatalf("failed to resolve issue: %v", err)
	}

	issueID3, err := svc.EnsureIssue(ctx, tenantID, evidenceID, issueType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issueID3 == "" {
		t.Fatal("expected a non-empty issue ID")
	}
	if issueID3 == issueID1 {
		t.Error("expected a new issue ID, but got the same as resolved one")
	}

	// Verify the new issue is stored and unresolved
	err = st.Read(func(state *persistence.State) error {
		arr := state.UnresolvedIssues[tenantID]
		if len(arr) != 2 {
			t.Fatalf("expected 2 issues in history, got %d", len(arr))
		}

		foundNew := false
		for _, issue := range arr {
			if issue.ID == issueID3 {
				foundNew = true
				if issue.ResolvedAt != nil {
					t.Error("expected new issue to be unresolved")
				}
			}
		}
		if !foundNew {
			t.Errorf("new issue %s not found in state", issueID3)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
