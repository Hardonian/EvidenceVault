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
	TenantID        string         `json:"tenantId"`
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
	store      persistence.Store
	evidence   *evidence.Service
	operations *operations.Service
	Now        func() time.Time
}

type graphData struct {
	proofpacks           []persistence.ProofpackMeta
	reviewSnapshots      []persistence.ReviewSnapshot
	operationalSnapshots []persistence.OperationalSnapshot
	operationalEvents    []persistence.OperationalEvent
	reviewReports        []persistence.ReviewReport
}

func NewBuilder(store persistence.Store, ev *evidence.Service, ops *operations.Service) *Builder {
	return &Builder{store: store, evidence: ev, operations: ops, Now: func() time.Time { return time.Now().UTC() }}
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
	data := graphData{}
	if b.store != nil {
		_ = b.store.Read(func(st *persistence.State) error {
			data.proofpacks = append([]persistence.ProofpackMeta{}, st.Proofpacks[tenantID]...)
			data.reviewSnapshots = append([]persistence.ReviewSnapshot{}, st.ReviewSnapshots[tenantID]...)
			data.operationalSnapshots = append([]persistence.OperationalSnapshot{}, st.OperationalSnapshots[tenantID]...)
			data.operationalEvents = append([]persistence.OperationalEvent{}, st.OperationalEvents[tenantID]...)
			data.reviewReports = append([]persistence.ReviewReport{}, st.ReviewReports[tenantID]...)
			return nil
		})
	}
	sortData(&data)
	gb := newWorkingGraph(tenantID, now)
	gb.addNode(Node{ID: tenantNodeID(tenantID), Type: "tenant", Label: tenantID, TenantID: tenantID, Status: "active", CreatedAt: now, UpdatedAt: now, Summary: "Tenant-scoped graph root.", Metadata: map[string]any{"graph_version": Version}})
	buildEvidenceLayer(gb, tenantID, now, items)
	buildProofpackLayer(gb, tenantID, now, data.proofpacks, items)
	buildHistoryLayer(gb, tenantID, now, data)
	buildOperationsLayer(ctx, gb, tenantID, now, b.operations, data, len(items))
	gb.actions = generateActions(tenantID, now, items, data)
	for _, action := range gb.actions {
		gb.addActionNode(action, now)
	}
	linkActions(gb, tenantID, now, gb.actions)
	gb.summary = buildSummary(tenantID, now, items, data, gb.actions)
	gb.degradedReasons = buildDegradedReasons(items, data, gb.summary)
	gb.sort()
	return gb.graph(), nil
}

func (b *Builder) now() time.Time {
	if b.Now != nil {
		return b.Now().UTC()
	}
	return time.Now().UTC()
}

type workingGraph struct {
	tenantID        string
	generatedAt     time.Time
	nodes           map[string]Node
	edges           map[string]Edge
	actions         []NextAction
	summary         Summary
	degradedReasons []string
}

func newWorkingGraph(tenantID string, generatedAt time.Time) *workingGraph {
	return &workingGraph{tenantID: tenantID, generatedAt: generatedAt, nodes: map[string]Node{}, edges: map[string]Edge{}}
}

