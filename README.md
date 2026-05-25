# EvidenceVault
Upload, track, remind, export.

EvidenceVault is an SMB compliance operations micro-service for ecommerce brands, agencies, and small regulated teams. It provides evidence CRUD, file uploads, reminders, JSON proofpack export, and Stripe billing.

## Quick start
1. Copy `.env.example` to `.env` and set secrets.
2. Start Postgres: `docker compose up db -d`
3. Run migration `psql "$DATABASE_URL" -f migrations/0001_init.sql`
4. Start app: `go run ./cmd/server`
5. Open `http://localhost:8080`.

## Verification
Run:
- `gofmt -w ./cmd ./internal`
- `go vet ./...`
- `go test ./...`
- `go build ./cmd/server`
- `make smoke`
