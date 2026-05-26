# Security at EvidenceVault

## Core Principles

1. **Tenant Isolation:** EvidenceVault employs strict logical boundaries between tenants. All operational routes, graph construction layers, and persistence mechanisms require validated tenant scope. A failure to identify a tenant results in immediate rejection (fail-closed).
2. **Operator Truth First:** We do not rewrite history. Actions are additive (events, snapshots). The Evidence Graph is deterministically reconstructed from raw data to ensure auditability and prevent "graph theatre" or hallucinated compliance.
3. **No External Intelligence Hooks:** The system does not transmit tenant evidence to external Large Language Models or third-party inference services. All intelligence (readiness scoring, next actions) is computed locally, deterministically, and explainably.
4. **Degraded State Safety:** The application must remain operational even if subsystems fail or persistence modes shift. Graph computations that fail must expose the exact degraded reasons rather than masking the error.

## Reporting a Vulnerability

If you discover a security issue or isolation flaw, please report it immediately to <security@example.com> rather than opening a public issue. We will respond within 48 hours.

## Architecture Security Notes

- Persistence relies on single-file JSON maps per entity (in FileStore mode). File system permissions must strictly protect the `data/` directory.
- The Evidence Graph API route (`/app/api/evidence-graph`) does not execute arbitrary queries. The graph engine enforces strict predefined traversal rules.
