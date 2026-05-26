# Operator Manual

Purpose: Daily/weekly operating procedure for deterministic continuity.
Audience: Operators.
Canonical status: Canonical operating doc.

## Daily Expectations
- Keep evidence records current.
- Attach source files to records where relevant.
- Resolve obvious expirations/missing ownership.

## Weekly Review Cadence
1. Trigger review snapshot (`POST /app/reviews`).
2. Check latest vs previous comparison state.
3. Review narratives for continuity and degradation signals.
4. Export reports for internal review.

## Degraded States
- Ephemeral memory persistence mode: non-durable continuity risk.
- Storage unavailable: file upload fails explicitly.
- Email adapter failure: reminder send logged as failed.
- Billing integration unavailable: billing routes return explicit errors.

## Exports and Proof
Use app export routes each week; use proofpack workflow for bundled evidence/proof outputs.

## Related
- `docs/operator/ONBOARDING.md`
- `docs/product/PILOT_MODEL.md`
- `docs/architecture/SYSTEM_ARCHITECTURE.md`
