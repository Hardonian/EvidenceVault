# Deployment
Supported targets: Fly.io, Render, Railway, Cloud Run.

## Required env vars
`DATABASE_URL`, `CRON_SECRET`, `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, `STRIPE_PRICE_ID`, `BASE_URL`.

Deploy container built from `Dockerfile`. Ensure migrations run before traffic.
