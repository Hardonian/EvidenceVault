# Security
- Fail closed on CRON endpoint via `CRON_SECRET` bearer token.
- Tenant isolation by mandatory `tenant_id` query scope in data access.
- Stripe webhook signature verification and event-id idempotency table.
- Do not log secrets or full cardholder data.
