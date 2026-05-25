# Security

- Development mode (`APP_ENV=development`) allows header-based tenant/user auth for local testing only.
- Production requires either:
  - signed session cookie (`SESSION_SECRET`), or
  - API key mapped through `API_KEYS` (`key:tenant_id,key2:tenant2`).
- Tenant scope is enforced in service queries by `tenant_id` filters.
- Stripe webhooks are signature-verified and idempotent (`stripe_events`).

Limitations:
- No RBAC yet beyond tenant scoping and auth boundary.
