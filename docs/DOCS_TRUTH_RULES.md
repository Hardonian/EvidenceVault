# Documentation Truth Rules

Purpose: Prevent documentation drift and capability theater.
Audience: Contributors and maintainers.
Canonical status: Policy source for documentation integrity.

## Rules
1. Label every major statement as implemented, staged, exploratory, or deferred when ambiguity exists.
2. No speculative capability claims in canonical docs.
3. Degraded states must be explicit; never describe degraded behavior as healthy.
4. Architecture docs must match executable paths/services/routes in code.
5. One canonical document per major topic; supporting docs must link to canonical doc.
6. Internal operations material belongs under `docs/internal/`; public/evaluator narrative stays outside it.
7. Superseded docs move to `docs/archive/` with no silent loss of institutional knowledge.

## Canonical Ownership
- Product/repo truth: `README.md`
- Founder workflow and value loop: `docs/founder/FOUNDER_GUIDE.md`
- Operator behavior and cadence: `docs/operator/OPERATOR_MANUAL.md`
- Setup/onboarding sequence: `docs/operator/ONBOARDING.md`
- Architecture truth: `docs/architecture/SYSTEM_ARCHITECTURE.md`
- Pilot semantics: `docs/product/PILOT_MODEL.md`
- Delivery planning: `docs/internal/ROADMAP.md`

## Terminology Standard
- Use **app workspace** (not dashboard/console variants).
- Use **review snapshot** for weekly persisted review.
- Use **continuity history** for append-only institutional memory.
- Use **export** for narratives/comparisons/proof outputs; avoid mixed “bundle/proofpack” unless route/model specifically uses proofpack.
- Use **founder surface** vs **operator surface** to distinguish audience.
