# Evidence Graph

The **Evidence Graph** is the canonical operational relationship layer for EvidenceVault.

It transforms isolated evidence artifacts, ownership states, and temporal drift into a deterministic, queryable directed acyclic graph (DAG) representing the organization's true compliance posture and operational readiness.

## Core Philosophy

- **Operator Truth First:** The graph is derived exclusively from persisted application state (the "source of truth"). It is deterministic, reproducible, and explainable.
- **No Theatre:** We do not visualize the graph with physics-based canvas widgets (D3, force-directed graphs) because compliance is an operational discipline, not a data visualization exercise. The graph is presented via calm, table-first UIs.
- **Tenant Safety:** Graph construction and export are strictly tenant-scoped.
- **Continuity > NovelTY:** The graph extends the existing operational paradigm (snapshots, proofpacks, evidence). It does not fork reality.

## Components

The graph system (`internal/graph`) consists of:

1. **Domain Model (`model.go`)**
   - Canonical nodes (`evidence`, `owner`, `control`, `risk`, `snapshot`, `proofpack`).
   - Canonical edges (`belongs_to`, `owned_by`, `mitigates`, `threatens`, `review_lineage`).
   - Strong determinism (sorted arrays, predictable ID generation: `type:tenant:naturalkey`).
2. **Builder Engine (`builder.go`)**
   - Reconstructs the graph on demand from the `persistence.Store`.
   - Favors freshness and correctness over caching.
3. **Readiness Intelligence (`readiness.go`)**
   - Multi-dimensional scoring (Coverage, Continuity, Verification, Pilot Readiness).
   - Temporal drift detection (cadence deterioration, persistent gaps).
4. **Next-Action Engine (`actions.go`)**
   - Priority-ranked deterministic actions.
   - Converts abstract graph insights into specific, routable operator workflows (e.g., "Assign owner to X", "Renew Y").
5. **Export Subsystem (`export.go`)**
   - Supports Markdown, Text, and JSON.
   - Built for founder trust, buyer confidence, and auditor verification.

## Usage

The graph is accessible via the dashboard (`/app`), full explorer (`/app/graph`), API (`/api/graph`), and exports (`/app/export/graph.json`).
