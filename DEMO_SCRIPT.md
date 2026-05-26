# DEMO SCRIPT

1. Open `/app` with pilot tenant.
2. Show pilot ritual state: week, last review date, next action.
3. Verify empty-state readability when no reviews exist yet (next action should be first snapshot).
4. Trigger/observe deterministic seeded 4-week progression and week-4 readiness.
5. Export `/app/export/pilot-proof.md` (includes narrative + comparison readiness + next action).
6. Export `/app/export/review-comparison.md` (latest-vs-previous default).
7. Open `/app/evidence-graph` and export `/app/export/evidence-graph.md` to show linked evidence, explicit mappings, degraded states, and next actions.

Demo seed includes explicit evidence examples for owners, vendors, controls, risks, proofpacks, review progression, expired evidence, stale evidence, ownerless evidence, and pilot readiness. Demo seed remains blocked in production.
7. Explain deterministic retention loop: weekly review -> persisted memory -> historical comparison -> portable proof (no analytics SDK, no AI summaries, no hidden scoring).
