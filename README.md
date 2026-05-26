# EvidenceVault

EvidenceVault is a deterministic compliance-operations system for small teams that need auditable continuity, weekly review discipline, and portable proof exports.

Purpose: Top-level product and repository truth.
Audience: Founders, operators, contributors, evaluators.
Canonical status: Authoritative for what is currently implemented in this repository.

## Implemented Product Truth
- Tenant-scoped evidence lifecycle (create/list/update) with file attachment flow.
- Deterministic weekly review snapshots persisted as historical truth.
- Deterministic review comparison (latest vs previous) with stable/improving/degrading state.
- Operational narratives generated from persisted/current data only.
- Export surfaces:
  - `/app/export/narratives.md`
  - `/app/export/review-comparison.md`
  - `/app/export/review-comparison.txt`
- 4-week pilot ritual state derived from review continuity.
- Proofpack generation and persisted payload for support/share workflows.

## Explicitly Not Implemented
- AI-generated recommendations.
- Hidden analytics pipelines or user-tracking SDK claims.
- Compliance certification workflows or legal advice.

## Quickstart
1. Start the app locally (see `DEPLOYMENT.md`).
2. Create or select a tenant and open `/app`.
3. Create evidence records and attach files.
4. Run weekly review snapshots (`POST /app/reviews`).
5. Export narratives/comparison and proof materials from `/app/export/*`.

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
