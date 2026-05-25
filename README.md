# EvidenceVault

## Commands
- `go mod tidy`
- `make fmt`
- `make vet`
- `make test`
- `make build`
- `make smoke`

## Routes
- GET `/healthz`
- GET `/readyz`
- GET `/version`
- GET `/`
- GET `/app`
- GET `/app/evidence`
- POST `/app/evidence`
- POST `/app/evidence/upload`
- GET `/app/proofpacks`
- POST `/app/proofpacks`
- POST `/api/cron/reminders`
- POST `/billing/checkout`
- POST `/billing/portal`
- POST `/webhooks/stripe`
