# EvidenceVault Roadmap

## Phase 1: Core Operations & Evidence Graph (Completed)

- Manual file-based persistence (tenant isolation, explainability).
- Evidence tracking (categories, ownership, temporal drift).
- Review rituals and snapshot generation.
- Proofpack exports (CSV, MD, TXT).
- **Evidence Graph Engine**: Deterministic relationship layer deriving operational intelligence (readiness scoring, automated next actions, degraded state management). Calm, table-first UI.

## Phase 2: Pilot Conversion & Extensibility (Current Focus)

- Webhooks for external integration.
- Read-only API tokens for tenant export automation.
- Jira / Linear integrations (generating tickets from Graph Next Actions).
- Graph expansion: Introduce "Remediation" and "Exception" nodes to handle risk acceptance workflows.
- Improved starter templates (e.g., SOC2 fast-start).

## Phase 3: Scale & Resilience (Future)

- Transition `persistence.Store` to a relational backing store (Postgres) while preserving the identical domain boundaries and file-based exportability.
- Multi-region redundancy.
- Advanced tenant lifecycle management (archival, automated tenant restoration).
