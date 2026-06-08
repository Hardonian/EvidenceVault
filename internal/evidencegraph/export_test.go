package evidencegraph

import (
	"strings"
	"testing"
	"time"
)

func TestMarkdown(t *testing.T) {
	// A static time to ensure consistent snapshots
	staticTime := time.Date(2023, 10, 31, 12, 0, 0, 0, time.UTC)

	graph := Graph{
		TenantID:     "tenant-xyz",
		GeneratedAt:  staticTime,
		GraphVersion: "v1",
		Nodes: []Node{
			{ID: "n1", Type: "policy", Status: "active", Label: "Access Policy"},
			{ID: "n2", Type: "evidence", Status: "expired", Label: "Access Logs Q3"},
		},
		Edges: []Edge{
			{ID: "e1", Type: "satisfies", Status: "active", SourceID: "n2", TargetID: "n1", Reason: "direct match"},
		},
		Summary: Summary{
			TotalEvidence:       10,
			TotalOwners:         3,
			OwnerlessEvidence:   1,
			ExpiredEvidence:     2,
			StaleEvidence:       0,
			ProofpackCount:      5,
			ReviewCount:         12,
			LatestReviewAt:      &staticTime,
			ComparisonState:     "in_sync",
			PilotReadinessState: "ready",
			GraphHealthScore:    85,
			TopRisks:            []string{"Risk A: high severity", "Risk B: medium severity"},
			TopNextActions:      []string{"Action X", "Action Y"},
		},
		DegradedReasons: []string{
			"Some data sources unreachable",
		},
		NextActions: []NextAction{
			{
				ID:             "a1",
				Type:           "fix",
				Title:          "Update Expired Evidence",
				Description:    "Refresh Access Logs Q3",
				Severity:       "high",
				Impact:         9,
				SuggestedRoute: "Go to settings",
				ExportImpact:   "Critical",
			},
		},
	}

	expectedMarkdown := `# Evidence Graph

- Generated at: 2023-10-31T12:00:00Z
- Tenant: tenant-xyz
- Graph version: v1

## Health Summary

- Graph health score: 85
- Readiness state: ready
- Evidence count: 10
- Owners: 3
- Ownerless evidence: 1
- Expired evidence: 2
- Stale evidence: 0
- Proofpacks: 5
- Reviews: 12
- Comparison state: in_sync

## Degraded State

- State: degraded
- Summary: Explicit degraded inputs are present and called out below.

## Degraded Reasons

- Some data sources unreachable

## Top Risks

- Risk A: high severity
- Risk B: medium severity

## Next Actions

- [high] Update Expired Evidence (impact 9): Refresh Access Logs Q3 Route: Go to settings. Export impact: Critical

## Node Summary By Type

- evidence: 1
- policy: 1

## Edge Summary By Type

- satisfies: 1

## Critical Evidence Gaps

- [expired] evidence: Access Logs Q3

## Proofpack And Readiness State

- Proofpacks: 5
- Pilot readiness: ready

## Review And Comparison State

- Reviews: 12
- Latest review at: 2023-10-31T12:00:00Z
- Comparison state: in_sync

## Appendix: Nodes

- n1 | policy | active | Access Policy
- n2 | evidence | expired | Access Logs Q3

## Appendix: Edges

- e1 | satisfies | active | n2 -> n1 | direct match

`

	result := Markdown(graph)

	if result != expectedMarkdown {
		t.Errorf("Markdown() output did not match expected:\n\nEXPECTED:\n%s\n\nGOT:\n%s\n", expectedMarkdown, result)
	}

	// Test case for empty degraded state and no next actions, critical gaps
	emptyGraph := Graph{
		TenantID:     "tenant-abc",
		GeneratedAt:  staticTime,
		GraphVersion: "v1",
		Summary: Summary{
			ComparisonState:     "none",
			PilotReadinessState: "not_ready",
		},
	}

	expectedEmptyMarkdown := `# Evidence Graph

- Generated at: 2023-10-31T12:00:00Z
- Tenant: tenant-abc
- Graph version: v1

## Health Summary

- Graph health score: 0
- Readiness state: not_ready
- Evidence count: 0
- Owners: 0
- Ownerless evidence: 0
- Expired evidence: 0
- Stale evidence: 0
- Proofpacks: 0
- Reviews: 0
- Comparison state: none

## Degraded State

- State: healthy
- Summary: No degraded graph inputs detected.

## Degraded Reasons

- No degraded graph inputs detected.

## Top Risks

- No explicit risk references or critical gaps detected.

## Next Actions

- No actions generated.

## Node Summary By Type

- none

## Edge Summary By Type

- none

## Critical Evidence Gaps

- No critical evidence gaps detected in graph nodes.

## Proofpack And Readiness State

- Proofpacks: 0
- Pilot readiness: not_ready

## Review And Comparison State

- Reviews: 0
- Latest review at: not reviewed yet
- Comparison state: none

## Appendix: Nodes


## Appendix: Edges


`

	resultEmpty := Markdown(emptyGraph)
	if resultEmpty != expectedEmptyMarkdown {
		t.Errorf("Markdown() empty graph output did not match expected:\n\nEXPECTED:\n%s\n\nGOT:\n%s\n", expectedEmptyMarkdown, resultEmpty)
	}
}

func TestText(t *testing.T) {
	staticTime := time.Date(2023, 10, 31, 12, 0, 0, 0, time.UTC)

	graph := Graph{
		TenantID:     "tenant-xyz",
		GeneratedAt:  staticTime,
		GraphVersion: "v1",
	}

	// We don't need a perfectly matching string for everything since we know Markdown() works.
	// We just want to check if the substitution works correctly.
	result := Text(graph)

	if strings.Contains(result, "# ") {
		t.Errorf("Text() output contains '# ' which should have been stripped")
	}
	if strings.Contains(result, "## ") {
		t.Errorf("Text() output contains '## ' which should have been stripped")
	}
	if strings.Contains(result, "- ") {
		t.Errorf("Text() output contains '- ' which should have been replaced with '* '")
	}
	if !strings.Contains(result, "* Tenant: tenant-xyz") {
		t.Errorf("Text() output should have substituted list items '* ' instead of '- '")
	}
	if !strings.Contains(result, "Evidence Graph") {
		t.Errorf("Text() output should contain title text without hash")
	}
}
