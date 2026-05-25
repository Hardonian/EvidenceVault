# EvidenceVault

EvidenceVault is a compliance-operations utility. It helps teams track evidence, renewal reminders, audit history, and proofpack exports. It does **not** certify compliance or provide legal advice.

## Implemented now
- Tenant-scoped evidence creation/list/update.
- Free-tier write limit enforced in evidence service.
- Evidence file upload requires `evidence_id` and creates `evidence_files` + updates `evidence_items.source_file_path`.
- Reminder run uses evidence expiry data, logs `sent`/`failed`, and is idempotent per evidence/day/channel.
- Proofpack generation persists payload and includes tenant, evidence, files, reminders, audit summary, generated timestamp, app version, and limitations statement.
- Billing checkout/portal creation and stripe webhook processing write audit events.
- `/app` uses persisted evidence and proofpack state.

## Intentionally not implemented
- Compliance certification workflows.
- Legal interpretation of evidence quality.
- Multi-channel reminder transports beyond email adapter.

## Degraded modes
- Storage unavailable: upload route returns HTTP 503 with explicit message.
- Email adapter failure: reminder is logged as `failed` (not `sent`).
- Stripe not configured/unavailable: billing routes return explicit errors.

## Verification commands
- `go mod tidy`
- `gofmt -w ./...`
- `go vet ./...`
- `go test ./...`
- `go build ./cmd/server`
- `make smoke`

## Pilot workflow
1. Create dev tenant and auth headers.
2. Create evidence (`POST /app/evidence`).
3. Upload file with `evidence_id` (`POST /app/evidence/upload`).
4. Run reminders (`POST /api/cron/reminders`).
5. Generate proofpack (`POST /app/proofpacks`).
6. Open `/app` for operational state.

## Persistence Modes
- Default build remains zero-dependency and uses `PERSISTENCE_MODE=memory` (ephemeral/degraded).
- Pilot durable mode: `PERSISTENCE_MODE=file` with `DATA_DIR` writable; single-instance only.
- Production fail-closed: memory mode in production requires `ALLOW_EPHEMERAL_PRODUCTION=true`.
- Cloud Run caveat: file mode requires a writable mounted volume/path.
- Postgres adapter is roadmap hardening, not active default.
- Verify: `go mod tidy && test -z "$(gofmt -l .)" && go vet ./... && go test ./... && go build ./cmd/server && make smoke`.
\n## Pilot truth update\n- Zero external Go dependencies.\n- Persistence modes: memory (degraded) and file (pilot durable).\n- No compliance certification or legal advice.\n- No AI claims.\n- Production fail-closed if persistence is memory unless explicitly overridden.\n

## Operational continuity memory
EvidenceVault emphasizes deterministic operational memory, recurring reviews, and historical truth without AI or analytics theatre.

## Institutional Memory and Operational Continuity
EvidenceVault is evolving into an operational continuity and institutional memory system. New deterministic layers include operational narratives, review comparisons, continuity heuristics, and founder conversion intelligence derived from persisted history only.
