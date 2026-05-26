# PILOT

## Objective
Run one tenant for 4 weeks with deterministic weekly review continuity and portable exports.

## Weekly operating loop
- Run weekly snapshot (`POST /app/reviews`).
- Review dashboard ritual state (current week, next action, missed cadence warning).
- Export narratives/comparison for founder check-ins.

## Success criteria
- >=4 persisted weekly reviews.
- comparison export available.
- continuity dependence forming from historical truth (not live-only status).
