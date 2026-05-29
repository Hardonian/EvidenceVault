# Evidence Graph

Evidence Graph is the server-generated operating layer that connects tenant evidence to persisted owners, explicit vendor/control/risk mappings, proofpack exports, operational events, review history, deterministic narratives, comparison state, pilot readiness, exports, and next actions.

It is deterministic and source-backed. It does not certify compliance, provide legal advice, infer unverifiable relationships, or use an LLM for core graph construction.

## Routes

- `/app/evidence-graph` - table-first operator page.
- `/app/api/evidence-graph` - canonical graph JSON.
- `/app/export/evidence-graph.md` - Markdown export.
- `/app/export/evidence-graph.txt` - plain text export.
- `/app/export/evidence-graph.json` - canonical graph JSON export.

## Graph Model

Graph output:

```json
{
  "tenantId": "tenant-id",
  "generatedAt": "timestamp",
  "graphVersion": "evidence-graph/v1",
  "nodes": [],
  "edges": [],
  "summary": {},
  "degradedReasons": [],
  "nextActions": []
}
```

Supported node types:

- `tenant`
- `evidence`
- `owner`
- `vendor`
- `control`
- `risk`
- `proofpack`
- `operational_event`
- `snapshot`
- `review_report`
- `review_comparison`
- `narrative`
- `pilot_readiness`
- `export`
- `action`

Supported edge types:

- `OWNS`
- `SUPPORTS_CONTROL`
- `RELATES_TO_VENDOR`
- `MITIGATES_RISK`
- `INCLUDED_IN_PROOFPACK`
- `GENERATED_EVENT`
- `APPEARS_IN_SNAPSHOT`
- `REVIEWED_IN`
- `COMPARED_WITH`
- `PRODUCED_NARRATIVE`
- `CONTRIBUTES_TO_PILOT_READINESS`
- `REQUIRES_ACTION`
- `BLOCKED_BY`
- `STALE_BECAUSE`
- `EXPIRED_BECAUSE`
- `OWNERLESS_BECAUSE`
- `MISSING_EVIDENCE_FOR`
- `READY_FOR_EXPORT`
- `NOT_COMPARABLE_YET`

Every node carries `id`, `type`, `label`, `tenantId`, `status`, timestamps, `summary`, and `metadata`.

Every edge carries `id`, `type`, `sourceId`, `targetId`, `tenantId`, `reason`, `evidenceSource`, `confidence`, `status`, timestamp, and `metadata`.

## Source Rules

- Owner edges come only from `evidence_items.owner_name` or `evidence_items.owner_email`.
- Control/vendor/risk edges come only from explicit `control_refs`, `vendor_refs`, and `risk_refs` fields on evidence records.
- Proofpack inclusion edges come only from persisted proofpack `evidence_ids`. Older proofpacks without manifests are marked degraded.
- Review, snapshot, event, report, comparison, and narrative nodes derive from persisted operational state and deterministic operations summaries.
- Missing links become degraded reasons and next actions; they are not silently inferred.

## Scoring

Graph health score is deterministic and bounded from 0 to 100.

Starting score: `100`.

Penalties:

- `14` per expired evidence item.
- `40` when no evidence exists yet.
- `10` per ownerless evidence item.
- `8` per stale evidence item (`updated_at` older than 180 days).
- `4` per evidence item without an explicit control mapping.
- `2` per evidence item without an explicit vendor mapping.
- `12` when no review snapshot exists.
- Up to `18` for unresolved issues in the latest review (`3` per issue).
- `10` when latest review is older than 9 days.
- `8` when no proofpack exists.
- `6` when fewer than two reviews exist and comparison is not available.

The score is clamped to `0..100`. It is an operational hygiene score, not a compliance certification or audit-readiness guarantee.

## Degraded States

Normal degraded states return a graph instead of a hard 500:

- Empty tenant: first-run guidance and actions.
- Ownerless evidence: owner gap edges and assign-owner actions.
- Expired evidence: expired edges and replacement actions.
- Stale evidence: stale edges and refresh actions.
- Missing mappings: degraded reasons and mapping actions.
- No review history: review action and not-comparable state.
- No proofpacks: proofpack action.
- Historical proofpacks without evidence manifests: degraded proofpack state.

## Tenant Safety

Graph generation requires a tenant id at the boundary. Routes obtain tenant context through the existing auth layer and pass only that tenant id into the builder. The builder reads tenant-local maps from persistence and never scans or joins across tenants. Every node, edge, export, and next action carries the same tenant id.

Graph metadata does not include environment variables or secrets. File evidence is represented as `has_source_file` instead of exporting raw file paths.

## Verification

Attempt the repo verification commands:

- `go mod tidy`
- `make fmt` or `test -z "$(gofmt -l ./cmd ./internal)"`
- `go vet ./...`
- `go test ./...`
- `go build ./cmd/server`
- `make smoke`
