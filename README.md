# EvidenceVault

EvidenceVault is a deterministic compliance-operations utility for small teams running evidence hygiene and weekly operational reviews.

## What is implemented now
- Tenant-scoped evidence + file attachment workflows.
- Deterministic weekly review snapshots persisted as canonical history.
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

## Intentionally not implemented
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
