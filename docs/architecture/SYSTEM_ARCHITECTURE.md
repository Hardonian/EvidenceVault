# System Architecture

Purpose: Deterministic architecture reference tied to implemented behavior.
Audience: Contributors/operators/founders needing technical truth.
Canonical status: Canonical architecture doc.

## Deterministic Continuity Model
- Weekly review snapshots are persisted and used for historical comparisons.
- Narratives and comparisons are derived from persisted continuity history + current state.

## Persistence and Truth
- Memory mode: degraded/ephemeral.
- File mode: pilot-durable single-instance continuity.
- Production memory mode requires explicit override.

## Export Architecture
- Direct export routes serve narratives and review comparison artifacts.
- Proofpack generation persists payload and includes evidence/reminder/audit context.

## Append-Only Operational Truth
Operational continuity relies on append-style history retention for review artifacts and related timeline context.

## Explicit Non-Architecture Claims
No hidden analytics engine and no AI inference pipeline in current implementation.
