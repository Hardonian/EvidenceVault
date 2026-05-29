# PILOT

## Objective
Run one tenant for 4 weeks with deterministic weekly review continuity and portable exports.

## Weekly operating loop
- Run weekly snapshot (`POST /app/reviews`).
- Review dashboard ritual state (current week, next action, missed cadence warning).
- Export pilot proof (`/app/export/pilot-proof.md`) for founder check-ins.
- Review Evidence Graph (`/app/evidence-graph`) for linked evidence, owner gaps, mappings, proofpack history, comparison state, and next actions.
- Export graph proof (`/app/export/evidence-graph.md`, `.txt`, or `.json`) when buyers or internal operators need a readable source-backed map.
- Latest-vs-previous comparison remains default continuity proof.
- Pilot week progression is review-count based, not calendar-enforced.

## Success criteria
- >=4 persisted weekly reviews.
- comparison export available.
- continuity dependence forming from historical truth (not live-only status).
- no analytics SDKs, no AI summaries, no hidden scoring.
- graph edges remain source-backed; missing mappings are shown as gaps, not inferred.
