# Contributing (Documentation + Implementation Truth)

Purpose: Contributor guardrails for truth-aligned changes.
Audience: Contributors.
Canonical status: Reference doc.

- Read canonical docs in `docs/README.md` before adding new documentation.
- Prefer updating canonical docs over creating parallel markdown.
- If replacing a doc, move prior version to `docs/archive/`.
- Keep implemented/planned boundaries explicit.
- Run verification before merge:
  - `go test ./...`
  - `go build ./cmd/server`
  - `make smoke`
