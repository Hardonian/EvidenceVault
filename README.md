# EvidenceVault

<!-- BEGIN: REPO HERO -->
![EvidenceVault — hero generated locally on the GPU stack](assets/repo-hero.png)
<!-- END: REPO HERO -->

EvidenceVault is a deterministic compliance-operations system for small teams that need auditable continuity, weekly review discipline, and portable proof exports.

Purpose: Top-level product and repository truth.
Audience: Founders, operators, contributors, evaluators.
Canonical status: Authoritative for what is currently implemented in this repository.

## Implemented Product Truth
- Tenant-scoped evidence lifecycle (create/list/update) with file attachment flow.
- Deterministic weekly review snapshots persisted as historical truth.
- Deterministic review comparison (latest vs previous) with stable/improving/degrading state.
- Operational narratives generated only from persisted/current truth.
- Evidence Graph generated from tenant-scoped persisted records, explicit mappings, operational history, proofpacks, review comparisons, pilot readiness, and deterministic next actions.
- Export routes:
  - `/app/api/evidence-graph`
  - `/app/evidence-graph`
  - `/app/export/evidence-graph.md`
  - `/app/export/evidence-graph.txt`
  - `/app/export/evidence-graph.json`
  - `/app/export/narratives.md`
  - `/app/export/review-comparison.md`
  - `/app/export/review-comparison.txt`
  - `/app/export/pilot-proof.md`
  - `/app/export/pilot-proof.txt`
- 4-week pilot ritual state (week, cadence, next action, export readiness).
- Proofpack generation and persisted payload for support/share workflows.

## Explicitly Not Implemented
- AI-generated recommendations.
- Analytics SDK or user tracking pipelines.
- Compliance certification/legal advice.
- Hidden scoring or inference outside persisted operational history.

## Evidence Graph
- Server-generated and tenant-scoped; no client-side tenant trust.
- Edges include source, reason, confidence, status, and timestamp.
- Control, vendor, and risk links require explicit persisted evidence mappings.
- Empty/degraded states return first-run guidance, degraded reasons, and next actions.
- See `docs/evidence-graph.md`.

## 4-week pilot ritual
1. Week 1: generate first review snapshot.
2. Week 2: continue weekly review and compare latest vs previous.
3. Week 3: continue weekly review and track recurring friction.
4. Week 4: export review comparison + narratives for founder conversion proof.
5. Pilot week is review-count-based (persisted snapshots), not calendar-enforced automation.

## Verification
- `go mod tidy`
- `test -z "$(gofmt -l .)"`
- `go vet ./...`
- `go test ./...`
- `go build ./cmd/server`
- `make smoke`

## Architecture Summary
- Canonical truth is persisted history and append-only review continuity.
- Comparison/narrative outputs depend on historical state, not transient UI memory.
- Degraded modes are explicit (for example ephemeral memory persistence, unavailable storage/email/billing integrations).

## Pilot and Demo Entry Points
- Founder path: `docs/founder/FOUNDER_GUIDE.md`
- Operator first run: `docs/operator/ONBOARDING.md`
- Pilot mechanics: `docs/product/PILOT_MODEL.md`
- Demo walkthrough: `docs/pilot/DEMO_WALKTHROUGH.md`
- Full docs map: `docs/README.md`

## Deterministic Truth Policy
- Documentation must distinguish implemented vs planned behavior.
- Docs must not claim capability that code does not provide.
- Internal operating guidance is separated from outward-facing product framing.
- Source-of-truth ownership is defined in `docs/DOCS_TRUTH_RULES.md`.
