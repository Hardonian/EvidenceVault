# Operational Memory
EvidenceVault persists deterministic operational snapshots and review reports to preserve historical operational truth.

- Daily snapshot generation (bounded 90-day retention)
- Tenant-local trend windows (7/30 day)
- Append-only timeline events for continuity signals
- Evidence Graph rebuilds tenant-local links from persisted evidence, explicit mappings, proofpacks, events, snapshots, reports, comparisons, and pilot-readiness state
- Missing graph inputs are represented as degraded reasons and next actions
- No AI summaries, no external analytics SDKs
