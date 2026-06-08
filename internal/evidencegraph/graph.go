package evidencegraph

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"evidencevault/internal/evidence"
	"evidencevault/internal/operations"
	"evidencevault/internal/persistence"
)

const Version = "evidence-graph/v1"

const (
	DefaultMaxNodes = 500
	DefaultMaxEdges = 1000
)

var ErrTenantRequired = errors.New("tenant id is required")

type Graph struct {
	TenantID        string       `json:"tenantId"`
	GeneratedAt     time.Time    `json:"generatedAt"`
	GraphVersion    string       `json:"graphVersion"`
	Nodes           []Node       `json:"nodes"`
	Edges           []Edge       `json:"edges"`
	Summary         Summary      `json:"summary"`
	DegradedReasons []string     `json:"degradedReasons"`
	NextActions     []NextAction `json:"nextActions"`
}

type Node struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Label     string         `json:"label"`
	TenantID  string         `json:"tenantId"`
	Status    string         `json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Summary   string         `json:"summary"`
	Metadata  map[string]any `json:"metadata"`
}

type Edge struct {
	ID             string         `json:"id"`
	Type           string         `json:"type"`
	SourceID       string         `json:"sourceId"`
	TargetID       string         `json:"targetId"`
	TenantID       string         `json:"tenantId"`
	Reason         string         `json:"reason"`
	EvidenceSource string         `json:"evidenceSource"`
	Confidence     string         `json:"confidence"`
	Status         string         `json:"status"`
	CreatedAt      time.Time      `json:"createdAt"`
	Metadata       map[string]any `json:"metadata"`
}

type Summary struct {
	TotalEvidence       int        `json:"totalEvidence"`
	TotalOwners         int        `json:"totalOwners"`
	OwnerlessEvidence   int        `json:"ownerlessEvidence"`
	ExpiredEvidence     int        `json:"expiredEvidence"`
	StaleEvidence       int        `json:"staleEvidence"`
	ProofpackCount      int        `json:"proofpackCount"`
	ReviewCount         int        `json:"reviewCount"`
	LatestReviewAt      *time.Time `json:"latestReviewAt,omitempty"`
	ComparisonState     string     `json:"comparisonState"`
	PilotReadinessState string     `json:"pilotReadinessState"`
	GraphHealthScore    int        `json:"graphHealthScore"`
	TopRisks            []string   `json:"topRisks"`
	TopNextActions      []string   `json:"topNextActions"`
}

type NextAction struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Severity       string   `json:"severity"`
	Impact         int      `json:"impact"`
	TargetNodeIDs  []string `json:"targetNodeIds"`
	Reason         string   `json:"reason"`
	SuggestedRoute string   `json:"suggestedRoute,omitempty"`
	ExportImpact   string   `json:"exportImpact,omitempty"`
}

type Builder struct {
	evidence   *evidence.Service
	operations *operations.Service
	source     TenantSource
	MaxNodes   int
	MaxEdges   int
	Now        func() time.Time
}

func NewBuilder(store persistence.Store, ev *evidence.Service, ops *operations.Service) *Builder {
	return &Builder{
		evidence:   ev,
		operations: ops,
		source:     StoreTenantSource{Store: store},
		MaxNodes:   DefaultMaxNodes,
		MaxEdges:   DefaultMaxEdges,
		Now:        func() time.Time { return time.Now().UTC() },
	}
}

func (b *Builder) Build(ctx context.Context, tenantID string) (Graph, error) {
	if strings.TrimSpace(tenantID) == "" {
		return Graph{}, ErrTenantRequired
	}
	now := b.now()
	items := []evidence.Item{}
	if b.evidence != nil {
		var err error
		items, err = b.evidence.List(ctx, tenantID)
		if err != nil {
			return Graph{}, err
		}
	}
	data, err := b.source.LoadTenantGraphData(ctx, tenantID)
	if err != nil {
		return Graph{}, err
	}
	sortTenantData(&data)
	w := newWork(tenantID, now)
	w.node(Node{ID: tenantNodeID(tenantID), Type: "tenant", Label: tenantID, TenantID: tenantID, Status: "active", CreatedAt: now, UpdatedAt: now, Summary: "Tenant-scoped graph root.", Metadata: map[string]any{"graph_version": Version}})
	buildEvidence(w, tenantID, now, items)
	buildProofpacks(w, tenantID, data, items)
	buildHistory(ctx, w, tenantID, now, data, b.operations)
	actions := nextActions(tenantID, now, items, data)
	for _, a := range actions {
		w.node(Node{ID: actionNodeID(a.ID), Type: "action", Label: a.Title, TenantID: tenantID, Status: a.Severity, CreatedAt: now, UpdatedAt: now, Summary: a.Description, Metadata: map[string]any{"action_type": a.Type, "impact": a.Impact, "route": a.SuggestedRoute}})
		for _, target := range a.TargetNodeIDs {
			w.edge(Edge{Type: "REQUIRES_ACTION", SourceID: target, TargetID: actionNodeID(a.ID), TenantID: tenantID, Reason: a.Reason, EvidenceSource: "evidence_graph.next_actions", Confidence: "derived", Status: edgeStatus(a.Severity), CreatedAt: now})
		}
	}
	out := w.graph()
	out.NextActions = actions
	out.Summary = summary(tenantID, now, items, data, actions)
	out.DegradedReasons = degradedReasons(items, data, out.Summary)
	sortGraph(&out)
	out = applyCaps(out, b.maxNodes(), b.maxEdges(), now)
	return out, nil
}

func (b *Builder) now() time.Time {
	if b.Now != nil {
		return b.Now().UTC()
	}
	return time.Now().UTC()
}

func (b *Builder) maxNodes() int {
	if b.MaxNodes <= 0 {
		return DefaultMaxNodes
	}
	return b.MaxNodes
}

func (b *Builder) maxEdges() int {
	if b.MaxEdges <= 0 {
		return DefaultMaxEdges
	}
	return b.MaxEdges
}

type work struct {
	tenantID string
	now      time.Time
	nodes    map[string]Node
	edges    map[string]Edge
}

func newWork(tenantID string, now time.Time) *work {
	return &work{tenantID: tenantID, now: now, nodes: map[string]Node{}, edges: map[string]Edge{}}
}

func (w *work) node(n Node) {
	if n.Metadata == nil {
		n.Metadata = map[string]any{}
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = w.now
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = n.CreatedAt
	}
	w.nodes[n.ID] = n
}

func (w *work) edge(e Edge) {
	if e.Metadata == nil {
		e.Metadata = map[string]any{}
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = w.now
	}
	e.ID = edgeID(e.Type, e.SourceID, e.TargetID)
	w.edges[e.ID] = e
}

func (w *work) graph() Graph {
	g := Graph{TenantID: w.tenantID, GeneratedAt: w.now, GraphVersion: Version}
	for _, n := range w.nodes {
		g.Nodes = append(g.Nodes, n)
	}
	for _, e := range w.edges {
		g.Edges = append(g.Edges, e)
	}
	return g
}

func buildEvidence(w *work, tenantID string, now time.Time, items []evidence.Item) {
	for _, it := range items {
		eid := evidenceNodeID(it.ID)
		status := evidenceStatus(it, now)
		w.node(Node{ID: eid, Type: "evidence", Label: it.Title, TenantID: tenantID, Status: status, CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt, Summary: it.Category + " evidence.", Metadata: map[string]any{"raw_evidence_id": it.ID, "category": it.Category, "has_source_file": it.SourceFilePath != "", "controls": it.ControlRefs, "vendors": it.VendorRefs, "risks": it.RiskRefs}})
		if oid, label, ok := ownerNode(it); ok {
			w.node(Node{ID: oid, Type: "owner", Label: label, TenantID: tenantID, Status: "active", CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt, Summary: "Owner explicitly assigned on evidence record.", Metadata: map[string]any{"owner_name": it.OwnerName, "owner_email": it.OwnerEmail}})
			w.edge(Edge{Type: "OWNS", SourceID: oid, TargetID: eid, TenantID: tenantID, Reason: "Evidence record has an explicit owner.", EvidenceSource: "evidence_items.owner_name/owner_email", Confidence: "explicit", Status: "active", CreatedAt: it.UpdatedAt})
		}
		for _, ref := range it.ControlRefs {
			id := typedNodeID("control", ref)
			w.node(Node{ID: id, Type: "control", Label: ref, TenantID: tenantID, Status: "active", CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt, Summary: "Explicit evidence control mapping.", Metadata: map[string]any{"source_evidence_id": it.ID}})
			w.edge(Edge{Type: "SUPPORTS_CONTROL", SourceID: eid, TargetID: id, TenantID: tenantID, Reason: "Evidence record lists this control reference.", EvidenceSource: "evidence_items.control_refs", Confidence: "explicit", Status: "active", CreatedAt: it.UpdatedAt})
		}
		for _, ref := range it.VendorRefs {
			id := typedNodeID("vendor", ref)
			w.node(Node{ID: id, Type: "vendor", Label: ref, TenantID: tenantID, Status: "active", CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt, Summary: "Explicit evidence vendor mapping.", Metadata: map[string]any{"source_evidence_id": it.ID}})
			w.edge(Edge{Type: "RELATES_TO_VENDOR", SourceID: eid, TargetID: id, TenantID: tenantID, Reason: "Evidence record lists this vendor reference.", EvidenceSource: "evidence_items.vendor_refs", Confidence: "explicit", Status: "active", CreatedAt: it.UpdatedAt})
		}
		for _, ref := range it.RiskRefs {
			id := typedNodeID("risk", ref)
			w.node(Node{ID: id, Type: "risk", Label: ref, TenantID: tenantID, Status: riskStatus(status), CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt, Summary: "Explicit evidence risk mapping.", Metadata: map[string]any{"source_evidence_id": it.ID}})
			w.edge(Edge{Type: "MITIGATES_RISK", SourceID: eid, TargetID: id, TenantID: tenantID, Reason: "Evidence record lists this risk reference.", EvidenceSource: "evidence_items.risk_refs", Confidence: "explicit", Status: edgeStatus(status), CreatedAt: it.UpdatedAt})
		}
		if it.Status == "expired" {
			w.edge(Edge{Type: "EXPIRED_BECAUSE", SourceID: eid, TargetID: actionNodeID(actionID("replace_expired_evidence", it.ID)), TenantID: tenantID, Reason: "Evidence status is expired.", EvidenceSource: "evidence_items.status", Confidence: "explicit", Status: "expired", CreatedAt: now})
		}
		if it.IsOwnerless() {
			w.edge(Edge{Type: "OWNERLESS_BECAUSE", SourceID: eid, TargetID: actionNodeID(actionID("assign_owner", it.ID)), TenantID: tenantID, Reason: "Evidence lacks owner fields.", EvidenceSource: "evidence_items.owner_name/owner_email", Confidence: "explicit", Status: "missing", CreatedAt: now})
		}
		if it.IsStale(now) {
			w.edge(Edge{Type: "STALE_BECAUSE", SourceID: eid, TargetID: actionNodeID(actionID("refresh_stale_evidence", it.ID)), TenantID: tenantID, Reason: "Evidence updated_at is older than 180 days.", EvidenceSource: "evidence_items.updated_at", Confidence: "derived", Status: "stale", CreatedAt: now})
		}
	}
}

func buildProofpacks(w *work, tenantID string, data TenantData, items []evidence.Item) {
	known := map[string]struct{}{}
	for _, it := range items {
		known[it.ID] = struct{}{}
	}
	for _, p := range data.Proofpacks {
		pid := proofpackNodeID(p.ID)
		status := "active"
		if len(p.EvidenceIDs) == 0 {
			status = "degraded"
		}
		w.node(Node{ID: pid, Type: "proofpack", Label: p.ID, TenantID: tenantID, Status: status, CreatedAt: p.CreatedAt, UpdatedAt: p.CreatedAt, Summary: "Persisted proofpack export metadata.", Metadata: map[string]any{"evidence_count": len(p.EvidenceIDs)}})
		for _, evidenceID := range p.EvidenceIDs {
			if _, ok := known[evidenceID]; ok {
				w.edge(Edge{Type: "INCLUDED_IN_PROOFPACK", SourceID: evidenceNodeID(evidenceID), TargetID: pid, TenantID: tenantID, Reason: "Proofpack manifest includes this evidence ID.", EvidenceSource: "proofpacks.evidence_ids", Confidence: "explicit", Status: "active", CreatedAt: p.CreatedAt})
			}
		}
	}
}

func buildHistory(ctx context.Context, w *work, tenantID string, now time.Time, data TenantData, ops *operations.Service) {
	for i, snap := range data.ReviewSnapshots {
		id := snapshotNodeID(snap.LastReviewedAt, i)
		w.node(Node{ID: id, Type: "snapshot", Label: "Review snapshot " + snap.LastReviewedAt.Format("2006-01-02"), TenantID: tenantID, Status: reviewStatus(snap), CreatedAt: snap.GeneratedAt, UpdatedAt: snap.LastReviewedAt, Summary: fmt.Sprintf("Health %d, unresolved %d.", snap.HealthScore, snap.UnresolvedIssues), Metadata: map[string]any{"health_score": snap.HealthScore, "unresolved_issues": snap.UnresolvedIssues}})
		w.edge(Edge{Type: "APPEARS_IN_SNAPSHOT", SourceID: tenantNodeID(tenantID), TargetID: id, TenantID: tenantID, Reason: "Review snapshot was generated for this tenant.", EvidenceSource: "review_snapshots", Confidence: "explicit", Status: "active", CreatedAt: snap.GeneratedAt})
	}
	for i, ev := range data.OperationalEvents {
		id := fmt.Sprintf("operational_event:%s:%s:%d", slug(ev.Type), ev.CreatedAt.Format("20060102T150405"), i)
		w.node(Node{ID: id, Type: "operational_event", Label: ev.Type, TenantID: tenantID, Status: "active", CreatedAt: ev.CreatedAt, UpdatedAt: ev.CreatedAt, Summary: ev.Message, Metadata: map[string]any{"entity_id": ev.EntityID}})
		source := tenantNodeID(tenantID)
		if ev.EntityID != "" {
			source = evidenceNodeID(ev.EntityID)
		}
		w.edge(Edge{Type: "GENERATED_EVENT", SourceID: source, TargetID: id, TenantID: tenantID, Reason: "Operational event is persisted for this tenant.", EvidenceSource: "operational_events", Confidence: "explicit", Status: "active", CreatedAt: ev.CreatedAt})
	}
	for _, r := range data.ReviewReports {
		id := "review_report:" + slug(r.ID)
		w.node(Node{ID: id, Type: "review_report", Label: r.ID, TenantID: tenantID, Status: "active", CreatedAt: r.GeneratedAt, UpdatedAt: r.GeneratedAt, Summary: r.Summary, Metadata: map[string]any{"has_markdown": r.Markdown != ""}})
		w.edge(Edge{Type: "REVIEWED_IN", SourceID: tenantNodeID(tenantID), TargetID: id, TenantID: tenantID, Reason: "Review report was generated from persisted tenant state.", EvidenceSource: "review_reports", Confidence: "explicit", Status: "active", CreatedAt: r.GeneratedAt})
	}
	buildComparison(w, tenantID, now, data)
	buildPilotAndExports(w, tenantID, now, data)
	if ops != nil {
		if sum, err := ops.BuildSummary(ctx, tenantID, nil); err == nil {
			for i, n := range sum.Narratives {
				id := fmt.Sprintf("narrative:%s:%d", slug(n.Scope), i)
				w.node(Node{ID: id, Type: "narrative", Label: n.Scope, TenantID: tenantID, Status: "active", CreatedAt: n.GeneratedAt, UpdatedAt: n.GeneratedAt, Summary: n.Message, Metadata: map[string]any{"evidence": n.Evidence}})
				w.edge(Edge{Type: "PRODUCED_NARRATIVE", SourceID: tenantNodeID(tenantID), TargetID: id, TenantID: tenantID, Reason: "Narrative is deterministically generated from persisted operational summary inputs.", EvidenceSource: "operations.BuildSummary", Confidence: "derived", Status: "active", CreatedAt: n.GeneratedAt})
			}
		}
	}
}

func buildComparison(w *work, tenantID string, now time.Time, data TenantData) {
	if len(data.ReviewSnapshots) < 2 {
		w.node(Node{ID: "review_comparison:not_comparable", Type: "review_comparison", Label: "Review comparison unavailable", TenantID: tenantID, Status: "not_comparable", CreatedAt: now, UpdatedAt: now, Summary: "No review comparison available yet.", Metadata: map[string]any{"required_reviews": 2, "current_reviews": len(data.ReviewSnapshots)}})
		return
	}
	state := "active"
	if data.ReviewSnapshots[0].UnresolvedIssues > data.ReviewSnapshots[1].UnresolvedIssues || data.ReviewSnapshots[0].HealthScore < data.ReviewSnapshots[1].HealthScore-2 {
		state = "unresolved"
	}
	w.node(Node{ID: "review_comparison:latest_previous", Type: "review_comparison", Label: "Latest vs previous review", TenantID: tenantID, Status: state, CreatedAt: data.ReviewSnapshots[0].GeneratedAt, UpdatedAt: data.ReviewSnapshots[0].GeneratedAt, Summary: "Deterministic comparison of the two latest persisted review snapshots.", Metadata: map[string]any{"state": state}})
	w.edge(Edge{Type: "COMPARED_WITH", SourceID: snapshotNodeID(data.ReviewSnapshots[0].LastReviewedAt, 0), TargetID: snapshotNodeID(data.ReviewSnapshots[1].LastReviewedAt, 1), TenantID: tenantID, Reason: "Latest and previous review snapshots are available.", EvidenceSource: "review_snapshots", Confidence: "derived", Status: state, CreatedAt: data.ReviewSnapshots[0].GeneratedAt})
}

func buildPilotAndExports(w *work, tenantID string, now time.Time, data TenantData) {
	readiness := "degraded"
	if len(data.ReviewSnapshots) >= 4 && len(data.Proofpacks) > 0 {
		readiness = "active"
	} else if len(data.ReviewSnapshots) >= 2 {
		readiness = "not_comparable"
	}
	pilotID := "pilot_readiness:" + slug(tenantID)
	w.node(Node{ID: pilotID, Type: "pilot_readiness", Label: "Pilot readiness", TenantID: tenantID, Status: readiness, CreatedAt: now, UpdatedAt: now, Summary: "Pilot readiness derived from review and proofpack history.", Metadata: map[string]any{"review_count": len(data.ReviewSnapshots), "proofpack_count": len(data.Proofpacks)}})
	for _, export := range []string{"markdown", "text", "json"} {
		id := "export:evidence-graph:" + export
		route := "/app/export/evidence-graph." + map[string]string{"markdown": "md", "text": "txt", "json": "json"}[export]
		w.node(Node{ID: id, Type: "export", Label: "Evidence graph " + export + " export", TenantID: tenantID, Status: "active", CreatedAt: now, UpdatedAt: now, Summary: "Server-generated Evidence Graph export.", Metadata: map[string]any{"route": route}})
		w.edge(Edge{Type: "READY_FOR_EXPORT", SourceID: tenantNodeID(tenantID), TargetID: id, TenantID: tenantID, Reason: "Evidence Graph export is generated from tenant-scoped records.", EvidenceSource: "evidence_graph_builder", Confidence: "derived", Status: "active", CreatedAt: now})
	}
}

func nextActions(tenantID string, now time.Time, items []evidence.Item, data TenantData) []NextAction {
	var out []NextAction
	for _, it := range items {
		id := evidenceNodeID(it.ID)
		if it.Status == "expired" {
			out = append(out, action("replace_expired_evidence", "expired", 25, "Replace expired evidence: "+it.Title, "Upload or record current evidence for this expired item.", it.ID, id, "/app", "Evidence status is expired."))
		}
		if it.IsOwnerless() {
			out = append(out, action("assign_owner", "ownerless", 18, "Assign owner to "+it.Title, "Add an explicit owner name or email.", it.ID, id, "/app", "Evidence has no owner_name or owner_email."))
		}
		if it.IsStale(now) {
			out = append(out, action("refresh_stale_evidence", "stale", 12, "Refresh stale evidence: "+it.Title, "Review and update this evidence.", it.ID, id, "/app", "Evidence updated_at is older than 180 days."))
		}
		if len(it.ControlRefs) == 0 {
			out = append(out, action("add_control_mapping", "missing", 10, "Add control mapping for "+it.Title, "Link this evidence to at least one explicit control reference.", it.ID, id, "/app/evidence/mappings?evidence_id="+it.ID, "Evidence has no control_refs."))
		}
		if len(it.VendorRefs) == 0 {
			out = append(out, action("add_vendor_mapping", "missing", 7, "Add vendor mapping for "+it.Title, "Add an explicit vendor reference when applicable.", it.ID, id, "/app/evidence/mappings?evidence_id="+it.ID, "Evidence has no vendor_refs."))
		}
	}
	if len(data.Proofpacks) == 0 {
		out = append(out, NextAction{ID: actionID("create_proofpack", tenantID), Type: "create_proofpack", Title: "Create first proofpack", Description: "Generate a proofpack so current evidence has export history.", Severity: "missing", Impact: 16, TargetNodeIDs: []string{tenantNodeID(tenantID)}, Reason: "No persisted proofpack metadata exists.", SuggestedRoute: "/app/proofpacks", ExportImpact: "Adds buyer/pilot proofpack history."})
	}
	if len(data.ReviewSnapshots) == 0 {
		out = append(out, NextAction{ID: actionID("complete_review", tenantID), Type: "complete_review", Title: "Complete first operational review", Description: "Generate the first review snapshot.", Severity: "missing", Impact: 20, TargetNodeIDs: []string{tenantNodeID(tenantID)}, Reason: "No review snapshots exist.", SuggestedRoute: "/app/reviews"})
	}
	if len(data.ReviewSnapshots) < 2 {
		out = append(out, NextAction{ID: actionID("compare_reviews", tenantID), Type: "compare_reviews", Title: "Generate enough reviews to compare", Description: "Create at least two review snapshots before comparing latest vs previous.", Severity: "not_comparable", Impact: 11, TargetNodeIDs: []string{"review_comparison:not_comparable"}, Reason: "Fewer than two review snapshots exist.", SuggestedRoute: "/app/reviews"})
	}
	if len(data.ReviewSnapshots) >= 4 && len(data.Proofpacks) > 0 {
		out = append(out, NextAction{ID: actionID("generate_pilot_proof_bundle", tenantID), Type: "generate_pilot_proof_bundle", Title: "Export pilot proof bundle", Description: "Export pilot proof from persisted reviews and proofpack history.", Severity: "active", Impact: 9, TargetNodeIDs: []string{"pilot_readiness:" + slug(tenantID)}, Reason: "Week-4 review history and proofpack history are available.", SuggestedRoute: "/app/export/pilot-proof.md", ExportImpact: "Produces founder-facing pilot proof."})
	}
	out = append(out, NextAction{ID: actionID("export_graph", tenantID), Type: "export_graph", Title: "Export evidence graph", Description: "Export the current canonical graph.", Severity: "active", Impact: 6, TargetNodeIDs: []string{tenantNodeID(tenantID)}, Reason: "Evidence Graph exports are available for this tenant.", SuggestedRoute: "/app/export/evidence-graph.md", ExportImpact: "Produces graph proof for operators, buyers, or auditors."})
	sortActions(out)
	return out
}

func action(typ, severity string, impact int, title, desc, rawTarget, nodeID, route, reason string) NextAction {
	return NextAction{ID: actionID(typ, rawTarget), Type: typ, Title: title, Description: desc, Severity: severity, Impact: impact, TargetNodeIDs: []string{nodeID}, Reason: reason, SuggestedRoute: route}
}

type evidenceStats struct {
	owners    int
	ownerless int
	expired   int
	stale     int
	risks     map[string]struct{}
}

func summarizeEvidenceInfo(now time.Time, items []evidence.Item) evidenceStats {
	owners := map[string]struct{}{}
	stats := evidenceStats{
		risks: map[string]struct{}{},
	}

	for _, it := range items {
		if it.IsOwnerless() {
			stats.ownerless++
		} else {
			owners[strings.ToLower(strings.TrimSpace(it.OwnerEmail)+"|"+strings.TrimSpace(it.OwnerName))] = struct{}{}
		}
		if it.Status == "expired" {
			stats.expired++
		}
		if it.IsStale(now) {
			stats.stale++
		}
		for _, r := range it.RiskRefs {
			stats.risks[r] = struct{}{}
		}
	}
	stats.owners = len(owners)
	return stats
}

func deriveTopRisks(risks map[string]struct{}, expired, ownerless, stale int) []string {
	topRisks := make([]string, 0, len(risks)+3)
	for r := range risks {
		topRisks = append(topRisks, r)
	}
	if expired > 0 {
		topRisks = append(topRisks, "expired evidence")
	}
	if ownerless > 0 {
		topRisks = append(topRisks, "ownerless evidence")
	}
	if stale > 0 {
		topRisks = append(topRisks, "stale evidence")
	}
	sort.Strings(topRisks)
	if len(topRisks) > 5 {
		topRisks = topRisks[:5]
	}
	return topRisks
}

func evaluateTenantData(data TenantData) (string, string, *time.Time) {
	comparison := "not_comparable"
	if len(data.ReviewSnapshots) >= 2 {
		comparison = "active"
	}
	readiness := "not_ready"
	if len(data.ReviewSnapshots) >= 4 && len(data.Proofpacks) > 0 {
		readiness = "ready_for_export"
	} else if len(data.ReviewSnapshots) >= 2 {
		readiness = "comparison_ready"
	}
	var latest *time.Time
	if len(data.ReviewSnapshots) > 0 {
		t := data.ReviewSnapshots[0].LastReviewedAt
		latest = &t
	}
	return comparison, readiness, latest
}

func extractTopActions(actions []NextAction) []string {
	topActions := []string{}
	for i, a := range actions {
		if i >= 5 {
			break
		}
		topActions = append(topActions, a.Title)
	}
	return topActions
}

func summary(tenantID string, now time.Time, items []evidence.Item, data TenantData, actions []NextAction) Summary {
	stats := summarizeEvidenceInfo(now, items)
	topRisks := deriveTopRisks(stats.risks, stats.expired, stats.ownerless, stats.stale)
	comparison, readiness, latest := evaluateTenantData(data)
	topActions := extractTopActions(actions)

	return Summary{
		TotalEvidence:       len(items),
		TotalOwners:         stats.owners,
		OwnerlessEvidence:   stats.ownerless,
		ExpiredEvidence:     stats.expired,
		StaleEvidence:       stats.stale,
		ProofpackCount:      len(data.Proofpacks),
		ReviewCount:         len(data.ReviewSnapshots),
		LatestReviewAt:      latest,
		ComparisonState:     comparison,
		PilotReadinessState: readiness,
		GraphHealthScore:    graphHealth(now, items, data),
		TopRisks:            topRisks,
		TopNextActions:      topActions,
	}
}

func degradedReasons(items []evidence.Item, data TenantData, s Summary) []string {
	var out []string
	if len(items) == 0 {
		out = append(out, "No evidence records exist yet. Add evidence, assign an owner, then generate a review or proofpack.")
	}
	if s.OwnerlessEvidence > 0 {
		out = append(out, "Some evidence has no explicit owner.")
	}
	if s.ExpiredEvidence > 0 {
		out = append(out, "Some evidence is expired.")
	}
	if s.StaleEvidence > 0 {
		out = append(out, "Some evidence is stale because updated_at is older than 180 days.")
	}
	if len(data.ReviewSnapshots) == 0 {
		out = append(out, "No review history exists yet.")
	}
	if len(data.ReviewSnapshots) < 2 {
		out = append(out, "No review comparison available yet.")
	}
	if len(data.Proofpacks) == 0 {
		out = append(out, "No proofpack history exists yet.")
	}
	missingControls, missingVendors := 0, 0
	for _, it := range items {
		if len(it.ControlRefs) == 0 {
			missingControls++
		}
		if len(it.VendorRefs) == 0 {
			missingVendors++
		}
	}
	if missingControls > 0 {
		out = append(out, "Some evidence lacks explicit control mappings.")
	}
	if missingVendors > 0 {
		out = append(out, "Some evidence lacks explicit vendor mappings.")
	}
	return out
}

func graphHealth(now time.Time, items []evidence.Item, data TenantData) int {
	score := 100
	if len(items) == 0 {
		score -= 40
	}
	for _, it := range items {
		if it.Status == "expired" {
			score -= 14
		}
		if it.IsOwnerless() {
			score -= 10
		}
		if it.IsStale(now) {
			score -= 8
		}
		if len(it.ControlRefs) == 0 {
			score -= 4
		}
		if len(it.VendorRefs) == 0 {
			score -= 2
		}
	}
	if len(data.ReviewSnapshots) == 0 {
		score -= 12
	} else if now.Sub(data.ReviewSnapshots[0].LastReviewedAt) > 9*24*time.Hour {
		score -= 10
	}
	if len(data.Proofpacks) == 0 {
		score -= 8
	}
	if len(data.ReviewSnapshots) < 2 {
		score -= 6
	}
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func applyCaps(g Graph, maxNodes, maxEdges int, now time.Time) Graph {
	originalNodes, originalEdges := len(g.Nodes), len(g.Edges)

	if maxNodes > 0 && len(g.Nodes) > maxNodes {
		kept := make(map[string]struct{}, maxNodes)

		g.Nodes = g.Nodes[:maxNodes]
		for i := 0; i < maxNodes; i++ {
			kept[g.Nodes[i].ID] = struct{}{}
		}

		var k int
		for i := 0; i < len(g.Edges); i++ {
			if _, ok := kept[g.Edges[i].SourceID]; !ok {
				continue
			}
			if _, ok := kept[g.Edges[i].TargetID]; !ok {
				continue
			}
			g.Edges[k] = g.Edges[i]
			k++
		}
		g.Edges = g.Edges[:k]

		g.Nodes = append(g.Nodes, Node{
			ID:        "graph_cap:omitted_nodes",
			Type:      "action",
			Label:     "Graph output capped",
			TenantID:  g.TenantID,
			Status:    "degraded",
			CreatedAt: now,
			UpdatedAt: now,
			Summary:   "Some nodes were omitted to keep graph output bounded.",
			Metadata:  map[string]any{"omitted_nodes": originalNodes - maxNodes, "max_nodes": maxNodes},
		})
		g.DegradedReasons = append(g.DegradedReasons, fmt.Sprintf("Graph node output capped at %d of %d nodes.", maxNodes, originalNodes))
	}

	if maxEdges > 0 && len(g.Edges) > maxEdges {
		g.Edges = g.Edges[:maxEdges]
		g.DegradedReasons = append(g.DegradedReasons, fmt.Sprintf("Graph edge output capped at %d of %d edges.", maxEdges, originalEdges))
	}

	sortGraph(&g)
	return g
}

func sortTenantData(d *TenantData) {
	sort.SliceStable(d.Proofpacks, func(i, j int) bool { return d.Proofpacks[i].CreatedAt.After(d.Proofpacks[j].CreatedAt) })
	sort.SliceStable(d.ReviewSnapshots, func(i, j int) bool {
		return d.ReviewSnapshots[i].LastReviewedAt.After(d.ReviewSnapshots[j].LastReviewedAt)
	})
	sort.SliceStable(d.OperationalEvents, func(i, j int) bool { return d.OperationalEvents[i].CreatedAt.After(d.OperationalEvents[j].CreatedAt) })
	sort.SliceStable(d.ReviewReports, func(i, j int) bool { return d.ReviewReports[i].GeneratedAt.After(d.ReviewReports[j].GeneratedAt) })
}

func sortGraph(g *Graph) {
	sort.SliceStable(g.Nodes, func(i, j int) bool {
		if g.Nodes[i].Type != g.Nodes[j].Type {
			return g.Nodes[i].Type < g.Nodes[j].Type
		}
		if severityRank(g.Nodes[i].Status) != severityRank(g.Nodes[j].Status) {
			return severityRank(g.Nodes[i].Status) < severityRank(g.Nodes[j].Status)
		}
		if g.Nodes[i].Label != g.Nodes[j].Label {
			return g.Nodes[i].Label < g.Nodes[j].Label
		}
		return g.Nodes[i].ID < g.Nodes[j].ID
	})
	sort.SliceStable(g.Edges, func(i, j int) bool {
		if g.Edges[i].Type != g.Edges[j].Type {
			return g.Edges[i].Type < g.Edges[j].Type
		}
		if g.Edges[i].SourceID != g.Edges[j].SourceID {
			return g.Edges[i].SourceID < g.Edges[j].SourceID
		}
		return g.Edges[i].TargetID < g.Edges[j].TargetID
	})
	sortActions(g.NextActions)
}

func sortActions(actions []NextAction) {
	sort.SliceStable(actions, func(i, j int) bool {
		if severityRank(actions[i].Severity) != severityRank(actions[j].Severity) {
			return severityRank(actions[i].Severity) < severityRank(actions[j].Severity)
		}
		if actions[i].Impact != actions[j].Impact {
			return actions[i].Impact > actions[j].Impact
		}
		return actions[i].ID < actions[j].ID
	})
}

func severityRank(status string) int {
	switch status {
	case "expired":
		return 1
	case "missing":
		return 2
	case "ownerless":
		return 3
	case "unresolved":
		return 4
	case "stale":
		return 5
	case "not_comparable":
		return 6
	case "degraded":
		return 7
	case "active":
		return 8
	default:
		return 9
	}
}

func evidenceStatus(it evidence.Item, now time.Time) string {
	return it.CanonicalStatus(now)
}

func edgeStatus(status string) string {
	if status == "ownerless" {
		return "missing"
	}
	switch status {
	case "active", "stale", "expired", "missing", "unresolved", "not_comparable", "degraded":
		return status
	default:
		return "degraded"
	}
}

func riskStatus(status string) string {
	if status == "active" {
		return "active"
	}
	return "unresolved"
}

func reviewStatus(snap persistence.ReviewSnapshot) string {
	if snap.ExpiredEvidence > 0 {
		return "expired"
	}
	if snap.MissingOwners > 0 {
		return "ownerless"
	}
	if snap.StaleEvidence > 0 {
		return "stale"
	}
	if snap.UnresolvedIssues > 0 {
		return "unresolved"
	}
	return "active"
}

func ownerNode(it evidence.Item) (string, string, bool) {
	name := strings.TrimSpace(it.OwnerName)
	email := strings.TrimSpace(it.OwnerEmail)
	if name == "" && email == "" {
		return "", "", false
	}
	label := name
	if label == "" {
		label = email
	}
	key := email
	if key == "" {
		key = name
	}
	return typedNodeID("owner", key), label, true
}

func tenantNodeID(id string) string        { return "tenant:" + slug(id) }
func evidenceNodeID(id string) string      { return "evidence:" + slug(id) }
func proofpackNodeID(id string) string     { return "proofpack:" + slug(id) }
func typedNodeID(typ, label string) string { return typ + ":" + slug(label) }
func actionNodeID(id string) string        { return "action:" + slug(id) }
func actionID(typ, target string) string   { return typ + ":" + slug(target) }
func edgeID(typ, source, target string) string {
	return "edge:" + slug(typ) + ":" + slug(source) + ":" + slug(target)
}
func snapshotNodeID(t time.Time, i int) string {
	return fmt.Sprintf("snapshot:review:%s:%d", t.Format("20060102T150405"), i)
}

func slug(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	var b strings.Builder
	lastDash := false
	for _, r := range v {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}
