# Support Playbook
## Triage
1. Check /healthz and /readyz.
2. Verify persistence mode and data dir access.
3. Validate tenant/user headers.
4. Reproduce with scripts/pilot_flow.sh.

## Known limitations
- No compliance certification.
- No legal advice.
- File mode is single-node.

## Upgrade path
Move from memory/file to Postgres in future release while keeping API contracts stable.
