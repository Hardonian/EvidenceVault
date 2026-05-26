package evidencegraph

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func Markdown(g Graph) string {
	var b strings.Builder
	b.WriteString("# Evidence Graph\n\n")
	b.WriteString("- Generated at: " + g.GeneratedAt.Format(time.RFC3339) + "\n")
	b.WriteString("- Tenant: " + g.TenantID + "\n")
	b.WriteString("- Graph version: " + g.GraphVersion + "\n\n")
	writeSummary(&b, g)
	writeList(&b, "Degraded Reasons", g.DegradedReasons, "No degraded graph inputs detected.")
	writeList(&b, "Top Risks", g.Summary.TopRisks, "No explicit risk references or critical gaps detected.")
	writeActions(&b, g.NextActions)
	writeCounts(&b, "Node Summary By Type", nodeCounts(g.Nodes))
	writeCounts(&b, "Edge Summary By Type", edgeCounts(g.Edges))
	writeCriticalGaps(&b, g)
	b.WriteString("## Proofpack And Readiness State\n\n")
	b.WriteString("- Proofpacks: " + itoa(g.Summary.ProofpackCount) + "\n")
	b.WriteString("- Pilot readiness: " + g.Summary.PilotReadinessState + "\n\n")
	b.WriteString("## Review And Comparison State\n\n")
	b.WriteString("- Reviews: " + itoa(g.Summary.ReviewCount) + "\n")
	if g.Summary.LatestReviewAt != nil {
		b.WriteString("- Latest review at: " + g.Summary.LatestReviewAt.Format(time.RFC3339) + "\n")
	} else {
		b.WriteString("- Latest review at: not reviewed yet\n")
	}
	b.WriteString("- Comparison state: " + g.Summary.ComparisonState + "\n\n")
	b.WriteString("## Appendix: Nodes\n\n")
	for _, n := range g.Nodes {
		b.WriteString("- " + n.ID + " | " + n.Type + " | " + n.Status + " | " + n.Label + "\n")
	}
	b.WriteString("\n## Appendix: Edges\n\n")
	for _, e := range g.Edges {
		b.WriteString("- " + e.ID + " | " + e.Type + " | " + e.Status + " | " + e.SourceID + " -> " + e.TargetID + " | " + e.Reason + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

func Text(g Graph) string {
	out := Markdown(g)
	out = strings.ReplaceAll(out, "# ", "")
	out = strings.ReplaceAll(out, "## ", "")
	out = strings.ReplaceAll(out, "- ", "* ")
	return out
}

func writeSummary(b *strings.Builder, g Graph) {
	b.WriteString("## Health Summary\n\n")
	b.WriteString("- Graph health score: " + itoa(g.Summary.GraphHealthScore) + "\n")
	b.WriteString("- Readiness state: " + g.Summary.PilotReadinessState + "\n")
	b.WriteString("- Evidence count: " + itoa(g.Summary.TotalEvidence) + "\n")
	b.WriteString("- Owners: " + itoa(g.Summary.TotalOwners) + "\n")
	b.WriteString("- Ownerless evidence: " + itoa(g.Summary.OwnerlessEvidence) + "\n")
	b.WriteString("- Expired evidence: " + itoa(g.Summary.ExpiredEvidence) + "\n")
	b.WriteString("- Stale evidence: " + itoa(g.Summary.StaleEvidence) + "\n")
	b.WriteString("- Proofpacks: " + itoa(g.Summary.ProofpackCount) + "\n")
	b.WriteString("- Reviews: " + itoa(g.Summary.ReviewCount) + "\n")
	b.WriteString("- Comparison state: " + g.Summary.ComparisonState + "\n\n")
}

func writeList(b *strings.Builder, title string, values []string, empty string) {
	b.WriteString("## " + title + "\n\n")
	if len(values) == 0 {
		b.WriteString("- " + empty + "\n\n")
		return
	}
	for _, value := range values {
		b.WriteString("- " + value + "\n")
	}
	b.WriteString("\n")
}

func writeActions(b *strings.Builder, actions []NextAction) {
	b.WriteString("## Next Actions\n\n")
	if len(actions) == 0 {
		b.WriteString("- No actions generated.\n\n")
		return
	}
	for _, a := range actions {
		b.WriteString("- [" + a.Severity + "] " + a.Title + " (impact " + itoa(a.Impact) + "): " + a.Description)
		if a.SuggestedRoute != "" {
			b.WriteString(" Route: " + a.SuggestedRoute + ".")
		}
		if a.ExportImpact != "" {
			b.WriteString(" Export impact: " + a.ExportImpact)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func writeCounts(b *strings.Builder, title string, counts map[string]int) {
	b.WriteString("## " + title + "\n\n")
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		b.WriteString("- none\n\n")
		return
	}
	for _, k := range keys {
		b.WriteString("- " + k + ": " + itoa(counts[k]) + "\n")
	}
	b.WriteString("\n")
}

func writeCriticalGaps(b *strings.Builder, g Graph) {
	b.WriteString("## Critical Evidence Gaps\n\n")
	found := false
	for _, n := range g.Nodes {
		switch n.Status {
		case "expired", "ownerless", "stale", "missing", "unresolved", "not_comparable", "degraded":
			b.WriteString("- [" + n.Status + "] " + n.Type + ": " + n.Label + "\n")
			found = true
		}
	}
	if !found {
		b.WriteString("- No critical evidence gaps detected in graph nodes.\n")
	}
	b.WriteString("\n")
}

func nodeCounts(nodes []Node) map[string]int {
	out := map[string]int{}
	for _, n := range nodes {
		out[n.Type]++
	}
	return out
}

func edgeCounts(edges []Edge) map[string]int {
	out := map[string]int{}
	for _, e := range edges {
		out[e.Type]++
	}
	return out
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