func (g *workingGraph) graph() Graph {
	nodes := make([]Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	sortNodes(nodes)
	edges := make([]Edge, 0, len(g.edges))
	for _, e := range g.edges {
		edges = append(edges, e)
	}
	sortEdges(edges)
	sortActions(g.actions)
	return Graph{TenantID: g.tenantID, GeneratedAt: g.generatedAt, GraphVersion: Version, Nodes: nodes, Edges: edges, Summary: g.summary, DegradedReasons: g.degradedReasons, NextActions: g.actions}
}

func (g *workingGraph) addNode(n Node) {
	if n.Metadata == nil {
		n.Metadata = map[string]any{}
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = g.generatedAt
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = n.CreatedAt
	}
	g.nodes[n.ID] = n
}

func (g *workingGraph) addEdge(e Edge) {
	if e.Metadata == nil {
		e.Metadata = map[string]any{}
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = g.generatedAt
	}
	e.ID = edgeID(e.Type, e.SourceID, e.TargetID)
	g.edges[e.ID] = e
}

func (g *workingGraph) addActionNode(a NextAction, now time.Time) {
	g.addNode(Node{ID: actionNodeID(a.ID), Type: "action", Label: a.Title, TenantID: g.tenantID, Status: a.Severity, CreatedAt: now, UpdatedAt: now, Summary: a.Description, Metadata: map[string]any{"action_type": a.Type, "impact": a.Impact, "reason": a.Reason, "suggested_route": a.SuggestedRoute, "export_impact": a.ExportImpact}})
}

func (g *workingGraph) sort() {
	nodes := make([]Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	sortNodes(nodes)
	g.nodes = map[string]Node{}
	for _, n := range nodes {
		g.nodes[n.ID] = n
	}
	edges := make([]Edge, 0, len(g.edges))
	for _, e := range g.edges {
		edges = append(edges, e)
	}
	sortEdges(edges)
	g.edges = map[string]Edge{}
	for _, e := range edges {
		g.edges[e.ID] = e
	}
	sortActions(g.actions)
}

func sortData(data *graphData) {
	sort.SliceStable(data.proofpacks, func(i, j int) bool { return data.proofpacks[i].CreatedAt.After(data.proofpacks[j].CreatedAt) })
	sort.SliceStable(data.reviewSnapshots, func(i, j int) bool { return data.reviewSnapshots[i].LastReviewedAt.After(data.reviewSnapshots[j].LastReviewedAt) })
	sort.SliceStable(data.operationalSnapshots, func(i, j int) bool { return data.operationalSnapshots[i].CreatedAt.After(data.operationalSnapshots[j].CreatedAt) })
	sort.SliceStable(data.operationalEvents, func(i, j int) bool { return data.operationalEvents[i].CreatedAt.After(data.operationalEvents[j].CreatedAt) })
	sort.SliceStable(data.reviewReports, func(i, j int) bool { return data.reviewReports[i].GeneratedAt.After(data.reviewReports[j].GeneratedAt) })
}

func buildEvidenceLayer(g *workingGraph, tenantID string, now time.Time, items []evidence.Item) {
	for _, it := range items {
		evID := evidenceNodeID(it.ID)
		status := evidenceStatus(it, now)
		g.addNode(Node{ID: evID, Type: "evidence", Label: it.Title, TenantID: tenantID, Status: status, CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt, Summary: evidenceSummary(it), Metadata: map[string]any{"category": it.Category, "has_source_file": it.SourceFilePath != "", "has_expiry": it.ExpiryDate != nil}})
		if ownerID, ownerLabel, ok := ownerNode(it); ok {
			g.addNode(Node{ID: ownerID, Type: "owner", Label: ownerLabel, TenantID: tenantID, Status: "active", CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt, Summary: "Owner explicitly assigned on evidence record.", Metadata: map[string]any{"owner_email": it.OwnerEmail, "owner_name": it.OwnerName}})
			g.addEdge(Edge{Type: "OWNS", SourceID: ownerID, TargetID: evID, TenantID: tenantID, Reason: "Evidence record has an explicit owner.", EvidenceSource: "evidence_items.owner_name/owner_email", Confidence: "explicit", Status: "active", CreatedAt: it.UpdatedAt})
		}
		for _, control := range it.ControlRefs {
			id := typedNodeID("control", control)
			g.addNode(Node{ID: id, Type: "control", Label: control, TenantID: tenantID, Status: "active", CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt, Summary: "Control explicitly mapped on evidence record.", Metadata: map[string]any{"source_evidence_id": it.ID}})
			g.addEdge(Edge{Type: "SUPPORTS_CONTROL", SourceID: evID, TargetID: id, TenantID: tenantID, Reason: "Evidence record lists this control reference.", EvidenceSource: "evidence_items.control_refs", Confidence: "explicit", Status: "active", CreatedAt: it.UpdatedAt})
		}
		for _, vendor := range it.VendorRefs {
			id := typedNodeID("vendor", vendor)
			g.addNode(Node{ID: id, Type: "vendor", Label: vendor, TenantID: tenantID, Status: "active", CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt, Summary: "Vendor explicitly mapped on evidence record.", Metadata: map[string]any{"source_evidence_id": it.ID}})
			g.addEdge(Edge{Type: "RELATES_TO_VENDOR", SourceID: evID, TargetID: id, TenantID: tenantID, Reason: "Evidence record lists this vendor reference.", EvidenceSource: "evidence_items.vendor_refs", Confidence: "explicit", Status: "active", CreatedAt: it.UpdatedAt})
		}
		for _, risk := range it.RiskRefs {
			id := typedNodeID("risk", risk)
			g.addNode(Node{ID: id, Type: "risk", Label: risk, TenantID: tenantID, Status: riskStatus(it, now), CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt, Summary: "Risk explicitly mapped on evidence record.", Metadata: map[string]any{"source_evidence_id": it.ID}})
			g.addEdge(Edge{Type: "MITIGATES_RISK", SourceID: evID, TargetID: id, TenantID: tenantID, Reason: "Evidence record lists this risk reference.", EvidenceSource: "evidence_items.risk_refs", Confidence: "explicit", Status: evidenceStatus(it, now), CreatedAt: it.UpdatedAt})
		}
	}
}

func buildProofpackLayer(g *workingGraph, tenantID string, now time.Time, packs []persistence.ProofpackMeta, items []evidence.Item) {
	evidenceByID := map[string]evidence.Item{}
	for _, it := range items {
		evidenceByID[it.ID] = it
	}
	for _, pack := range packs {
		packID := proofpackNodeID(pack.ID)
		status := "active"
		if len(pack.EvidenceIDs) == 0 {
			status = "degraded"
		}
		g.addNode(Node{ID: packID, Type: "proofpack", Label: pack.ID, TenantID: tenantID, Status: status, CreatedAt: pack.CreatedAt, UpdatedAt: pack.CreatedAt, Summary: "Persisted proofpack export metadata.", Metadata: map[string]any{"evidence_count": len(pack.EvidenceIDs)}})
		if len(pack.EvidenceIDs) == 0 {
			g.addEdge(Edge{Type: "READY_FOR_EXPORT", SourceID: tenantNodeID(tenantID), TargetID: packID, TenantID: tenantID, Reason: "Legacy proofpack metadata exists but lacks persisted evidence manifest.", EvidenceSource: "proofpacks.created_at", Confidence: "degraded", Status: "degraded", CreatedAt: pack.CreatedAt})
			continue
		}
		for _, evidenceID := range pack.EvidenceIDs {
			if _, ok := evidenceByID[evidenceID]; !ok {
				continue
			}
			g.addEdge(Edge{Type: "INCLUDED_IN_PROOFPACK", SourceID: evidenceNodeID(evidenceID), TargetID: packID, TenantID: tenantID, Reason: "Proofpack manifest includes this evidence ID.", EvidenceSource: "proofpacks.evidence_ids", Confidence: "explicit", Status: "active", CreatedAt: pack.CreatedAt})
		}
	}
}

func buildHistoryLayer(g *workingGraph, tenantID string, now time.Time, data graphData) {
	for i, snap := range data.reviewSnapshots {
		id := fmt.Sprintf("snapshot:review:%s:%d", snap.LastReviewedAt.Format("20060102T150405"), i)
		g.addNode(Node{ID: id, Type: "snapshot", Label: "Review snapshot " + snap.LastReviewedAt.Format("2006-01-02"), TenantID: tenantID, Status: reviewSnapshotStatus(snap), CreatedAt: snap.GeneratedAt, UpdatedAt: snap.LastReviewedAt, Summary: fmt.Sprintf("Health %d, unresolved %d.", snap.HealthScore, snap.UnresolvedIssues), Metadata: map[string]any{"snapshot_kind": "review", "health_score": snap.HealthScore, "unresolved_issues": snap.UnresolvedIssues, "expired_evidence": snap.ExpiredEvidence, "stale_evidence": snap.StaleEvidence, "missing_owners": snap.MissingOwners}})
		g.addEdge(Edge{Type: "APPEARS_IN_SNAPSHOT", SourceID: tenantNodeID(tenantID), TargetID: id, TenantID: tenantID, Reason: "Review snapshot was generated for this tenant.", EvidenceSource: "review_snapshots", Confidence: "explicit", Status: "active", CreatedAt: snap.GeneratedAt})
	}
	for i, snap := range data.operationalSnapshots {
		id := fmt.Sprintf("snapshot:operational:%s:%d", snap.CreatedAt.Format("20060102T150405"), i)
		g.addNode(Node{ID: id, Type: "snapshot", Label: "Operational snapshot " + snap.Date, TenantID: tenantID, Status: "active", CreatedAt: snap.CreatedAt, UpdatedAt: snap.CreatedAt, Summary: fmt.Sprintf("Operational health %d.", snap.HealthScore), Metadata: map[string]any{"snapshot_kind": "operational", "health_score": snap.HealthScore, "review_streak": snap.ReviewStreak, "activation_percent": snap.ActivationPercent}})
		g.addEdge(Edge{Type: "APPEARS_IN_SNAPSHOT", SourceID: tenantNodeID(tenantID), TargetID: id, TenantID: tenantID, Reason: "Operational snapshot was generated for this tenant.", EvidenceSource: "operational_snapshots", Confidence: "explicit", Status: "active", CreatedAt: snap.CreatedAt})
	}
	for i, event := range data.operationalEvents {
		id := fmt.Sprintf("operational_event:%s:%s:%d", slug(event.Type), event.CreatedAt.Format("20060102T150405"), i)
		g.addNode(Node{ID: id, Type: "operational_event", Label: event.Type, TenantID: tenantID, Status: "active", CreatedAt: event.CreatedAt, UpdatedAt: event.CreatedAt, Summary: event.Message, Metadata: map[string]any{"entity_id": event.EntityID}})
		source := tenantNodeID(tenantID)
		if event.EntityID != "" {
			if _, ok := g.nodes[evidenceNodeID(event.EntityID)]; ok {
				source = evidenceNodeID(event.EntityID)
			}
		}
		g.addEdge(Edge{Type: "GENERATED_EVENT", SourceID: source, TargetID: id, TenantID: tenantID, Reason: "Operational event is persisted for this tenant and entity.", EvidenceSource: "operational_events", Confidence: "explicit", Status: "active", CreatedAt: event.CreatedAt})
	}
	for _, report := range data.reviewReports {
		id := reviewReportNodeID(report.ID)
		g.addNode(Node{ID: id, Type: "review_report", Label: report.ID, TenantID: tenantID, Status: "active", CreatedAt: report.GeneratedAt, UpdatedAt: report.GeneratedAt, Summary: report.Summary, Metadata: map[string]any{"has_markdown": report.Markdown != "", "has_plain_text": report.PlainText != ""}})
		g.addEdge(Edge{Type: "REVIEWED_IN", SourceID: tenantNodeID(tenantID), TargetID: id, TenantID: tenantID, Reason: "Review report was generated from persisted tenant state.", EvidenceSource: "review_reports", Confidence: "explicit", Status: "active", CreatedAt: report.GeneratedAt})
	}
	if len(data.reviewSnapshots) >= 2 {
		id := "review_comparison:latest_previous"
		state := comparisonState(data.reviewSnapshots[0], data.reviewSnapshots[1])
		g.addNode(Node{ID: id, Type: "review_comparison", Label: "Latest vs previous review", TenantID: tenantID, Status: state, CreatedAt: data.reviewSnapshots[0].GeneratedAt, UpdatedAt: data.reviewSnapshots[0].GeneratedAt, Summary: "Deterministic comparison of the two latest persisted review snapshots.", Metadata: map[string]any{"state": state}})
		g.addEdge(Edge{Type: "COMPARED_WITH", SourceID: snapshotNodeForReview(data.reviewSnapshots[0], 0), TargetID: snapshotNodeForReview(data.reviewSnapshots[1], 1), TenantID: tenantID, Reason: "Latest and previous review snapshots are available.", EvidenceSource: "review_snapshots", Confidence: "derived", Status: state, CreatedAt: data.reviewSnapshots[0].GeneratedAt})
	} else {
		id := "review_comparison:not_comparable"
		g.addNode(Node{ID: id, Type: "review_comparison", Label: "Review comparison unavailable", TenantID: tenantID, Status: "not_comparable", CreatedAt: now, UpdatedAt: now, Summary: "No review comparison available yet.", Metadata: map[string]any{"required_reviews": 2, "current_reviews": len(data.reviewSnapshots)}})
	}
}

func buildOperationsLayer(ctx context.Context, g *workingGraph, tenantID string, now time.Time, ops *operations.Service, data graphData, totalEvidence int) {
	pilotID := "pilot_readiness:" + slug(tenantID)
	pilotStatus := "degraded"
	pilotSummary := "Pilot readiness is not established yet."
	if len(data.reviewSnapshots) >= 4 && len(data.proofpacks) > 0 {
		pilotStatus = "active"
		pilotSummary = "Pilot proof bundle inputs are available from persisted reviews and proofpacks."
	} else if len(data.reviewSnapshots) >= 2 {
		pilotStatus = "not_comparable"
		pilotSummary = "Comparison history exists, but week-4 proof readiness is not complete."
	}
	g.addNode(Node{ID: pilotID, Type: "pilot_readiness", Label: "Pilot readiness", TenantID: tenantID, Status: pilotStatus, CreatedAt: now, UpdatedAt: now, Summary: pilotSummary, Metadata: map[string]any{"review_count": len(data.reviewSnapshots), "proofpack_count": len(data.proofpacks), "total_evidence": totalEvidence}})
	for i, snap := range data.reviewSnapshots {
		g.addEdge(Edge{Type: "CONTRIBUTES_TO_PILOT_READINESS", SourceID: snapshotNodeForReview(snap, i), TargetID: pilotID, TenantID: tenantID, Reason: "Review snapshot contributes to pilot readiness state.", EvidenceSource: "review_snapshots", Confidence: "derived", Status: "active", CreatedAt: snap.GeneratedAt})
	}
	for _, pack := range data.proofpacks {
		g.addEdge(Edge{Type: "CONTRIBUTES_TO_PILOT_READINESS", SourceID: proofpackNodeID(pack.ID), TargetID: pilotID, TenantID: tenantID, Reason: "Proofpack export history contributes to pilot proof readiness.", EvidenceSource: "proofpacks", Confidence: "explicit", Status: "active", CreatedAt: pack.CreatedAt})
	}
	g.addNode(Node{ID: "export:evidence-graph:markdown", Type: "export", Label: "Evidence graph Markdown export", TenantID: tenantID, Status: "active", CreatedAt: now, UpdatedAt: now, Summary: "Deterministic Markdown export of the canonical graph.", Metadata: map[string]any{"route": "/app/export/evidence-graph.md"}})
	g.addNode(Node{ID: "export:evidence-graph:text", Type: "export", Label: "Evidence graph text export", TenantID: tenantID, Status: "active", CreatedAt: now, UpdatedAt: now, Summary: "Deterministic plain text export of the canonical graph.", Metadata: map[string]any{"route": "/app/export/evidence-graph.txt"}})
	g.addNode(Node{ID: "export:evidence-graph:json", Type: "export", Label: "Evidence graph JSON export", TenantID: tenantID, Status: "active", CreatedAt: now, UpdatedAt: now, Summary: "Canonical graph JSON export.", Metadata: map[string]any{"route": "/app/export/evidence-graph.json"}})
	g.addEdge(Edge{Type: "READY_FOR_EXPORT", SourceID: tenantNodeID(tenantID), TargetID: "export:evidence-graph:markdown", TenantID: tenantID, Reason: "Evidence Graph export is generated server-side from tenant-scoped records.", EvidenceSource: "evidence_graph_builder", Confidence: "derived", Status: "active", CreatedAt: now})
	g.addEdge(Edge{Type: "READY_FOR_EXPORT", SourceID: tenantNodeID(tenantID), TargetID: "export:evidence-graph:text", TenantID: tenantID, Reason: "Evidence Graph export is generated server-side from tenant-scoped records.", EvidenceSource: "evidence_graph_builder", Confidence: "derived", Status: "active", CreatedAt: now})
	g.addEdge(Edge{Type: "READY_FOR_EXPORT", SourceID: tenantNodeID(tenantID), TargetID: "export:evidence-graph:json", TenantID: tenantID, Reason: "Evidence Graph export is generated server-side from tenant-scoped records.", EvidenceSource: "evidence_graph_builder", Confidence: "derived", Status: "active", CreatedAt: now})
	if ops == nil {
		return
	}
	sum, err := ops.BuildSummary(ctx, tenantID, nil)
	if err != nil {
		return
	}
	for i, n := range sum.Narratives {
		id := fmt.Sprintf("narrative:%s:%d", slug(n.Scope), i)
		g.addNode(Node{ID: id, Type: "narrative", Label: n.Scope, TenantID: tenantID, Status: "active", CreatedAt: n.GeneratedAt, UpdatedAt: n.GeneratedAt, Summary: n.Message, Metadata: map[string]any{"evidence": n.Evidence}})
		source := tenantNodeID(tenantID)
		if len(data.reviewSnapshots) >= 2 {
			source = "review_comparison:latest_previous"
		}
		g.addEdge(Edge{Type: "PRODUCED_NARRATIVE", SourceID: source, TargetID: id, TenantID: tenantID, Reason: "Narrative is deterministically generated from persisted operational summary inputs.", EvidenceSource: "operations.BuildSummary", Confidence: "derived", Status: "active", CreatedAt: n.GeneratedAt})
	}
	_ = sum
}

func generateActions(tenantID string, now time.Time, items []evidence.Item, data graphData) []NextAction {
	actions := []NextAction{}
	for _, it := range items {
		nodeID := evidenceNodeID(it.ID)
		if isOwnerless(it) {
			actions = append(actions, NextAction{ID: actionID("assign_owner", it.ID), Type: "assign_owner", Title: "Assign owner to " + it.Title, Description: "Add an explicit owner name or email to make this evidence accountable.", Severity: "ownerless", Impact: 18, TargetNodeIDs: []string{nodeID}, Reason: "Evidence has no owner_name or owner_email.", SuggestedRoute: "/app"})
		}
		if it.Status == "expired" {
			actions = append(actions, NextAction{ID: actionID("replace_expired_evidence", it.ID), Type: "replace_expired_evidence", Title: "Replace expired evidence: " + it.Title, Description: "Upload or record current evidence for this expired item.", Severity: "expired", Impact: 25, TargetNodeIDs: []string{nodeID}, Reason: "Evidence status is expired.", SuggestedRoute: "/app"})
		}
		if isStale(it, now) {
			actions = append(actions, NextAction{ID: actionID("refresh_stale_evidence", it.ID), Type: "refresh_stale_evidence", Title: "Refresh stale evidence: " + it.Title, Description: "Review and update this evidence because it has not changed in over 180 days.", Severity: "stale", Impact: 12, TargetNodeIDs: []string{nodeID}, Reason: "Evidence updated_at is older than 180 days.", SuggestedRoute: "/app"})
		}
		if len(it.ControlRefs) == 0 {
			actions = append(actions, NextAction{ID: actionID("add_control_mapping", it.ID), Type: "add_control_mapping", Title: "Add control mapping for " + it.Title, Description: "Link this evidence to at least one explicit control reference.", Severity: "missing", Impact: 10, TargetNodeIDs: []string{nodeID}, Reason: "Evidence has no control_refs.", SuggestedRoute: "/app"})
		}
		if len(it.VendorRefs) == 0 {
			actions = append(actions, NextAction{ID: actionID("add_vendor_mapping", it.ID), Type: "add_vendor_mapping", Title: "Add vendor mapping for " + it.Title, Description: "Add an explicit vendor reference when this evidence relates to a vendor.", Severity: "missing", Impact: 7, TargetNodeIDs: []string{nodeID}, Reason: "Evidence has no vendor_refs.", SuggestedRoute: "/app"})
		}
	}
	if len(data.proofpacks) == 0 {
		actions = append(actions, NextAction{ID: actionID("create_proofpack", tenantID), Type: "create_proofpack", Title: "Create first proofpack", Description: "Generate a proofpack so current evidence has export history.", Severity: "missing", Impact: 16, TargetNodeIDs: []string{tenantNodeID(tenantID)}, Reason: "No persisted proofpack metadata exists.", SuggestedRoute: "/app/proofpacks", ExportImpact: "Adds buyer/pilot proofpack history."})
	}
	if len(data.reviewSnapshots) == 0 {
		actions = append(actions, NextAction{ID: actionID("complete_review", tenantID), Type: "complete_review", Title: "Complete first operational review", Description: "Generate the first review snapshot to establish baseline history.", Severity: "missing", Impact: 20, TargetNodeIDs: []string{tenantNodeID(tenantID)}, Reason: "No review snapshots exist.", SuggestedRoute: "/app/reviews"})
	} else if data.reviewSnapshots[0].UnresolvedIssues > 0 {
		actions = append(actions, NextAction{ID: actionID("resolve_unresolved_item", tenantID), Type: "resolve_unresolved_item", Title: "Resolve unresolved review items", Description: "Work down expired, stale, or ownerless evidence from the latest review.", Severity: "unresolved", Impact: 18, TargetNodeIDs: []string{snapshotNodeForReview(data.reviewSnapshots[0], 0)}, Reason: "Latest review snapshot has unresolved issues.", SuggestedRoute: "/app"})
	}
	if len(data.reviewSnapshots) < 2 {
		actions = append(actions, NextAction{ID: actionID("compare_reviews", tenantID), Type: "compare_reviews", Title: "Generate enough reviews to compare", Description: "Create at least two review snapshots before comparing latest vs previous.", Severity: "not_comparable", Impact: 11, TargetNodeIDs: []string{"review_comparison:not_comparable"}, Reason: "Fewer than two review snapshots exist.", SuggestedRoute: "/app/reviews"})
	} else {
		actions = append(actions, NextAction{ID: actionID("compare_reviews", tenantID), Type: "compare_reviews", Title: "Compare latest and previous reviews", Description: "Use the review comparison export to explain operational change.", Severity: "active", Impact: 8, TargetNodeIDs: []string{"review_comparison:latest_previous"}, Reason: "At least two review snapshots exist.", SuggestedRoute: "/app/export/review-comparison.md", ExportImpact: "Adds continuity proof for buyer or founder review."})
	}
	if len(data.reviewSnapshots) >= 4 && len(data.proofpacks) > 0 {
		actions = append(actions, NextAction{ID: actionID("generate_pilot_proof_bundle", tenantID), Type: "generate_pilot_proof_bundle", Title: "Export pilot proof bundle", Description: "Export pilot proof from persisted reviews, narratives, comparison, and proofpack history.", Severity: "active", Impact: 9, TargetNodeIDs: []string{"pilot_readiness:" + slug(tenantID)}, Reason: "Week-4 review history and proofpack history are available.", SuggestedRoute: "/app/export/pilot-proof.md", ExportImpact: "Produces founder-facing pilot proof."})
	}
	actions = append(actions, NextAction{ID: actionID("export_graph", tenantID), Type: "export_graph", Title: "Export evidence graph", Description: "Export the current canonical evidence graph as Markdown, text, or JSON.", Severity: "active", Impact: 6, TargetNodeIDs: []string{tenantNodeID(tenantID)}, Reason: "Evidence Graph exports are available for this tenant.", SuggestedRoute: "/app/export/evidence-graph.md", ExportImpact: "Produces graph proof for operators, buyers, or auditors."})
	sortActions(actions)
	return actions
}

func linkActions(g *workingGraph, tenantID string, now time.Time, actions []NextAction) {
	pilotID := "pilot_readiness:" + slug(tenantID)
	for _, action := range actions {
		actionNode := actionNodeID(action.ID)
		for _, target := range action.TargetNodeIDs {
			g.addEdge(Edge{Type: "REQUIRES_ACTION", SourceID: target, TargetID: actionNode, TenantID: tenantID, Reason: action.Reason, EvidenceSource: "evidence_graph.next_actions", Confidence: "derived", Status: action.Severity, CreatedAt: now})
			switch action.Type {
			case "replace_expired_evidence":
				g.addEdge(Edge{Type: "EXPIRED_BECAUSE", SourceID: target, TargetID: actionNode, TenantID: tenantID, Reason: "Evidence is expired and blocks readiness until replaced.", EvidenceSource: "evidence_items.status", Confidence: "explicit", Status: "expired", CreatedAt: now})
			case "refresh_stale_evidence":
				g.addEdge(Edge{Type: "STALE_BECAUSE", SourceID: target, TargetID: actionNode, TenantID: tenantID, Reason: "Evidence has not been updated within the stale threshold.", EvidenceSource: "evidence_items.updated_at", Confidence: "derived", Status: "stale", CreatedAt: now})
			case "assign_owner":
				g.addEdge(Edge{Type: "OWNERLESS_BECAUSE", SourceID: target, TargetID: actionNode, TenantID: tenantID, Reason: "Evidence lacks owner_name and owner_email.", EvidenceSource: "evidence_items.owner_name/owner_email", Confidence: "explicit", Status: "ownerless", CreatedAt: now})
			case "add_control_mapping":
				g.addEdge(Edge{Type: "MISSING_EVIDENCE_FOR", SourceID: target, TargetID: actionNode, TenantID: tenantID, Reason: "Evidence has no explicit control mapping.", EvidenceSource: "evidence_items.control_refs", Confidence: "explicit", Status: "missing", CreatedAt: now})
			case "compare_reviews":
				if action.Severity == "not_comparable" {
					g.addEdge(Edge{Type: "NOT_COMPARABLE_YET", SourceID: target, TargetID: actionNode, TenantID: tenantID, Reason: "At least two review snapshots are required for comparison.", EvidenceSource: "review_snapshots", Confidence: "derived", Status: "not_comparable", CreatedAt: now})
				}
			}
		}
		if action.Severity != "active" && g.nodes[pilotID].ID != "" {
			g.addEdge(Edge{Type: "BLOCKED_BY", SourceID: pilotID, TargetID: actionNode, TenantID: tenantID, Reason: "Pilot readiness is blocked by this graph action.", EvidenceSource: "evidence_graph.next_actions", Confidence: "derived", Status: action.Severity, CreatedAt: now})
		}
	}
}

func buildSummary(tenantID string, now time.Time, items []evidence.Item, data graphData, actions []NextAction) Summary {
	owners := map[string]struct{}{}
	ownerless := 0
	expired := 0
	stale := 0
	risks := map[string]struct{}{}
	for _, it := range items {
		if isOwnerless(it) {
			ownerless++
		} else {
			owners[strings.ToLower(strings.TrimSpace(it.OwnerEmail)+"|"+strings.TrimSpace(it.OwnerName))] = struct{}{}
		}
		if it.Status == "expired" {
			expired++
		}
		if isStale(it, now) {
			stale++
		}
		for _, risk := range it.RiskRefs {
			risks[risk] = struct{}{}
		}
	}
	topRisks := make([]string, 0, len(risks)+3)
	for risk := range risks {
		topRisks = append(topRisks, risk)
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
	comparison := "not_comparable"
	if len(data.reviewSnapshots) >= 2 {
		comparison = comparisonState(data.reviewSnapshots[0], data.reviewSnapshots[1])
	}
	readiness := "not_ready"
	if len(data.reviewSnapshots) >= 4 && len(data.proofpacks) > 0 {
		readiness = "ready_for_export"
	} else if len(data.reviewSnapshots) >= 2 {
		readiness = "comparison_ready"
	}
	var latest *time.Time
	if len(data.reviewSnapshots) > 0 {
		t := data.reviewSnapshots[0].LastReviewedAt
		latest = &t
	}
	topActions := []string{}
	for i, a := range actions {
		if i >= 5 {
			break
		}
		topActions = append(topActions, a.Title)
	}
	return Summary{TotalEvidence: len(items), TotalOwners: len(owners), OwnerlessEvidence: ownerless, ExpiredEvidence: expired, StaleEvidence: stale, ProofpackCount: len(data.proofpacks), ReviewCount: len(data.reviewSnapshots), LatestReviewAt: latest, ComparisonState: comparison, PilotReadinessState: readiness, GraphHealthScore: graphHealthScore(now, items, data), TopRisks: topRisks, TopNextActions: topActions}
}

func buildDegradedReasons(items []evidence.Item, data graphData, summary Summary) []string {
	reasons := []string{}
	if len(items) == 0 {
		reasons = append(reasons, "No evidence records exist yet. Add evidence, assign an owner, then generate a review or proofpack.")
	}
	if summary.OwnerlessEvidence > 0 {
		reasons = append(reasons, "Some evidence has no explicit owner.")
	}
	if summary.ExpiredEvidence > 0 {
		reasons = append(reasons, "Some evidence is expired.")
	}
	if summary.StaleEvidence > 0 {
		reasons = append(reasons, "Some evidence is stale because updated_at is older than 180 days.")
	}
	if len(data.reviewSnapshots) == 0 {
		reasons = append(reasons, "No review history exists yet.")
	}
	if len(data.reviewSnapshots) < 2 {
		reasons = append(reasons, "No review comparison available yet.")
	}
	if len(data.proofpacks) == 0 {
		reasons = append(reasons, "No proofpack history exists yet.")
	}
	controlMissing := 0
	vendorMissing := 0
	for _, it := range items {
		if len(it.ControlRefs) == 0 {
			controlMissing++
		}
		if len(it.VendorRefs) == 0 {
			vendorMissing++
		}
	}
	if controlMissing > 0 {
		reasons = append(reasons, "Some evidence lacks explicit control mappings.")
	}
	if vendorMissing > 0 {
		reasons = append(reasons, "Some evidence lacks explicit vendor mappings.")
	}
	for _, pack := range data.proofpacks {
		if len(pack.EvidenceIDs) == 0 {
			reasons = append(reasons, "At least one historical proofpack lacks a persisted evidence manifest.")
			break
		}
	}
	return reasons
}

func graphHealthScore(now time.Time, items []evidence.Item, data graphData) int {
	score := 100
	missingControlLinks := 0
	missingVendorLinks := 0
	for _, it := range items {
		if it.Status == "expired" {
			score -= 14
		}
		if isOwnerless(it) {
			score -= 10
		}
		if isStale(it, now) {
			score -= 8
		}
		if len(it.ControlRefs) == 0 {
			missingControlLinks++
		}
		if len(it.VendorRefs) == 0 {
			missingVendorLinks++
		}
	}
	score -= missingControlLinks * 4
	score -= missingVendorLinks * 2
	if len(data.reviewSnapshots) == 0 {
		score -= 12
	} else {
		latest := data.reviewSnapshots[0]
		score -= min(latest.UnresolvedIssues*3, 18)
		if now.Sub(latest.LastReviewedAt) > 9*24*time.Hour {
			score -= 10
		}
	}
	if len(data.proofpacks) == 0 {
		score -= 8
	}
	if len(data.reviewSnapshots) < 2 {
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

func comparisonState(latest, previous persistence.ReviewSnapshot) string {
	healthDelta := latest.HealthScore - previous.HealthScore
	unresolvedDelta := latest.UnresolvedIssues - previous.UnresolvedIssues
	if healthDelta > 2 && unresolvedDelta <= 0 {
		return "active"
	}
	if healthDelta < -2 || unresolvedDelta > 0 {
		return "unresolved"
	}
	return "active"
}

func evidenceStatus(it evidence.Item, now time.Time) string {
	if it.Status == "expired" {
		return "expired"
	}
	if it.Status == "missing" {
		return "missing"
	}
	if isOwnerless(it) {
		return "ownerless"
	}
	if isStale(it, now) {
		return "stale"
	}
	if it.Status == "" {
		return "degraded"
	}
	return it.Status
}

func riskStatus(it evidence.Item, now time.Time) string {
	status := evidenceStatus(it, now)
	if status == "active" {
		return "active"
	}
	return "unresolved"
}

func reviewSnapshotStatus(s persistence.ReviewSnapshot) string {
	if s.ExpiredEvidence > 0 {
		return "expired"
	}
	if s.MissingOwners > 0 {
		return "ownerless"
	}
	if s.StaleEvidence > 0 {
		return "stale"
	}
	if s.UnresolvedIssues > 0 {
		return "unresolved"
	}
	return "active"
}

func evidenceSummary(it evidence.Item) string {
	out := it.Category + " evidence."
	if it.OwnerEmail != "" || it.OwnerName != "" {
		out += " Owner assigned."
	}
	if it.ExpiryDate != nil {
		out += " Expires " + it.ExpiryDate.Format("2006-01-02") + "."
	}
	return out
}

func isOwnerless(it evidence.Item) bool {
	return strings.TrimSpace(it.OwnerEmail) == "" && strings.TrimSpace(it.OwnerName) == ""
}

func isStale(it evidence.Item, now time.Time) bool {
	if it.UpdatedAt.IsZero() {
		return false
	}
	return it.UpdatedAt.Before(now.AddDate(0, 0, -180))
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

func sortNodes(nodes []Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Type != nodes[j].Type {
			return nodes[i].Type < nodes[j].Type
		}
		if severityRank(nodes[i].Status) != severityRank(nodes[j].Status) {
			return severityRank(nodes[i].Status) < severityRank(nodes[j].Status)
		}
		if nodes[i].Label != nodes[j].Label {
			return nodes[i].Label < nodes[j].Label
		}
		return nodes[i].ID < nodes[j].ID
	})
}

func sortEdges(edges []Edge) {
	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].Type != edges[j].Type {
			return edges[i].Type < edges[j].Type
		}
		if edges[i].SourceID != edges[j].SourceID {
			return edges[i].SourceID < edges[j].SourceID
		}
		if edges[i].TargetID != edges[j].TargetID {
			return edges[i].TargetID < edges[j].TargetID
		}
		return edges[i].ID < edges[j].ID
	})
}

func sortActions(actions []NextAction) {
	sort.SliceStable(actions, func(i, j int) bool {
		if severityRank(actions[i].Severity) != severityRank(actions[j].Severity) {
			return severityRank(actions[i].Severity) < severityRank(actions[j].Severity)
		}
		if actions[i].Impact != actions[j].Impact {
			return actions[i].Impact > actions[j].Impact
		}
		if actions[i].Title != actions[j].Title {
			return actions[i].Title < actions[j].Title
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

func tenantNodeID(tenantID string) string { return "tenant:" + slug(tenantID) }
func evidenceNodeID(id string) string    { return "evidence:" + slug(id) }
func proofpackNodeID(id string) string   { return "proofpack:" + slug(id) }
func reviewReportNodeID(id string) string {
	return "review_report:" + slug(id)
}
func typedNodeID(typ, label string) string { return typ + ":" + slug(label) }
func actionNodeID(actionID string) string  { return "action:" + slug(actionID) }
func actionID(typ, target string) string   { return typ + ":" + slug(target) }
func edgeID(typ, source, target string) string {
	return "edge:" + slug(typ) + ":" + slug(source) + ":" + slug(target)
}

func snapshotNodeForReview(s persistence.ReviewSnapshot, index int) string {
	return fmt.Sprintf("snapshot:review:%s:%d", s.LastReviewedAt.Format("20060102T150405"), index)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
