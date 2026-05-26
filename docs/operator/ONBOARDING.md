# Operator Onboarding

Purpose: First-run path from empty tenant to first continuity cycle.
Audience: New operators.
Canonical status: Canonical onboarding procedure.

## First Run Setup
1. Start application (`DEPLOYMENT.md`).
2. Open app workspace at `/app`.
3. Confirm tenant context/auth headers for API operations.

## First 3 Records
Create at least three evidence records with owners and expected renewal context.

## First Review Snapshot
Run `POST /app/reviews` after initial records are present.

## Understand Urgency Ordering
Use review outputs to prioritize expired/missing-owner and unresolved operational items first.

## Continuity Accumulation
Continuity only compounds when weekly snapshots are persisted and compared over time.
