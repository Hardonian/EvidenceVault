# DEMO SCRIPT

1. Open `/app` with pilot tenant.
2. Show pilot ritual state: week, last review date, next action.
3. Verify empty-state readability when no reviews exist yet (next action should be first snapshot).
4. Trigger/observe deterministic seeded 4-week progression and week-4 readiness.
5. Export `/app/export/pilot-proof.md` (includes narrative + comparison readiness + next action).
6. Export `/app/export/review-comparison.md` (latest-vs-previous default).
7. Explain deterministic retention loop: weekly review -> persisted memory -> historical comparison -> portable proof (no analytics SDK, no AI summaries, no hidden scoring).
