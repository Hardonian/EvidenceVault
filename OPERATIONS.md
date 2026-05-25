# Operations
- `/healthz` liveness.
- `/readyz` readiness.
- `/version` release identity.
- Run reminders using `POST /internal/reminders/run` with `Authorization: Bearer $CRON_SECRET`.
