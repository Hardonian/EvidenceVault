# EvidenceVault

EvidenceVault is a deterministic compliance-operations utility for small teams running evidence hygiene and weekly operational reviews.

## What is implemented now
- Tenant-scoped evidence + file attachment workflows.
- Deterministic weekly review snapshots persisted as canonical history.
- Deterministic review comparison (latest vs previous) with stable/improving/degrading state.
- Operational narratives generated only from persisted/current truth.
- Export routes:
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
