#!/bin/sh
set -eu
: "${CRON_SECRET:=pilot-secret}"
: "${BASE_URL:=http://127.0.0.1:8080}"
TENANT=${TENANT:-pilot-tenant}
USER_ID=${USER_ID:-pilot-user}
export APP_ENV=${APP_ENV:-development}
export PERSISTENCE_MODE=${PERSISTENCE_MODE:-file}
export DATA_DIR=${DATA_DIR:-./data}
go run ./cmd/server >/tmp/evidencevault-pilot-flow.log 2>&1 &
PID=$!
trap 'kill "$PID" 2>/dev/null || true' EXIT INT TERM
sleep 2
curl -fsS "$BASE_URL/healthz" >/dev/null
ID=$(curl -fsS -X POST "$BASE_URL/app/evidence" -H "X-Tenant-ID: $TENANT" -H "X-User-ID: $USER_ID" -H 'Content-Type: application/json' -d '{"title":"Pilot Insurance","category":"Compliance","status":"active","owner_email":"ops@example.com","reminder_days_before":30}' | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
printf 'pilot file' >/tmp/pilot-evidence.txt
curl -fsS -X POST "$BASE_URL/app/evidence/upload" -H "X-Tenant-ID: $TENANT" -H "X-User-ID: $USER_ID" -F "evidence_id=$ID" -F "file=@/tmp/pilot-evidence.txt;type=text/plain" >/dev/null
curl -fsS -X POST "$BASE_URL/app/proofpacks" -H "X-Tenant-ID: $TENANT" -H "X-User-ID: $USER_ID" >/dev/null
curl -fsS "$BASE_URL/app" -H "X-Tenant-ID: $TENANT" -H "X-User-ID: $USER_ID" >/dev/null
curl -fsS "$BASE_URL/healthz" >/dev/null
printf 'Pilot flow completed successfully.\n'
